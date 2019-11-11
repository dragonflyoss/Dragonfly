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

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/sirupsen/logrus"
)

func (gcm *Manager) gcDisk(ctx context.Context) {
	gcTaskIDs, err := gcm.cdnMgr.GetGCTaskIDs(ctx, gcm.taskMgr)
	if err != nil {
		logrus.Errorf("gc disk: failed to get gc tasks: %v", err)
		return
	}

	if len(gcTaskIDs) == 0 {
		return
	}

	logrus.Debugf("gc disk: success to get gcTaskIDs(%d)", len(gcTaskIDs))
	gcm.deleteTaskDisk(ctx, gcTaskIDs)
}

func (gcm *Manager) deleteTaskDisk(ctx context.Context, gcTaskIDs []string) {
	// NOTE: We only gc a certain percentage of tasks which calculated by the config.CleanRatio.
	gcLen := (len(gcTaskIDs)*gcm.cfg.CleanRatio + 9) / 10

	count := 0
	for _, taskID := range gcTaskIDs {
		if count >= gcLen {
			break
		}

		util.GetLock(taskID, false)

		// try to ensure the taskID is not using again
		if _, err := gcm.taskMgr.Get(ctx, taskID); err == nil || !errortypes.IsDataNotFound(err) {
			if err != nil {
				logrus.Errorf("gc disk: failed to get taskID(%s): %v", taskID, err)
			}
			util.ReleaseLock(taskID, false)
			continue
		}

		if err := gcm.cdnMgr.Delete(ctx, taskID, true); err != nil {
			logrus.Errorf("gc disk: failed to delete disk files with taskID(%s): %v", taskID, err)
			util.ReleaseLock(taskID, false)
			continue
		}
		util.ReleaseLock(taskID, false)
		count++
	}
	gcm.metrics.gcDisksCount.WithLabelValues().Add(float64(count))
	gcm.metrics.lastGCDisksTime.WithLabelValues().SetToCurrentTime()

	logrus.Debugf("gc disk: success to gc task count(%d), remainder count(%d)", count, len(gcTaskIDs)-count)
}
