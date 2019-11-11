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

package util

import (
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"
)

// Store maintains some metadata information in memory.
type Store struct {
	*syncmap.SyncMap
}

// NewStore returns a new Store.
func NewStore() *Store {
	return &Store{syncmap.NewSyncMap()}
}

// Put a key-value pair into the store.
func (s *Store) Put(key string, value interface{}) error {
	return s.Add(key, value)
}

// Delete a key-value pair from the store with specified key.
func (s *Store) Delete(key string) error {
	return s.Remove(key)
}

// List returns all key-value pairs in the store.
// And the order of results is random.
func (s *Store) List() []interface{} {
	metaSlice := make([]interface{}, 0)
	rangeFunc := func(key, value interface{}) bool {
		metaSlice = append(metaSlice, value)
		return true
	}
	s.Range(rangeFunc)

	return metaSlice
}
