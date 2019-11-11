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
)

// CDNMgr as an interface defines all operations against CDN and
// operates on the underlying files stored on the local disk, etc.
type CDNMgr interface {
	// TriggerCDN will trigger CDN to download the file from sourceUrl.
	// It includes the following steps:
	// 1). download the source file
	// 2). write the file to disk
	//
	// In fact, it's a very time consuming operation.
	// So if not necessary, it should usually be executed concurrently.
	// In addition, it's not thread-safe.
	TriggerCDN(ctx context.Context, taskInfo *types.TaskInfo) (*types.TaskInfo, error)

	// GetHTTPPath returns the http download path of taskID.
	GetHTTPPath(ctx context.Context, taskID string) (path string, err error)

	// GetStatus gets the status of the file.
	GetStatus(ctx context.Context, taskID string) (cdnStatus string, err error)

	// GetGCTaskIDs returns the taskIDs that should exec GC operations as a string slice.
	//
	// It should return nil when the free disk of cdn storage is lager than config.YoungGCThreshold.
	// It should return all taskIDs that are not running when the free disk of cdn storage is less than config.FullGCThreshold.
	GetGCTaskIDs(ctx context.Context, taskMgr TaskMgr) ([]string, error)

	// GetPieceMD5 gets the piece Md5 accorrding to the specified taskID and pieceNum.
	GetPieceMD5(ctx context.Context, taskID string, pieceNum int, pieceRange, source string) (pieceMd5 string, err error)

	// CheckFile checks the file whether exists.
	CheckFile(ctx context.Context, taskID string) bool

	// Delete the cdn meta with specified taskID.
	// The file on the disk will be deleted when the force is true.
	Delete(ctx context.Context, taskID string, force bool) error
}
