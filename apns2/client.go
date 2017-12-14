// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"context"
	"crypto/tls"
	"errors"
	"sync"

	"github.com/baobabus/go-apns/syncx"
)

// Gateway holds APN service's Development & Production urls.
// These use default HTTPS port 443. According to Apple you can
// alternatively use port 2197 if needed.
var Gateway = struct {
	Development string
	Production  string
}{
	Development: "https://api.development.push.apple.com",
	Production:  "https://api.push.apple.com",
}

// APNS default root URL path.
const RequestRoot = "/3/device/"

var (
	ErrMissingAuth = errors.New("apns2: authentication is not possible with no client certificate and no signer")
	ErrClientNotRunning = errors.New("apns2: client processing pipeline not running")
	ErrClientAlreadyStarted = errors.New("apns2: client processing pipeline already started")
	ErrClientAlreadyClosed  = errors.New("apns2: client processing pipeline already closed")
	ErrPushInterrupted = errors.New("apns2: push request interrupted")
	ErrCanceled = errors.New("apns2: push request canceled")
)

// NoSigner can be used where a RequestSigner is required when a push request
// need not be signed.
var NoSigner RequestSigner

// DefaultSigner can be used instead of nil value where a RequestSigner
// is required to indicate that a push request should be signed with client's
// default signer.
var DefaultSigner RequestSigner

// NoContext can be used instead of nil value to indicate no cancellation
// context.
var NoContext context.Context

// NoCallback is used to indicate that results of push notification requests
// should be silently discarded.
var NoCallback chan *Result

// DefaultCallback can be used instead of nil value to idicate that client's
// default callback channel should be used to communicate back the result
// of a push request.
var DefaultCallback chan<- *Result

// Client provides the means for asynchronous communication with APN service.
// It is safe to use one client in concurrent goroutines and issue concurrent
// push requests.
//
// As per APN service guidelines, you should keep a handle on this client
// so that you can keep your connections with APN servers open.
// Repeatedly opening and closing connections in rapid succession is
// treated by Apple as a denial-of-service attack.
type Client struct {

	// Id identifies client in log entries.
	Id string

	// Gateway is the APN service connection endpoint.
	// Apple publishes two public endpoints: production and development.
	// They are preconfigured in Gateway.Production and Gateway.Development.
	Gateway     string

	// CommsCfg contains communication settings to be used by the client.
	// See CommsCfg type declaration for additional details.
	CommsCfg    CommsCfg

	// ProcCfg contains autoscaling settings.
	// See ProcCfg type declaration for additional details.
	ProcCfg    ProcCfg

	// Certificate, if not nil, is used in the client side configuration
	// of the TLS connections to APN servers.
	// This is one of the authentication methods supported by APN service.
	Certificate *tls.Certificate

	// RootCA, if not nil, can be used to specify an alternative root
	// certificate authority. This should only be needed in testing, or
	// if you system's root certificate authorities are not set up.
	RootCA *tls.Certificate

	// Signer, if not nil, is used to sign individual requests to APN service.
	Signer      RequestSigner

	// Queue for submitting push requests.
	//
	// You can use it directly in your code, especially in select statements
	// when coordination with other channels is desired.
	// Alternatively client's Push method can be used.
	Queue      <-chan *Request

	// Callback, if not nil, specifies the channel to which the outcome of
	// the push request executions should be delivered.
	// If Callback is nil and a request doesn't specify an alternative callback,
	// requests execution result is silently dropped.
	Callback chan<- *Result

	retry chan *Request

	out chan *Request
	gov *governor

	mu    sync.RWMutex
	state uint
	wg    sync.WaitGroup
	ctl   chan struct{} // our control channel
	cctl  chan struct{} // submitter control channel
	gctl  chan struct{} // governor control channel
	cdone chan struct{} // pipeline done processing signal

	// counter for waits on outbound channel
	waitCtr  syncx.TickTockCounter
	// counter of processed requests
	rateCtr  syncx.Counter
}

const (
	stateInitial  uint = iota
	stateStarting
	stateRunning
	stateStopping
	stateTerminating
	stateClosed
)

// Start starts Client processing pipeline. If the client has already
// been started, ErrClientAlreadyStarted error is returned.
func (c *Client) Start(wg *sync.WaitGroup) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.Id) == 0 {
		c.Id = "Client"
	}
	if c.state >= stateStarting {
		return ErrClientAlreadyStarted
	}
	c.state = stateStarting
	logInfo(c.Id, "Starting.")
	if wg != nil {
		wg.Add(1)
	}
	c.wg.Add(1)
	c.ctl = make(chan struct{})
	c.cctl = make(chan struct{})
	c.gctl = make(chan struct{})
	c.cdone = make(chan struct{})
	c.out = make(chan *Request)
	c.retry = make(chan *Request)
	c.gov = &governor{
		id:      c.Id + "-Governor",
		c:       c,
		ctl:     c.gctl,
		done:    c.cdone,
		cfg:     c.ProcCfg,
		minSust: c.ProcCfg.minSustainPollPeriods(),
	}
	// TODO Figure out coordination of governor and retrier shutdowns.
	go c.gov.run()
	go c.runSubmitter(wg)
	return nil
}

// Stop performs soft shutdown of the Client. All inflight requests are
// given the chance to be executed.
func (c *Client) Stop() error {
	c.mu.Lock()
	if c.state >= stateStopping {
		c.mu.Unlock()
		return ErrClientAlreadyClosed
	}
	c.state = stateStopping
	logInfo(c.Id, "Stopping.")
	close(c.cctl) // stop submitter
	c.cctl = nil
	c.mu.Unlock()
	c.wg.Wait()
	close(c.out)
	// Block until all processing is complete
	// or we are signaled to terminate.
	select {
	case <-c.cdone:
	case <-c.ctl:
	}
	if c.Callback != nil && c.Callback != NoCallback {
		close(c.Callback)
	}
	logInfo(c.Id, "Stopped.")
	return nil
}

// Kill performs hard shutdown of the Client without waiting for the processing
// pipeline to unwind. Inflight requests are discarded.
func (c *Client) Kill() error {
	c.mu.Lock()
	if c.state >= stateTerminating {
		c.mu.Unlock()
		return ErrClientAlreadyClosed
	}
	c.state = stateTerminating
	logInfo(c.Id, "Terminating.")
	if c.cctl != nil {
		close(c.cctl)
	}
	close(c.gctl)
	close(c.ctl) // unblock pending Stop() if there's one
	c.mu.Unlock()
	logInfo(c.Id, "Terminated.")
	return nil
}

// Push asynchronously sends a Notification to the APN service.
// Context carries a deadline and a cancellation signal and allows you to close
// long running requests when the context timeout is exceeded.
// Context can be nil or NoContext if no cancellation functionality
// is desired.
//
// If not nil, the supplied signer is asked to sign the request before
// submitting it to APN service. If the supplied signer is nil, but client's
// signer was configured at the initialization time, the client's signer will
// sign the request. NoSigner can be specified if the request must not be signed.
//
// This method will block if downstream capacity is exceeded. For non-blocking
// behavior or to allow coordination with activity on other channels consider
// creating a Request instance and writing it to client's Queue directly.
func (c *Client) Push(n *Notification, signer RequestSigner, ctx context.Context, callback chan<- *Result) error {
	c.mu.RLock()
	state := c.state
	c.mu.RUnlock()
	if state < stateStarting || state > stateRunning {
		return ErrClientNotRunning
	}
	// Ensure that authentication is possible
	if c.Certificate == nil && (signer == NoSigner || !c.HasSigner() && signer == DefaultSigner) {
		return ErrMissingAuth
	}
	// Everything else is done asynchronously
	req := &Request{
		Notification: n,
		Signer: signer,
		Context: ctx,
		Callback: callback,
	}
	err := c.submit(req)
	return err
}

// HasSigner returns `true` if there is a non-default signer configured
// for signing push requests.
func (c *Client) HasSigner() bool {
	return c.Signer != DefaultSigner
}

// TODO Separate submitter out
func (c *Client) runSubmitter(wg *sync.WaitGroup) {
	done := false
	c.mu.Lock()
	if c.state != stateStarting {
		done = true
	} else {
		c.state = stateRunning
	}
	c.mu.Unlock()
	if !done {
		logInfo(c.Id + "-Submitter", "Running.")
	}
	for ; !done; {
		select {
		case req, _ := <-c.retry:
			c.submit(req)
		case req, ok := <-c.Queue:
			if !ok {
				// Queue is closed and we must do s soft shutdown.
				// TODO Rework soft shutdown to account for retries.
				done = true
			}
			c.submit(req)
		case <-c.cctl:
			done = true
		}
	}
	c.mu.Lock()
	c.state = stateClosed
	c.mu.Unlock()
	logInfo(c.Id + "-Submitter", "Stopped.")
	c.wg.Done()
	if wg != nil {
		wg.Done()
	}
}

func (c *Client)submit(req *Request) (rerr error) {
	if c.state < stateStarting || c.state > stateRunning {
		return
	}
	c.rateCtr.Add(1)
	// TODO implement ctx timing out and cancellation checks
	isBlocked := false
	select {
	case c.out<- req:
	default:
		isBlocked = true
	}
	if !isBlocked {
		return
	}
	c.waitCtr.Tick()
	select {
	case c.out<- req:
	case <-c.cctl:
		rerr = ErrPushInterrupted
	}
	c.waitCtr.Tock()
	return
}

func init() {
	NoSigner = noSigner{}
	NoCallback = make(chan *Result)
}
