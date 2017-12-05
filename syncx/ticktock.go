// Copyright 2017 Aleksey Blinov. All rights reserved.

package syncx

import (
	"sync/atomic"
)

// TickTockCounter counts the number of Tick and Tock calls that it receives.
// Balancing ticks and tocks can then be "folded", such that both counts are
// reduced by the number of "tocks". I.e. 4 calls of Tick and 3 calls of Tock
// would result in the counts to be reduced to 1 and 0 respectively.
//
// TickTockCounter places following constraints on its use:
//
// 1. A call to Tick must be guaranteed to have completed before calling its
// balancing Tock. Typically such balancing calls would be made serially
// from a single goroutine.
//
// 2. Concurrent calls to Fold are not allowed. Folding would typically be done
// in a single goroutine in a serial manner.
//
// TickTockCounter is optimized for use with multiple concurrent "tikers" and
// a single folder. If concurrent calls to Fold are anticipated, they must be
// guarded by a mutex.
type TickTockCounter uint64

// Tick atomically increments "tick" counter. It is safe for use in concurrent
// gorotines as long as the caller ensures that corresponding Tock call is only
// made after Tick call.
func (c *TickTockCounter) Tick() {
	atomic.AddUint64((*uint64)(c), 1 << 32)
}

// Tick atomically increments "tock" counter. It is safe for use in concurrent
// gorotines as long as the caller ensures that corresponding Tick call has
// already been made.
func (c *TickTockCounter) Tock() {
	atomic.AddUint64((*uint64)(c), 1)
}

// Fold collapses balancing "ticks" and "tocks" by reducing the counts by the
// number of "tocks" and returns pre-folded counts. Folding is done atomically,
// such that concurrent calls to Tick and Tock do not result in an imbalance
// as well as ensuring that no "ticks" or "tocks" are dropped or double-counted.
//
// For performance reasons this method is not safe for use in concurrent
// gorotines. It is however safe for use concurrently with Tick and Tock calls.
// If concurrent calls to Draw are anticipated they must be protected
// by a mutex.
func (c *TickTockCounter) Fold() (ticks uint32, tocks uint32) {
	cntr := atomic.LoadUint64((*uint64)(c))
	tocks = uint32(cntr)
	ticks = uint32(cntr >> 32)
	// It is possible for the counts to have increased since the load call
	// as Tick and Tock are called from concurrent goroutines.
	// Atomically subtracting previously read tock count from both counters
	// is still safe as the counts would have never decreased (as long as Fold
	// is not called concurrently for another goroutine).
	// We may end up with a non-zero tock count at the end of the subtraction,
	// but it is not wrong. These "extra" counts will be picked up by the
	// subsequent call to Fold. No counts are dropped or double-counted.
	atomic.AddUint64((*uint64)(c), ^((uint64(tocks) << 32) + uint64(tocks) - 1))
	return
}

// TickTockFolder counts the number of balanced Tick/Tock calls.
// Conceptually it comprises two counters, one for the number of complete
// pairs and one the number of pending ones, with each being a uint32.
//
// TickTockFolder is optimized for concurrent use by multiple "tickers",
// but is restricted to a single concurrent consumer. Protect access
// to the counter with a mutex if concurrent Draw attempts are anticipated.
type TickTockFolder uint64

// Tick atomically increments pending cycle counter.
//
// This method is safe for use in concurrent gorotines.
func (f *TickTockFolder) Tick() {
	atomic.AddUint64((*uint64)(f), 1)
}

// Tock atomically decrements pending cycle counter and increments
// complete cycle counter.
//
// This method is safe for use in concurrent gorotines.
func (f *TickTockFolder) Tock() {
	atomic.AddUint64((*uint64)(f), uint64(^uint32(0)))
}

// Draw atomically draws complete cycle counter. The counter's complete cycles
// count is set to 0 and the previous value is returned. Pending cycle count
// is also returned.
//
// This method is not safe for use in concurrent gorotines. It is however safe
// for use concurrently with Tick and Tock methods.
// If concurrent calls to Draw are anticipated they must be protected
// by a mutex.
func (f *TickTockFolder) Draw() (complete uint32, pending uint32) {
	cntr := atomic.LoadUint64((*uint64)(f))
	pending = uint32(cntr)
	cntr = cntr >> 32
	complete = uint32(cntr)
	// It's possible for the complete count to have increased by this point,
	// but we are only subtracting the value previously read.
	// This is safe as long as we are not calling Draw concurrently from more
	// than one goroutine.
	atomic.AddUint64((*uint64)(f), ^((cntr << 32) - 1))
	return
}
