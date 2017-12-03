// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"errors"
	"crypto/tls"
	"crypto/x509"
  	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/baobabus/go-apns/http2x"
	"golang.org/x/net/http2"
	"golang.org/x/net/idna"
)

// HTTP layer-related errors.
var (
	ErrHTTPClientClosed = errors.New("HTTPClient: attempt to close already closed client")
	ErrNoConnectionPool = errors.New("HTTPClient: no connection pool")
)

// HTTPClient wraps http.Client and augments it with HTTP/2 stream
// reservation facility.
//
// Due to current limitations of Go http2 client implementation, only a single
// underlying http2.ClientConn is intended to be supported by the client. This
// means that correct communication behavior is limited to a single HTTP/2
// server endpoint. Note, however, that no attempt is made to restrict the way
// in which the client is used, including handling of any encountered redirect
// responses.
type HTTPClient struct {
	http.Client

	addr    string
	precise bool
	pollInt time.Duration
	cfgCap uint32

	mu       sync.Mutex
	cond     *sync.Cond
	connPool http2.ClientConnPool
	actCap   uint32
	effCap   uint32
	cnt      uint32
	closed   bool

	tkr    *time.Ticker
	ctl    chan struct{}

	initOnce sync.Once
}

// NewHTTPClient creates a new HTTPClient for handling HTTP requests
// to a single specified gateway.
// TLS client certificate cCert and custom root certificate authority rootCA
// certificate are optional and can be nil.
func NewHTTPClient(gateway string, commsCfg CommsCfg, cCert *tls.Certificate, rootCA *tls.Certificate) (*HTTPClient, error) {
	t := &http2.Transport{
		DialTLS: makeDialer(commsCfg),
		DisableCompression: true, // As per Apple spec
	}
	tlsConfig := t.TLSClientConfig
	if cCert != nil {
		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}
		tlsConfig.Certificates = []tls.Certificate{*cCert}
		if len(cCert.Certificate) > 0 {
			tlsConfig.BuildNameToCertificate()
		}
	}
	if rootCA != nil && len(rootCA.Certificate[0]) > 0 {
		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}
		rCert, err := x509.ParseCertificate(rootCA.Certificate[0])
		if err != nil {
			return nil, err
		}
		certpool := x509.NewCertPool()
		certpool.AddCert(rCert)
		tlsConfig.RootCAs = certpool
  	}
	t.TLSClientConfig = tlsConfig
	url, _ := url.ParseRequestURI(gateway)
	res := &HTTPClient{
		Client: http.Client{
			Transport: t,
			Timeout:   commsCfg.RequestTimeout,
		},
		addr:    AuthorityAddr(url.Scheme, url.Host),
		precise: false,
		pollInt: 0,
		cfgCap:  1,
	}
	return res, nil
}

func (c *HTTPClient) init() {
	c.cond = sync.NewCond(&c.mu)
	c.effCap = 1 // assume just 1 until connection is open
	if c.precise || c.pollInt > 0 {
		c.connPool, _ = http2x.GetClientConnPool(c.Client.Transport)
		c.refreshCap()
	}
	if c.connPool != nil && c.pollInt > 0 {
		c.tkr = time.NewTicker(c.pollInt)
		c.ctl = make(chan struct{})
		go func() {
			select {
			case <-c.tkr.C:
				c.refreshCap()
			case <-c.ctl:
				return
			}
		}()
	}
}

func (c *HTTPClient) GetClientConn() (*http2.ClientConn, error) {
	c.initOnce.Do(c.init)
	if c.connPool == nil {
		// http2 incursion is disabled, so this it not an error
		return nil, nil
	}
	return http2x.GetClientConn(c.connPool, c.addr)
}

// ReservedStream returns a reserved HTTP2Stream in the client's 
// HTTP/2 connections, or a non-nil error
func (c *HTTPClient) ReservedStream(cancel func(<-chan struct{}) error) (*HTTP2Stream, error) {
	c.initOnce.Do(c.init)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.precise {
		c.refreshCapLocked()
	}
	var cerr error
	for cnlLaunched := false; c.effCap > 0 && c.cnt >= c.effCap && cerr == nil; {
		if !cnlLaunched && cancel != nil {
			done := make(chan struct{})
			defer close(done)
			go func() {
				if err := cancel(done); err != nil {
					// Must guard access to cerr.
					// Atomic store and load could be more efficient.
					c.mu.Lock()
					cerr = err
					c.mu.Unlock()
					c.cond.Broadcast()
				}
			}()
			cnlLaunched = true
		}
		c.cond.Wait()
	}
	if cerr != nil {
		return nil, cerr
	}
	// This may need to be its own error
	// if c.effCap == 0 {
	// 	return nil, ErrZeroCapacity
	// }
	c.cnt++
	// TODO Consider using sync.Pool for HTTP2Stream instances.
	return &HTTP2Stream{client: c}, nil
}

func (c *HTTPClient) Close() error {
	c.initOnce.Do(c.init)
	if c.closed {
		return ErrHTTPClientClosed
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tkr != nil {
		c.tkr.Stop()
		close(c.ctl)
	}
	// Client and everything underneath should be GC'd soon
	// and that should take care of closing any open connections.
	// Not sure if present http2 state of client can be trusted, though.
	if c.Client.Transport != nil {
		if t2, ok := c.Client.Transport.(*http2.Transport); ok {
			// All streams must be closed for the below to have any effect.
			t2.CloseIdleConnections()
		}
	}
	c.closed = true
	return nil
}

func (c *HTTPClient) release() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cnt > 0 {
		c.cnt--
		if c.cnt < c.effCap {
			c.cond.Broadcast()
		}
	}
}

func (c *HTTPClient) refreshCap() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refreshCapLocked()
}

func (c *HTTPClient) refreshCapLocked() {
	if c.connPool == nil {
		return
	}
	conn, err := http2x.GetClientConn(c.connPool, c.addr)
	if err != nil {
		return
	}
	c.actCap = http2x.GetMaxConcurrentStreams(conn)
	logTrace(0, "HTTClient", "Max streams = %d\n", c.actCap)
	v := c.actCap
	if v > c.cfgCap {
		v = c.cfgCap
	}
	notif := c.effCap < v || (c.effCap > 0 && v == 0)
	c.effCap = v
	if notif {
		c.cond.Broadcast()
	}
}

// HTTP2Stream is a token indicating a stream reservation in one
// of the HTTPClient's HTTP/2 onnections.
type HTTP2Stream struct {
	client *HTTPClient
}

// Close releases an HTTP/2 stream reservation.
func (s *HTTP2Stream) Close() {
	s.client.release()
}

// AuthorityAddr returns a given authority (a host/IP, or host:port / ip:port)
// and returns a host:port. The port 443 is added if needed.
func AuthorityAddr(scheme string, authority string) string {
	host, port, err := net.SplitHostPort(authority)
	if err != nil { // authority didn't have a port
		port = "443"
		if scheme == "http" {
			port = "80"
		}
		host = authority
	}
	if a, err := idna.ToASCII(host); err == nil {
		host = a
	}
	// IPv6 address literal, without a port:
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return host + ":" + port
	}
	return net.JoinHostPort(host, port)
}
