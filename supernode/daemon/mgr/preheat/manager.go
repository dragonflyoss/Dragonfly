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

package preheat

import (
	"context"
	"fmt"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

var _ mgr.PreheatManager = &Manager{}

// Manager is an implementation of interface PreheatManager.
type Manager struct {
}

func NewManager(cfg *config.Config) (mgr.PreheatManager, error) {
	return &Manager{}, nil
}

func (m *Manager) Create(ctx context.Context, task *types.PreheatCreateRequest) (preheatID string, err error) {
	return "", fmt.Errorf("not implement")
}

func (m *Manager) Get(ctx context.Context, preheatID string) (preheatTask *mgr.PreheatTask, err error) {
	return nil, fmt.Errorf("not implement")
}

func (m *Manager) Delete(ctx context.Context, preheatID string) (err error) {
	return fmt.Errorf("not implement")
}

func (m *Manager) GetAll(ctx context.Context) (preheatTasks []*mgr.PreheatTask, err error) {
	return nil, fmt.Errorf("not implement")
}
