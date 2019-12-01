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

package client

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// CommonAPIClient defines common methods of api client.
type CommonAPIClient interface {
	PreheatAPIClient
	PeerAPIClient
	TaskAPIClient
}

// PreheatAPIClient defines methods of Container client.
type PreheatAPIClient interface {
	PreheatCreate(ctx context.Context, request *types.PreheatCreateRequest) (preheatCreateResponse *types.PreheatCreateResponse, err error)
	PreheatInfo(ctx context.Context, id string) (preheatInfoResponse *types.PreheatInfo, err error)
}

// PeerAPIClient defines methods of peer related client.
type PeerAPIClient interface {
	PeerCreate(ctx context.Context, request *types.PeerCreateRequest) (peerCreateResponse *types.PeerCreateResponse, err error)
	PeerDelete(ctx context.Context, id string) error
	PeerInfo(ctx context.Context, id string) (peerInfoResponse *types.PeerInfo, err error)
	PeerList(ctx context.Context, id string) (peersInfoResponse []*types.PeerInfo, err error)
}

// TaskAPIClient defines methods of task related client.
type TaskAPIClient interface {
	TaskCreate(ctx context.Context, request *types.TaskCreateRequest) (taskCreateResponse *types.TaskCreateResponse, err error)
	TaskDelete(ctx context.Context, id string) error
	TaskInfo(ctx context.Context, id string) (taskInfoResponse *types.TaskInfo, err error)
	TaskUpdate(ctx context.Context, id string, config *types.TaskUpdateRequest) error
}
