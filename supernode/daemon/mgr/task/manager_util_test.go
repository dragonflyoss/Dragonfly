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
	"github.com/dragonflyoss/Dragonfly/apis/types"
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

func (s *TaskUtilTestSuite) TestEqualsTask(c *check.C) {
	var cases = []struct {
		existTask *types.TaskInfo
		task      *types.TaskInfo
		result    bool
	}{
		{
			existTask: &types.TaskInfo{
				ID:             generateTaskID("http://aa.bb.com", "", ""),
				CdnStatus:      types.TaskInfoCdnStatusRUNNING,
				HTTPFileLength: 1000,
				PieceSize:      config.DefaultPieceSize,
				PieceTotal:     1,
				RawURL:         "http://aa.bb.com?page=1",
				TaskURL:        "http://aa.bb.com",
				Md5:            "fooMD5",
			},
			task: &types.TaskInfo{
				ID:             generateTaskID("http://aa.bb.com", "", ""),
				CdnStatus:      types.TaskInfoCdnStatusWAITING,
				HTTPFileLength: 1000,
				PieceSize:      config.DefaultPieceSize,
				PieceTotal:     1,
				RawURL:         "http://aa.bb.com",
				TaskURL:        "http://aa.bb.com",
				Md5:            "fooMD5",
			},
			result: true,
		},
		{

			existTask: &types.TaskInfo{
				ID:             generateTaskID("http://aa.bb.com", "", ""),
				CdnStatus:      types.TaskInfoCdnStatusWAITING,
				HTTPFileLength: 1000,
				PieceSize:      config.DefaultPieceSize,
				PieceTotal:     1,
				RawURL:         "http://aa.bb.com",
				TaskURL:        "http://aa.bb.com",
				Headers:        map[string]string{"aaa": "bbb"},
				Md5:            "fooMD5",
			},
			task: &types.TaskInfo{
				ID:             generateTaskID("http://aa.bb.com", "", ""),
				CdnStatus:      types.TaskInfoCdnStatusWAITING,
				HTTPFileLength: 1000,
				PieceSize:      config.DefaultPieceSize,
				PieceTotal:     1,
				RawURL:         "http://aa.bb.com",
				TaskURL:        "http://aa.bb.com",
				Headers:        map[string]string{"aaa": "bbb"},
				Md5:            "otherMD5",
			},
			result: false,
		},
	}

	for _, v := range cases {
		result := equalsTask(v.existTask, v.task)
		c.Check(result, check.DeepEquals, v.result)
	}
}
