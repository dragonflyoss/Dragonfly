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

package task

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/mock"

	"github.com/go-check/check"
	"github.com/golang/mock/gomock"
	"github.com/prashantv/gostub"
)

func init() {
	check.Suite(&TaskUtilTestSuite{})
}

type TaskUtilTestSuite struct {
	mockCtl          *gomock.Controller
	mockCDNMgr       *mock.MockCDNMgr
	mockDfgetTaskMgr *mock.MockDfgetTaskMgr
	mockPeerMgr      *mock.MockPeerMgr
	mockProgressMgr  *mock.MockProgressMgr
	mockSchedulerMgr *mock.MockSchedulerMgr

	taskManager       *Manager
	contentLengthStub *gostub.Stubs
}

func (s *TaskUtilTestSuite) SetUpSuite(c *check.C) {
	s.mockCtl = gomock.NewController(c)

	s.mockPeerMgr = mock.NewMockPeerMgr(s.mockCtl)
	s.mockCDNMgr = mock.NewMockCDNMgr(s.mockCtl)
	s.mockDfgetTaskMgr = mock.NewMockDfgetTaskMgr(s.mockCtl)
	s.mockProgressMgr = mock.NewMockProgressMgr(s.mockCtl)
	s.mockSchedulerMgr = mock.NewMockSchedulerMgr(s.mockCtl)
	s.taskManager, _ = NewManager(config.NewConfig(), s.mockPeerMgr, s.mockDfgetTaskMgr,
		s.mockProgressMgr, s.mockCDNMgr, s.mockSchedulerMgr)

	s.contentLengthStub = gostub.Stub(&getContentLength, func(url string, headers map[string]string) (int64, int, error) {
		return 1000, 200, nil
	})
}

func (s *TaskUtilTestSuite) TearDownSuite(c *check.C) {
	s.contentLengthStub.Reset()
}

func (s *TaskUtilTestSuite) TestAddOrUpdateTask(c *check.C) {
	var cases = []struct {
		req    *types.TaskCreateRequest
		task   *types.TaskInfo
		errNil bool
	}{
		{
			req: &types.TaskCreateRequest{
				CID:        "cid",
				CallSystem: "foo",
				Dfdaemon:   true,
				Path:       "/peer/file/foo",
				RawURL:     "http://aa.bb.com",
			},
			task: &types.TaskInfo{
				ID:             generateTaskID("http://aa.bb.com", "", ""),
				CallSystem:     "foo",
				CdnStatus:      types.TaskInfoCdnStatusWAITING,
				Dfdaemon:       true,
				HTTPFileLength: 1000,
				PieceSize:      config.DefaultPieceSize,
				PieceTotal:     1,
				RawURL:         "http://aa.bb.com",
				TaskURL:        "http://aa.bb.com",
			},
			errNil: true,
		},
		{
			req: &types.TaskCreateRequest{
				CID:        "cid2",
				CallSystem: "foo2",
				Dfdaemon:   false,
				Path:       "/peer/file/foo2",
				RawURL:     "http://aa.bb.com",
				Headers:    map[string]string{"aaa": "bbb"},
			},
			task: &types.TaskInfo{
				ID:             generateTaskID("http://aa.bb.com", "", ""),
				CallSystem:     "foo",
				CdnStatus:      types.TaskInfoCdnStatusWAITING,
				Dfdaemon:       true,
				HTTPFileLength: 1000,
				PieceSize:      config.DefaultPieceSize,
				PieceTotal:     1,
				RawURL:         "http://aa.bb.com",
				TaskURL:        "http://aa.bb.com",
				Headers:        map[string]string{"aaa": "bbb"},
			},
			errNil: true,
		},
	}

	for _, v := range cases {
		task, err := s.taskManager.addOrUpdateTask(context.Background(), v.req)
		c.Check(cutil.IsNil(err), check.Equals, v.errNil)
		taskInfo, err := s.taskManager.getTask(task.ID)
		c.Check(err, check.IsNil)
		c.Check(taskInfo, check.DeepEquals, v.task)
	}
}
