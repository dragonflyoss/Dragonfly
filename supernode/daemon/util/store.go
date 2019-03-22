package util

import (
	"sync"

	errorType "github.com/dragonflyoss/Dragonfly/common/errors"

	"github.com/pkg/errors"
)

// Store maintains some metadata information in memory.
type Store struct {
	metaMap sync.Map
}

// NewStore returns a new Store.
func NewStore() *Store {
	return &Store{}
}

// Put a key-value pair into the store.
func (s *Store) Put(key string, value interface{}) error {
	s.metaMap.Store(key, value)
	return nil
}

// Get a key-value pair from the store.
func (s *Store) Get(key string) (interface{}, error) {
	v, ok := s.metaMap.Load(key)
	if !ok {
		return nil, errors.Wrapf(errorType.ErrDataNotFound, "key (%s)", key)
	}

	return v, nil
}

// Delete a key-value pair from the store with sepcified key.
func (s *Store) Delete(key string) error {
	_, ok := s.metaMap.Load(key)
	if !ok {
		return errors.Wrapf(errorType.ErrDataNotFound, "key (%s)", key)
	}

	s.metaMap.Delete(key)

	return nil
}

// List returns all key-value pairs in the store.
// And the order of results is random.
func (s *Store) List() []interface{} {
	metaSlice := make([]interface{}, 0)
	rangeFunc := func(key, value interface{}) bool {
		metaSlice = append(metaSlice, value)
		return true
	}
	s.metaMap.Range(rangeFunc)

	return metaSlice
}
