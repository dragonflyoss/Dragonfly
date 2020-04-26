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
	"testing"
	"github.com/go-check/check"
	"github.com/dragonflyoss/Dragonfly/apis/types"
	"sort"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/pkg/digest"
	"net/http"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&SeedTaskMgrTestSuite{})
}

type SeedTaskMgrTestSuite struct {
	seedTaskMgr *Manager
}

func (s *SeedTaskMgrTestSuite) SetUpSuite(c *check.C) {
	baseConfig := config.NewBaseProperties()
	baseConfig.MaxSeedPerObject = 2
	s.seedTaskMgr, _ = NewManager(&config.Config{BaseProperties: baseConfig})
}

func (mgr *SeedTaskMgrTestSuite) TestInvalidTask(c *check.C) {
	c.Check(mgr.seedTaskMgr.IsSeedTask(context.Background(), &http.Request{}), check.Equals, false)
}

func (mgr *SeedTaskMgrTestSuite) TestRegistryTask(c *check.C) {
	for _, url := range []string{"http://abc-2.com", "http://abc-2-1.com",
		"http://abc-2-2.com", "http://abc-2-3.com", "http://abc-2-4.com"} {
		resp, err := mgr.seedTaskMgr.Register(context.Background(), &types.TaskRegisterRequest{
			CID:         "c01",
			IP:          "192.168.1.1",
			Port:        16543,
			SuperNodeIP: "10.10.10.10",
			TaskURL:     url,
			AsSeed:      true,
		})
		c.Check(err, check.IsNil)
		c.Check(resp.AsSeed, check.Equals, true)
	}

	resp, err := mgr.seedTaskMgr.Register(context.Background(), &types.TaskRegisterRequest{
		CID:         "c03",
		IP:          "192.168.1.1",
		Port:        16545,
		SuperNodeIP: "10.10.10.10",
		TaskURL:     "http://abc-2.com",
		AsSeed:      true,
	})

	resp, err = mgr.seedTaskMgr.Register(context.Background(), &types.TaskRegisterRequest{
		CID:         "c02",
		IP:          "192.168.1.1",
		Port:        16544,
		SuperNodeIP: "10.10.10.10",
		TaskURL:     "http://abc-2.com",
		AsSeed:      true,
	})
	c.Check(err, check.IsNil)
	c.Check(resp.AsSeed, check.Equals, true)

	tasksInfo, err := mgr.seedTaskMgr.GetTasksInfo(context.Background(), digest.Sha256("http://abc-2.com"))
	c.Check(err, check.IsNil)

	expectPeers := []string{"c02", "c03"}
	var peers []string
	for _, item := range tasksInfo {
		peers = append(peers, item.P2pInfo.peerId)
	}
	sort.Strings(peers)
	c.Check(peers, check.DeepEquals, expectPeers)

	mgr.seedTaskMgr.EvictTask(context.Background(), digest.Sha256("http://abc-2.com"))
	tasksInfo, err = mgr.seedTaskMgr.GetTasksInfo(context.Background(), digest.Sha256("http://abc-2.com"))
	c.Check(err, check.NotNil)

	mgr.seedTaskMgr.DeRegisterPeer(context.Background(), "c01")
	mgr.seedTaskMgr.DeRegisterPeer(context.Background(), "c02")
	mgr.seedTaskMgr.DeRegisterPeer(context.Background(), "c03")
}

func (mgr *SeedTaskMgrTestSuite) TestDownPeers(c *check.C) {
	request := &types.TaskRegisterRequest{
		CID:         "c01",
		IP:          "192.168.1.1",
		HostName:    "node01",
		Path:        "abc",
		Port:        16543,
		RawURL:      "http://abc.com",
		SuperNodeIP: "10.10.10.10",
		TaskURL:     "http://abc.com",
	}

	_, err := mgr.seedTaskMgr.Register(context.Background(), request)
	c.Check(err, check.IsNil)

	p2pInfo, _ := mgr.seedTaskMgr.getP2pInfo(context.Background(), "c01")
	p2pInfo.hbTime = 0

	result := mgr.seedTaskMgr.ScanDownPeers(context.Background())
	c.Check(len(result), check.Equals, 1)
	c.Check(result[0], check.Equals, "c01")
	mgr.seedTaskMgr.DeRegisterPeer(context.Background(), "c01")
}

func (mgr *SeedTaskMgrTestSuite) TestHeartBeat(c *check.C) {
	request := &types.TaskRegisterRequest{
		CID:         "c01",
		IP:          "192.168.1.1",
		HostName:    "node01",
		Path:        "abc",
		Port:        16543,
		RawURL:      "http://abc.com",
		SuperNodeIP: "10.10.10.10",
		TaskURL:     "http://abc.com",
	}
	_, err := mgr.seedTaskMgr.Register(context.Background(), request)
	c.Check(err, check.IsNil)

	p2pInfo, _ := mgr.seedTaskMgr.getP2pInfo(context.Background(), "c01")
	p2pInfo.hbTime = 0

	resp, err := mgr.seedTaskMgr.ReportPeerHealth(context.Background(), "c01")
	c.Check(err, check.IsNil)
	c.Check(p2pInfo.hbTime > 0, check.Equals, true)
	c.Check(resp.SeedTaskIds, check.DeepEquals, []string{digest.Sha256(request.TaskURL)})
	mgr.seedTaskMgr.DeRegisterPeer(context.Background(), "c01")
}
