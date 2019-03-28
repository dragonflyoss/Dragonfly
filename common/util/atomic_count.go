package util

import (
	"sync/atomic"
)

// AtomicInt is a struct that can be added or subtracted atomically.
type AtomicInt struct {
	count *int32
}

// NewAtomicInt returns a new AtomicInt.
func NewAtomicInt(value int32) *AtomicInt {
	return &AtomicInt{
		count: &value,
	}
}

// Add atomically adds delta to count and returns the new value.
func (ac *AtomicInt) Add(delta int32) int32 {
	if ac != nil {
		return atomic.AddInt32(ac.count, delta)
	}
	return 0
}

// Get the value atomically.
func (ac *AtomicInt) Get() int32 {
	if ac != nil {
		return *ac.count
	}
	return 0
}

// Set to value atomically and returns the previous value.
func (ac *AtomicInt) Set(value int32) int32 {
	return atomic.SwapInt32(ac.count, value)
}
