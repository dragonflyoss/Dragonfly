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

package dfgettask

import (
	"context"
	"testing"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/supernode/config"

	"github.com/go-check/check"
	"github.com/prometheus/client_golang/prometheus"
	prom_testutil "github.com/prometheus/client_golang/prometheus/testutil"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&DfgetTaskMgrTestSuite{})
}

type DfgetTaskMgrTestSuite struct {
	cfg *config.Config
}

func (s *DfgetTaskMgrTestSuite) SetUpSuite(c *check.C) {
	s.cfg = config.NewConfig()
	s.cfg.SetCIDPrefix("127.0.0.1")
}

func (s *DfgetTaskMgrTestSuite) TestDfgetTaskAdd(c *check.C) {
	manager, _ := NewManager(s.cfg, prometheus.NewRegistry())
	dfgetTasks := manager.metrics.dfgetTasks
	dfgetTasksRegisterCount := manager.metrics.dfgetTasksRegisterCount

	var testCases = []struct {
		dfgetTask *types.DfGetTask
		Expect    *types.DfGetTask
	}{
		{
			dfgetTask: &types.DfGetTask{
				CID:        "foo",
				CallSystem: "foo",
				Dfdaemon:   true,
				Path:       "/peer/file/taskFileName",
				PieceSize:  4 * 1024 * 1024,
				TaskID:     "test1",
				PeerID:     "peer1",
			},
			Expect: &types.DfGetTask{
				CID:        "foo",
				CallSystem: "foo",
				Dfdaemon:   true,
				Path:       "/peer/file/taskFileName",
				PieceSize:  4 * 1024 * 1024,
				TaskID:     "test1",
				PeerID:     "peer1",
				Status:     types.DfGetTaskStatusWAITING,
			},
		},
		{
			dfgetTask: &types.DfGetTask{
				CID:        "bar",
				CallSystem: "bar",
				Dfdaemon:   false,
				Path:       "/peer/file/taskFileName",
				PieceSize:  4 * 1024 * 1024,
				TaskID:     "test2",
				PeerID:     "peer2",
			},
			Expect: &types.DfGetTask{
				CID:        "bar",
				CallSystem: "bar",
				Dfdaemon:   false,
				Path:       "/peer/file/taskFileName",
				PieceSize:  4 * 1024 * 1024,
				TaskID:     "test2",
				PeerID:     "peer2",
				Status:     types.DfGetTaskStatusWAITING,
			},
		},
	}

	for _, tc := range testCases {
		err := manager.Add(context.Background(), tc.dfgetTask)
		c.Check(err, check.IsNil)
		c.Assert(1, check.Equals,
			int(prom_testutil.ToFloat64(
				dfgetTasks.WithLabelValues(tc.dfgetTask.CallSystem, tc.dfgetTask.Status))))

		c.Assert(1, check.Equals,
			int(prom_testutil.ToFloat64(
				dfgetTasksRegisterCount.WithLabelValues(tc.dfgetTask.CallSystem))))
		dt, err := manager.Get(context.Background(), tc.dfgetTask.CID, tc.dfgetTask.TaskID)
		c.Check(err, check.IsNil)
		c.Check(dt, check.DeepEquals, tc.Expect)
	}
}

func (s *DfgetTaskMgrTestSuite) TestDfgetTaskUpdate(c *check.C) {
	manager, _ := NewManager(s.cfg, prometheus.NewRegistry())
	dfgetTasksFailCount := manager.metrics.dfgetTasksFailCount

	var testCases = []struct {
		dfgetTask  *types.DfGetTask
		taskStatus string
		Expect     *types.DfGetTask
	}{
		{
			dfgetTask: &types.DfGetTask{
				CID:        "foo",
				CallSystem: "foo",
				Dfdaemon:   true,
				Path:       "/peer/file/taskFileName",
				PieceSize:  4 * 1024 * 1024,
				TaskID:     "test1",
				PeerID:     "peer1",
			},
			taskStatus: types.DfGetTaskStatusFAILED,
			Expect: &types.DfGetTask{
				CID:        "foo",
				CallSystem: "foo",
				Dfdaemon:   true,
				Path:       "/peer/file/taskFileName",
				PieceSize:  4 * 1024 * 1024,
				TaskID:     "test1",
				PeerID:     "peer1",
				Status:     types.DfGetTaskStatusFAILED,
			},
		},
		{
			dfgetTask: &types.DfGetTask{
				CID:        "bar",
				CallSystem: "bar",
				Dfdaemon:   false,
				Path:       "/peer/file/taskFileName",
				PieceSize:  4 * 1024 * 1024,
				TaskID:     "test2",
				PeerID:     "peer2",
			},
			taskStatus: types.DfGetTaskStatusSUCCESS,
			Expect: &types.DfGetTask{
				CID:        "bar",
				CallSystem: "bar",
				Dfdaemon:   false,
				Path:       "/peer/file/taskFileName",
				PieceSize:  4 * 1024 * 1024,
				TaskID:     "test2",
				PeerID:     "peer2",
				Status:     types.DfGetTaskStatusSUCCESS,
			},
		},
	}

	for _, tc := range testCases {
		err := manager.Add(context.Background(), tc.dfgetTask)
		c.Check(err, check.IsNil)

		err = manager.UpdateStatus(context.Background(), tc.dfgetTask.CID, tc.dfgetTask.TaskID, tc.taskStatus)
		c.Check(err, check.IsNil)

		if tc.taskStatus == types.DfGetTaskStatusFAILED {
			c.Assert(1, check.Equals,
				int(prom_testutil.ToFloat64(
					dfgetTasksFailCount.WithLabelValues(tc.dfgetTask.CallSystem))))
		}

		dt, err := manager.Get(context.Background(), tc.dfgetTask.CID, tc.dfgetTask.TaskID)
		c.Check(err, check.IsNil)
		c.Check(dt, check.DeepEquals, tc.Expect)
	}
}

func (s *DfgetTaskMgrTestSuite) TestDfgetTaskDelete(c *check.C) {
	manager, _ := NewManager(s.cfg, prometheus.NewRegistry())
	dfgetTasks := manager.metrics.dfgetTasks

	var testCases = []struct {
		dfgetTask *types.DfGetTask
	}{
		{
			dfgetTask: &types.DfGetTask{
				CID:        "foo",
				CallSystem: "foo",
				Dfdaemon:   false,
				Path:       "/peer/file/taskFileName",
				PieceSize:  4 * 1024 * 1024,
				TaskID:     "test1",
				PeerID:     "peer1",
			},
		},
		{
			dfgetTask: &types.DfGetTask{
				CID:        "bar",
				CallSystem: "bar",
				Dfdaemon:   true,
				Path:       "/peer/file/taskFileName",
				PieceSize:  4 * 1024 * 1024,
				TaskID:     "test2",
				PeerID:     "peer2",
			},
		},
	}

	for _, tc := range testCases {
		err := manager.Add(context.Background(), tc.dfgetTask)
		c.Check(err, check.IsNil)

		err = manager.Delete(context.Background(), tc.dfgetTask.CID, tc.dfgetTask.TaskID)
		c.Check(err, check.IsNil)
		c.Assert(0, check.Equals,
			int(prom_testutil.ToFloat64(
				dfgetTasks.WithLabelValues(tc.dfgetTask.CallSystem, tc.dfgetTask.Status))))

		_, err = manager.Get(context.Background(), tc.dfgetTask.CID, tc.dfgetTask.TaskID)
		c.Check(errortypes.IsDataNotFound(err), check.Equals, true)
	}
}
