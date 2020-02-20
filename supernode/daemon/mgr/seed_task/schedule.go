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

import "github.com/sirupsen/logrus"

type seedScheduler interface {
	// try to schedule a new seed
	Schedule(nowTasks []*SeedTaskInfo, newTask *SeedTaskInfo) bool
}

type defaultScheduler struct { }

func (scheduler *defaultScheduler) Schedule (nowTasks []*SeedTaskInfo, newTask *SeedTaskInfo) bool {
	busyPeer := newTask.P2pInfo
	newTaskInfo := newTask.TaskInfo
	pos := -1
	idx := 0
	for idx < len(nowTasks) {
		if nowTasks[idx] == nil {
			if busyPeer != nil {
				busyPeer = nil
				pos = idx
			}
			idx += 1
			continue
		}
		p2pInfo := nowTasks[idx].P2pInfo
		if p2pInfo != nil && p2pInfo.peerId == newTask.P2pInfo.peerId {
			// Hardly run here
			// This peer already have this task
			logrus.Warnf("peer %s registry same taskid %s twice", p2pInfo.peerId, newTaskInfo.ID)
			return false
		}
		if p2pInfo.Load() < newTask.P2pInfo.Load() + 3 {
			idx += 1
			continue
		}
		if busyPeer != nil && p2pInfo.Load() > busyPeer.Load() {
			busyPeer = p2pInfo
			pos = idx
		}
		idx += 1
	}
	if pos >= 0 {
		nowTasks[pos] = newTask
		newTask.P2pInfo.addTask(newTaskInfo.ID)
	}
	if busyPeer != nil && busyPeer != newTask.P2pInfo {
		logrus.Infof("seed %s: peer %s up, peer %s down",
			newTask.TaskInfo.TaskURL,
			newTask.P2pInfo.peerId,
			busyPeer.peerId)
		busyPeer.deleteTask(newTaskInfo.ID)
		// tell busy peer to evict task of blob file next heartbeat period
		busyPeer.rmTaskIds.add(newTaskInfo.ID)
	}

	return pos >= 0
}