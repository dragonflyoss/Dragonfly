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

package cdn

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	"github.com/dragonflyoss/Dragonfly/supernode/store"

	"github.com/emirpasic/gods/maps/treemap"
	godsutils "github.com/emirpasic/gods/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// GetGCTaskIDs returns the taskIDs that should exec GC operations as a string slice.
//
// It should return nil when the free disk of cdn storage is lager than config.YoungGCThreshold.
// It should return all taskIDs that are not running when the free disk of cdn storage is less than config.FullGCThreshold.
func (cm *Manager) GetGCTaskIDs(ctx context.Context, taskMgr mgr.TaskMgr) ([]string, error) {
	var gcTaskIDs []string

	freeDisk, err := cm.cacheStore.GetAvailSpace(ctx, getHomeRawFunc())
	if err != nil {
		if store.IsKeyNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get avail space")
	}
	if freeDisk > cm.cfg.YoungGCThreshold {
		return nil, nil
	}

	fullGC := false
	if freeDisk <= cm.cfg.FullGCThreshold {
		fullGC = true
	}
	logrus.Debugf("start to exec gc with fullGC: %t", fullGC)

	gapTasks := treemap.NewWith(godsutils.Int64Comparator)
	intervalTasks := treemap.NewWith(godsutils.Int64Comparator)

	// walkTaskIDs is used to avoid processing multiple times for the same taskID
	// which is extracted from file name.
	walkTaskIDs := make(map[string]bool)
	walkFn := func(path string, info os.FileInfo, err error) error {
		logrus.Debugf("start to walk path(%s)", path)

		if err != nil {
			logrus.Errorf("failed to access path(%s): %v", path, err)
			return err
		}
		if info.IsDir() {
			return nil
		}
		taskID := strings.Split(info.Name(), ".")[0]

		// If the taskID has been handled, and no need to do that again.
		if walkTaskIDs[taskID] {
			return nil
		}
		walkTaskIDs[taskID] = true

		// we should return directly when we success to get info which means it is being used
		if _, err := taskMgr.Get(ctx, taskID); err == nil || !errortypes.IsDataNotFound(err) {
			if err != nil {
				logrus.Errorf("failed to get taskID(%s): %v", taskID, err)
			}
			return nil
		}

		// add taskID to gcTaskIDs slice directly when fullGC equals true.
		if fullGC {
			gcTaskIDs = append(gcTaskIDs, taskID)
			return nil
		}

		metaData, err := cm.metaDataManager.readFileMetaData(ctx, taskID)
		if err != nil || metaData == nil {
			logrus.Debugf("failed to get metadata taskID(%s): %v", taskID, err)
			// TODO: delete the file when failed to get metadata
			return nil
		}
		// put taskID into gapTasks or intervalTasks which will sort by some rules
		if err := cm.sortInert(ctx, gapTasks, intervalTasks, metaData); err != nil {
			logrus.Errorf("failed to parse inert metaData(%+v): %v", metaData, err)
		}

		return nil
	}

	raw := &store.Raw{
		Bucket: config.DownloadHome,
		WalkFn: walkFn,
	}
	if err := cm.cacheStore.Walk(ctx, raw); err != nil {
		return nil, err
	}

	if !fullGC {
		gcTaskIDs = append(gcTaskIDs, getGCTasks(gapTasks, intervalTasks)...)
	}

	return gcTaskIDs, nil
}

func (cm *Manager) sortInert(ctx context.Context, gapTasks, intervalTasks *treemap.Map, metaData *fileMetaData) error {
	gap := getCurrentTimeMillisFunc() - metaData.AccessTime

	if metaData.Interval > 0 &&
		gap <= metaData.Interval+(int64(cm.cfg.IntervalThreshold.Seconds())*int64(time.Millisecond)) {
		info, err := cm.cacheStore.Stat(ctx, getDownloadRaw(metaData.TaskID))
		if err != nil {
			return err
		}

		v, found := intervalTasks.Get(info.Size)
		if !found {
			v = make([]string, 0)
		}
		tasks := v.([]string)
		tasks = append(tasks, metaData.TaskID)
		intervalTasks.Put(info.Size, tasks)
		return nil
	}

	v, found := gapTasks.Get(gap)
	if !found {
		v = make([]string, 0)
	}
	tasks := v.([]string)
	tasks = append(tasks, metaData.TaskID)
	gapTasks.Put(gap, tasks)
	return nil
}

func getGCTasks(gapTasks, intervalTasks *treemap.Map) []string {
	var gcTasks = make([]string, 0)

	for _, v := range gapTasks.Values() {
		if taskIDs, ok := v.([]string); ok {
			gcTasks = append(gcTasks, taskIDs...)
		}
	}

	for _, v := range intervalTasks.Values() {
		if taskIDs, ok := v.([]string); ok {
			gcTasks = append(gcTasks, taskIDs...)
		}
	}

	return gcTasks
}
