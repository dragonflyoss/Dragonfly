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

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
)

// PieceStatusMap maintains the mapping relationship between PieceUpdateRequestResult and PieceStatus code.
var PieceStatusMap = map[string]int{
	types.PieceUpdateRequestPieceStatusFAILED:  config.PieceFAILED,
	types.PieceUpdateRequestPieceStatusSEMISUC: config.PieceSEMISUC,
	types.PieceUpdateRequestPieceStatusSUCCESS: config.PieceSUCCESS,
}

// TaskMgr as an interface defines all operations against Task.
// A Task will store some meta info about the taskFile, pieces and something else.
// A Task has a one-to-one correspondence with a file on the disk which is identified by taskID.
type TaskMgr interface {
	// Register a task represents that someone wants to download a file.
	// Supernode will get the task file meta and return taskID.
	// NOTE: If supernode cannot find the task file, the CDN download will be triggered.
	Register(ctx context.Context, taskCreateRequest *types.TaskCreateRequest) (taskCreateResponse *types.TaskCreateResponse, err error)

	// Get the task Info with specified taskID.
	Get(ctx context.Context, taskID string) (*types.TaskInfo, error)

	// GetAccessTime gets all task accessTime.
	GetAccessTime(ctx context.Context) (*syncmap.SyncMap, error)

	// List returns the list tasks with filter.
	List(ctx context.Context, filter map[string]string) ([]*types.TaskInfo, error)

	// CheckTaskStatus checks whether the taskID corresponding file exists.
	CheckTaskStatus(ctx context.Context, taskID string) (bool, error)

	// Delete deletes a task.
	Delete(ctx context.Context, taskID string) error

	// Update updates the task info with specified info.
	// In common, there are several situations that we will use this method:
	// 1. when finished to download, update task status.
	// 2. for operation usage.
	// TODO: define a struct of TaskUpdateRequest?
	Update(ctx context.Context, taskID string, taskInfo *types.TaskInfo) error

	// GetPieces gets the pieces to be downloaded based on the scheduling result,
	// just like this: which pieces can be downloaded from which peers.
	GetPieces(ctx context.Context, taskID, clientID string, piecePullRequest *types.PiecePullRequest) (isFinished bool, data interface{}, err error)

	// UpdatePieceStatus updates the piece status with specified parameters.
	// A task file is divided into several pieces logically.
	// We use a sting called pieceRange to identify a piece.
	// A pieceRange is separated by a dash, like this: 0-45565, etc.
	UpdatePieceStatus(ctx context.Context, taskID, pieceRange string, pieceUpdateRequest *types.PieceUpdateRequest) error
}
