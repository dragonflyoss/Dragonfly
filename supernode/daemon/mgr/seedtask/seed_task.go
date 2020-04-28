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
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

type P2pInfo struct {
	peerID   string
	PeerInfo *types.PeerInfo
	hbTime   int64
	taskIDs  *idSet // seed tasks
}

func (p2p *P2pInfo) Load() int { return p2p.taskIDs.size() }

func (p2p *P2pInfo) addTask(id string) { p2p.taskIDs.add(id) }

func (p2p *P2pInfo) deleteTask(id string) { p2p.taskIDs.delete(id) }

func (p2p *P2pInfo) hasTask(id string) bool { return p2p.taskIDs.has(id) }

func (p2p *P2pInfo) update() { p2p.hbTime = time.Now().Unix() }

// a tuple of TaskInfo and P2pInfo
type SeedInfo struct {
	RequestPath string
	TaskInfo    *types.TaskInfo
	P2pInfo     *P2pInfo
}

// point to a real-time task
type SeedMap struct {
	/* seed task id */
	taskID string
	lock   *sync.RWMutex
	/* latest access time */
	accessTime int64
	/* store all task-peer info */
	tasks []*SeedInfo
	/* seed schedule method */
	scheduler seedScheduler
}

func newSeedTaskMap(taskID string, maxTaskPeers int) *SeedMap {
	return &SeedMap{
		taskID:     taskID,
		tasks:      make([]*SeedInfo, maxTaskPeers),
		lock:       new(sync.RWMutex),
		accessTime: -1,
		scheduler:  &defaultScheduler{},
	}
}

func (taskMap *SeedMap) tryAddNewTask(p2pInfo *P2pInfo, taskRequest *types.TaskCreateRequest) bool {
	taskMap.lock.Lock()
	defer taskMap.lock.Unlock()

	newTaskInfo := &types.TaskInfo{
		ID:             taskRequest.TaskID,
		CdnStatus:      types.TaskInfoCdnStatusSUCCESS,
		FileLength:     taskRequest.FileLength,
		Headers:        taskRequest.Headers,
		HTTPFileLength: taskRequest.FileLength,
		Identifier:     taskRequest.Identifier,
		Md5:            taskRequest.Md5,
		PieceSize:      int32(taskRequest.FileLength),
		PieceTotal:     1,
		RawURL:         taskRequest.RawURL,
		RealMd5:        taskRequest.Md5,
		TaskURL:        taskRequest.TaskURL,
		AsSeed:         true,
	}
	return taskMap.scheduler.Schedule(
		taskMap.tasks,
		&SeedInfo{
			RequestPath: taskRequest.Path,
			TaskInfo:    newTaskInfo,
			P2pInfo:     p2pInfo,
		})
}

func (taskMap *SeedMap) listTasks() []*SeedInfo {
	taskMap.lock.RLock()
	defer taskMap.lock.RUnlock()

	result := make([]*SeedInfo, 0)
	for _, v := range taskMap.tasks {
		if v == nil {
			continue
		}
		result = append(result, v)
	}

	return result
}

func (taskMap *SeedMap) update() {
	unixNow := time.Now().Unix()
	if unixNow > taskMap.accessTime {
		taskMap.accessTime = unixNow
	}
}

func (taskMap *SeedMap) remove(id string) bool {
	taskMap.lock.Lock()
	defer taskMap.lock.Unlock()

	i := 0
	left := len(taskMap.tasks)
	for i < len(taskMap.tasks) {
		if taskMap.tasks[i] == nil {
			left--
		} else if taskMap.tasks[i].P2pInfo.peerID == id {
			taskMap.tasks[i].P2pInfo.deleteTask(taskMap.taskID)
			taskMap.tasks[i] = nil
			left--
		}
		i++
	}

	return left == 0
}

func (taskMap *SeedMap) removeAllPeers() {
	taskMap.lock.Lock()
	defer taskMap.lock.Unlock()

	for idx, task := range taskMap.tasks {
		if task == nil {
			continue
		}
		taskMap.tasks[idx] = nil
		task.P2pInfo.deleteTask(task.TaskInfo.ID)
	}
}
