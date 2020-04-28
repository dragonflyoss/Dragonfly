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
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/digest"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	dutil "github.com/dragonflyoss/Dragonfly/supernode/daemon/util"
	"github.com/go-openapi/strfmt"
	"github.com/sirupsen/logrus"
)

type TaskRegistryResponce struct {
	TaskID string
	AsSeed bool
}

type Manager struct {
	/* interested in MaxSeedPerObject & PeerExpireTime */
	cfg *config.Config
	/* store all seed task info */
	taskStore *dutil.Store
	/* store all seed peer info */
	p2pInfoStore *dutil.Store
	ipPortMap    *safeMap
	/* create time of seed manager */
	timeStamp time.Time
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

func (mgr *Manager) getTaskMap(ctx context.Context, taskID string) (*SeedMap, error) {
	item, err := mgr.taskStore.Get(taskID)
	if err != nil {
		return nil, err
	}
	taskMap, _ := item.(*SeedMap)
	return taskMap, nil
}

func (mgr *Manager) getOrCreateTaskMap(ctx context.Context, taskID string) *SeedMap {
	ret, err := mgr.getTaskMap(ctx, taskID)
	if err != nil {
		item, _ := mgr.taskStore.LoadOrStore(taskID,
			newSeedTaskMap(taskID, mgr.cfg.MaxSeedPerObject))
		ret, _ = item.(*SeedMap)
	}
	return ret
}

func (mgr *Manager) getP2pInfo(ctx context.Context, peerID string) (*P2pInfo, error) {
	item, err := mgr.p2pInfoStore.Get(peerID)
	if err != nil {
		return nil, err
	}
	peerInfo, _ := item.(*P2pInfo)

	return peerInfo, nil
}

func (mgr *Manager) getOrCreateP2pInfo(ctx context.Context, peerID string, peerRequest *types.PeerCreateRequest) *P2pInfo {
	peerInfo, err := mgr.getP2pInfo(ctx, peerID)
	if err != nil {
		newPeerInfo := &types.PeerInfo{
			ID:       peerID,
			IP:       peerRequest.IP,
			Created:  strfmt.DateTime(time.Now()),
			HostName: peerRequest.HostName,
			Port:     peerRequest.Port,
			Version:  peerRequest.Version,
		}
		item, _ := mgr.p2pInfoStore.LoadOrStore(
			peerID,
			&P2pInfo{
				peerID:   peerID,
				PeerInfo: newPeerInfo,
				taskIDs:  newIDSet(),
				hbTime:   time.Now().Unix()})
		peerInfo, _ = item.(*P2pInfo)
	}
	return peerInfo
}

func convertToCreateRequest(request *types.TaskRegisterRequest, peerID string) *types.TaskCreateRequest {
	return &types.TaskCreateRequest{
		CID:         request.CID,
		CallSystem:  request.CallSystem,
		Dfdaemon:    request.Dfdaemon,
		Headers:     netutils.ConvertHeaders(request.Headers),
		Identifier:  request.Identifier,
		Md5:         request.Md5,
		Path:        request.Path,
		PeerID:      peerID,
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

/*
	1. Register peer(if not registered) & update heartbeat timestamp
	2. if peer was restarted(with same ip-port pair), remove old peer info
	3. try schedule a new seed task
*/
func (mgr *Manager) Register(ctx context.Context, request *types.TaskRegisterRequest) (*TaskRegistryResponce, error) {
	logrus.Debugf("registry seed task %v", request)
	request.TaskID = digest.Sha256(request.TaskURL)
	resp := &TaskRegistryResponce{TaskID: request.TaskID}

	peerCreateReq := &types.PeerCreateRequest{
		IP:       request.IP,
		HostName: strfmt.Hostname(request.HostName),
		Port:     request.Port,
		Version:  request.Version,
	}
	// cid == peer id
	peerID := request.CID
	p2pInfo := mgr.getOrCreateP2pInfo(ctx, peerID, peerCreateReq)
	// update peer hb time
	p2pInfo.update()
	// check if peer was restarted
	ipPortStr := ipPortToStr(request.IP, request.Port)
	oldPeerID := mgr.ipPortMap.get(ipPortStr)
	if oldPeerID != peerID {
		mgr.DeRegisterPeer(ctx, oldPeerID)
		mgr.ipPortMap.add(ipPortStr, peerID)
	}
	if p2pInfo.hasTask(request.TaskID) {
		return resp, nil
	}
	taskMap := mgr.getOrCreateTaskMap(ctx, request.TaskID)
	taskMap.update()
	if taskMap.tryAddNewTask(p2pInfo, convertToCreateRequest(request, peerID)) {
		resp.AsSeed = true
		logrus.Infof("peer %s becomes seed of %s", p2pInfo.peerID, request.TaskURL)
	}

	return resp, nil
}

func (mgr *Manager) DeRegisterTask(ctx context.Context, peerID, taskID string) error {
	if !mgr.HasTasks(ctx, taskID) {
		return nil
	}
	taskMap, err := mgr.getTaskMap(ctx, taskID)
	if err != nil {
		return err
	}
	if taskMap.remove(peerID) {
		mgr.taskStore.Delete(taskID)
		logrus.Debugf("Task %s has no peers", taskID)
	}

	return nil
}

func (mgr *Manager) EvictTask(ctx context.Context, taskID string) error {
	taskMap, err := mgr.getTaskMap(ctx, taskID)
	if err != nil {
		return err
	}
	taskMap.removeAllPeers()
	mgr.taskStore.Delete(taskID)

	return nil
}

func (mgr *Manager) DeRegisterPeer(ctx context.Context, peerID string) error {
	if peerID == "" {
		return nil
	}
	logrus.Infof("DeRegister peer %s", peerID)
	p2pInfo, err := mgr.getP2pInfo(ctx, peerID)
	if err != nil {
		logrus.Warnf("No peer %s", peerID)
		return err
	}
	for _, id := range p2pInfo.taskIDs.list() {
		mgr.DeRegisterTask(ctx, peerID, id)
	}
	// remove from hash table
	mgr.p2pInfoStore.Delete(peerID)
	mgr.ipPortMap.remove(ipPortToStr(p2pInfo.PeerInfo.IP, p2pInfo.PeerInfo.Port))
	return nil
}

func (mgr *Manager) GetTasksInfo(ctx context.Context, taskID string) ([]*SeedInfo, error) {
	taskMap, err := mgr.getTaskMap(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return taskMap.listTasks(), nil
}

func (mgr *Manager) HasTasks(ctx context.Context, taskID string) bool {
	_, err := mgr.taskStore.Get(taskID)

	return err == nil
}

func (mgr *Manager) IsSeedTask(ctx context.Context, request *http.Request) bool {
	return request.Header.Get("X-register-seed") != "" ||
		request.Header.Get("X-report-resource") != ""
}

func (mgr *Manager) ReportPeerHealth(ctx context.Context, peerID string) (*types.HeartBeatResponse, error) {
	p2pInfo, err := mgr.getP2pInfo(ctx, peerID)
	if err != nil {
		// tell peer to register again
		return &types.HeartBeatResponse{NeedRegister: true, Version: mgr.timeStamp.String()}, nil
	}
	p2pInfo.update()

	// return all tasks peer owned
	return &types.HeartBeatResponse{
		SeedTaskIds: p2pInfo.taskIDs.list(),
		Version:     mgr.timeStamp.String(),
	}, nil
}

func (mgr *Manager) ScanDownPeers(ctx context.Context) []string {
	nowTime := time.Now().Unix()

	result := make([]string, 0)
	for _, iter := range mgr.p2pInfoStore.List() {
		p2pInfo, ok := iter.(*P2pInfo)
		if !ok {
			continue
		}
		if nowTime < p2pInfo.hbTime+mgr.cfg.PeerExpireTime {
			continue
		}
		result = append(result, p2pInfo.peerID)
	}

	return result
}
