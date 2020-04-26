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

package seed_task

import (
	"sync"
	"github.com/dragonflyoss/Dragonfly/apis/types"
	"time"
)

type P2pInfo struct {
	peerId   string
	PeerInfo *types.PeerInfo
	hbTime   int64
	taskIds  *idSet // seed tasks
}

func (p2p *P2pInfo) Load() int { return p2p.taskIds.size() }

func (p2p *P2pInfo) addTask(id string) { p2p.taskIds.add(id) }

func (p2p *P2pInfo) deleteTask(id string) { p2p.taskIds.delete(id) }

func (p2p *P2pInfo) hasTask(id string) bool { return p2p.taskIds.has(id) }

func (p2p *P2pInfo) update() { p2p.hbTime = time.Now().Unix() }

// a tuple of TaskInfo and P2pInfo
type SeedTaskInfo struct {
	RequestPath string
	TaskInfo    *types.TaskInfo
	P2pInfo     *P2pInfo
}

// point to a real-time task
type SeedTaskMap struct {
	taskId     string
	lock       *sync.RWMutex
	accessTime int64
	tasks      []*SeedTaskInfo
	availTasks int
	scheduler  seedScheduler
}

func newSeedTaskMap(taskId string, maxTaskPeers int) *SeedTaskMap {
	return &SeedTaskMap{
		taskId:     taskId,
		tasks:      make([]*SeedTaskInfo, maxTaskPeers),
		lock:       new(sync.RWMutex),
		availTasks: 0,
		accessTime: -1,
		scheduler:  &defaultScheduler{},
	}
}

func (taskMap *SeedTaskMap) tryAddNewTask(p2pInfo *P2pInfo, taskRequest *types.TaskCreateRequest) bool {
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
		&SeedTaskInfo{
			RequestPath: taskRequest.Path,
			TaskInfo:    newTaskInfo,
			P2pInfo:     p2pInfo,
		})
}

func (taskMap *SeedTaskMap) listTasks() []*SeedTaskInfo {
	taskMap.lock.RLock()
	defer taskMap.lock.RUnlock()

	result := make([]*SeedTaskInfo, 0)
	for _, v := range taskMap.tasks {
		if v == nil {
			continue
		}
		result = append(result, v)
	}

	return result
}

func (taskMap *SeedTaskMap) update() {
	unixNow := time.Now().Unix()
	if unixNow > taskMap.accessTime {
		taskMap.accessTime = unixNow
	}
}

func (taskMap *SeedTaskMap) remove(id string) bool {
	taskMap.lock.Lock()
	defer taskMap.lock.Unlock()

	i := 0
	left := len(taskMap.tasks)
	for i < len(taskMap.tasks) {
		if taskMap.tasks[i] == nil {
			left -= 1
		} else if taskMap.tasks[i].P2pInfo.peerId == id {
			taskMap.tasks[i].P2pInfo.deleteTask(taskMap.taskId)
			taskMap.tasks[i] = nil
			left -= 1
			taskMap.availTasks -= 1
		}
		i += 1
	}

	return left == 0
}

func (taskMap *SeedTaskMap) removeAllPeers() {
	taskMap.lock.Lock()
	defer taskMap.lock.Unlock()

	for idx, task := range taskMap.tasks {
		if task == nil {
			continue
		}
		taskMap.tasks[idx] = nil
		task.P2pInfo.deleteTask(task.TaskInfo.ID)
	}
	taskMap.availTasks = 0
}
