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

package gc

import (
	"context"

	"github.com/pkg/errors"
)

func (gcm *Manager) gcDfgetTasksWithTaskID(ctx context.Context, taskID string, cids []string) []error {
	var errSlice []error
	for _, cid := range cids {
		// NOTE: At present, we create a peer for every dfgetTask.
		// So we should also delete the peer when the dfgetTask deleted.
		// In addition,if a peer crashes when servers,we can gc it by this way
		//
		// TODO: In the future, we need a heartbeat mechanism to get the peer status and
		// implement a one-to-many relationship between peers and dfgetTasks.
		// For compatibility with older versions, we can perform different operations according to the version of peer.
		dfgetTask, err := gcm.dfgetTaskMgr.Get(ctx, cid, taskID)
		if err == nil {
			gcm.progressMgr.UpdatePeerServiceDown(ctx, dfgetTask.PeerID)
		} else {
			errSlice = append(errSlice, errors.Wrapf(err, "failed to get dfgetTask by cid(%s)", cid))
		}

		if err := gcm.progressMgr.DeleteCID(ctx, cid); err != nil {
			errSlice = append(errSlice, errors.Wrapf(err, "failed to delete dfgetTask(%s) progress info", cid))
		}
		if err := gcm.dfgetTaskMgr.Delete(ctx, cid, taskID); err != nil {
			errSlice = append(errSlice, errors.Wrapf(err, "failed to delete dfgetTask(%s) info", cid))
		}
	}

	if len(errSlice) != 0 {
		return nil
	}

	return errSlice
}

func (gcm *Manager) gcDfgetTasks(ctx context.Context, keys map[string]string, cids []string) []error {
	var errSlice []error
	for _, cid := range cids {
		if err := gcm.progressMgr.DeleteCID(ctx, cid); err != nil {
			errSlice = append(errSlice, errors.Wrapf(err, "failed to delete dfgetTask(%s) progress info", cid))
		}
		if err := gcm.dfgetTaskMgr.Delete(ctx, cid, keys[cid]); err != nil {
			errSlice = append(errSlice, errors.Wrapf(err, "failed to delete dfgetTask(%s) info", cid))
		}
	}

	if len(errSlice) != 0 {
		return nil
	}

	return errSlice
}
