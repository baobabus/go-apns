// Copyright 2017 Aleksey Blinov. All rights reserved.

package funit

import "time"

const (
	Second Measure = 1.0
	Minute         = 60.0 * Second
	Hour           = 60.0 * Minute
	Sec            = Second
	Min            = Minute
	Hr             = Hour
	Millisecond    = Milli * Second
	Microsecond    = Micro * Second
	Nanosecond     = Nano * Second
	Picosecond     = Pico * Second
	Femtosecond    = Femto * Second
)

func (m Measure) AsDuration() time.Duration {
	return time.Duration(1000000000.0 * m)
}
