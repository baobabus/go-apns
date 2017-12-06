// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMovingAcc(t *testing.T) {
	var v uint64
	var s *movingAcc
	// -1 samples
	s = newMovingAcc(-1)
	assert.Nil(t, s)
	// 0 samples
	s = newMovingAcc(0)
	assert.Nil(t, s)
	// 1 sample
	s = newMovingAcc(1)
	assert.Equal(t, 1, len(s.samples))
	assert.Equal(t, uint64(0), s.sum)
	assert.Equal(t, 0, s.pos)
	v = s.accumulate(2)
	assert.Equal(t, uint64(2), s.sum)
	assert.Equal(t, 0, s.pos)
	assert.Equal(t, uint64(2), v)
	v = s.accumulate(4)
	assert.Equal(t, uint64(4), s.sum)
	assert.Equal(t, 0, s.pos)
	assert.Equal(t, uint64(4), v)
	// 2 samples
	s = newMovingAcc(2)
	assert.Equal(t, 2, len(s.samples))
	assert.Equal(t, uint64(0), s.sum)
	assert.Equal(t, 0, s.pos)
	v = s.accumulate(2)
	assert.Equal(t, uint64(2), s.sum)
	assert.Equal(t, 1, s.pos)
	assert.Equal(t, uint64(2), v)
	v = s.accumulate(4)
	assert.Equal(t, uint64(6), s.sum)
	assert.Equal(t, 0, s.pos)
	assert.Equal(t, uint64(6), v)
	v = s.accumulate(6)
	assert.Equal(t, uint64(10), s.sum)
	assert.Equal(t, 1, s.pos)
	assert.Equal(t, uint64(10), v)
}

