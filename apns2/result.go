// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"context"
)

// Result represents the outcome of an asynchronous push operation.
// The original notification is included along with any optional
// agruments supplied to the push request.
type Result struct {

	// Notification is the original notification.
	Notification *Notification

	// Signer is the one-off signer that was supplied in the push request.
	Signer RequestSigner

	// Context is the cancellation context instance passed to the original
	// push request.
	Context context.Context

	// Response represents a result from the APN service. If a push operation
	// fails prior to communicating with APN servers, Response will be nil and
	// Err field will have a non-nil value.
	Response *Response

	// Err, if not nil, is an error encontered while attempting a push.
	// Note that nil Err does not necessarily indicate a successful attempt.
	// You must also examine Response for additional status details.
	Err error
}

// IsAccepted returns whether or not the notification was accepted by APN service.
func (r *Result) IsAccepted() bool {
	return r.Err == nil && r.Response != nil && r.Response.StatusCode == StatusAcccepted
}
