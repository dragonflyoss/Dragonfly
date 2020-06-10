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
	"encoding/json"
	"sync"
	"time"

	api_types "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/basic"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/config"
	"github.com/dragonflyoss/Dragonfly/dfget/local/seed"
	"github.com/dragonflyoss/Dragonfly/dfget/locator"
	"github.com/dragonflyoss/Dragonfly/pkg/algorithm"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"

	"github.com/go-openapi/strfmt"
	"github.com/sirupsen/logrus"
)

type superNodeWrapper struct {
	superNode string

	// version of supernode, if changed, it indicates supernode has been restarted.
	version string

	// scheduler which the task is belong to the supernode.
	sm *scheduleManager

	disconnectCount int

	evQueue queue.Queue
}

func (s *superNodeWrapper) versionChanged(version string) bool {
	return s.version != "" && version != "" && s.version != version
}

func (s *superNodeWrapper) setVersion(version string) {
	if s.versionChanged(version) {
		s.handleEvent(reconnectedEv)
		s.disconnectCount = 0
	}

	s.version = version
}

func (s *superNodeWrapper) disconnect() {
	s.disconnectCount++
	if s.disconnectCount == 3 {
		s.handleEvent(disconnectedEv)
	}
}

func (s *superNodeWrapper) connect() {
	if s.disconnectCount != 0 {
		s.handleEvent(connectedEv)
	}
	s.disconnectCount = 0
}

func (s *superNodeWrapper) handleEvent(evType string) {
	s.evQueue.Put(&supernodeEvent{evType: evType, node: s.superNode})
}

const (
	reconnectedEv  = "reconnect"
	connectedEv    = "connect"
	disconnectedEv = "disconnect"
)

type supernodeEvent struct {
	evType string
	node   string
}

type activeFetchSt struct {
	url    string
	waitCh chan struct{}
}

type intervalOpt struct {
	heartBeatInterval    time.Duration
	fetchNetworkInterval time.Duration
}

// supernodeManager manages the supernodes which may be connected/disconnected. Url will be hashed to
// different supernodes, and schedule is binding to supernode.
type supernodeManager struct {
	locator        locator.SupernodeLocator
	supernodeMap   map[string]*superNodeWrapper
	superEvQueue   queue.Queue
	supernodeAPI   api.SupernodeAPI
	locatorEvQueue queue.Queue
	// innerEvQueue is used to get event from superNodeWrapper.
	innerEvQueue queue.Queue
	cfg          *Config

	syncP2PNetworkCh chan activeFetchSt
	syncTimeLock     sync.Mutex
	syncTime         time.Time

	// recentFetchUrls is the urls which as the parameters to fetch the p2p network recently
	recentFetchUrls []string
	rm              *requestManager

	heartBeatInterval    time.Duration
	fetchNetworkInterval time.Duration
}

func NewSupernodeManager(ctx context.Context, cfg *Config, nodes []string, supernodeAPI api.SupernodeAPI, opt intervalOpt) *supernodeManager {
	locatorEvQueue := queue.NewQueue(0)
	lc, err := locator.NewHashCirclerLocator("default", nodes, locatorEvQueue)
	if err != nil {
		panic(err)
	}

	innerEvQueue := queue.NewQueue(0)

	localPeer := &api_types.PeerInfo{
		IP:   strfmt.IPv4(cfg.IP),
		Port: int32(cfg.Port),
		ID:   cfg.Cid,
	}

	ma := make(map[string]*superNodeWrapper)
	for _, node := range nodes {
		ma[node] = &superNodeWrapper{
			superNode: node,
			sm:        newScheduleManager(localPeer),
			evQueue:   innerEvQueue,
		}
	}

	if opt.heartBeatInterval == 0 {
		opt.heartBeatInterval = time.Second * 30
	}

	if opt.fetchNetworkInterval == 0 {
		opt.fetchNetworkInterval = time.Second * 5
	}

	m := &supernodeManager{
		supernodeMap:         ma,
		locator:              lc,
		locatorEvQueue:       locatorEvQueue,
		syncP2PNetworkCh:     make(chan activeFetchSt, 20),
		rm:                   newRequestManager(),
		cfg:                  cfg,
		innerEvQueue:         innerEvQueue,
		supernodeAPI:         supernodeAPI,
		superEvQueue:         queue.NewQueue(0),
		heartBeatInterval:    opt.heartBeatInterval,
		fetchNetworkInterval: opt.fetchNetworkInterval,
	}
	go m.heartbeatLoop(ctx)
	go m.fetchP2PNetworkInfoLoop(ctx)
	go m.handleEventLoop(ctx)

	return m
}

// GetSupernode gets supernode by url.
func (sm *supernodeManager) GetSupernode(url string) string {
	s := sm.locator.Select(url)
	if s != nil {
		return s.String()
	}

	return ""
}

// AddLocalSeed adds local seed to local scheduler.
func (sm *supernodeManager) AddLocalSeed(path string, taskID string, sd seed.Seed) {
	for _, sw := range sm.supernodeMap {
		sw.sm.AddLocalSeedInfo(&config.TaskFetchInfo{
			Task: &config.SeedTaskInfo{
				TaskID:     taskID,
				AsSeed:     true,
				FileLength: sd.GetFullSize(),
				RawURL:     sd.GetURL(),
				TaskURL:    sd.GetURL(),
			},
			Path: path,
		})
	}
}

// RemoveLocalSeed removes local seed from local scheduler.
func (sm *supernodeManager) RemoveLocalSeed(url string) {
	for _, sw := range sm.supernodeMap {
		sw.sm.DeleteLocalSeedInfo(url)
	}
}

// AddRequest adds request which recently is made.
func (sm *supernodeManager) AddRequest(url string) {
	sm.rm.addRequest(url)
}

// ActiveFetchP2PNetwork fetches p2p network actively.
func (sm *supernodeManager) ActiveFetchP2PNetwork(st activeFetchSt) {
	sm.syncP2PNetworkCh <- st
}

// GetSupernodeEvent gets events which shows the connect state of supernode.
func (sm *supernodeManager) GetSupernodeEvent(timeout time.Duration) (*supernodeEvent, bool) {
	ev, ok := sm.superEvQueue.PollTimeout(timeout)
	if !ok {
		return nil, false
	}

	return ev.(*supernodeEvent), true
}

// Schedule schedules peers by input RangeRequest.
func (sm *supernodeManager) Schedule(ctx context.Context, rr basic.RangeRequest) []*basic.SchedulePeerInfo {
	dwInfos := []*basic.SchedulePeerInfo{}

	for _, sw := range sm.supernodeMap {
		result, err := sw.sm.Schedule(ctx, rr, nil)
		if err != nil || result == nil {
			continue
		}

		dr := result.Result()
		for _, r := range dr {
			dwInfos = append(dwInfos, r.PeerInfos...)
		}
	}

	return dwInfos
}

func (sm *supernodeManager) handleEventLoop(ctx context.Context) {
	timeout := time.Millisecond * 500
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		ev, ok := sm.innerEvQueue.PollTimeout(timeout)
		if !ok {
			continue
		}

		// DO NOT handle event by goroutine because of importance of making sure order of events.
		sm.handleSupernodeEvent(ev.(*supernodeEvent))
	}
}

func (sm *supernodeManager) handleSupernodeEvent(ev *supernodeEvent) {
	switch ev.evType {
	case disconnectedEv:
		sm.locatorEvQueue.Put(locator.NewDisableEvent(ev.node))
	case connectedEv:
		sm.locatorEvQueue.Put(locator.NewEnableEvent(ev.node))
	}

	// send event to supernodeEvQueue to notify out system.
	sm.superEvQueue.Put(ev)
}

func (sm *supernodeManager) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(sm.heartBeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.heartbeat()
		}
	}
}

func (sm *supernodeManager) heartbeat() {
	for node, sw := range sm.supernodeMap {
		resp, err := sm.supernodeAPI.HeartBeat(node, &api_types.HeartBeatRequest{
			IP:   strfmt.IPv4(sm.cfg.IP),
			Port: int32(sm.cfg.Port),
			CID:  sm.cfg.Cid,
		})

		if resp != nil && logrus.GetLevel() == logrus.DebugLevel {
			d, _ := json.Marshal(resp)
			logrus.Debugf("heart beat node %s, resp: %v", node, string(d))
		}

		if err != nil {
			logrus.Errorf("failed to heart beat: %v", err)
			sw.disconnect()
			continue
		}

		if resp.Data == nil {
			sw.disconnect()
			continue
		}

		sw.setVersion(resp.Data.Version)
		sw.connect()
	}
}

// sync p2p network to local scheduler.
func (sm *supernodeManager) syncP2PNetworkInfo(urls []string) {
	if len(urls) == 0 {
		logrus.Debugf("no urls to syncP2PNetworkInfo")
		return
	}

	for node, sw := range sm.supernodeMap {
		sm.syncP2PNetworkInfoFromSuperNode(node, sw, urls)
	}

	sm.syncTimeLock.Lock()
	defer sm.syncTimeLock.Unlock()
	sm.syncTime = time.Now()
	sm.recentFetchUrls = urls
}

// sync p2p network to local scheduler from supernode
func (sm *supernodeManager) syncP2PNetworkInfoFromSuperNode(supernode string, sw *superNodeWrapper, urls []string) {
	if len(urls) == 0 {
		logrus.Debugf("no urls to syncP2PNetworkInfo")
		return
	}

	resp, err := sm.fetchP2PNetwork(supernode, urls)
	if err != nil {
		logrus.Error(err)
		return
	}

	nodes := make([]*config.Node, len(resp.Nodes))
	for i, node := range resp.Nodes {
		tasks := make([]*config.TaskFetchInfo, len(node.Tasks))
		for i, task := range node.Tasks {
			tasks[i] = &config.TaskFetchInfo{
				Task: &config.SeedTaskInfo{
					RawURL:  task.Task.RawURL,
					TaskURL: task.Task.TaskURL,
					TaskID:  task.Task.ID,
					//Headers: task.Task.Headers,
					AsSeed:     task.Task.AsSeed,
					FileLength: task.Task.FileLength,
				},
				Path: task.Pieces[0].Path,
			}
		}

		nodes[i] = &config.Node{
			Basic: node.Basic,
			Load:  node.Load,
			Extra: node.Extra,
			Tasks: tasks,
		}
	}

	// update nodes info to internal scheduler
	sw.sm.SyncSchedulerInfo(nodes)
}

func (sm *supernodeManager) fetchP2PNetworkInfoLoop(ctx context.Context) {
	var (
		lastTime time.Time
	)
	ticker := time.NewTicker(sm.fetchNetworkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.syncTimeLock.Lock()
			lastTime = sm.syncTime
			sm.syncTimeLock.Unlock()

			if lastTime.Add(sm.fetchNetworkInterval).After(time.Now()) {
				continue
			}

			sm.syncP2PNetworkInfo(sm.rm.getRecentRequest(0))
		case active := <-sm.syncP2PNetworkCh:
			if sm.isRecentFetch(active.url) {
				if active.waitCh != nil {
					close(active.waitCh)
				}
				// the url is fetch recently, directly ignore it
				continue
			}
			urls := sm.rm.getRecentRequest(0)
			urls = append(urls, active.url)
			sm.syncP2PNetworkInfo(algorithm.DedupStringArr(urls))
			if active.waitCh != nil {
				close(active.waitCh)
			}
		}
	}
}

func (sm *supernodeManager) isRecentFetch(url string) bool {
	sm.syncTimeLock.Lock()
	defer sm.syncTimeLock.Unlock()

	for _, u := range sm.recentFetchUrls {
		if u == url {
			return true
		}
	}

	return false
}

func (sm *supernodeManager) fetchP2PNetwork(supernode string, urls []string) (resp *api_types.NetworkInfoFetchResponse, e error) {
	req := &api_types.NetworkInfoFetchRequest{
		Urls: urls,
	}
	return sm.supernodeAPI.FetchP2PNetworkInfo(supernode, 0, 0, req)
}
