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

package seedtask

import (
	"context"
	"net/http"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	seedmgr "github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

type Manager struct {
	Version string
}

func NewManager() (*Manager, error) {
	return &Manager{
		Version: time.Now().String(),
	}, nil
}

func (mgr *Manager) Register(ctx context.Context, taskReq *types.TaskRegisterRequest) (*seedmgr.SeedRegisterResponse, error) {
	return &seedmgr.SeedRegisterResponse{
		SeedTaskID: "",
		AsSeed:     false,
	}, nil
}

func (mgr *Manager) GetTaskInfo(ctx context.Context, taskID string) ([]*seedmgr.SeedTaskInfo, error) {
	return nil, nil
}

func (mgr *Manager) DeRegisterTask(ctx context.Context, peerID, taskID string) error {
	return nil
}

func (mgr *Manager) DeRegisterPeer(ctx context.Context, peerID string) error {
	return nil
}

func (mgr *Manager) EvictTask(ctx context.Context, taskID string) error {
	return nil
}

func (mgr *Manager) HasTasks(ctx context.Context, taskID string) bool {
	return false
}

func (mgr *Manager) ReportPeerHealth(ctx context.Context, peerID string) (*types.HeartBeatResponse, error) {
	return &types.HeartBeatResponse{
		NeedRegister: false,
		SeedTaskIds:  nil,
		Version:      mgr.Version,
	}, nil
}

func (mgr *Manager) ScanDownPeers(ctx context.Context) []string {
	return nil
}

func (mgr *Manager) IsSeedTask(ctx context.Context, request *http.Request) bool {
	return false
}
