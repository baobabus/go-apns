// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"errors"
	"testing"
	"time"

	"github.com/baobabus/go-apns/funit"
	"github.com/stretchr/testify/assert"
)

var backOffTesterErr = errors.New("")

const backOffTesterTimeDelta float64 = 10000000 // 10 millisecond

func TestZeroBackOffTracker(t *testing.T) {
	// Failure first
	s := backOffTracker{}
	assert.Exactly(t, time.Time{}, s.blackoutEnd())
	s.update(backOffTesterErr)
	assert.InDelta(t, time.Now().UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	s.update(backOffTesterErr)
	assert.InDelta(t, time.Now().UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	s.update(nil)
	assert.InDelta(t, time.Now().UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	s.update(nil)
	assert.InDelta(t, time.Now().UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	// Success first
	s = backOffTracker{}
	assert.Exactly(t, time.Time{}, s.blackoutEnd())
	s.update(nil)
	assert.Exactly(t, time.Time{}, s.blackoutEnd())
	s.update(nil)
	assert.Exactly(t, time.Time{}, s.blackoutEnd())
	s.update(backOffTesterErr)
	assert.InDelta(t, time.Now().UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	s.update(backOffTesterErr)
	assert.InDelta(t, time.Now().UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
}

func TestNoJitterBackOffTracker(t *testing.T) {
	// Failure first
	d := time.Millisecond
	s := backOffTracker{initial: d, jitter: 0 * funit.Percent}
	assert.Exactly(t, time.Time{}, s.blackoutEnd())
	s.update(backOffTesterErr)
	assert.InDelta(t, time.Now().Add(d).UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	s.update(backOffTesterErr)
	assert.InDelta(t, time.Now().Add(d).UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	time.Sleep(d)
	s.update(backOffTesterErr)
	d = d << 1
	last := time.Now().Add(d).UnixNano()
	assert.InDelta(t, last, s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	time.Sleep(d)
	s.update(nil)
	assert.InDelta(t, last, s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	s.update(backOffTesterErr)
	d = d >> 1
	assert.InDelta(t, time.Now().Add(d).UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	// Success first
	d = time.Millisecond
	s = backOffTracker{initial: d, jitter: 0 * funit.Percent}
	assert.Exactly(t, time.Time{}, s.blackoutEnd())
	s.update(nil)
	assert.Exactly(t, time.Time{}, s.blackoutEnd())
	s.update(backOffTesterErr)
	assert.InDelta(t, time.Now().Add(d).UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	s.update(backOffTesterErr)
	assert.InDelta(t, time.Now().Add(d).UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	time.Sleep(d)
	s.update(backOffTesterErr)
	d = d << 1
	last = time.Now().Add(d).UnixNano()
	assert.InDelta(t, last, s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	time.Sleep(d)
	s.update(nil)
	assert.InDelta(t, last, s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	s.update(backOffTesterErr)
	d = d >> 1
	assert.InDelta(t, time.Now().Add(d).UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
}

func TestNoJitterCappedBackOffTracker(t *testing.T) {
	// Failure first
	d := time.Millisecond
	max := 3 * time.Millisecond
	s := backOffTracker{initial: d, max: max, jitter: 0 * funit.Percent}
	assert.Exactly(t, time.Time{}, s.blackoutEnd())
	s.update(backOffTesterErr)
	assert.InDelta(t, time.Now().Add(d).UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	s.update(backOffTesterErr)
	assert.InDelta(t, time.Now().Add(d).UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	time.Sleep(d)
	s.update(backOffTesterErr)
	d = max
	last := time.Now().Add(d).UnixNano()
	assert.InDelta(t, last, s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	time.Sleep(d)
	s.update(nil)
	assert.InDelta(t, last, s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	s.update(backOffTesterErr)
	d = time.Millisecond
	assert.InDelta(t, time.Now().Add(d).UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	// Success first
	d = time.Millisecond
	s = backOffTracker{initial: d, jitter: 0 * funit.Percent}
	assert.Exactly(t, time.Time{}, s.blackoutEnd())
	s.update(nil)
	assert.Exactly(t, time.Time{}, s.blackoutEnd())
	s.update(backOffTesterErr)
	assert.InDelta(t, time.Now().Add(d).UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	s.update(backOffTesterErr)
	assert.InDelta(t, time.Now().Add(d).UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	time.Sleep(d)
	s.update(backOffTesterErr)
	d = max
	last = time.Now().Add(d).UnixNano()
	assert.InDelta(t, last, s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	time.Sleep(d)
	s.update(nil)
	assert.InDelta(t, last, s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
	s.update(backOffTesterErr)
	d = time.Millisecond
	assert.InDelta(t, time.Now().Add(d).UnixNano(), s.blackoutEnd().UnixNano(), backOffTesterTimeDelta)
}
