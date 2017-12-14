// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"context"
)

// Request holds all necessary information needed to submit a notification
// to APN service. Requests can be directly submitted to Client's Queue.
type Request struct {

	// Notification is the notification to push to APN service
	Notification *Notification

	// Signer, if not nil, is used to sign the request before submitting it
	// to APN service. If Signer is nil, but client's signer was configured
	// at the initialization time, the client's signer will sign the request.
	Signer RequestSigner

	// Context carries a deadline and a cancellation signal and allows you
	// to close long running requests when the context timeout is exceeded.
	// Context can be nil, for backwards compatibility.
	Context context.Context

	// Callback, if not nil, specifies the channel to which the outcome of
	// the push execution should be delivered. If Callback is nil and client's
	// Callback was configured at the initialization time, the result
	// will be delivered to client's Callback.
	Callback chan<- *Result

	attemptCnt int
}

// HasSigner returns true if the request has a custom signer supplied or if
// no signing should be performed for this request.
func (r *Request) HasSigner() bool {
	return r.Signer != DefaultSigner
}

// RequestError indicates a request-level error. This helps distinguishing
// errors that are only scoped to a single request from those related to wider
// scope, such as transport layer errors.
type RequestError struct {
	error
}
