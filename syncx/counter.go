// Copyright 2017 Aleksey Blinov. All rights reserved.

package syncx

import (
	"sync/atomic"
)

// Counter is a convenience unit64 value counter with atomic operations.
// It is optimized for concurrent use by multiple incrementers,
// but is restricted to a single concurrent consumer. Protect access
// to the counter with a mutex if concurrent Draw attempts are anticipated.
type Counter uint64

// Add atomically adds the supplied value to the counter.
//
// This method is safe for use in concurrent gorotines.
func (f *Counter) Add(v uint64) {
	atomic.AddUint64((*uint64)(f), v)
}

// Draw atomically draws the counter counter. The counter's value it set to 0
// and its previous value is returned.
//
// This method is not safe for use in concurrent gorotines. It is however safe
// for use concurrently with Add method.
// If concurrent calls to Draw are anticipated they must be protected
// by a mutex.
func (f *Counter) Draw() (uint64) {
	res := atomic.LoadUint64((*uint64)(f))
	// It's possible for the count to have increased by this point,
	// but we are only subtracting the value previously read.
	// This is safe as long as we are not calling Draw concurrently from more
	// than one goroutine.
	atomic.AddUint64((*uint64)(f), ^(res - 1))
	return res
}
