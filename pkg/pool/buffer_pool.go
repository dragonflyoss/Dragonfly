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
	"bytes"
	"io"
	"sync"
)

const (
	// defaultAllocatedSize 2MB
	defaultAllocatedSize int = 2 * 1024 * 1024

	maxPoolCount = 8
	minPoolCount = 1
)

// bufferPools is the default pool which has 4 intervals of buffers'
// initial capacity:
//   (0,2MB], (2MB, 4MB], (4MB, 8MB], (8MB, +∞)
var bufferPool = NewBufferPool(4, defaultAllocatedSize)

// AcquireBufferSize returns an empty Buffer instance from buffer pool,
// whose capacity is greater than or equal to the giving size.
//
// The returned Buffer instance may be passed to ReleaseBuffer when it is
// no longer needed. This allows Buffer recycling, reduces GC pressure
// and usually improves performance.
//
// This function is recommended when you know how much memory you need
// actually.
func AcquireBufferSize(size int) *Buffer {
	if buf := bufferPool.Get(size); buf != nil {
		return buf
	}
	return NewBuffer(size)
}

// AcquireBuffer returns an empty Buffer instance from buffer pool.
//
// The returned Buffer instance may be passed to ReleaseBuffer when it is
// no longer needed. This allows Buffer recycling, reduces GC pressure
// and usually improves performance.
func AcquireBuffer() *Buffer {
	return AcquireBufferSize(defaultAllocatedSize)
}

// ReleaseBuffer returns buf acquired via AcquireBuffer/AcquireBufferSize
// to buffer pool.
//
// It is forbidden accessing buf and/or its' members after returning
// it to buffer pool.
func ReleaseBuffer(buf *Buffer) {
	if buf == nil {
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}

// ----------------------------------------------------------------------------
// struct BufferPool

func NewBufferPool(count, base int) *BufferPool {
	if count < minPoolCount {
		count = minPoolCount
	} else if count > maxPoolCount {
		count = maxPoolCount
	}
	if base <= 0 {
		base = defaultAllocatedSize
	}
	pool := &BufferPool{
		pools:    make([]sync.Pool, count),
		baseSize: base,
	}
	return pool
}

// BufferPool stores several intervals of buffer's initial capacity, which can
// minimizing the allocation times by bytes.Buffer.grow(n int).
//
// It groups the scenarios of using buffer, tries to avoid a large buffer not
// being recycling because it's used by callers which only need a small one.
type BufferPool struct {
	pools    []sync.Pool
	baseSize int
}

// Get returns a buffer with a capacity from the buffer pool.
func (bp *BufferPool) Get(size int) *Buffer {
	idx := bp.index(size)
	if buf := bp.pools[idx].Get(); buf != nil {
		return buf.(*Buffer)
	}
	return nil
}

// Put puts the buf to the buffer pool.
func (bp *BufferPool) Put(buf *Buffer) {
	if buf != nil {
		idx := bp.index(buf.allocatedSize)
		bp.pools[idx].Put(buf)
	}
}

// index finds the first index of pool whose buffer's capacity is greater than
// the giving capacity.
func (bp *BufferPool) index(allocatedSize int) int {
	i, length := 0, len(bp.pools)
	for c := bp.baseSize; i < length && c < allocatedSize; c *= 2 {
		i++
	}
	if i < length {
		return i
	}
	return length - 1
}

// ----------------------------------------------------------------------------
// struct Buffer

// NewBuffer creates a new Buffer which initialized by empty content.
func NewBuffer(size int) *Buffer {
	return &Buffer{
		Buffer:        bytes.NewBuffer(make([]byte, 0, size)),
		allocatedSize: size,
	}
}

// NewBufferString creates and initializes a new Buffer using string s as its
// content.
func NewBufferString(s string) *Buffer {
	return &Buffer{
		Buffer:        bytes.NewBufferString(s),
		allocatedSize: len(s),
	}
}

var (
	_ io.ReaderFrom      = &Buffer{}
	_ io.WriterTo        = &Buffer{}
	_ io.ReadWriteCloser = &Buffer{}
)

// Buffer provides byte buffer, which can be used for minimizing memory
// allocations and implements interfaces: io.ReaderFrom, io.WriterTo,
// io.ReadWriteCloser.
//
// The allocatedSize is the buffer's initial size.
type Buffer struct {
	*bytes.Buffer
	allocatedSize int
}

func (b *Buffer) Close() error {
	b.Reset()
	return nil
}
