// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

// Priority is the priority of the notification.
// Allowable values are defined by APNs and are listed below.
type Priority int

const (
	// PriorityLow instructs APNs to send the push message at a time
	// that takes into account power considerations for the device.
	// Notifications with this priority might be grouped and delivered
	//in bursts. They are throttled, and in some cases are not delivered.
	PriorityLow Priority = 5

	// PriorityHigh instructs APNs to send the push message immediately.
	// Notifications with this priority must trigger an alert, sound,
	// or badge on the target device.
	// It is an error to use this priority for a push notification
	// that contains only the content-available key.
	PriorityHigh = 10
)

// Notification holds the data that is to be pushed to the recipient
// as well as any routing information required to deliver it.
// Routing headers and the notification payload are meant to remain immutable
// and are intended to be shared accross multiple notifications if needed.
// This is usefull when the same message needs to be deliverd to
// many recipients.
type Notification struct {
	// ApnsID is a canonical UUID that identifies the notification.
	// If there is an error sending the notification, APNs uses this value
	// to identify the notification in its response.
	// The canonical form is 32 lowercase hexadecimal digits,
	// displayed in five groups separated by hyphens in the form 8-4-4-4-12.
	// An example ApnsID is as follows: 123e4567-e89b-12d3-a456-42665544000
	// If omitted, a new ApnsID is created by APNs and returned in the response.
	ApnsID string

	// Recipient is the device token of the notification target.
	Recipient string

	// Header is a reference to a structure containing routing information.
	Header *Header

	// Payload is the notification data that is passed to the recipient.
	// Payload can be of any type that can be marshalled into a valid
	// JSON dictionary, a string representation of such dictionaty or
	// a slice of bytes of JSON encoding of such dictionary.
	Payload interface{}
}

// Header is a container for the routing information.
// How a header is constructed and utilized is not constrained,
// but the intent is to never modify it once created and assigned.
// The same header can then be used for routing of any number of notifications.
type Header struct {
	// The topic of the remote notification, which is typically the bundle ID
	// for your app. The certificate you create in your developer account
	// must include the capability for this topic.
	// If your certificate includes multiple topics, you must specify a value for this header.
	// If you omit this request header and your APNs certificate does not specify
	// multiple topics, the APNs server uses the certificate’s Subject as the default topic.
	// If you are using a provider token instead of a certificate, you must specify a value
	// for this request header. The topic you provide should be provisioned for the your team
	// named in your developer account.
	Topic string

	// CollapseID, if set, allows grouping of multiple notifications by apns2.
	// Multiple notifications with the same collapse identifier are displayed
	// to the user as a single notification.
	// The value of this field must not exceed 64 bytes.
	CollapseID string

	// Priority is the priority of the notification.
	// Specify ether apns2.PriorityHigh (10) or apns2.PriorityLow (5)
	// If you don't set this, the APNs server will set the priority to 10.
	Priority Priority

	// Expiration identifies the date when the notification is no longer valid
	// and can be discarded.
	// If this value is nonzero, APNs stores the notification
	// and tries to deliver it at least once, repeating the attempt as needed
	// if it is unable to deliver the notification the first time.
	// If the value is 0, APNs treats the notification as if it expires immediately
	// and does not store the notification or attempt to redeliver it.
	Expiration time.Time

	httpHeaders atomic.Value
}

func (n *Notification) write(r *http.Request) error {
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	if n.ApnsID != "" {
		r.Header.Set("apns-id", n.ApnsID)
	}
	n.Header.write(r)
	body, err := n.newPayloadReader()
	if err != nil {
		return err
	}
	r.Body = body
	r.ContentLength = body.Len()
	// TODO Move to a separate func congitional on go1.8
	// r.GetBody = func() (io.ReadCloser, error) {
	// 	return body.ResetClone(), nil
	// }
	return nil
}

func (n *Notification) newPayloadReader() (*sliceReader, error) {
	var buf []byte
	switch n.Payload.(type) {
	case []byte:
		buf = n.Payload.([]byte)
	case string:
		buf = []byte(n.Payload.(string))
	default:
		var err error
		buf, err = json.Marshal(n.Payload)
		if err != nil {
			return nil, err
		}
	}
	return newSliceReader(buf), nil
}

func (h *Header) getHTTPHeaders() [][2]string {
	res := h.httpHeaders.Load()
	if res != nil {
		return res.([][2]string)
	}
	// We could protect this with a Mutex, but for improved throughput
	// it is probably better to avoid resource contention here and just
	// duplicate the work in case we have concurrent calls.
	hdrs := make([][2]string, 0, 4)
	if h.Topic != "" {
		hdrs = append(hdrs, [...]string{"apns-topic", h.Topic})
	}
	if h.CollapseID != "" {
		hdrs = append(hdrs, [...]string{"apns-collapse-id", h.CollapseID})
	}
	if h.Priority > 0 {
		hdrs = append(hdrs, [...]string{"apns-priority", fmt.Sprintf("%v", h.Priority)})
	}
	if !h.Expiration.IsZero() {
		hdrs = append(hdrs, [...]string{"apns-expiration", fmt.Sprintf("%v", h.Expiration.Unix())})
	}
	h.httpHeaders.Store(hdrs)
	return hdrs
}

func (h *Header) write(r *http.Request) error {
	for _, h := range h.getHTTPHeaders() {
		r.Header.Set(h[0], h[1])
	}
	return nil
}

// sliceReader is ReaderCloser that doesn't take ownership of the slice.
type sliceReader struct {
	buf []byte
	off int
}

func newSliceReader(buf []byte) *sliceReader {
	return &sliceReader{buf: buf}
}

// Read reads the next len(p) bytes from the buffer or until the buffer
// is drained. The return value n is the number of bytes read. If the
// buffer has no data to return, err is io.EOF (unless len(p) is zero);
// otherwise it is nil.
func (r *sliceReader) Read(p []byte) (n int, err error) {
	if r.off >= len(r.buf) {
		if len(p) == 0 {
			return
		}
		return 0, io.EOF
	}
	n = copy(p, r.buf[r.off:])
	r.off += n
	return
}

func (r *sliceReader) Close() error {
	return nil
}

func (r *sliceReader) Len() int64 {
	return int64(len(r.buf))
}

func (r *sliceReader) ResetClone() *sliceReader {
	return &sliceReader{buf: r.buf}
}
