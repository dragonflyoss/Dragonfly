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

package locator

import (
	"sync"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
)

const (
	// GroupDefaultName indicates all the supernodes in this group are set as
	// default value. It's only used when the supernode list from configuration
	// or CLI is empty.
	GroupDefaultName = "default"

	// GroupConfigName indicates all the supernodes in this group come from
	// configuration or CLI.
	GroupConfigName = "config"
)

var mutex sync.Mutex

// DefaultBuilder returns a supernode locator with default value when supernodes
// in the config file is empty.
var DefaultBuilder Builder = func(cfg *config.Config) SupernodeLocator {
	if cfg == nil || len(cfg.Nodes) == 0 {
		return NewStaticLocator(GroupDefaultName, config.GetDefaultSupernodesValue())
	}
	locator, _ := NewStaticLocatorFromStr(GroupConfigName, cfg.Nodes)
	return locator
}

// Builder defines the constructor of SupernodeLocator.
type Builder func(cfg *config.Config) SupernodeLocator

// CreateLocator creates a supernode locator with the giving config.
func CreateLocator(cfg *config.Config) SupernodeLocator {
	mutex.Lock()
	defer mutex.Unlock()
	return DefaultBuilder(cfg)
}

// RegisterLocator provides a way for users to customize SupernodeLocator.
// This function should be invoked before CreateLocator.
func RegisterLocator(builder Builder) {
	mutex.Lock()
	defer mutex.Unlock()
	DefaultBuilder = builder
}
