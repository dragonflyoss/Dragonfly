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

package hashcircler

import (
	"fmt"
	"hash/fnv"
	"sync"

	"github.com/HuKeping/rbtree"
	"github.com/pkg/errors"
)

// HashCircler hashes input string to target key, and the key could be enabled or disabled.
// And the keys array is preset, only the keys could be enable or disable.
type HashCircler interface {
	// Add adds the target key in hash circle.
	Add(key string)

	// Hash hashes the input and output the target key which hashes.
	Hash(input string) (key string, err error)

	// Delete deletes the target key
	Delete(key string)
}

var (
	ErrKeyNotPresent = errors.New("key is not present")
)

// consistentHashCircler is an implementation of HashCircler. And the keys is preset.
type consistentHashCircler struct {
	sync.RWMutex
	hashFunc func(string) uint64

	keysMap           map[uint64]string
	replicationPerKey int
	rb                *rbtree.Rbtree
}

// NewConsistentHashCircler constructs an instance of HashCircler from keys. And this is thread safety.
// if hashFunc is nil, it will be set to default hash func.
func NewConsistentHashCircler(keys []string, hashFunc func(string) uint64) (HashCircler, error) {
	if hashFunc == nil {
		hashFunc = fnvHashFunc
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("empty keys")
	}

	hc := &consistentHashCircler{
		hashFunc:          hashFunc,
		keysMap:           make(map[uint64]string),
		replicationPerKey: 16,
		rb:                rbtree.New(),
	}

	for _, k := range keys {
		hc.Add(k)
	}

	return hc, nil
}

func (h *consistentHashCircler) Add(key string) {
	h.Lock()
	defer h.Unlock()

	for i := 0; i < h.replicationPerKey; i++ {
		m := h.hashFunc(fmt.Sprintf("%s-%d", key, i))
		if _, exist := h.keysMap[m]; exist {
			continue
		}
		h.keysMap[m] = key
		h.addToRbTree(m, key)
	}

	return
}

func (h *consistentHashCircler) Hash(input string) (key string, err error) {
	h.RLock()
	defer h.RUnlock()

	if len(h.keysMap) == 0 {
		return "", ErrKeyNotPresent
	}

	index := h.hashFunc(input)

	return h.searchFromRbTree(index), nil
}

func (h *consistentHashCircler) Delete(key string) {
	h.Lock()
	defer h.Unlock()

	for i := 0; i < h.replicationPerKey; i++ {
		m := h.hashFunc(fmt.Sprintf("%s-%d", key, i))
		delete(h.keysMap, m)
		h.deleteFromRbTree(m)
	}

	return
}

func fnvHashFunc(input string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(input))
	return h.Sum64()
}

func (h *consistentHashCircler) addToRbTree(index uint64, key string) {
	i := &item{
		index: index,
		key:   key,
	}

	h.rb.Insert(i)
}

func (h *consistentHashCircler) deleteFromRbTree(index uint64) {
	i := &item{
		index: index,
	}

	h.rb.Delete(i)
}

func (h *consistentHashCircler) searchFromRbTree(index uint64) string {
	comp := &item{
		index: index,
	}

	target := ""

	// find the key which index of item greater or equal than input index.
	h.rb.Ascend(comp, func(i rbtree.Item) bool {
		o := i.(*item)
		target = o.key
		return false
	})

	// if not found the target, return the max item.
	if target == "" {
		i := h.rb.Max()
		target = i.(*item).key
	}

	return target
}

type item struct {
	index uint64
	key   string
}

func (i *item) Less(than rbtree.Item) bool {
	other := than.(*item)
	return i.index < other.index
}
