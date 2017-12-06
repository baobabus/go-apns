// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/baobabus/go-apns/syncx"
)

// Each streamer "owns" a single HTTPClient on top of an HTTP/2 transport.
// It streams requests to and handles responses from APN servers while
// coordinating HTTP/2 stream utilization.
type streamer struct {
	id         string
	c          *Client
	gov        *governor
	in         <-chan *Request
	out        chan<- *Result
	ctl        chan struct{}
	done       chan<- *streamer

	warmStart  bool

	startOnce  sync.Once
	startErr   error

	httpClient *HTTPClient

	// counter for waits on outbound channel
	waitCtr  syncx.TickTockCounter
	// cumulative request sizes in bytes
	sizeCtr  syncx.Counter

	// wait group for spawned HTTP/2 roundrips
	wg sync.WaitGroup

	didQuit  bool
	inClosed bool
}

func (s *streamer) start(wg *sync.WaitGroup) error {
	s.startOnce.Do(func() {
		logInfo(s.id, "Starting.")
		s.httpClient, s.startErr = NewHTTPClient(s.c.Gateway, s.c.CommsCfg, s.c.Certificate, s.c.RootCA)
		if s.startErr != nil {
			return
		}
		var pollInt time.Duration
		if s.gov.cfg.AllowHTTP2Incursion && !s.gov.cfg.UsePreciseHTTP2Metrics {
			pollInt = s.gov.cfg.HTTP2MetricsRefreshPeriod
		}
		s.httpClient.precise = s.gov.cfg.AllowHTTP2Incursion && s.gov.cfg.UsePreciseHTTP2Metrics
		s.httpClient.pollInt = pollInt
		s.httpClient.cfgCap = s.c.CommsCfg.MaxConcurrentStreams
		if s.warmStart {
			// This can also be accomplished by sending a malformed http.Request.
			// No reflection is required, but it's still a kludge and results
			// in error being logged.
			_, s.startErr = s.httpClient.getClientConn()
			// TODO Should we wait for OPTIONS frame to arrive and set MAXCONCURRENTSTREAMS?
		}
		if s.startErr != nil {
			return
		}
		if wg != nil {
			wg.Add(1)
		}
		go s.run(wg)
	})
	return s.startErr
}

func (s *streamer) run(wg *sync.WaitGroup) {
	logInfo(s.id, "Running.")
	for done := false; !done; {
		select {
		case req, ok := <- s.in:
			if !ok {
				// soft shutdown - wait for pending roundtrips to complete
				logInfo(s.id, "Stopping.")
				// TODO Switch from WaitGroup to channel signal
				s.wg.Wait()
				done = true
				s.inClosed = true
				break
			}
			s.exec(req)
		case _, ok := <-s.ctl:
			if ok {
				// unusable connection
				s.didQuit = true
				logInfo(s.id, "Quitting.")
			} else {
				// hard shutdown - do not wait for pending roundtrips to complete
				logInfo(s.id, "Terminating.")
			}
			// TODO Cancel pending roundtrips' contexts.
			done = true
		}
	}
	// This will only have effect if all roundtrips are finished.
	s.httpClient.Close()
	// read from ctl prevents blocking on done if the governor
	// was commanded to terminate in the meantime
	select {
	case s.done<- s:
	case <-s.ctl:
	}
	if wg != nil {
		wg.Done()
	}
	logInfo(s.id, "Stopped.")
}

func (s *streamer) exec(req *Request) {
	logTrace(0, s.id, "Serving %v.", req)
	if s.c.Certificate == nil && (req.Signer == NoSigner || !s.c.HasSigner() && !req.HasSigner()) {
		s.callBack(req, nil, ErrMissingAuth)
		return
	}
	hasCtx := req.Context != NoContext
	canceled := false
	// TODO Move the below to HTTP/2 stream wait code
	if hasCtx {
		select {
		case <-req.Context.Done():
			canceled = true
		default:
		}
	}
	if canceled {
		s.callBack(req, nil, ErrCanceled)
		return
	}
	var cancel func (done <-chan struct{}) error
	if hasCtx {
		// Waits for the user to cancel a request's context.
		cancel = func (done <-chan struct{}) error {
			ctx := req.Context
			if ctx.Done() == nil {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-done:
				return nil
			}
		}
	}
	// 1. Acquire HTTP/2 stream
	// This can block and is the primary source of back pressure.
	st, err := s.httpClient.ReservedStream(cancel)
	if err != nil {
		s.callBack(req, nil, err)
		return
	}
	// 2. go submit()
	s.wg.Add(1)
	go func() {
		defer st.Close()
		defer s.wg.Done()
		resp, err := s.submit(req)
		if err != nil && uint32(req.attemptCnt) < s.gov.cfg.MaxRetries && s.isRetriable(resp, err) {
			req.attemptCnt++
			// Retry is serviced in a timely manner, so no need to worry about blocking.
			// There's just a potential issue with retry forwarder stopping reads
			// due to a signal on its ctl channel with streamers still running.
			// Forwarder's ctl channel shoulnd't be shared with governor.
			s.gov.retry <- req
			return
		}
		s.callBack(req, resp, err)
		if !s.isConnUsable(resp, err) {
			// Each worker is given its own ctl channel, but we cannot close it here.
			// Writing to it accomplishes the same thing. Just do not block.
			var v struct{}
			select {
			case s.ctl <- v:
			default:
			}
		}
	}()
}

// Submits request to APN service and returns APN response or an error.
func (s *streamer) submit(req *Request) (*Response, error) {
	url := s.c.Gateway + RequestRoot + req.Notification.Recipient
	httpReq, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, &RequestError{err}
	}
	if err := req.Notification.write(httpReq); err != nil {
		return nil, &RequestError{err}
	}
	signer := req.Signer
	if signer == nil {
		signer = s.c.Signer
	}
	if signer != nil {
		if err := signer.SignRequest(httpReq); err != nil {
			return nil, &RequestError{err}
		}
	}
	if req.Context != NoContext {
		httpReq = httpReq.WithContext(req.Context)
	}
	logTrace(2, s.id, "http.Request: %v\n", httpReq)
	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	s.sizeCtr.Add(uint64(estimatedRequestWireSize(httpReq)))
	logTrace(2, s.id, "http.Response: %v\n", httpResp)
	defer httpResp.Body.Close()
	res := &Response{
		StatusCode: httpResp.StatusCode,
		ApnsID:     httpResp.Header.Get("apns-id"),
	}
	decoder := json.NewDecoder(httpResp.Body)
	if err := decoder.Decode(&res); err != nil && err != io.EOF {
		return &Response{}, &RequestError{err}
	}
	return res, nil
}

func (s *streamer) callBack(req *Request, resp *Response, err error) {
	res := &Result{
		Notification: req.Notification,
		Signer: req.Signer,
		Context: req.Context,
		Response: resp,
		Err: err,
	}
	if req.Callback == NoCallback {
		return
	}
	tgt := s.out
	if req.Callback != nil {
		tgt = req.Callback
	}
	if tgt != nil {
		isBlocked := false
		select {
		case tgt<- res:
		default:
			isBlocked = true
		}
		if !isBlocked {
			return
		}
		s.waitCtr.Tick()
		select {
		case tgt<- res:
		case <-s.ctl:
		}
		s.waitCtr.Tock()
	}
}

func (s *streamer) isRetriable(resp *Response, err error) bool {
	if resp == nil && err != nil {
		return false
	}
	if s.gov.cfg.RetryEval != nil {
		return s.gov.cfg.RetryEval(resp, err)
	}
	return false
}

func (s *streamer) isConnUsable(resp *Response, err error) bool {
	if resp == nil && err != nil {
		switch err.(type) {
		case *RequestError:
			// Request-level error
			return true
		default:
			// Error from http.Client.Do()
			// "Invalid method" is our fault and not recoverable.
			if strings.HasPrefix(err.Error(), "net/http: ") {
				return false
			}
			// TODO Consider other possibilities
			return false
		}
	}
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusServiceUnavailable,
			http.StatusMethodNotAllowed:
			return true
		case http.StatusBadRequest:
			return resp.RejectionReason != ReasonIdleTimeout
		case http.StatusForbidden:
			return resp.RejectionReason != ReasonBadCertificate && resp.RejectionReason != ReasonBadCertificateEnvironment
		}
	}
	return true
}

var baseReqWireSizeSize  = uint64(5 + len(RequestRoot))

// Only an estimate and only based on the fields we use. I.e. cookie sizes
// are not included.
func estimatedRequestWireSize(req *http.Request) (res int) {
	res = len(req.Host) + // this needs to be counted in addition
	      len(req.URL.RawPath) + // not .EscapedPath() as no escaping is needed in our case
	      int(req.ContentLength) + // We know we set it
	      14 + // for "POST " and " HTTP/2.0"
	      estimatedHeaderWireSize(req.Header)
	return res
}

// Only an estimate and under the assupmtion that no duplicates are present
func estimatedHeaderWireSize(hs http.Header) (res int) {
	for h, vs := range hs {
		res += len(h) + 4 // account for ": " and "\r\n"
		for _, v := range vs {
			res += len(v)
			break // no duplicates allowed
		}
	}
	return res
}
