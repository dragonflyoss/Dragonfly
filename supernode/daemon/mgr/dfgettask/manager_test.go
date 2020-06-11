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
	"reflect"
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

func (s *DfgetTaskMgrTestSuite) TestDfgetTaskGetCIDByPeerIDAndTaskID(c *check.C) {
	manager, _ := NewManager(s.cfg, prometheus.NewRegistry())
	// peer1
	manager.Add(context.Background(), &types.DfGetTask{
		CID:        "foo",
		CallSystem: "foo",
		Dfdaemon:   false,
		Path:       "/peer/file/taskFileName",
		PieceSize:  4 * 1024 * 1024,
		TaskID:     "test1",
		PeerID:     "peer1",
	})
	// peer2
	manager.Add(context.Background(), &types.DfGetTask{
		CID:        "bar",
		CallSystem: "bar",
		Dfdaemon:   true,
		Path:       "/peer/file/taskFileName",
		PieceSize:  4 * 1024 * 1024,
		TaskID:     "test2",
		PeerID:     "peer2",
	})

	cases := []struct {
		peerID        string
		taskID        string
		expected      string
		expectedError bool
	}{
		{
			peerID:        "",
			taskID:        "",
			expectedError: true,
		},
		{
			peerID:        "peer1",
			taskID:        "",
			expectedError: true,
		},
		{
			peerID:        "",
			taskID:        "test1",
			expectedError: true,
		},
		{
			peerID:        "peer1",
			taskID:        "test2",
			expectedError: true,
		},
		{
			peerID:        "peer2",
			taskID:        "test1",
			expectedError: true,
		},
		{
			peerID:   "peer1",
			taskID:   "test1",
			expected: "foo",
		},
	}

	for _, tc := range cases {
		got, err := manager.GetCIDByPeerIDAndTaskID(context.Background(), tc.peerID, tc.taskID)
		c.Check(err != nil, check.Equals, tc.expectedError)
		c.Check(got, check.Equals, tc.expected)
	}
}

func (s *DfgetTaskMgrTestSuite) TestDfgetTaskGetCIDsByTaskID(c *check.C) {
	manager, _ := NewManager(s.cfg, prometheus.NewRegistry())
	// peer1
	manager.Add(context.Background(), &types.DfGetTask{
		CID:        "foo",
		CallSystem: "foo",
		Dfdaemon:   false,
		Path:       "/peer/file/taskFileName",
		PieceSize:  4 * 1024 * 1024,
		TaskID:     "test1",
		PeerID:     "peer1",
	})
	// peer2
	manager.Add(context.Background(), &types.DfGetTask{
		CID:        "foo",
		CallSystem: "foo",
		Dfdaemon:   true,
		Path:       "/peer/file/taskFileName",
		PieceSize:  4 * 1024 * 1024,
		TaskID:     "test1",
		PeerID:     "peer2",
	})
	manager.Add(context.Background(), &types.DfGetTask{
		CID:        "bar",
		CallSystem: "bar",
		Dfdaemon:   true,
		Path:       "/peer/file/taskFileName",
		PieceSize:  4 * 1024 * 1024,
		TaskID:     "test2",
		PeerID:     "peer2",
	})

	cases := []struct {
		taskID        string
		expected      []string
		expectedError bool
	}{
		{
			taskID:   "",
			expected: nil,
		},
		{
			taskID:   "test1",
			expected: []string{"foo", "foo"},
		},
		{
			taskID:   "test2",
			expected: []string{"bar"},
		},
	}

	for _, tc := range cases {
		got, err := manager.GetCIDsByTaskID(context.Background(), tc.taskID)
		c.Check(err != nil, check.Equals, tc.expectedError)
		c.Check(reflect.DeepEqual(got, tc.expected), check.Equals, true)
	}
}

func (s *DfgetTaskMgrTestSuite) TestDfgetTaskGetCIDAndTaskIDsByPeerID(c *check.C) {
	manager, _ := NewManager(s.cfg, prometheus.NewRegistry())
	// peer1
	manager.Add(context.Background(), &types.DfGetTask{
		CID:        "foo",
		CallSystem: "foo",
		Dfdaemon:   false,
		Path:       "/peer/file/taskFileName",
		PieceSize:  4 * 1024 * 1024,
		TaskID:     "test1",
		PeerID:     "peer1",
	})
	manager.Add(context.Background(), &types.DfGetTask{
		CID:        "foo",
		CallSystem: "foo",
		Dfdaemon:   false,
		Path:       "/peer/file/taskFileName",
		PieceSize:  4 * 1024 * 1024,
		TaskID:     "test2",
		PeerID:     "peer1",
	})
	// peer2
	manager.Add(context.Background(), &types.DfGetTask{
		CID:        "foo",
		CallSystem: "foo",
		Dfdaemon:   true,
		Path:       "/peer/file/taskFileName",
		PieceSize:  4 * 1024 * 1024,
		TaskID:     "test1",
		PeerID:     "peer2",
	})
	manager.Add(context.Background(), &types.DfGetTask{
		CID:        "bar",
		CallSystem: "bar",
		Dfdaemon:   true,
		Path:       "/peer/file/taskFileName",
		PieceSize:  4 * 1024 * 1024,
		TaskID:     "test2",
		PeerID:     "peer2",
	})

	cases := []struct {
		peerID        string
		expected      map[string]string
		expectedError bool
	}{
		{
			peerID:   "",
			expected: map[string]string{},
		},
		{
			peerID: "peer1",
			expected: map[string]string{
				"foo": "test2",
			},
		},
		{
			peerID: "peer2",
			expected: map[string]string{
				"foo": "test1",
				"bar": "test2",
			},
		},
	}

	for _, tc := range cases {
		got, err := manager.GetCIDAndTaskIDsByPeerID(context.Background(), tc.peerID)
		c.Check(err != nil, check.Equals, tc.expectedError)
		c.Check(reflect.DeepEqual(got, tc.expected), check.Equals, true)
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
