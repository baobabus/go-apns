// Copyright 2017 Aleksey Blinov. All rights reserved.

package scale

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstant(t *testing.T) {
	assert.True(t, Constant.IsValid())
	assert.Exactly(t, uint32(0), Constant.Apply(0))
	assert.Exactly(t, uint32(1), Constant.Apply(1))
	assert.Exactly(t, uint32(2), Constant.Apply(2))
	assert.Exactly(t, uint32(0), Constant.ApplyInverse(0))
	assert.Exactly(t, uint32(1), Constant.ApplyInverse(1))
	assert.Exactly(t, uint32(2), Constant.ApplyInverse(2))
}

func TestIncremental(t *testing.T) {
	var s Incremental
	s = Incremental(0)
	assert.False(t, s.IsValid())
	s = Incremental(1)
	assert.True(t, s.IsValid())
	assert.Exactly(t, uint32(1), s.Apply(0))
	assert.Exactly(t, uint32(2), s.Apply(1))
	assert.Exactly(t, uint32(3), s.Apply(2))
	assert.Exactly(t, uint32(0), s.ApplyInverse(0))
	assert.Exactly(t, uint32(0), s.ApplyInverse(1))
	assert.Exactly(t, uint32(1), s.ApplyInverse(2))
	s = Incremental(10)
	assert.True(t, s.IsValid())
	assert.Exactly(t, uint32(10), s.Apply(0))
	assert.Exactly(t, uint32(11), s.Apply(1))
	assert.Exactly(t, uint32(12), s.Apply(2))
	assert.Exactly(t, uint32(0), s.ApplyInverse(0))
	assert.Exactly(t, uint32(0), s.ApplyInverse(9))
	assert.Exactly(t, uint32(0), s.ApplyInverse(10))
	assert.Exactly(t, uint32(1), s.ApplyInverse(11))
}

func TestExponential(t *testing.T) {
	var s Exponential
	s = Exponential(0)
	assert.False(t, s.IsValid())
	s = Exponential(1)
	assert.False(t, s.IsValid())
	s = Exponential(2)
	assert.True(t, s.IsValid())
	assert.Exactly(t, uint32(1), s.Apply(0))
	assert.Exactly(t, uint32(2), s.Apply(1))
	assert.Exactly(t, uint32(4), s.Apply(2))
	assert.Exactly(t, uint32(0), s.ApplyInverse(0))
	assert.Exactly(t, uint32(0), s.ApplyInverse(1))
	assert.Exactly(t, uint32(1), s.ApplyInverse(2))
	assert.Exactly(t, uint32(1), s.ApplyInverse(3))
	assert.Exactly(t, uint32(2), s.ApplyInverse(4))
	s = Exponential(1.25)
	assert.True(t, s.IsValid())
	assert.Exactly(t, uint32(1), s.Apply(0))
	assert.Exactly(t, uint32(2), s.Apply(1))
	assert.Exactly(t, uint32(3), s.Apply(2))
	assert.Exactly(t, uint32(12), s.Apply(10))
	assert.Exactly(t, uint32(0), s.ApplyInverse(0))
	assert.Exactly(t, uint32(0), s.ApplyInverse(1))
	assert.Exactly(t, uint32(1), s.ApplyInverse(2))
	assert.Exactly(t, uint32(2), s.ApplyInverse(3))
	assert.Exactly(t, uint32(3), s.ApplyInverse(4))
	assert.Exactly(t, uint32(4), s.ApplyInverse(5))
	assert.Exactly(t, uint32(4), s.ApplyInverse(6))
	assert.Exactly(t, uint32(9), s.ApplyInverse(12))
	assert.Exactly(t, uint32(10), s.ApplyInverse(13))
}

