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

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/basic"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/config"

	"github.com/go-check/check"
	strfmt "github.com/go-openapi/strfmt"
)

type mockRangeRequest struct {
	url  string
	off  int64
	size int64
}

func (r mockRangeRequest) URL() string {
	return r.url
}
func (r mockRangeRequest) Offset() int64 {
	return r.off
}
func (r mockRangeRequest) Size() int64 {
	return r.size
}
func (r mockRangeRequest) Header() map[string]string {
	return nil
}
func (r mockRangeRequest) Extra() interface{} {
	return nil
}

func initTaskFetchInfoForTest(id string, asSeed bool, size int64, url string, path string) *config.TaskFetchInfo {
	return &config.TaskFetchInfo{
		Task: &config.SeedTaskInfo{
			TaskID:     id,
			AsSeed:     asSeed,
			FileLength: size,
			TaskURL:    url,
		},
		Path: path,
	}
}

func initPeerInfoForTest(id string, ip string, port int) *types.PeerInfo {
	return &types.PeerInfo{
		ID:   id,
		IP:   strfmt.IPv4(ip),
		Port: int32(port),
	}
}

func isInArrayForTest(cid string, path string, result []*basic.SchedulePeerInfo, c *check.C) {
	for _, r := range result {
		if r.Path == path && r.PeerInfo.ID == cid {
			return
		}
	}

	c.Fatalf("failed to get cid %s, path %s in array %v", cid, path, result)
}

func (suite *seedSuite) TestNormalScheduler(c *check.C) {
	sm := newScheduleManager(&types.PeerInfo{
		ID:   "local_cid",
		IP:   "127.0.0.1",
		Port: 20001,
	})

	node1 := initPeerInfoForTest("node1", "1.1.1.1", 20001)
	node2 := initPeerInfoForTest("node2", "1.1.1.2", 20001)

	task1 := initTaskFetchInfoForTest("task1", true, 100, "http://url1", "seed1")
	task2 := initTaskFetchInfoForTest("task2", true, 200, "http://url2", "seed2")
	task3 := initTaskFetchInfoForTest("task3", true, 300, "http://url3", "seed3")

	nodes := []*config.Node{
		{
			Basic: node1,
			Tasks: []*config.TaskFetchInfo{task1, task2},
			Load:  5,
		},
		{
			Basic: node2,
			Tasks: []*config.TaskFetchInfo{task2, task3},
			Load:  3,
		},
	}

	sm.SyncSchedulerInfo(nodes)
	rs, err := sm.Schedule(context.Background(), &mockRangeRequest{url: "http://url1"}, nil)
	c.Assert(err, check.IsNil)
	result := rs.Result()
	c.Assert(len(result), check.Equals, 1)
	c.Assert(len(result[0].PeerInfos), check.Equals, 1)
	c.Assert(result[0].PeerInfos[0].ID, check.Equals, "node1")
	c.Assert(result[0].PeerInfos[0].Path, check.Equals, "seed1")

	rs, err = sm.Schedule(context.Background(), &mockRangeRequest{url: "http://url2"}, nil)
	c.Assert(err, check.IsNil)
	result = rs.Result()
	c.Assert(len(result), check.Equals, 1)
	c.Assert(len(result[0].PeerInfos), check.Equals, 2)
	isInArrayForTest("node1", "seed2", result[0].PeerInfos, c)
	isInArrayForTest("node2", "seed2", result[0].PeerInfos, c)

	rs, err = sm.Schedule(context.Background(), &mockRangeRequest{url: "http://url3"}, nil)
	c.Assert(err, check.IsNil)
	result = rs.Result()
	c.Assert(len(result), check.Equals, 1)
	c.Assert(len(result[0].PeerInfos), check.Equals, 1)
	c.Assert(result[0].PeerInfos[0].ID, check.Equals, "node2")
	c.Assert(result[0].PeerInfos[0].Path, check.Equals, "seed3")

	rs, err = sm.Schedule(context.Background(), &mockRangeRequest{url: "http://url4"}, nil)
	c.Assert(err, check.IsNil)
	result = rs.Result()
	c.Assert(len(result), check.Equals, 1)
	c.Assert(len(result[0].PeerInfos), check.Equals, 0)
}
