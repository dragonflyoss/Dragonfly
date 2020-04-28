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

package mgr

import (
	"context"
	"net/http"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/seedtask"
)

type SeedTaskMgr interface {
	Register(ctx context.Context, request *types.TaskRegisterRequest) (*seedtask.TaskRegistryResponce, error)

	GetTasksInfo(ctx context.Context, taskID string) ([]*seedtask.SeedInfo, error)

	DeRegisterTask(ctx context.Context, peerID, taskID string) error

	DeRegisterPeer(ctx context.Context, peerID string) error

	EvictTask(ctx context.Context, taskID string) error

	HasTasks(ctx context.Context, taskID string) bool

	ReportPeerHealth(ctx context.Context, peerID string) (*types.HeartBeatResponse, error)

	ScanDownPeers(ctx context.Context) []string

	IsSeedTask(ctx context.Context, request *http.Request) bool
}
