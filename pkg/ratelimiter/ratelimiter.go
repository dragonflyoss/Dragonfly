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

package ratelimiter

import (
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/util"
)

// RateLimiter is used for limiting the rate of transporting.
type RateLimiter struct {
	capacity      int64
	bucket        int64
	rate          int64
	ratePerWindow int64
	window        int64
	last          int64

	mu sync.Mutex
}

// NewRateLimiter creates a RateLimiter instance.
// rate: how many tokens are generated per second. 0 represents that don't limit the rate.
// window: generating tokens interval (millisecond, [1,1000]).
// The production of rate and window should be division by 1000.
func NewRateLimiter(rate int64, window int64) *RateLimiter {
	rl := new(RateLimiter)
	rl.capacity = rate
	rl.bucket = 0
	rl.rate = rate
	rl.setWindow(window)
	rl.computeRatePerWindow()
	rl.last = time.Now().UnixNano()
	return rl
}

// AcquireBlocking acquires tokens. It will be blocking unit the bucket has enough required
// number of tokens.
func (rl *RateLimiter) AcquireBlocking(token int64) int64 {
	return rl.acquire(token, true)
}

// AcquireNonBlocking acquires tokens. It will return -1 immediately when there is no enough
// number of tokens.
func (rl *RateLimiter) AcquireNonBlocking(token int64) int64 {
	return rl.acquire(token, false)
}

// SetRate sets rate of RateLimiter.
func (rl *RateLimiter) SetRate(rate int64) {
	if rl.rate != rate {
		rl.capacity = rate
		rl.rate = rate
		rl.computeRatePerWindow()
	}
}

func (rl *RateLimiter) acquire(token int64, blocking bool) int64 {
	if rl.capacity <= 0 || token < 1 {
		return token
	}
	tmpCapacity := util.Max(rl.capacity, token)

	var process func() int64
	process = func() int64 {
		now := time.Now().UnixNano()

		newTokens := rl.createTokens(now)
		curTotal := util.Min(newTokens+rl.bucket, tmpCapacity)

		if curTotal >= token {
			rl.bucket = curTotal - token
			rl.last = now
			return token
		}
		if blocking {
			rl.blocking(token - curTotal)
			return process()
		}
		return -1
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()
	return process()
}

func (rl *RateLimiter) setWindow(window int64) {
	if window >= 1 && window <= 1000 {
		rl.window = window
	} else if window < 1 {
		rl.window = 1
	} else {
		rl.window = 1000
	}
}

func (rl *RateLimiter) computeRatePerWindow() {
	if rl.rate <= 0 {
		return
	}
	ratePerWindow := int64(rl.rate) * int64(rl.window) / 1000
	if ratePerWindow > 0 {
		rl.ratePerWindow = ratePerWindow
		return
	}
	rl.ratePerWindow = 1
	rl.setWindow(int64(rl.ratePerWindow * 1000 / rl.rate))
}

func (rl *RateLimiter) createTokens(timeNano int64) int64 {
	diff := timeNano - rl.last
	if diff < time.Millisecond.Nanoseconds() {
		return 0
	}
	return diff / (rl.window * time.Millisecond.Nanoseconds()) * rl.ratePerWindow
}

func (rl *RateLimiter) blocking(requiredToken int64) {
	if requiredToken <= 0 {
		return
	}
	windowCount := int64(util.Max(requiredToken/rl.ratePerWindow, 1))
	time.Sleep(time.Duration(windowCount * rl.window * time.Millisecond.Nanoseconds()))
}

// TransRate trans the rate to multiples of 1000.
// For NewRateLimiter, the production of rate should be division by 1000.
func TransRate(rate int64) int64 {
	if rate <= 0 {
		rate = 10 * 1024 * 1024
	}
	rate = (rate/1000 + 1) * 1000
	return rate
}
