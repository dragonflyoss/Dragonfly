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
	"sort"
	"sync"

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
	sortedSet         []uint64
	replicationPerKey int
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
		sortedSet:         []uint64{},
		replicationPerKey: 16,
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
		h.sortedSet = append(h.sortedSet, m)
	}

	// sort hashes ascendingly
	sort.Slice(h.sortedSet, func(i int, j int) bool {
		if h.sortedSet[i] < h.sortedSet[j] {
			return true
		}
		return false
	})

	return
}

func (h *consistentHashCircler) Hash(input string) (key string, err error) {
	h.RLock()
	defer h.RUnlock()

	if len(h.keysMap) == 0 {
		return "", ErrKeyNotPresent
	}

	hashN := h.hashFunc(input)
	index := h.search(hashN)

	return h.keysMap[h.sortedSet[index]], nil
}

func (h *consistentHashCircler) Delete(key string) {
	h.Lock()
	defer h.Unlock()

	for i := 0; i < h.replicationPerKey; i++ {
		m := h.hashFunc(fmt.Sprintf("%s-%d", key, i))
		delete(h.keysMap, m)
		h.delSlice(m)
	}

	return
}

func (h *consistentHashCircler) search(key uint64) int {
	idx := sort.Search(len(h.sortedSet), func(i int) bool {
		return h.sortedSet[i] >= key
	})

	if idx >= len(h.sortedSet) {
		idx = 0
	}
	return idx
}

func (h *consistentHashCircler) delSlice(val uint64) {
	idx := -1
	l := 0
	r := len(h.sortedSet) - 1
	for l <= r {
		m := (l + r) / 2
		if h.sortedSet[m] == val {
			idx = m
			break
		} else if h.sortedSet[m] < val {
			l = m + 1
		} else if h.sortedSet[m] > val {
			r = m - 1
		}
	}

	if idx != -1 {
		h.sortedSet = append(h.sortedSet[:idx], h.sortedSet[idx+1:]...)
	}
}

func fnvHashFunc(input string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(input))
	return h.Sum64()
}
