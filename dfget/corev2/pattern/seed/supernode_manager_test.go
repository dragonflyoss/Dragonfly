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
	"errors"
	"fmt"
	"sync"
	"time"

	api_types "github.com/dragonflyoss/Dragonfly/apis/types"
	dfCfg "github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/basic"
	"github.com/dragonflyoss/Dragonfly/dfget/types"

	"github.com/go-check/check"
	"github.com/go-openapi/strfmt"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

type nodeTaskBinding struct {
	taskPath string

	task *api_types.TaskInfo
}

type mockNode struct {
	cid  string
	ip   string
	port int
	// key is taskID
	tasks map[string]*api_types.TaskFetchInfo
}

type mockTaskWrapper struct {
	task *api_types.TaskInfo
	// key is cid, value is path in node.
	nodeMap map[string]string
}

type mockSupernode struct {
	mutex   sync.RWMutex
	version string
	// key is url.
	seeds map[string][]*mockTaskWrapper

	// key is node cid.
	nodes map[string]*mockNode
}

func (sp *mockSupernode) addNode(cid, ip string, port int) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	_, ok := sp.nodes[cid]
	if ok {
		return
	}

	sp.nodes[cid] = &mockNode{
		cid:   cid,
		ip:    ip,
		port:  port,
		tasks: make(map[string]*api_types.TaskFetchInfo),
	}
}

func (sp *mockSupernode) deleteNode(cid string) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	node, ok := sp.nodes[cid]
	if !ok {
		return
	}

	for _, task := range node.tasks {
		mt, ok := sp.seeds[task.Task.TaskURL]
		if !ok {
			continue
		}

		newTs := []*mockTaskWrapper{}
		for _, m := range mt {
			delete(m.nodeMap, cid)
			if len(m.nodeMap) > 0 {
				newTs = append(newTs, m)
			}
		}

		if len(newTs) == 0 {
			delete(sp.seeds, task.Task.TaskURL)
		} else {
			sp.seeds[task.Task.TaskURL] = newTs
		}
	}

	delete(sp.nodes, cid)
}

func (sp *mockSupernode) addTask(task *api_types.TaskInfo, cid string, path string, allowSeedDownload bool) error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	// add task to node.
	node, ok := sp.nodes[cid]
	if !ok {
		return errors.New("not set node")
	}

	_, ok = node.tasks[task.ID]
	if !ok {
		node.tasks[task.ID] = &api_types.TaskFetchInfo{
			Task:              task,
			Path:              path,
			AllowSeedDownload: allowSeedDownload,
		}
	}

	// add task to supernode
	ts, ok := sp.seeds[task.TaskURL]
	if !ok {
		ts = []*mockTaskWrapper{}
	}

	found := false
	for _, t := range ts {
		if t.task.ID == task.ID {
			found = true
			t.nodeMap[cid] = path
			break
		}
	}

	if !found {
		ts = append(ts, &mockTaskWrapper{
			task: task,
			nodeMap: map[string]string{
				cid: path,
			},
		})
	}

	sp.seeds[task.TaskURL] = ts
	return nil
}

func (sp *mockSupernode) delTask(task *api_types.TaskInfo, cid string) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	node, ok := sp.nodes[cid]
	if !ok {
		return
	}

	delete(node.tasks, task.ID)

	ts, ok := sp.seeds[task.TaskURL]
	if !ok {
		return
	}

	newTs := []*mockTaskWrapper{}

	for _, t := range ts {
		delete(t.nodeMap, cid)
		if len(t.nodeMap) == 0 {
			continue
		}
		newTs = append(newTs, t)
	}

	if len(newTs) == 0 {
		delete(sp.seeds, task.TaskURL)
	} else {
		sp.seeds[task.TaskURL] = newTs
	}
}

func (sp *mockSupernode) getTasks(filterURLs []string) *api_types.NetworkInfoFetchResponse {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	nm := map[string][]*api_types.TaskFetchInfo{}

	for _, url := range filterURLs {
		infos, ok := sp.seeds[url]
		if !ok {
			continue
		}

		for _, info := range infos {
			for cid := range info.nodeMap {
				node, ok := sp.nodes[cid]
				if !ok {
					continue
				}

				bindings, ok := nm[cid]
				if !ok {
					bindings = []*api_types.TaskFetchInfo{}
				}

				bd, ok := node.tasks[info.task.ID]
				if !ok {
					continue
				}

				bindings = append(bindings, bd)
				nm[cid] = bindings
			}
		}
	}

	nodes := []*api_types.Node{}

	for cid, bd := range nm {
		nInfo, ok := sp.nodes[cid]
		if !ok {
			continue
		}

		node := &api_types.Node{
			Basic: &api_types.PeerInfo{
				ID:   nInfo.cid,
				IP:   strfmt.IPv4(nInfo.ip),
				Port: int32(nInfo.port),
			},
		}

		//infos := []*api_types.TaskFetchInfo{}
		//for _, b := range bd {
		//	infos = append(infos, &api_types.TaskFetchInfo{
		//		Task: b.task,
		//		Path: b.taskPath,
		//	})
		//}

		node.Tasks = bd
		nodes = append(nodes, node)
	}

	return &api_types.NetworkInfoFetchResponse{
		Nodes: nodes,
	}
}

func (sp *mockSupernode) getVersion() string {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	return sp.version
}

func (sp *mockSupernode) updateVersion() {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	time.Sleep(time.Millisecond * 10)
	sp.version = time.Now().Format(time.RFC3339Nano)
}

type mockSupernodeSet struct {
	mutex      sync.RWMutex
	supernodes map[string]*mockSupernode
	enableMap  map[string]struct{}
}

func (ms *mockSupernodeSet) getSupernode(node string) *mockSupernode {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	_, ok := ms.enableMap[node]
	if !ok {
		return nil
	}

	s, ok := ms.supernodes[node]
	if ok {
		return s
	}

	return nil
}

func (ms *mockSupernodeSet) addSupernode(node string) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	_, ok := ms.supernodes[node]
	if ok {
		return
	}

	time.Sleep(time.Millisecond * 10)
	ms.supernodes[node] = &mockSupernode{
		version: time.Now().Format(time.RFC3339Nano),
		seeds:   make(map[string][]*mockTaskWrapper),
		nodes:   make(map[string]*mockNode),
	}

	ms.enableMap[node] = struct{}{}
}

func (ms *mockSupernodeSet) enable(node string) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	ms.enableMap[node] = struct{}{}
}

func (ms *mockSupernodeSet) disable(node string) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	delete(ms.enableMap, node)
}

func newMockSupernodeAPI(set *mockSupernodeSet) api.SupernodeAPI {
	return &helper.MockSupernodeAPI{
		HeartBeatFunc: func(node string, req *api_types.HeartBeatRequest) (response *types.HeartBeatResponse, e error) {
			sp := set.getSupernode(node)
			if sp == nil || sp.getVersion() == "" {
				return nil, errors.New("version not set")
			}

			return &types.HeartBeatResponse{
				BaseResponse: types.NewBaseResponse(200, ""),
				Data: &api_types.HeartBeatResponse{
					Version: sp.getVersion(),
				},
			}, nil
		},
		FetchP2PNetworkInfoFunc: func(node string, start int, limit int, req *api_types.NetworkInfoFetchRequest) (resp *api_types.NetworkInfoFetchResponse, e error) {
			sp := set.getSupernode(node)
			if sp == nil {
				return nil, errors.New("supernode not found")
			}

			return sp.getTasks(req.Urls), nil
		},
	}
}

func checkInArray(info *basic.SchedulePeerInfo, cids []string, paths []string) bool {
	for i := range cids {
		if info.ID == cids[i] && info.Path == paths[i] {
			return true
		}
	}

	return false
}

func (suite *seedSuite) TestSupernodeManager(c *check.C) {
	mss := &mockSupernodeSet{
		supernodes: map[string]*mockSupernode{},
		enableMap:  map[string]struct{}{},
	}
	supernodes := []string{"1.1.1.1:8002", "2.2.2.2:8002", "3.3.3.3:8002"}
	for _, s := range supernodes {
		mss.addSupernode(s)
	}

	nodes := []dfCfg.DFGetCommonConfig{
		{
			Cid:  "local",
			IP:   "10.1.1.1",
			Port: 40901,
		},
		{
			Cid:  "remote1",
			IP:   "10.1.1.2",
			Port: 40901,
		},
	}

	tasks := []*api_types.TaskInfo{
		{
			ID:         "task1",
			TaskURL:    "http://task1",
			FileLength: 1000,
			AsSeed:     true,
		},
		{
			ID:         "task2",
			TaskURL:    "http://task2",
			FileLength: 2000,
			AsSeed:     true,
		},
		{
			ID:         "task3",
			TaskURL:    "http://task3",
			FileLength: 3000,
			AsSeed:     true,
		},
	}

	sAPI := newMockSupernodeAPI(mss)

	localCfg := &Config{
		DFGetCommonConfig: nodes[0],
	}
	// new supernodeManager
	manager := newSupernodeManager(context.Background(), localCfg, supernodes, sAPI,
		intervalOpt{heartBeatInterval: 3 * time.Second, fetchNetworkInterval: 3 * time.Second})

	matchUrls := []string{tasks[0].TaskURL, tasks[1].TaskURL, tasks[2].TaskURL, "url1", "url2", "url3", "url4", "url5", "url6", "url7", "url8"}
	origins := make([]string, len(matchUrls))

	for i, url := range matchUrls {
		node := manager.GetSupernode(url)
		c.Assert(node, check.Not(check.Equals), "")
		origins[i] = node
	}

	// add tasks[0] to all nodes
	snode := manager.locator.Select(tasks[0].TaskURL)
	c.Assert(snode, check.Not(check.Equals), "")

	sp := mss.getSupernode(snode.String())
	c.Assert(sp, check.NotNil)
	sp.addNode(nodes[0].Cid, nodes[0].IP, nodes[0].Port)
	err := sp.addTask(tasks[0], nodes[0].Cid, fmt.Sprintf("%s-%s", tasks[0].ID, nodes[0].Cid), true)
	c.Assert(err, check.IsNil)

	sp.addNode(nodes[1].Cid, nodes[1].IP, nodes[1].Port)
	err = sp.addTask(tasks[0], nodes[1].Cid, fmt.Sprintf("%s-%s", tasks[0].ID, nodes[1].Cid), false)
	c.Assert(err, check.IsNil)

	// add tasks[1] to local
	snode = manager.locator.Select(tasks[1].TaskURL)
	c.Assert(snode, check.Not(check.Equals), "")

	sp = mss.getSupernode(snode.String())
	c.Assert(sp, check.NotNil)
	sp.addNode(nodes[0].Cid, nodes[0].IP, nodes[0].Port)
	err = sp.addTask(tasks[1], nodes[0].Cid, fmt.Sprintf("%s-%s", tasks[1].ID, nodes[0].Cid), false)
	c.Assert(err, check.IsNil)

	// add tasks[2] to remote.
	snode = manager.locator.Select(tasks[2].TaskURL)
	c.Assert(snode, check.Not(check.Equals), "")

	sp = mss.getSupernode(snode.String())
	c.Assert(sp, check.NotNil)
	sp.addNode(nodes[1].Cid, nodes[1].IP, nodes[1].Port)
	err = sp.addTask(tasks[2], nodes[1].Cid, fmt.Sprintf("%s-%s", tasks[2].ID, nodes[1].Cid), true)
	c.Assert(err, check.IsNil)

	// firstly, try to schedule task[0], expect schedule failed.
	infos := manager.Schedule(context.Background(), &rangeRequest{url: tasks[0].TaskURL})
	c.Assert(len(infos), check.Equals, 0)

	time.Sleep(time.Second * 5)

	// add request
	//manager.AddRequest(tasks[0].TaskURL)
	waitCh := make(chan struct{})
	manager.ActiveFetchP2PNetwork(activeFetchSt{url: tasks[0].TaskURL, waitCh: waitCh})

	timer := time.NewTimer(time.Second * 2)
	defer timer.Stop()
	failed := false
	select {
	case <-waitCh:
		break
	case <-timer.C:
		failed = true
		break
	}

	c.Assert(failed, check.Equals, false)

	// try to schedule task[0] again, expect schedule success.
	infos = manager.Schedule(context.Background(), &rangeRequest{url: tasks[0].TaskURL})
	c.Assert(len(infos), check.Equals, 2)
	c.Assert(checkInArray(infos[0], []string{nodes[0].Cid, nodes[1].Cid},
		[]string{fmt.Sprintf("%s-%s", tasks[0].ID, nodes[0].Cid),
			fmt.Sprintf("%s-%s", tasks[0].ID, nodes[1].Cid)}), check.Equals, true)

	c.Assert(checkInArray(infos[1], []string{nodes[0].Cid, nodes[1].Cid},
		[]string{fmt.Sprintf("%s-%s", tasks[0].ID, nodes[0].Cid),
			fmt.Sprintf("%s-%s", tasks[0].ID, nodes[1].Cid)}), check.Equals, true)

	// update version of supernode[0]
	sp = mss.getSupernode(supernodes[0])
	sp.updateVersion()

	time.Sleep(12 * time.Second)
	ev, ok := manager.GetSupernodeEvent(time.Second * 2)
	c.Assert(ok, check.Equals, true)
	c.Assert(ev.evType, check.Equals, reconnectedEv)
	c.Assert(ev.node, check.Equals, supernodes[0])

	// disable supernode[1]
	mss.disable(supernodes[1])
	time.Sleep(15 * time.Second)
	ev, ok = manager.GetSupernodeEvent(time.Second * 2)
	c.Assert(ok, check.Equals, true)
	c.Assert(ev.evType, check.Equals, disconnectedEv)
	c.Assert(ev.node, check.Equals, supernodes[1])

	// try to match urls
	for i, url := range matchUrls {
		node := manager.GetSupernode(url)
		c.Assert(node, check.Not(check.Equals), "")
		if origins[i] == supernodes[1] {
			c.Assert(node, check.Not(check.Equals), origins[i])
		} else {
			c.Assert(node, check.Equals, origins[i])
		}
	}

	// enable supernode[1]
	mss.enable(supernodes[1])
	time.Sleep(10 * time.Second)
	ev, ok = manager.GetSupernodeEvent(time.Second * 2)
	c.Assert(ok, check.Equals, true)
	c.Assert(ev.evType, check.Equals, connectedEv)
	c.Assert(ev.node, check.Equals, supernodes[1])

	// try to match urls
	for i, url := range matchUrls {
		node := manager.GetSupernode(url)
		c.Assert(node, check.Not(check.Equals), "")
		c.Assert(node, check.Equals, origins[i])
	}
}
