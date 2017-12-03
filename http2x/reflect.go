// Copyright 2017 Aleksey Blinov. All rights reserved.

package http2x

import (
	"errors"
	"net/http"
	"reflect"
	"sync"
	"unsafe"

	"golang.org/x/net/http2"
)

var (
	ErrIncompatibleHTTP2Layer = errors.New("http2x: incompatible http2 client library")
	ErrUnsupportedTransport = errors.New("http2x: unsupported transport layer")
)

// GetMaxConcurrentStreams returns the value of maxConcurrentStreams
// private field of c using reflection. It properly guards its read
// with c's mutex.
//
// If c is nil, 1 is returned. If c is closed, 0 is returned.
// Otherwise, if maxConcurrentStreams cannot be determined
// due to http2.ClientConn incompatibility, maximum uint32 value is returned.
func GetMaxConcurrentStreams(c *http2.ClientConn) uint32 {
	if c == nil {
		return 1
	}
	rc := reflect.Indirect(reflect.ValueOf(c))
	if !c.CanTakeNewRequest() {
		return 0
	}
	if !http2Compat {
		return ^uint32(0)
	}
	// This is all of the data we are currently interested in.
	// If the need for other fields arises, we should ensure
	// to retrive them all together while holding the c's Mutex.
	mu := (*sync.Mutex)(ptrToFieldValue(rc, clientConn.mu))
	mu.Lock()
	defer mu.Unlock()
	// This may be better than c.CanTakeNewRequest() as it is being guarded.
	// closed := (*bool)(ptrToFieldValue(rc, clientConn.closed))
	// goAway := (**http2.GoAwayFrame)(ptrToFieldValue(rc, clientConn.goAway))
	// if *closed || *goAway != nil {
	// 	return 0
	// }
	res := (*uint32)(ptrToFieldValue(rc, clientConn.maxConcurrentStreams))
	return *res
}

var dummyReq http.Request

// GetClientConnPool returns http2.Transport t's ClientConnPool. If t is not a
// *http2.Transport, ErrUnsupportedTransport error is returned.
//
// GetClientConnPool must be used with extreme caution. It relies on the side
// effect of http2.Transport.CloseIdleConnections to ensure t's connection pool
// is initialized before trying to access it. The only safe time for this call
// is before t has had a chance to open its first connection.
func GetClientConnPool(t http.RoundTripper) (http2.ClientConnPool, error) {
	if !http2Compat {
		return nil, ErrIncompatibleHTTP2Layer
	}
	t2, ok := t.(*http2.Transport)
	if !ok || t2 == nil {
		return nil, ErrUnsupportedTransport
	}
	// Hack
	// This has a side effect of the transport initializing its connection pool
	t2.CloseIdleConnections()
	rt := reflect.Indirect(reflect.ValueOf(t2))
	res := (*(*http2.ClientConnPool)(ptrToFieldValue(rt, transport.connPoolOrDef)))
	return res, nil
}

// GetClientConn returns http2.ClientConn from the pool that can be used to
// communicate with the endpoint specified by the addr. Note that the pool
// may initiate a dial operation at this time if no active connection
// to addr exists.
func GetClientConn(pool http2.ClientConnPool, addr string) (*http2.ClientConn, error) {
	return pool.GetClientConn(&dummyReq, addr)
}

func ptrToFieldValue(v reflect.Value, fieldIndex []int) unsafe.Pointer {
	return unsafe.Pointer(v.FieldByIndex(fieldIndex).UnsafeAddr())
}

// True if it is confirmed that http2.ClientConn structure is
// as expected and can be used with our reflection code.
var http2Compat = true

var clientConn struct {
	mu []int
	maxConcurrentStreams []int
	closed []int
	goAway []int
}

var transport struct {
	connPoolOrDef []int
}

func init() {
	// Validate http2.ClientConn structure
	c := reflect.TypeOf(&http2.ClientConn{}).Elem()
	if f, ok := c.FieldByName("mu"); ok {
		if f.Type.AssignableTo(reflect.TypeOf(sync.Mutex{})) {
			clientConn.mu = f.Index
		} else {
			http2Compat = false
		}
	} else {
		http2Compat = false
	}
	if f, ok := c.FieldByName("maxConcurrentStreams"); ok {
		if f.Type.Kind() == reflect.Uint32 {
			clientConn.maxConcurrentStreams = f.Index
		} else {
			http2Compat = false
		}
	} else {
		http2Compat = false
	}
	if f, ok := c.FieldByName("closed"); ok {
		if f.Type.Kind() == reflect.Bool {
			clientConn.closed = f.Index
		} else {
			http2Compat = false
		}
	} else {
		http2Compat = false
	}
	if f, ok := c.FieldByName("goAway"); ok {
		if f.Type.AssignableTo(reflect.TypeOf(&http2.GoAwayFrame{})) {
			clientConn.goAway = f.Index
		} else {
			http2Compat = false
		}
	} else {
		http2Compat = false
	}
	// Validate http2.Transport structure
	t := reflect.TypeOf(&http2.Transport{}).Elem()
	if f, ok := t.FieldByName("connPoolOrDef"); ok {
		if f.Type.AssignableTo(reflect.TypeOf((*http2.ClientConnPool)(nil)).Elem()) {
			transport.connPoolOrDef = f.Index
		} else {
			http2Compat = false
		}
	} else {
		http2Compat = false
	}
}
