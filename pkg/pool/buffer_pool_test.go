// BufferPool is no-op under race detector, so all these tests do not work.
// +build !race

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

package pool

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestSuite(t *testing.T) {
	suite.Run(t, new(BufferPoolTestSuite))
}

type BufferPoolTestSuite struct {
	suite.Suite
	tmpBufferPool *BufferPool
}

func (s *BufferPoolTestSuite) SetupSuite() {
	s.tmpBufferPool = bufferPool
	bufferPool = NewBufferPool(8, defaultAllocatedSize)
}

func (s *BufferPoolTestSuite) TearDownSuite() {
	bufferPool = s.tmpBufferPool
	s.tmpBufferPool = nil
}

func (s *BufferPoolTestSuite) TestAcquireBuffer() {
	buf := AcquireBuffer()
	defer ReleaseBuffer(buf)

	s.NotNil(buf)
	s.Equal(0, buf.Len(), "not empty")
}

func (s *BufferPoolTestSuite) TestReleaseBuffer() {
	// Limit to 1 processor to make sure that the goroutine doesn't migrate
	// to another P between AcquireBuffer and ReleaseBuffer calls.
	prev := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(prev)

	buf1 := AcquireBuffer()
	ReleaseBuffer(buf1)
	ReleaseBuffer(nil)

	buf2 := AcquireBuffer()
	buf3 := AcquireBuffer()
	defer func() {
		ReleaseBuffer(buf2)
		ReleaseBuffer(buf3)
		buf2, buf3 = nil, nil
	}()

	s.NotNil(buf1)
	s.NotNil(buf2)
	s.NotNil(buf3)
	s.True(buf1 == buf2, "should reuse an old buffer but got a new one")
	s.True(buf1 != buf3, "should create a new buffer but got an old one")
}

func (s *BufferPoolTestSuite) TestNewBufferPool() {
	cases := []struct {
		count         int
		base          int
		expectedCount int
		expectedBase  int
	}{
		{0, 0, minPoolCount, defaultAllocatedSize},
		{maxPoolCount + 1, 0, maxPoolCount, defaultAllocatedSize},
		{2, 4, 2, 4},
	}

	for i, c := range cases {
		msg := fmt.Sprintf("case %d: %v", i, c)
		p := NewBufferPool(c.count, c.base)
		s.Equal(c.expectedCount, len(p.pools), msg)
		s.Equal(c.expectedBase, p.baseSize, msg)
	}

}

func (s *BufferPoolTestSuite) TestBufferPool_index() {
	count := 2
	base := 4
	p := NewBufferPool(count, base)
	idx := func(i int) int {
		if i < count {
			return i
		}
		return count - 1
	}

	capacity := base
	for i := 0; i <= count; i++ {
		s.Equal(idx(i), p.index(capacity-1))
		s.Equal(idx(i), p.index(capacity))
		s.Equal(idx(i+1), p.index(capacity+1))
		capacity *= 2
	}
	s.Equal(0, p.index(base))
}

func (s *BufferPoolTestSuite) TestBuffer_NewBufferString() {
	buf := NewBufferString("hello")
	s.Equal([]byte("hello"), buf.Bytes())
	buf.Close()
	s.Equal(0, buf.Len())
}

func BenchmarkAcquireBuffer(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := AcquireBuffer()
		buf.WriteString("hello")
		ReleaseBuffer(buf)
	}
}
