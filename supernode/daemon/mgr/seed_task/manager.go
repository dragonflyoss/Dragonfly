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
	"context"
	dutil "github.com/dragonflyoss/Dragonfly/supernode/daemon/util"
	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/sirupsen/logrus"
	"github.com/go-openapi/strfmt"
	"time"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/pkg/digest"
	"fmt"
	"net/http"
)

type TaskRegistryResponce struct {
	TaskID   string
	AsSeed   bool
}

type Manager struct {
	cfg          *config.Config
	taskStore    *dutil.Store /* taskid --> peerid set */
	p2pInfoStore *dutil.Store
	ipPortMap    *safeMap
	timeStamp    time.Time
}

func NewManager(cfg *config.Config) (*Manager, error) {
	return &Manager{
		cfg:          cfg,
		taskStore:    dutil.NewStore(),
		p2pInfoStore: dutil.NewStore(),
		ipPortMap:    newSafeMap(),
		timeStamp:    time.Now(),
	}, nil
}

func (mgr *Manager) getTaskMap(ctx context.Context, taskId string) (*SeedTaskMap, error) {
	item, err := mgr.taskStore.Get(taskId)
	if err != nil {
		return nil, err
	}
	taskMap, _ := item.(*SeedTaskMap)
	return taskMap, nil
}

func (mgr *Manager) getOrCreateTaskMap(ctx context.Context, taskId string) *SeedTaskMap {
	ret, err := mgr.getTaskMap(ctx, taskId)
	if err != nil {
		item, _ := mgr.taskStore.LoadOrStore(taskId,
							newSeedTaskMap(taskId, mgr.cfg.MaxSeedPerObject))
		ret, _ = item.(*SeedTaskMap)
	}
	return ret
}

func (mgr *Manager) getP2pInfo(ctx context.Context, peerId string) (*P2pInfo, error) {
	item, err := mgr.p2pInfoStore.Get(peerId)
	if err != nil {
		return nil, err
	}
	peerInfo, _ := item.(*P2pInfo)

	return peerInfo, nil
}

func (mgr *Manager) getOrCreateP2pInfo(ctx context.Context, peerId string, peerRequest *types.PeerCreateRequest) *P2pInfo {
	peerInfo, err := mgr.getP2pInfo(ctx, peerId)
	if err != nil {
		newPeerInfo := &types.PeerInfo{
			ID:       peerId,
			IP:       peerRequest.IP,
			Created:  strfmt.DateTime(time.Now()),
			HostName: peerRequest.HostName,
			Port:     peerRequest.Port,
			Version:  peerRequest.Version,
		}
		item, _ := mgr.p2pInfoStore.LoadOrStore(
					peerId,
					&P2pInfo{
						peerId: peerId,
						PeerInfo: newPeerInfo,
						taskIds: newIdSet(),
						hbTime: time.Now().Unix()})
		peerInfo, _ = item.(*P2pInfo)
	}
	return peerInfo
}

func convertToCreateRequest(request *types.TaskRegisterRequest, peerId string) *types.TaskCreateRequest {
	return &types.TaskCreateRequest{
		CID:         request.CID,
		CallSystem:  request.CallSystem,
		Dfdaemon:    request.Dfdaemon,
		Headers:     netutils.ConvertHeaders(request.Headers),
		Identifier:  request.Identifier,
		Md5:         request.Md5,
		Path:        request.Path,
		PeerID:      peerId,
		RawURL:      request.RawURL,
		TaskURL:     request.TaskURL,
		SupernodeIP: request.SuperNodeIP,
		TaskID:      request.TaskID,
		FileLength:  request.FileLength,
	}
}

func ipPortToStr(ip strfmt.IPv4, port int32) string {
	return fmt.Sprintf("%s-%d", ip.String(), port)
}

func (mgr *Manager) Register(ctx context.Context, request *types.TaskRegisterRequest) (*TaskRegistryResponce, error) {
	logrus.Debugf("registry rt task %v", request)
	request.TaskID = digest.Sha256(request.TaskURL)
	resp := &TaskRegistryResponce{ TaskID: request.TaskID }

	peerCreateReq := &types.PeerCreateRequest{
		IP:       request.IP,
		HostName: strfmt.Hostname(request.HostName),
		Port:     request.Port,
		Version:  request.Version,
	}
	// In real-time situation, cid == peer id
	peerId := request.CID
	p2pInfo := mgr.getOrCreateP2pInfo(ctx, peerId, peerCreateReq)
	// update peer hb time
	p2pInfo.update()
	// check if peer was restarted
	ipPortStr := ipPortToStr(request.IP, request.Port)
	oldPeerId := mgr.ipPortMap.get(ipPortStr)
	if oldPeerId != peerId {
		mgr.DeRegisterPeer(ctx, oldPeerId)
		mgr.ipPortMap.add(ipPortStr, peerId)
	}
	if p2pInfo.hasTask(request.TaskID) {
		return resp, nil
	}
	taskMap := mgr.getOrCreateTaskMap(ctx, request.TaskID)
	taskMap.update()
	if taskMap.tryAddNewTask(p2pInfo, convertToCreateRequest(request, peerId)) {
		resp.AsSeed = true
		logrus.Infof("peer %s becomes seed of %s", p2pInfo.peerId, request.TaskURL)
	}

	return resp, nil
}

func (mgr *Manager) DeRegisterTask(ctx context.Context, peerId, taskId string) error {
	if !mgr.HasTasks(ctx, taskId) {
		return nil
	}
	taskMap, err := mgr.getTaskMap(ctx, taskId)
	if err != nil {
		return err
	}
	if taskMap.remove(peerId) {
		mgr.taskStore.Delete(taskId)
		logrus.Debugf("Task %s has no peers", taskId)
	}

	return nil
}

func (mgr *Manager) EvictTask(ctx context.Context, taskId string) error {
	taskMap, err := mgr.getTaskMap(ctx, taskId)
	if err != nil {
		return err
	}
	taskMap.removeAllPeers()
	mgr.taskStore.Delete(taskId)

	return nil
}

func (mgr *Manager) DeRegisterPeer(ctx context.Context, peerId string) error {
	if peerId == "" {
		return nil
	}
	logrus.Infof("DeRegister peer %s", peerId)
	p2pInfo, err := mgr.getP2pInfo(ctx, peerId)
	if err != nil {
		logrus.Warnf("No peer %s", peerId)
		return err
	}
	for _, id := range p2pInfo.taskIds.list() {
		mgr.DeRegisterTask(ctx, peerId, id)
	}
	// remove from hash table
	mgr.p2pInfoStore.Delete(peerId)
	mgr.ipPortMap.remove(ipPortToStr(p2pInfo.PeerInfo.IP, p2pInfo.PeerInfo.Port))
	return nil
}

func (mgr *Manager) GetTasksInfo(ctx context.Context, taskId string) ([]*SeedTaskInfo, error) {
	taskMap, err := mgr.getTaskMap(ctx, taskId)
	if err != nil {
		return nil, err
	}

	return taskMap.listTasks(), nil
}

func (mgr *Manager) HasTasks(ctx context.Context, taskId string) bool {
	_, err := mgr.taskStore.Get(taskId)

	return err == nil
}

func (mgr *Manager) IsSeedTask(ctx context.Context, request *http.Request) bool {
	return request.Header.Get("X-register-seed") != "" ||
				request.Header.Get("X-report-resource") != ""
}

func (mgr *Manager) ReportPeerHealth (ctx context.Context, peerId string) (*types.HeartBeatResponse, error) {
	p2pInfo, err := mgr.getP2pInfo(ctx, peerId)
	if err != nil {
		return &types.HeartBeatResponse{ NeedRegister:true, Version:mgr.timeStamp.String() }, nil
	}
	p2pInfo.update()

	return &types.HeartBeatResponse{
		SeedTaskIds: p2pInfo.taskIds.list(),
		Version:     mgr.timeStamp.String(),
	}, nil
}

func (mgr *Manager) ScanDownPeers (ctx context.Context) []string {
	nowTime := time.Now().Unix()

	result := make([]string, 0)
	for _, iter := range mgr.p2pInfoStore.List() {
		p2pInfo, ok := iter.(*P2pInfo)
		if !ok {
			continue
		}
		if nowTime < p2pInfo.hbTime + mgr.cfg.PeerExpireTime {
			continue
		}
		result = append(result, p2pInfo.peerId)
	}

	return result
}