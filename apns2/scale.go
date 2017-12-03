// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

// Scale must be implemented by scale-up and wind-down calculators.
// Three scale calculators come predefined: Incremental, Exponential
// and Constant.
type Scale interface {
	IsValid() bool
	Apply(n uint32) uint32
	ApplyInverse(n uint32) uint32
}

// Constant scaling mode does not allow scaling.
type constant struct{}

// IsValid always return true.
func (s constant) IsValid() bool {
	return true
}

// Apply returns supplied value unmodified.
func (s constant) Apply(n uint32) uint32 {
	return n
}

// ApplyInverse returns supplied value unmodified.
func (s constant) ApplyInverse(n uint32) uint32 {
	return n
}

// Constant scaler that does not allow scaling.
var Constant constant

// Incremental scaling mode specifies the number of new instances to be added
// during each scaling attempt. Must be 1 or greater.
type Incremental uint32

// IsValid checks that its value is greater than 1.
func (s Incremental) IsValid() bool {
	return s >= 1
}

// Apply adds itself to the supplied value and returns the sum.
func (s Incremental) Apply(n uint32) uint32 {
	return n + uint32(s)
}

// If Incremental is greater or equal to the supplied value, ApplyInverse
// subtracts itself to the argument and returns the difference. Otherwise
// 0 is returned.
func (s Incremental) ApplyInverse(n uint32) uint32 {
	if uint32(s) > n {
		return 0
	}
	return n - uint32(s)
}

// Exponential scaling mode specifies the factor by which the number of
// instances should be increased during each scaling attempt. Must be greater
// than 1.0.
type Exponential float32

// IsValid checks that its value is greater than 1.
func (s Exponential) IsValid() bool {
	return s > 1.0
}

// Apply scales the supplied value by its factor and returns the result.
// The result is guaranteed to be greater that the input by at least 1.
func (s Exponential) Apply(n uint32) uint32 {
	res := uint32(float32(s) * float32(n))
	// We must increase by at least 1.
	if res <= n {
		res = n + 1
	}
	return res
}

// Apply scales the supplied value by its inverse factor and returns the result.
// The result is guaranteed to be 0 or to be less that the nonzero input
// by at least 1.
func (s Exponential) ApplyInverse(n uint32) uint32 {
	res := uint32(float32(n) / float32(s))
	// We must decrease by at least 1, but not go below 0.
	if res >= n && n > 0 {
		res = n - 1
	}
	return res
}

