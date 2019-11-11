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

package queue

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type DFGetUtilSuite struct{}

func init() {
	check.Suite(&DFGetUtilSuite{})
}

func (suite *DFGetUtilSuite) SetUpTest(c *check.C) {
	rand.Seed(time.Now().UnixNano())
}

func (suite *DFGetUtilSuite) TestQueue_infiniteQueue(c *check.C) {
	timeout := 50 * time.Millisecond
	q := NewQueue(0)

	q.Put(nil)
	c.Assert(q.Len(), check.Equals, 0)

	q.PutTimeout(nil, 0)
	c.Assert(q.Len(), check.Equals, 0)

	q.Put(0)
	c.Assert(q.Len(), check.Equals, 1)
	c.Assert(q.Poll(), check.Equals, 0)
	c.Assert(q.Len(), check.Equals, 0)

	{ // test Poll
		time.AfterFunc(timeout, func() { q.Put(1) })
		start := time.Now()
		c.Assert(q.Poll(), check.Equals, 1)
		c.Assert(time.Since(start) > timeout, check.Equals, true)
	}

	{ // test PollTimeout
		v, ok := q.PollTimeout(0)
		c.Assert(v, check.IsNil)
		c.Check(ok, check.Equals, false)

		start := time.Now()
		v, ok = q.PollTimeout(timeout)
		c.Assert(v, check.IsNil)
		c.Check(ok, check.Equals, false)
		c.Assert(time.Since(start) > timeout, check.Equals, true)

		time.AfterFunc(timeout/2, func() { q.Put(1) })
		start = time.Now()
		v, ok = q.PollTimeout(timeout)
		c.Assert(ok, check.Equals, true)
		c.Assert(v, check.Equals, 1)
		c.Assert(time.Since(start) >= timeout/2, check.Equals, true)
		c.Assert(time.Since(start) < timeout, check.Equals, true)
	}
}
func (suite *DFGetUtilSuite) TestQueue_infiniteQueue_PollTimeout(c *check.C) {
	timeout := 50 * time.Millisecond
	q := NewQueue(0)

	wg := sync.WaitGroup{}
	var cnt int32
	f := func(i int) {
		if _, ok := q.PollTimeout(timeout); ok {
			atomic.AddInt32(&cnt, 1)
		}
		wg.Done()
	}
	start := time.Now()
	n := 6
	wg.Add(n)
	for i := 0; i < n; i++ {
		go f(i)
	}
	time.AfterFunc(timeout/2, func() {
		for i := 0; i < n-1; i++ {
			q.Put(i)
		}
	})
	wg.Wait()

	c.Assert(time.Since(start) > timeout, check.Equals, true)
	c.Assert(cnt, check.Equals, int32(n-1))
}

func (suite *DFGetUtilSuite) TestQueue_finiteQueue(c *check.C) {
	timeout := 50 * time.Millisecond
	q := NewQueue(2)

	q.Put(nil)
	c.Assert(q.Len(), check.Equals, 0)

	q.PutTimeout(nil, 0)
	c.Assert(q.Len(), check.Equals, 0)

	q.Put(1)
	c.Assert(q.Len(), check.Equals, 1)

	start := time.Now()
	q.PutTimeout(2, timeout)
	q.PutTimeout(3, timeout)
	q.PutTimeout(4, 0)
	c.Assert(q.Len(), check.Equals, 2)
	c.Assert(time.Since(start) >= timeout, check.Equals, true)
	c.Assert(time.Since(start) < 2*timeout, check.Equals, true)

	c.Assert(q.Poll(), check.Equals, 1)
	c.Assert(q.Len(), check.Equals, 1)
	c.Assert(q.Poll(), check.Equals, 2)

	{
		q.PutTimeout(1, 0)
		item, ok := q.PollTimeout(timeout)
		c.Assert(ok, check.Equals, true)
		c.Assert(item, check.Equals, 1)

		start = time.Now()
		item, ok = q.PollTimeout(timeout)
		c.Assert(ok, check.Equals, false)
		c.Assert(item, check.IsNil)
		c.Assert(time.Since(start) >= timeout, check.Equals, true)

		start = time.Now()
		q.PutTimeout(1, 0)
		item, ok = q.PollTimeout(0)
		c.Assert(ok, check.Equals, true)
		c.Assert(item, check.Equals, 1)
		item, ok = q.PollTimeout(0)
		c.Assert(ok, check.Equals, false)
		c.Assert(item, check.IsNil)
		c.Assert(time.Since(start) < timeout, check.Equals, true)
	}
}
