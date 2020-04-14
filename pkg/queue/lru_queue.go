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

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
)

// cQElementData is the value of list.Element.Value.
// It records the key and data of item.
type cQElementData struct {
	key  string
	data interface{}
}

// LRUQueue is implementation of LRU.
type LRUQueue struct {
	lock     sync.Mutex
	capacity int

	itemMap map[string]*list.Element
	l       *list.List
}

func NewLRUQueue(capacity int) *LRUQueue {
	return &LRUQueue{
		capacity: capacity,
		itemMap:  make(map[string]*list.Element, capacity),
		l:        list.New(),
	}
}

// Put puts item to front, return the obsolete item
func (q *LRUQueue) Put(key string, data interface{}) (obsoleteKey string, obsoleteData interface{}) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if i, ok := q.itemMap[key]; ok {
		i.Value.(*cQElementData).data = data
		q.putAtFront(i)
		return
	}

	if len(q.itemMap) >= q.capacity {
		// remove the earliest item
		i := q.removeFromTail()
		if i != nil {
			delete(q.itemMap, i.Value.(*cQElementData).key)
			obsoleteKey = i.Value.(*cQElementData).key
			obsoleteData = i.Value.(*cQElementData).data
		}
	}

	i := q.putValue(&cQElementData{key: key, data: data})
	q.itemMap[key] = i
	return
}

// Get will return the item by key. And it will put the item to front.
func (q *LRUQueue) Get(key string) (interface{}, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	data, exist := q.itemMap[key]
	if !exist {
		return nil, errortypes.ErrDataNotFound
	}

	q.putAtFront(data)

	return data.Value.(*cQElementData).data, nil
}

// GetFront will get several items from front and not poll out them.
func (q *LRUQueue) GetFront(count int) []interface{} {
	if count <= 0 {
		return nil
	}

	q.lock.Lock()
	defer q.lock.Unlock()

	result := make([]interface{}, count)
	item := q.l.Front()
	index := 0
	for {
		if item == nil {
			break
		}

		result[index] = item.Value.(*cQElementData).data
		index++
		if index >= count {
			break
		}

		item = item.Next()
	}

	return result[:index]
}

// GetItemByKey will return the item by key. But it will not put the item to front.
func (q *LRUQueue) GetItemByKey(key string) (interface{}, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if data, exist := q.itemMap[key]; exist {
		return data.Value.(*cQElementData).data, nil
	}

	return nil, errortypes.ErrDataNotFound
}

// Delete deletes the item by key, return the deleted item if item exists.
func (q *LRUQueue) Delete(key string) interface{} {
	q.lock.Lock()
	defer q.lock.Unlock()

	data, exist := q.itemMap[key]
	if !exist {
		return nil
	}

	retData := data.Value.(*cQElementData).data
	delete(q.itemMap, key)
	q.removeElement(data)

	return retData
}

func (q *LRUQueue) putAtFront(i *list.Element) {
	q.l.MoveToFront(i)
}

func (q *LRUQueue) putValue(data interface{}) *list.Element {
	e := q.l.PushFront(data)
	return e
}

func (q *LRUQueue) removeFromTail() *list.Element {
	e := q.l.Back()
	q.l.Remove(e)

	return e
}

func (q *LRUQueue) removeElement(i *list.Element) {
	q.l.Remove(i)
}
