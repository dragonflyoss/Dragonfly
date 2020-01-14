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
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/mock"
	cMock "github.com/dragonflyoss/Dragonfly/supernode/httpclient/mock"

	"github.com/go-check/check"
	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus"
	prom_testutil "github.com/prometheus/client_golang/prometheus/testutil"
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
	mockOriginClient *cMock.MockOriginHTTPClient

	taskManager *Manager
}

func (s *TaskUtilTestSuite) SetUpSuite(c *check.C) {
	s.mockCtl = gomock.NewController(c)

	s.mockPeerMgr = mock.NewMockPeerMgr(s.mockCtl)
	s.mockCDNMgr = mock.NewMockCDNMgr(s.mockCtl)
	s.mockDfgetTaskMgr = mock.NewMockDfgetTaskMgr(s.mockCtl)
	s.mockProgressMgr = mock.NewMockProgressMgr(s.mockCtl)
	s.mockSchedulerMgr = mock.NewMockSchedulerMgr(s.mockCtl)
	s.mockOriginClient = cMock.NewMockOriginHTTPClient(s.mockCtl)
	s.taskManager, _ = NewManager(config.NewConfig(), s.mockPeerMgr, s.mockDfgetTaskMgr,
		s.mockProgressMgr, s.mockCDNMgr, s.mockSchedulerMgr, s.mockOriginClient, prometheus.NewRegistry())

	s.mockOriginClient.EXPECT().GetContentLength(gomock.Any(), gomock.Any()).Return(int64(1000), 200, nil)
}

func (s *TaskUtilTestSuite) TearDownSuite(c *check.C) {
	s.mockCtl.Finish()
}

func (s *TaskUtilTestSuite) TestEqualsTask(c *check.C) {
	var cases = []struct {
		existTask *types.TaskInfo
		task      *types.TaskInfo
		result    bool
	}{
		{
			existTask: &types.TaskInfo{
				ID:             generateTaskID("http://aa.bb.com", "", "", nil),
				CdnStatus:      types.TaskInfoCdnStatusRUNNING,
				HTTPFileLength: 1000,
				PieceSize:      config.DefaultPieceSize,
				PieceTotal:     1,
				RawURL:         "http://aa.bb.com?page=1",
				TaskURL:        "http://aa.bb.com",
				Md5:            "fooMD5",
			},
			task: &types.TaskInfo{
				ID:             generateTaskID("http://aa.bb.com", "", "", nil),
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
				ID:             generateTaskID("http://aa.bb.com", "", "", nil),
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
				ID:             generateTaskID("http://aa.bb.com", "", "", nil),
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

func (s *TaskUtilTestSuite) TestTriggerCdnSyncAction(c *check.C) {
	var err error
	totalCounter := s.taskManager.metrics.triggerCdnCount

	var cases = []struct {
		task  *types.TaskInfo
		err   error
		skip  bool
		total float64
	}{
		{
			task: &types.TaskInfo{
				CdnStatus: types.TaskInfoCdnStatusRUNNING,
			},
			err:  nil,
			skip: true,
		},
		{
			task: &types.TaskInfo{
				CdnStatus: types.TaskInfoCdnStatusSUCCESS,
			},
			err:  nil,
			skip: true,
		},
		{
			task: &types.TaskInfo{
				ID:        "foo",
				CdnStatus: types.TaskInfoCdnStatusWAITING,
			},
			err:   nil,
			skip:  false,
			total: 1,
		},
		{
			task: &types.TaskInfo{
				ID:        "foo1",
				CdnStatus: types.TaskInfoCdnStatusWAITING,
			},
			err:   nil,
			skip:  false,
			total: 2,
		},
	}

	for _, tc := range cases {
		err = s.taskManager.triggerCdnSyncAction(context.Background(), tc.task)
		c.Assert(err, check.Equals, tc.err)
		if !tc.skip {
			c.Assert(tc.total, check.Equals,
				int(prom_testutil.ToFloat64(totalCounter.WithLabelValues())))
		}
	}
}
