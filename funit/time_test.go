// Copyright 2017 Aleksey Blinov. All rights reserved.

package funit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeAsDuration(t *testing.T) {
	assert.Exactly(t, time.Second, Second.AsDuration())
	assert.Exactly(t, time.Second, (1000*Millisecond).AsDuration())
	assert.Exactly(t, time.Minute, Minute.AsDuration())
	assert.Exactly(t, time.Minute, (60*Second).AsDuration())
	assert.Exactly(t, time.Hour, Hour.AsDuration())
	assert.Exactly(t, time.Hour, (60*Minute).AsDuration())
	assert.Exactly(t, time.Hour, (3600*Second).AsDuration())
	assert.Exactly(t, time.Hour + 30*time.Minute, (90*Minute).AsDuration())
	assert.Exactly(t, time.Millisecond, Millisecond.AsDuration())
	assert.Exactly(t, time.Microsecond, Microsecond.AsDuration())
	assert.Exactly(t, time.Nanosecond, Nanosecond.AsDuration())
}
