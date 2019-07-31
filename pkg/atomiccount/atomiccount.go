/*
 * Copyright The Dragonfly Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package atomiccount

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
