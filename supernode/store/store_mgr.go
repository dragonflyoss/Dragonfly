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
	"sync"
)

var (
	storeMap      sync.Map
	driverFactory = make(map[string]initFunc)
)

type initFunc func(config interface{}) (StorageDriver, error)

// Register defines an interface to register a driver with specified name.
// All drivers should call this function to register itself to the driverFactory.
func Register(name string, initializer initFunc) {
	driverFactory[name] = initializer
}

// Manager manage stores.
type Manager struct {
}

// ManagerConfig wraps the config that defined in the config file.
type ManagerConfig struct {
	driverName   string
	driverConfig interface{}
}

// NewManager create a store manager.
func NewManager(configs map[string]*ManagerConfig) (*Manager, error) {
	if configs == nil {
		return nil, fmt.Errorf("empty configs")
	}
	for name, config := range configs {
		// initialize store
		store, err := NewStore(config.driverName, config.driverConfig)
		if err != nil {
			return nil, err
		}
		storeMap.Store(name, store)
	}
	return &Manager{}, nil
}

// Get a store from manager with specified name.
func (sm *Manager) Get(name string) (*Store, error) {
	v, ok := storeMap.Load(name)
	if !ok {
		return nil, fmt.Errorf("not existed storage: %s", name)
	}
	if store, ok := v.(*Store); ok {
		return store, nil
	}
	return nil, fmt.Errorf("get store error: unknown reason")
}
