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

package syncmap

import (
	"strconv"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/atomiccount"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"

	"github.com/pkg/errors"
	"github.com/willf/bitset"
)

// SyncMap is a thread-safe map.
type SyncMap struct {
	*sync.Map
}

// NewSyncMap returns a new SyncMap.
func NewSyncMap() *SyncMap {
	return &SyncMap{&sync.Map{}}
}

// Add adds a key-value pair into the *sync.Map.
// The ErrEmptyValue error will be returned if the key is empty.
func (mmap *SyncMap) Add(key string, value interface{}) error {
	if stringutils.IsEmptyStr(key) {
		return errors.Wrap(errortypes.ErrEmptyValue, "key")
	}
	mmap.Store(key, value)
	return nil
}

// Get returns result as interface{} according to the key.
// The ErrEmptyValue error will be returned if the key is empty.
// And the ErrDataNotFound error will be returned if the key cannot be found.
func (mmap *SyncMap) Get(key string) (interface{}, error) {
	if stringutils.IsEmptyStr(key) {
		return nil, errors.Wrap(errortypes.ErrEmptyValue, "key")
	}

	if v, ok := mmap.Load(key); ok {
		return v, nil
	}

	return nil, errors.Wrapf(errortypes.ErrDataNotFound, "failed to get key %s from map", key)
}

// GetAsBitset returns result as *bitset.BitSet.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsBitset(key string) (*bitset.BitSet, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get key %s from map", key)
	}

	if value, ok := v.(*bitset.BitSet); ok {
		return value, nil
	}
	return nil, errors.Wrapf(errortypes.ErrConvertFailed, "failed to get key %s from map with value %s", key, v)
}

// GetAsMap returns result as SyncMap.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsMap(key string) (*SyncMap, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get key %s from map", key)
	}

	if value, ok := v.(*SyncMap); ok {
		return value, nil
	}
	return nil, errors.Wrapf(errortypes.ErrConvertFailed, "failed to get key %s from map with value %s", key, v)
}

// GetAsInt returns result as int.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsInt(key string) (int, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get key %s from map", key)
	}

	if value, ok := v.(int); ok {
		return value, nil
	}
	return 0, errors.Wrapf(errortypes.ErrConvertFailed, "failed to get key %s from map with value %s", key, v)
}

// GetAsInt64 returns result as int64.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsInt64(key string) (int64, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get key %s from map", key)
	}

	if value, ok := v.(int64); ok {
		return value, nil
	}
	return 0, errors.Wrapf(errortypes.ErrConvertFailed, "failed to get key %s from map with value %s", key, v)
}

// GetAsString returns result as string.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsString(key string) (string, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get key %s from map", key)
	}

	if value, ok := v.(string); ok {
		return value, nil
	}
	return "", errors.Wrapf(errortypes.ErrConvertFailed, "failed to get key %s from map with value %s", key, v)
}

// GetAsBool returns result as bool.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsBool(key string) (bool, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get key %s from map", key)
	}

	if value, ok := v.(bool); ok {
		return value, nil
	}
	return false, errors.Wrapf(errortypes.ErrConvertFailed, "failed to get key %s from map with value %s", key, v)
}

// GetAsAtomicInt returns result as *AtomicInt.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsAtomicInt(key string) (*atomiccount.AtomicInt, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get key %s from map", key)
	}

	if value, ok := v.(*atomiccount.AtomicInt); ok {
		return value, nil
	}
	return nil, errors.Wrapf(errortypes.ErrConvertFailed, "failed to get key %s from map with value %s", key, v)
}

// GetAsTime returns result as Time.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsTime(key string) (time.Time, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return time.Now(), errors.Wrapf(err, "failed to get key %s from map", key)
	}

	if value, ok := v.(time.Time); ok {
		return value, nil
	}
	return time.Now(), errors.Wrapf(errortypes.ErrConvertFailed, "failed to get key %s from map with value %s", key, v)
}

// Remove deletes the key-value pair from the mmap.
// The ErrEmptyValue error will be returned if the key is empty.
// And the ErrDataNotFound error will be returned if the key cannot be found.
func (mmap *SyncMap) Remove(key string) error {
	if stringutils.IsEmptyStr(key) {
		return errors.Wrap(errortypes.ErrEmptyValue, "key")
	}

	if _, ok := mmap.Load(key); !ok {
		return errors.Wrapf(errortypes.ErrDataNotFound, "failed to get key %s from map", key)
	}

	mmap.Delete(key)
	return nil
}

// ListKeyAsStringSlice returns the list of keys as a string slice.
func (mmap *SyncMap) ListKeyAsStringSlice() (result []string) {
	if mmap == nil {
		return []string{}
	}

	rangeFunc := func(key, value interface{}) bool {
		if v, ok := key.(string); ok {
			result = append(result, v)
			return true
		}
		return true
	}

	mmap.Range(rangeFunc)
	return
}

// ListKeyAsIntSlice returns the list of keys as an int slice.
func (mmap *SyncMap) ListKeyAsIntSlice() (result []int) {
	if mmap == nil {
		return []int{}
	}

	rangeFunc := func(key, value interface{}) bool {
		if v, ok := key.(string); ok {
			if value, err := strconv.Atoi(v); err == nil {
				result = append(result, value)
				return true
			}
		}
		return true
	}

	mmap.Range(rangeFunc)
	return
}
