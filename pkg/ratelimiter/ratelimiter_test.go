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
	"testing"
	"time"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type RateLimiterSuite struct{}

func init() {
	check.Suite(&RateLimiterSuite{})
}

func (suite *RateLimiterSuite) TestNewRateLimiter(c *check.C) {
	var cases = []struct {
		r int64
		w int64
		e *RateLimiter
	}{
		{0, 1, &RateLimiter{rate: 0, window: 1, ratePerWindow: 0}},
		{1000, 1, &RateLimiter{rate: 1000, window: 1, ratePerWindow: 1}},
		{500, 1, &RateLimiter{rate: 500, window: 2, ratePerWindow: 1}},
		{500, 1001, &RateLimiter{rate: 500, window: 1000, ratePerWindow: 500}},
		{500, 0, &RateLimiter{rate: 500, window: 2, ratePerWindow: 1}},
	}

	for _, cc := range cases {
		rl := NewRateLimiter(cc.r, cc.w)
		c.Assert(rl.capacity, check.Equals, cc.e.rate)
		c.Assert(rl.bucket, check.Equals, int64(0))
		c.Assert(rl.rate, check.Equals, cc.e.rate)
		c.Assert(rl.window, check.Equals, cc.e.window)
		c.Assert(rl.ratePerWindow, check.Equals, cc.e.ratePerWindow)
	}
}

func (suite *RateLimiterSuite) TestRateLimiter_SetRate(c *check.C) {
	var cases = []struct {
		r  int64
		w  int64
		nr int64
		e  *RateLimiter
	}{
		{0, 1, 500, &RateLimiter{rate: 500, window: 2, ratePerWindow: 1}},
		{1000, 1, 0, &RateLimiter{rate: 0, window: 1, ratePerWindow: 1}},
		{1000, 1, 500, &RateLimiter{rate: 500, window: 2, ratePerWindow: 1}},
		{1000, 2, 500, &RateLimiter{rate: 500, window: 2, ratePerWindow: 1}},
		{1000, 4, 500, &RateLimiter{rate: 500, window: 4, ratePerWindow: 2}},
		{1000, 1, 2000, &RateLimiter{rate: 2000, window: 1, ratePerWindow: 2}},
		{1000, 1, 1999, &RateLimiter{rate: 1999, window: 1, ratePerWindow: 1}},
	}
	for _, cc := range cases {
		rl := NewRateLimiter(cc.r, cc.w)
		rl.SetRate(cc.nr)
		c.Assert(rl.capacity, check.Equals, cc.e.rate)
		c.Assert(rl.rate, check.Equals, cc.e.rate)
		c.Assert(rl.window, check.Equals, cc.e.window)
		c.Assert(rl.ratePerWindow, check.Equals, cc.e.ratePerWindow)
	}
}

func (suite *RateLimiterSuite) TestRateLimiter_AcquireBlocking(c *check.C) {
	var cases = []struct {
		r     int64
		w     int64
		t     int64
		count int
		e     int64
	}{
		{0, 1, 1000, 1, 0},
		{1000, 1, 1000, 1, 1000},
		{1000, 1, 500, 1, 500},
		{1000, 1, 250, 4, 1000},
		{1000, 1, 2000, 1, 2000},
		{1000, 1000, 500, 1, 1000},
		{1000, 1000, 500, 2, 1000},
	}
	for _, cc := range cases {
		start := time.Now().UnixNano() / time.Millisecond.Nanoseconds()
		rl := NewRateLimiter(cc.r, cc.w)
		for i := 0; i < cc.count; i++ {
			c.Assert(rl.AcquireBlocking(cc.t), check.Equals, cc.t)
		}
		end := time.Now().UnixNano() / time.Millisecond.Nanoseconds()
		c.Assert(end-start >= cc.e, check.Equals, true)
		c.Assert(end-start < cc.e+10, check.Equals, true)
	}
}

func (suite *RateLimiterSuite) TestRateLimiter_AcquireNonBlocking(c *check.C) {
	rl := NewRateLimiter(1000, 1)
	c.Assert(rl.AcquireNonBlocking(1000), check.Equals, int64(-1))
	rl.blocking(1000)
	c.Assert(rl.AcquireNonBlocking(1000), check.Equals, int64(1000))
}

func (suite *RateLimiterSuite) TestTransRate(c *check.C) {
	var cases = []struct {
		r int64
		e int64
	}{
		{666, 1000},
		{2048, 3000},
		{123456, 124000},
		{0, 10486000},
		{-233, 10486000},
	}
	for _, cc := range cases {
		v := TransRate(cc.r)
		c.Assert(v, check.Equals, cc.e)
	}
}
