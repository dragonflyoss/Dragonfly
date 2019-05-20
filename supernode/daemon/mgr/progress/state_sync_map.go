package progress

import (
	"sync"

	errorType "github.com/dragonflyoss/Dragonfly/common/errors"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"

	"github.com/pkg/errors"
)

// stateSyncMap is a thread-safe map.
type stateSyncMap struct {
	*sync.Map
}

// newStateSyncMap returns a new stateSyncMap.
func newStateSyncMap() *stateSyncMap {
	return &stateSyncMap{&sync.Map{}}
}

// add a key-value pair into the *sync.Map.
// The ErrEmptyValue error will be returned if the key is empty.
func (mmap *stateSyncMap) add(key string, value interface{}) error {
	if cutil.IsEmptyStr(key) {
		return errors.Wrap(errorType.ErrEmptyValue, "key")
	}
	mmap.Store(key, value)
	return nil
}

// get returns result as interface{} according to the key.
// The ErrEmptyValue error will be returned if the key is empty.
// And the ErrDataNotFound error will be returned if the key cannot be found.
func (mmap *stateSyncMap) get(key string) (interface{}, error) {
	if cutil.IsEmptyStr(key) {
		return nil, errors.Wrap(errorType.ErrEmptyValue, "key")
	}

	if v, ok := mmap.Load(key); ok {
		return v, nil
	}

	return nil, errors.Wrapf(errorType.ErrDataNotFound, "key: %s", key)
}

// getAsSuperState returns result as *superState.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *stateSyncMap) getAsSuperState(key string) (*superState, error) {
	v, err := mmap.get(key)
	if err != nil {
		return nil, errors.Wrapf(err, "key: %s", key)
	}

	if value, ok := v.(*superState); ok {
		return value, nil
	}
	return nil, errors.Wrapf(errorType.ErrConvertFailed, "key %s: %v", key, v)
}

// getAsClientState returns result as *clientState.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *stateSyncMap) getAsClientState(key string) (*clientState, error) {
	v, err := mmap.get(key)
	if err != nil {
		return nil, errors.Wrapf(err, "key: %s", key)
	}

	if value, ok := v.(*clientState); ok {
		return value, nil
	}
	return nil, errors.Wrapf(errorType.ErrConvertFailed, "key %s: %v", key, v)
}

// getAsPeerState returns result as *peerState.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *stateSyncMap) getAsPeerState(key string) (*peerState, error) {
	v, err := mmap.get(key)
	if err != nil {
		return nil, errors.Wrapf(err, "key: %s", key)
	}

	if value, ok := v.(*peerState); ok {
		return value, nil
	}
	return nil, errors.Wrapf(errorType.ErrConvertFailed, "key %s: %v", key, v)
}

// getAsPieceState returns result as *pieceState.
// The ErrConvertFailed error will be returned if the assertion fails.
func (mmap *stateSyncMap) getAsPieceState(key string) (*pieceState, error) {
	v, err := mmap.get(key)
	if err != nil {
		return nil, errors.Wrapf(err, "key: %s", key)
	}

	if value, ok := v.(*pieceState); ok {
		return value, nil
	}
	return nil, errors.Wrapf(errorType.ErrConvertFailed, "key %s: %v", key, v)
}

// remove deletes the key-value pair from the mmap.
// The ErrEmptyValue error will be returned if the key is empty.
// And the ErrDataNotFound error will be returned if the key cannot be found.
func (mmap *stateSyncMap) remove(key string) error {
	if cutil.IsEmptyStr(key) {
		return errors.Wrap(errorType.ErrEmptyValue, "key")
	}

	if _, ok := mmap.Load(key); !ok {
		return errors.Wrapf(errorType.ErrDataNotFound, "key: %s", key)
	}

	mmap.Delete(key)
	return nil
}
