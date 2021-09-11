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
)

type SeedTaskInfo struct {
	Load     int64 // load of peer
	PeerInfo *types.PeerInfo
	TaskInfo *types.TaskFetchInfo
}

type SeedRegisterResponse struct {
	SeedTaskID string
	AsSeed     bool
}

type SeedTaskMgr interface {
	// register seed task & peer
	Register(ctx context.Context, taskReq *types.TaskRegisterRequest) (*SeedRegisterResponse, error)
	// get all seed info of specific task
	GetTaskInfo(ctx context.Context, taskID string) ([]*SeedTaskInfo, error)
	// peer didn't cache specific task anymore
	DeRegisterTask(ctx context.Context, peerID, taskID string) error
	// peer down, remove its all resources
	DeRegisterPeer(ctx context.Context, peerID string) error
	// remove specific task from cluster
	EvictTask(ctx context.Context, taskID string) error
	// whether task in cluster
	HasTasks(ctx context.Context, taskID string) bool
	// peer report health status
	ReportPeerHealth(ctx context.Context, peerID string) (*types.HeartBeatResponse, error)
	// list all peers which were already down
	ScanDownPeers(ctx context.Context) []string
	// tell whether is a seed task request from http header
	IsSeedTask(ctx context.Context, request *http.Request) bool
}
