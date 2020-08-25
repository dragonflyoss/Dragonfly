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
	"github.com/sirupsen/logrus"
	"net/http"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	dferr "github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

var _ mgr.PreheatManager = &Manager{}

// Manager is an implementation of interface PreheatManager.
type Manager struct {
	service *PreheatService
}

func NewManager(cfg *config.Config) (mgr.PreheatManager, error) {
	return &Manager{service: NewPreheatService(cfg.HomeDir)}, nil
}

func (m *Manager) Create(ctx context.Context, task *types.PreheatCreateRequest) (preheatID string, err error) {
	preheatTask := new(mgr.PreheatTask)
	preheatTask.Type = *task.Type
	preheatTask.URL = *task.URL
	preheatTask.Filter = task.Filter
	preheatTask.Identifier = task.Identifier
	preheatTask.Headers = task.Headers
	logrus.Debugf("create preheat: Type[%s] URL[%s] Filter[%s] Identifier[%s] Headers[%v]",
		preheatTask.Type, preheatTask.URL, preheatTask.Filter, preheatTask.Identifier, preheatTask.Headers)
	return m.service.Create(preheatTask)
}

func (m *Manager) Get(ctx context.Context, preheatID string) (preheatTask *mgr.PreheatTask, err error) {
	preheatTask = m.service.Get(preheatID)
	if preheatTask == nil {
		err = dferr.New(http.StatusNotFound, preheatID+" doesn't exists")
	}
	return
}

func (m *Manager) Delete(ctx context.Context, preheatID string) (err error) {
	m.service.Delete(preheatID)
	return nil
}

func (m *Manager) GetAll(ctx context.Context) (preheatTasks []*mgr.PreheatTask, err error) {
	preheatTasks = m.service.GetAll()
	return
}
