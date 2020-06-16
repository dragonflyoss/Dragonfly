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
	"github.com/sirupsen/logrus"
	"time"
)

type seedScheduler interface {
	// try to schedule a new seed
	Schedule(nowTasks []*SeedInfo, newTask *SeedInfo) bool
}

type defaultScheduler struct{}

func setAllowSeedDownload(newTask *SeedInfo) {
	// 100MB/s * 30s = 3GB
	time.Sleep(time.Duration(30*time.Second))
	newTask.AllowSeedDownload = true
}

func (scheduler *defaultScheduler) Schedule(nowTasks []*SeedInfo, newTask *SeedInfo) bool {
	busyPeer := newTask.P2pInfo
	newTaskInfo := newTask.TaskInfo
	pos := -1
	idx := 0
	nowAvail := 0
	for idx < len(nowTasks) {
		if nowTasks[idx] == nil {
			/* number of seed < MaxSeedPerObj */
			if busyPeer != nil {
				busyPeer = nil
				pos = idx
			}
			idx++
			continue
		}
		p2pInfo := nowTasks[idx].P2pInfo
		nowAvail++
		if p2pInfo != nil && p2pInfo.peerID == newTask.P2pInfo.peerID {
			// Hardly run here
			// This peer already have this task
			logrus.Warnf("peer %s registry same taskid %s twice", p2pInfo.peerID, newTaskInfo.ID)
			return false
		}
		if p2pInfo.Load() < newTask.P2pInfo.Load()+3 {
			idx++
			continue
		}
		if busyPeer != nil && p2pInfo.Load() > busyPeer.Load() {
			busyPeer = p2pInfo
			pos = idx
		}
		idx++
	}
	if pos >= 0 {
		nowTasks[pos] = newTask
		newTask.P2pInfo.addTask(newTaskInfo.ID)
		if nowAvail < 3 {
			newTask.AllowSeedDownload = true
		} else {
			go setAllowSeedDownload(newTask)
		}
	}
	if busyPeer != nil && busyPeer != newTask.P2pInfo {
		logrus.Infof("seed %s: peer %s up, peer %s down",
			newTask.TaskInfo.TaskURL,
			newTask.P2pInfo.peerID,
			busyPeer.peerID)
		busyPeer.deleteTask(newTaskInfo.ID)
	}

	return pos >= 0
}
