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

package plugins

import (
	"sync"

	"github.com/dragonflyoss/Dragonfly/supernode/config"
)

// NewManager creates a default plugin manager instant.
func NewManager() Manager {
	return &managerIml{
		builders: NewRepository(),
		plugins:  NewRepository(),
	}
}

// NewRepository creates a default repository instant.
func NewRepository() Repository {
	return &repositoryIml{
		repos: make(map[config.PluginType]*sync.Map),
	}
}

// Manager manages all plugin builders and plugin instants.
type Manager interface {
	// GetBuilder adds a Builder object with the giving plugin type and name.
	AddBuilder(pt config.PluginType, name string, b Builder)

	// GetBuilder returns a Builder object with the giving plugin type and name.
	GetBuilder(pt config.PluginType, name string) Builder

	// DeleteBuilder deletes a builder with the giving plugin type and name.
	DeleteBuilder(pt config.PluginType, name string)

	// AddPlugin adds a plugin into this manager.
	AddPlugin(p Plugin)

	// GetPlugin returns a plugin with the giving plugin type and name.
	GetPlugin(pt config.PluginType, name string) Plugin

	// DeletePlugin deletes a plugin with the giving plugin type and name.
	DeletePlugin(pt config.PluginType, name string)
}

// Plugin defines methods that plugins need to implement.
type Plugin interface {
	// Type returns the type of this plugin.
	Type() config.PluginType

	// Name returns the name of this plugin.
	Name() string
}

// Builder is a function that creates a new plugin instant with the giving conf.
type Builder func(conf string) (Plugin, error)

// Repository stores data related to plugin.
type Repository interface {
	// Add adds a data to this repository.
	Add(pt config.PluginType, name string, data interface{})

	// Get gets a data with the giving type and name from this
	// repository.
	Get(pt config.PluginType, name string) interface{}

	// Delete deletes a data with the giving type and name from
	// this repository.
	Delete(pt config.PluginType, name string)
}

// -----------------------------------------------------------------------------
// implementation of Manager

type managerIml struct {
	builders Repository
	plugins  Repository
}

var _ Manager = (*managerIml)(nil)

func (m *managerIml) AddBuilder(pt config.PluginType, name string, b Builder) {
	if b == nil {
		return
	}
	m.builders.Add(pt, name, b)
}

func (m *managerIml) GetBuilder(pt config.PluginType, name string) Builder {
	data := m.builders.Get(pt, name)
	if data == nil {
		return nil
	}
	if builder, ok := data.(Builder); ok {
		return builder
	}
	return nil
}

func (m *managerIml) DeleteBuilder(pt config.PluginType, name string) {
	m.builders.Delete(pt, name)
}

func (m *managerIml) AddPlugin(p Plugin) {
	if p == nil {
		return
	}
	m.plugins.Add(p.Type(), p.Name(), p)
}

func (m *managerIml) GetPlugin(pt config.PluginType, name string) Plugin {
	data := m.plugins.Get(pt, name)
	if data == nil {
		return nil
	}
	if plugin, ok := data.(Plugin); ok {
		return plugin
	}
	return nil
}

func (m *managerIml) DeletePlugin(pt config.PluginType, name string) {
	m.plugins.Delete(pt, name)
}

// -----------------------------------------------------------------------------
// implementation of Repository

type repositoryIml struct {
	repos map[config.PluginType]*sync.Map
	lock  sync.Mutex
}

var _ Repository = (*repositoryIml)(nil)

func (r *repositoryIml) Add(pt config.PluginType, name string, data interface{}) {
	if data == nil || !validate(pt, name) {
		return
	}

	m := r.getRepo(pt)
	m.Store(name, data)
}

func (r *repositoryIml) Get(pt config.PluginType, name string) interface{} {
	if !validate(pt, name) {
		return nil
	}

	m := r.getRepo(pt)
	if v, ok := m.Load(name); ok && v != nil {
		return v
	}
	return nil
}

func (r *repositoryIml) Delete(pt config.PluginType, name string) {
	if !validate(pt, name) {
		return
	}
	m := r.getRepo(pt)
	m.Delete(name)
}

func (r *repositoryIml) getRepo(pt config.PluginType) *sync.Map {
	var (
		m  *sync.Map
		ok bool
	)
	if m, ok = r.repos[pt]; ok && m != nil {
		return m
	}

	r.lock.Lock()
	if m, ok = r.repos[pt]; !ok || m == nil {
		m = &sync.Map{}
		r.repos[pt] = m
	}
	r.lock.Unlock()
	return m
}

// -----------------------------------------------------------------------------
// helper functions

func validate(pt config.PluginType, name string) bool {
	if name == "" {
		return false
	}
	for i := len(config.PluginTypes) - 1; i >= 0; i-- {
		if pt == config.PluginTypes[i] {
			return true
		}
	}
	return false
}
