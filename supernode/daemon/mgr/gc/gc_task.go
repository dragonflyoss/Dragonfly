package gc

import (
	"context"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/sirupsen/logrus"
)

func (gcm *Manager) gcTasks(ctx context.Context) {
	var removedTaskCount int

	// get all taskIDs and the corresponding accessTime
	taskAccessMap, err := gcm.taskMgr.GetAccessTime(ctx)
	if err != nil {
		logrus.Errorf("gc tasks: failed to get task accessTime map for GC: %v", err)
		return
	}

	// range all tasks and determine whether they are expired
	taskIDs := taskAccessMap.ListKeyAsStringSlice()
	totalTaskNums := len(taskIDs)
	for _, taskID := range taskIDs {
		atime, err := taskAccessMap.GetAsTime(taskID)
		if err != nil {
			logrus.Errorf("gc tasks: failed to get access time taskID(%s): %v", taskID, err)
			continue
		}
		if time.Since(atime) < gcm.cfg.TaskExpireTime {
			continue
		}

		if !gcm.gcTask(ctx, taskID) {
			continue
		}
		removedTaskCount++
	}

	logrus.Infof("gc tasks: success to full gc task count(%d), remainder count(%d)", removedTaskCount, totalTaskNums-removedTaskCount)
}

func (gcm *Manager) gcTask(ctx context.Context, taskID string) bool {
	logrus.Infof("start to gc task: %s", taskID)

	util.GetLock(taskID, false)
	defer util.ReleaseLock(taskID, false)

	var wg sync.WaitGroup
	wg.Add(3)

	go func(wg *sync.WaitGroup) {
		gcm.gcCIDsByTaskID(ctx, taskID)
		wg.Done()
	}(&wg)

	go func(wg *sync.WaitGroup) {
		gcm.gcCDNByTaskID(ctx, taskID)
		wg.Done()
	}(&wg)

	go func(wg *sync.WaitGroup) {
		gcm.gcTaskByTaskID(ctx, taskID)
		wg.Done()
	}(&wg)

	wg.Wait()
	return true
}

func (gcm *Manager) gcCIDsByTaskID(ctx context.Context, taskID string) {
	// get CIDs according to the taskID
	cids, err := gcm.dfgetTaskMgr.GetCIDsByTaskID(ctx, taskID)
	if err != nil {
		logrus.Errorf("gc task: failed to get cids taskID(%s): %v", taskID, err)
		return
	}
	if err := gcm.gcDfgetTasksWithTaskID(ctx, taskID, cids); err != nil {
		logrus.Errorf("gc task: failed to gc dfgetTasks taskID(%s): %v", taskID, err)
	}
}

func (gcm *Manager) gcCDNByTaskID(ctx context.Context, taskID string) {
	if err := gcm.cdnMgr.Delete(ctx, taskID, false); err != nil {
		logrus.Errorf("gc task: failed to gc cdn meta taskID(%s): %v", taskID, err)
	}
}

func (gcm *Manager) gcTaskByTaskID(ctx context.Context, taskID string) {
	if err := gcm.progressMgr.DeleteTaskID(ctx, taskID); err != nil {
		logrus.Errorf("gc task: failed to gc progress info taskID(%s): %v", taskID, err)
	}
	if err := gcm.taskMgr.Delete(ctx, taskID); err != nil {
		logrus.Errorf("gc task: failed to gc task info taskID(%s): %v", taskID, err)
	}
}
