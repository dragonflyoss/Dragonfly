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

package seed

import (
	"context"
	"sync"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/basic"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/datascheduler"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/config"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/sirupsen/logrus"
)

var (
	defaultSchedulerState = &schedulerState{}
)

const (
	defaultMaxPeers = 3
)

type localTaskState struct {
	task *config.TaskFetchInfo
	path string
}

// schedulerState is an implementation of datascheduler.ScheduleState.
type schedulerState struct{}

func (state *schedulerState) Continue() bool {
	return false
}

// schedulerResult is an implementation of datascheduler.SchedulerResult.
type schedulerResult struct {
	data *basic.SchedulePieceDataResult
}

func (result *schedulerResult) Result() []*basic.SchedulePieceDataResult {
	return []*basic.SchedulePieceDataResult{result.data}
}

func (result *schedulerResult) State() datascheduler.ScheduleState {
	return defaultSchedulerState
}

type scheduleManager struct {
	mutex         sync.Mutex
	localPeerInfo *types.PeerInfo

	// key is peerID, value is Node
	nodeContainer *dataMap

	// key is url, value is taskState
	seedContainer *dataMap

	// key is url, value is localTaskState
	localSeedContainer *dataMap
}

func newScheduleManager(localPeer *types.PeerInfo) *scheduleManager {
	sm := &scheduleManager{
		localSeedContainer: newDataMap(),
		nodeContainer:      newDataMap(),
		seedContainer:      newDataMap(),
		localPeerInfo:      localPeer,
	}

	return sm
}

func (sm *scheduleManager) Schedule(ctx context.Context, rr basic.RangeRequest, state datascheduler.ScheduleState) (datascheduler.SchedulerResult, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	url := rr.URL()
	result := []*basic.SchedulePeerInfo{}

	// get local seed
	localRs := sm.scheduleLocalPeer(url)
	if len(localRs) > 0 {
		result = append(result, localRs...)
	}

	remoteRs := sm.scheduleRemotePeer(ctx, url)
	if len(remoteRs) > 0 {
		result = append(result, remoteRs...)
	}

	return &schedulerResult{
		data: &basic.SchedulePieceDataResult{
			Off:       rr.Offset(),
			Size:      rr.Size(),
			PeerInfos: result,
		},
	}, nil
}

func (sm *scheduleManager) scheduleRemotePeer(ctx context.Context, url string) []*basic.SchedulePeerInfo {
	var (
		state *taskState
		err   error
	)

	state, err = sm.seedContainer.getAsTaskState(url)
	if err != nil {
		return nil
	}

	pns := state.getPeersByLoad(defaultMaxPeers)
	if len(pns) == 0 {
		return nil
	}

	result := make([]*basic.SchedulePeerInfo, len(pns))
	for i, pn := range pns {
		node, err := sm.nodeContainer.getAsNode(pn.peerID)
		if err != nil {
			logrus.Errorf("failed to get node: %v", err)
			continue
		}

		result[i] = &basic.SchedulePeerInfo{
			PeerInfo: &types.PeerInfo{
				ID:   pn.peerID,
				Port: node.Basic.Port,
				IP:   node.Basic.IP,
			},
			Path: pn.path,
		}
	}

	return result
}

func (sm *scheduleManager) SyncSchedulerInfo(nodes []*config.Node) {
	newNodeContainer := newDataMap()
	seedContainer := newDataMap()

	for _, node := range nodes {
		newNodeContainer.add(node.Basic.ID, node)
		sm.syncSeedContainerPerNode(node, seedContainer)
	}

	// replace the taskContainer and nodeContainer
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.nodeContainer = newNodeContainer
	sm.seedContainer = seedContainer
}

func (sm *scheduleManager) AddLocalSeedInfo(task *config.TaskFetchInfo) {
	if task.Path == "" {
		return
	}

	sm.localSeedContainer.add(task.Task.TaskURL, &localTaskState{task: task, path: task.Path})
}

func (sm *scheduleManager) DeleteLocalSeedInfo(url string) {
	sm.localSeedContainer.remove(url)
}

func (sm *scheduleManager) syncSeedContainerPerNode(node *config.Node, seedContainer *dataMap) {
	for _, task := range node.Tasks {
		if !task.Task.AsSeed {
			continue
		}

		if task.Path == "" {
			continue
		}

		ts, err := seedContainer.getAsTaskState(task.Task.TaskURL)
		if err != nil && !errortypes.IsDataNotFound(err) {
			logrus.Errorf("syncSeedContainerPerNode error: %v", err)
			continue
		}

		if ts == nil {
			ts = newTaskState()
			if err := seedContainer.add(task.Task.TaskURL, ts); err != nil {
				logrus.Errorf("syncSeedContainerPerNode add taskstate %v to taskContainer error: %v", ts, err)
				continue
			}
		}

		err = ts.add(node.Basic.ID, task.Path, task.Task)
		if err != nil {
			logrus.Errorf("syncSeedContainerPerNode error: %v", err)
		}
	}
}

func (sm *scheduleManager) scheduleLocalPeer(url string) []*basic.SchedulePeerInfo {
	var (
		lts *localTaskState
		err error
	)

	// seed file has the priority
	lts, err = sm.localSeedContainer.getAsLocalTaskState(url)
	if err == nil {
		return []*basic.SchedulePeerInfo{sm.covertLocalTaskStateToResult(lts)}
	}

	return nil
}

func (sm *scheduleManager) covertLocalTaskStateToResult(lts *localTaskState) *basic.SchedulePeerInfo {
	return &basic.SchedulePeerInfo{
		PeerInfo: &types.PeerInfo{
			ID:   sm.localPeerInfo.ID,
			Port: sm.localPeerInfo.Port,
			IP:   sm.localPeerInfo.IP,
		},
		Path: lts.path,
	}
}
