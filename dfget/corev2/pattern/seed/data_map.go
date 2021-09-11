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

package seed

import (
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/config"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"

	"github.com/pkg/errors"
)

type dataMap struct {
	*syncmap.SyncMap
}

func newDataMap() *dataMap {
	return &dataMap{
		syncmap.NewSyncMap(),
	}
}

func (dm *dataMap) add(key string, value interface{}) error {
	return dm.Add(key, value)
}

func (dm *dataMap) remove(key string) error {
	return dm.Remove(key)
}

func (dm *dataMap) getAsTaskState(key string) (*taskState, error) {
	if stringutils.IsEmptyStr(key) {
		return nil, errors.Wrap(errortypes.ErrEmptyValue, "taskID")
	}

	v, err := dm.Get(key)
	if err != nil {
		return nil, err
	}

	if ts, ok := v.(*taskState); ok {
		return ts, nil
	}

	return nil, errors.Wrapf(errortypes.ErrConvertFailed, "key %s: %v", key, v)
}

func (dm *dataMap) getAsNode(key string) (*config.Node, error) {
	if stringutils.IsEmptyStr(key) {
		return nil, errors.Wrap(errortypes.ErrEmptyValue, "taskID")
	}

	v, err := dm.Get(key)
	if err != nil {
		return nil, err
	}

	if ts, ok := v.(*config.Node); ok {
		return ts, nil
	}

	return nil, errors.Wrapf(errortypes.ErrConvertFailed, "key %s: %v", key, v)
}

func (dm *dataMap) getAsLocalTaskState(key string) (*localTaskState, error) {
	if stringutils.IsEmptyStr(key) {
		return nil, errors.Wrap(errortypes.ErrEmptyValue, "taskID")
	}

	v, err := dm.Get(key)
	if err != nil {
		return nil, err
	}

	if lts, ok := v.(*localTaskState); ok {
		return lts, nil
	}

	return nil, errors.Wrapf(errortypes.ErrConvertFailed, "key %s: %v", key, v)
}
