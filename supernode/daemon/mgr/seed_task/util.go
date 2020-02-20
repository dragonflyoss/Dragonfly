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

package seed_task

import (
	"sync"
)

type idSet struct {
	lock 	*sync.RWMutex
	set 	map[string]bool
}

func (s *idSet) add (id string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.set[id] = true
}

func (s *idSet) has(id string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, ok := s.set[id]
	return ok
}

func (s *idSet) delete(id string) {
	if !s.has(id) {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.set, id)
}

func (s *idSet) doRange (fn func (k, v interface{})) {
	for k, v := range s.set {
		fn(k, v)
	}
}

func (s *idSet) size() int {
	return len(s.set)
}

func (s *idSet) listWithLimit(maxNumber int) []string {
	i := 0
	result := make([]string, 0)
	rangeFn := func (k, v interface {}) {
		if maxNumber > 0 && i >= maxNumber {
			return
		}
		id, _ := k.(string)
		result = append(result, id)
		i += 1
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	s.doRange(rangeFn)
	return result
}

func (s *idSet) list () []string {
	return s.listWithLimit(0)
}

func newIdSet() *idSet {
	return &idSet{
		set: make(map[string]bool),
		lock: new(sync.RWMutex),
	}
}

type safeMap struct {
	lock  	*sync.RWMutex
	safeMap map[string]string
}

func (m *safeMap) add (key, value string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.safeMap[key] = value
}

func (m *safeMap) remove (key string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.safeMap, key)
}

func (m *safeMap) get (key string) string {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if v, ok := m.safeMap[key]; ok {
		return v
	}
	return ""
}

func newSafeMap() *safeMap {
	return &safeMap{
		lock: 		new(sync.RWMutex),
		safeMap: 	make(map[string]string),
	}
}