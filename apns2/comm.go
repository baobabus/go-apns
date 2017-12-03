// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"crypto/tls"
	"net"
	"time"
)

// CommsCfg is a set of parameters that govern communications with APN servers.
// Two baseline configuration sets are predefined by CommsFast and CommsSlow
// global variables. You may define your own sets as needed to address
// any specific requirements of your particular setup.
type CommsCfg struct {

	// DialTimeout is the maximum amount of time a dial will wait for a connect
	// to complete.
	DialTimeout time.Duration

	// RequestTimeout specifies a time limit for requests made by the
	// HTTPClient. The timeout includes connection time, any redirects,
	// and reading the response body.
	RequestTimeout time.Duration

	// KeepAlive specifies the keep-alive period for an active network
	// connection. If zero, keep-alives are not enabled.
	// Apple recommends not closing connections to APN service at all,
	// but a sinsibly long duration is acceptable.
	KeepAlive time.Duration

	// MaxConcurrentStreams is the maximum allowed number of concurrent streams
	// per HTTP/2 connection. If connection's MAX_CONCURRENT_STREAMS option
	// is invoked by the remote side with a lower value, the remote request
	// will be honored if possible.
	MaxConcurrentStreams uint32

}

// CommsFast is a baseline set of communication settings for situations where
// long delays cannot be tolerated.
var CommsFast = CommsCfg{
	DialTimeout:          20 * time.Second,
	RequestTimeout:       30 * time.Second,
	KeepAlive:            10 * time.Hour,
	MaxConcurrentStreams: 500,
}

// CommsSlow is a baseline set of communication settings accommodating
// wider range of network performance and APN service responsiveness scenarios.
var CommsSlow = CommsCfg{
	DialTimeout:          40 * time.Second,
	RequestTimeout:       60 * time.Second,
	KeepAlive:            10 * time.Hour,
	MaxConcurrentStreams: 500,
}

// CommsDefault is the set of communication settings that is used when
// you do not supply an explicit comms configuration where one is needed.
var CommsDefault = CommsSlow

func makeDialer(commsCfg CommsCfg) func(network, addr string, cfg *tls.Config) (net.Conn, error) {
	return func(network, addr string, cfg *tls.Config) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout:   commsCfg.DialTimeout,
			KeepAlive: commsCfg.KeepAlive,
		}
		return tls.DialWithDialer(dialer, network, addr, cfg)
	}
}
