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
	"path/filepath"
	"sync"

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

// Manager manages stores.
type Manager struct {
	cfg *config.Config

	defaultStorage *Store
	mutex          sync.Mutex
}

// NewManager creates a store manager.
func NewManager(cfg *config.Config) (*Manager, error) {
	return &Manager{
		cfg: cfg,
	}, nil
}

// Get a store from manager with specified name.
func (sm *Manager) Get(name string) (*Store, error) {
	v := plugins.GetPlugin(config.StoragePlugin, name)
	if v == nil {
		if name == LocalStorageDriver {
			return sm.getDefaultStorage()
		}
		return nil, fmt.Errorf("not existed storage: %s", name)
	}
	if store, ok := v.(*Store); ok {
		return store, nil
	}
	return nil, fmt.Errorf("get store error: unknown reason")
}

func (sm *Manager) getDefaultStorage() (*Store, error) {
	if sm.defaultStorage != nil {
		return sm.defaultStorage, nil
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// check again to avoid initializing repeatedly
	if sm.defaultStorage != nil {
		return sm.defaultStorage, nil
	}

	if sm.cfg == nil {
		return nil, fmt.Errorf("cannot init local storage without home path")
	}
	cfg := fmt.Sprintf("baseDir: %s", filepath.Join(sm.cfg.HomeDir, "repo"))
	s, err := NewStore(LocalStorageDriver, NewLocalStorage, cfg)
	if err != nil {
		return nil, err
	}
	sm.defaultStorage = s
	return sm.defaultStorage, nil
}
