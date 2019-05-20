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

package store

import (
	"fmt"

	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/plugins"
)

// StorageBuilder is a function that creates a new storage plugin instant
// with the giving conf.
type StorageBuilder func(conf string) (StorageDriver, error)

// Register defines an interface to register a driver with specified name.
// All drivers should call this function to register itself to the driverFactory.
func Register(name string, builder StorageBuilder) {
	var f plugins.Builder = func(conf string) (plugin plugins.Plugin, e error) {
		return NewStore(name, builder, conf)
	}
	plugins.RegisterPlugin(config.StoragePlugin, name, f)
}

// Manager manage stores.
type Manager struct {
}

// NewManager create a store manager.
func NewManager() (*Manager, error) {
	return &Manager{}, nil
}

// Get a store from manager with specified name.
func (sm *Manager) Get(name string) (*Store, error) {
	v := plugins.GetPlugin(config.StoragePlugin, name)
	if v == nil {
		return nil, fmt.Errorf("not existed storage: %s", name)
	}
	if store, ok := v.(*Store); ok {
		return store, nil
	}
	return nil, fmt.Errorf("get store error: unknown reason")
}
