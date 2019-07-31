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
	"container/list"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/dragonflyoss/Dragonfly/pkg/util"
)

// Queue blocking queue. The items putted into queue mustn't be nil.
type Queue interface {
	// Put puts item into the queue and keeps blocking if the queue is full.
	// It will return immediately and do nothing if the item is nil.
	Put(item interface{})

	// PutTimeout puts item into the queue and waits for timeout if the queue is full.
	// If timeout <= 0, it will return false immediately when queue is full.
	// It will return immediately and do nothing if the item is nil.
	PutTimeout(item interface{}, timeout time.Duration) bool

	// Poll gets an item from the queue and keeps blocking if the queue is empty.
	Poll() interface{}

	// PollTimeout gets an item from the queue and waits for timeout if the queue is empty.
	// If timeout <= 0, it will return (nil, bool) immediately when queue is empty.
	PollTimeout(timeout time.Duration) (interface{}, bool)

	// Len returns the current size of the queue.
	Len() int
}

// NewQueue creates a blocking queue.
// If capacity <= 0, the queue capacity is infinite.
func NewQueue(capacity int) Queue {
	if capacity <= 0 {
		c := make(chan struct{})
		return &infiniteQueue{
			store: list.New(),
			empty: unsafe.Pointer(&c),
		}
	}
	return &finiteQueue{
		store: make(chan interface{}, capacity),
	}
}

// infiniteQueue implements infinite blocking queue.
type infiniteQueue struct {
	sync.Mutex
	store *list.List
	empty unsafe.Pointer
}

var _ Queue = &infiniteQueue{}

func (q *infiniteQueue) Put(item interface{}) {
	if util.IsNil(item) {
		return
	}
	q.Lock()
	defer q.Unlock()
	q.store.PushBack(item)
	if q.store.Len() < 2 {
		// empty -> has one element
		q.broadcast()
	}
}

func (q *infiniteQueue) PutTimeout(item interface{}, timeout time.Duration) bool {
	q.Put(item)
	return !util.IsNil(item)
}

func (q *infiniteQueue) Poll() interface{} {
	q.Lock()
	defer q.Unlock()
	for q.store.Len() == 0 {
		q.wait()
	}
	item := q.store.Front()
	q.store.Remove(item)
	return item.Value
}

func (q *infiniteQueue) PollTimeout(timeout time.Duration) (interface{}, bool) {
	deadline := time.Now().Add(timeout)
	q.Lock()
	defer q.Unlock()
	for q.store.Len() == 0 {
		timeout = -time.Since(deadline)
		if timeout <= 0 || !q.waitTimeout(timeout) {
			return nil, false
		}
	}
	item := q.store.Front()
	q.store.Remove(item)
	return item.Value, true
}

func (q *infiniteQueue) Len() int {
	q.Lock()
	defer q.Unlock()
	return q.store.Len()
}

func (q *infiniteQueue) wait() {
	c := q.notifyChan()
	q.Unlock()
	defer q.Lock()
	<-c
}

func (q *infiniteQueue) waitTimeout(timeout time.Duration) bool {
	c := q.notifyChan()

	q.Unlock()
	defer q.Lock()
	select {
	case <-c:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (q *infiniteQueue) notifyChan() <-chan struct{} {
	ptr := atomic.LoadPointer(&q.empty)
	return *((*chan struct{})(ptr))
}

// broadcast notifies all the Poll goroutines to re-check whether the queue is empty.
func (q *infiniteQueue) broadcast() {
	c := make(chan struct{})
	old := atomic.SwapPointer(&q.empty, unsafe.Pointer(&c))
	close(*(*chan struct{})(old))
}

// finiteQueue implements finite blocking queue by buffered channel.
type finiteQueue struct {
	store chan interface{}
}

func (q *finiteQueue) Put(item interface{}) {
	if util.IsNil(item) {
		return
	}
	q.store <- item
}

func (q *finiteQueue) PutTimeout(item interface{}, timeout time.Duration) bool {
	if util.IsNil(item) {
		return false
	}
	if timeout <= 0 {
		select {
		case q.store <- item:
			return true
		default:
			return false
		}
	}
	select {
	case q.store <- item:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (q *finiteQueue) Poll() interface{} {
	item := <-q.store
	return item
}

func (q *finiteQueue) PollTimeout(timeout time.Duration) (interface{}, bool) {
	if timeout <= 0 {
		select {
		case item := <-q.store:
			return item, true
		default:
			return nil, false
		}
	}
	select {
	case item := <-q.store:
		return item, true
	case <-time.After(timeout):
		return nil, false
	}
}

func (q *finiteQueue) Len() int {
	return len(q.store)
}
