package util

import (
	"strconv"
	"sync"

	errorType "github.com/dragonflyoss/Dragonfly/common/errors"

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

// Add add a key-value pair into the *sync.Map.
// The ErrEmptyValue error will be returned if the key is empty.
func (mmap *SyncMap) Add(key string, value interface{}) error {
	if IsEmptyStr(key) {
		return errors.Wrap(errorType.ErrEmptyValue, "key")
	}
	mmap.Store(key, value)
	return nil
}

// Get returns result as interface{} according to the key.
// The ErrEmptyValue error will be returned if the key is empty.
// And the ErrDataNotFound error will be returned if the key cannot be found.
func (mmap *SyncMap) Get(key string) (interface{}, error) {
	if IsEmptyStr(key) {
		return nil, errors.Wrap(errorType.ErrEmptyValue, "key")
	}

	if v, ok := mmap.Load(key); ok {
		return v, nil
	}

	return nil, errors.Wrapf(errorType.ErrDataNotFound, "key: %s", key)
}

// GetAsBitset returns result as *bitset.BitSet.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsBitset(key string) (*bitset.BitSet, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return nil, errors.Wrapf(err, "key: %s", key)
	}

	if value, ok := v.(*bitset.BitSet); ok {
		return value, nil
	}
	return nil, errors.Wrapf(errorType.ErrConvertFailed, "key %s: %v", key, v)
}

// GetAsMap returns result as SyncMap.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsMap(key string) (*SyncMap, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return nil, errors.Wrapf(err, "key: %s", key)
	}

	if value, ok := v.(*SyncMap); ok {
		return value, nil
	}
	return nil, errors.Wrapf(errorType.ErrConvertFailed, "key %s: %v", key, v)
}

// GetAsInt returns result as int.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsInt(key string) (int, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return 0, errors.Wrapf(err, "key: %s", key)
	}

	if value, ok := v.(int); ok {
		return value, nil
	}
	return 0, errors.Wrapf(errorType.ErrConvertFailed, "key %s: %v", key, v)
}

// GetAsString returns result as string.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsString(key string) (string, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return "", errors.Wrapf(err, "key: %s", key)
	}

	if value, ok := v.(string); ok {
		return value, nil
	}
	return "", errors.Wrapf(errorType.ErrConvertFailed, "key %s: %v", key, v)
}

// GetAsBool returns result as bool.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsBool(key string) (bool, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return false, errors.Wrapf(err, "key: %s", key)
	}

	if value, ok := v.(bool); ok {
		return value, nil
	}
	return false, errors.Wrapf(errorType.ErrConvertFailed, "key %s: %v", key, v)
}

// GetAsAtomicInt returns result as *AtomicInt.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *SyncMap) GetAsAtomicInt(key string) (*AtomicInt, error) {
	v, err := mmap.Get(key)
	if err != nil {
		return nil, errors.Wrapf(err, "key: %s", key)
	}

	if value, ok := v.(*AtomicInt); ok {
		return value, nil
	}
	return nil, errors.Wrapf(errorType.ErrConvertFailed, "key %s: %v", key, v)
}

// Remove deletes the key-value pair from the mmap.
// The ErrEmptyValue error will be returned if the key is empty.
// And the ErrDataNotFound error will be returned if the key cannot be found.
func (mmap *SyncMap) Remove(key string) error {
	if IsEmptyStr(key) {
		return errors.Wrap(errorType.ErrEmptyValue, "key")
	}

	if _, ok := mmap.Load(key); !ok {
		return errors.Wrapf(errorType.ErrDataNotFound, "key: %s", key)
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

// ListKeyAsIntSlice returns the list of keys as a int slice.
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
