// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"math/rand"
	"time"

	"github.com/baobabus/go-apns/funit"
)

type backOffTracker struct {
	initial time.Duration
	jitter  funit.Measure
	current time.Duration
	end     time.Time
}

func (t *backOffTracker) update(status error) {
	if status != nil {
		if now := time.Now(); now.After(t.end) {
			// Ignore any failures before end time as it may be coming
			// from a concurrent attempt.
			if t.current == 0 {
				t.current = t.initial
			}
			d := t.current
			if t.jitter > 0 {
				jtr := rand.Int63n(int64(funit.Measure(d) * t.jitter))
				d += time.Duration(jtr)
			}
			t.end = now.Add(d)
			t.current = t.current << 1
			logTrace(1, "backoff", "backing off for %v until %v", d, t.end)
		}
	} else {
		if now := time.Now(); now.After(t.end) {
			// Ignore any success before end time as it may be coming
			// from a concurrent attempt.
			t.current = t.initial
			logTrace(1, "backoff", "resetting to &v", t.current)
		}
	}
}

func (t *backOffTracker) blackoutEnd() time.Time {
	return t.end
}
