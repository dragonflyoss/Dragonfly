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
	"testing"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/common/errors"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/mock"
	dutil "github.com/dragonflyoss/Dragonfly/supernode/daemon/util"

	"github.com/go-check/check"
	"github.com/golang/mock/gomock"
	"github.com/prashantv/gostub"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&TaskMgrTestSuite{})
}

type TaskMgrTestSuite struct {
	mockCtl          *gomock.Controller
	mockCDNMgr       *mock.MockCDNMgr
	mockDfgetTaskMgr *mock.MockDfgetTaskMgr
	mockPeerMgr      *mock.MockPeerMgr
	mockProgressMgr  *mock.MockProgressMgr
	mockSchedulerMgr *mock.MockSchedulerMgr

	taskManager       *Manager
	contentLengthStub *gostub.Stubs
}

func (s *TaskMgrTestSuite) SetUpSuite(c *check.C) {
	s.mockCtl = gomock.NewController(c)

	s.mockPeerMgr = mock.NewMockPeerMgr(s.mockCtl)
	s.mockCDNMgr = mock.NewMockCDNMgr(s.mockCtl)
	s.mockDfgetTaskMgr = mock.NewMockDfgetTaskMgr(s.mockCtl)
	s.mockProgressMgr = mock.NewMockProgressMgr(s.mockCtl)
	s.mockSchedulerMgr = mock.NewMockSchedulerMgr(s.mockCtl)

	s.mockCDNMgr.EXPECT().TriggerCDN(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	s.mockDfgetTaskMgr.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.mockProgressMgr.EXPECT().InitProgress(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	cfg := config.NewConfig()
	s.taskManager, _ = NewManager(cfg, s.mockPeerMgr, s.mockDfgetTaskMgr,
		s.mockProgressMgr, s.mockCDNMgr, s.mockSchedulerMgr)

	s.contentLengthStub = gostub.Stub(&getContentLength, func(url string, headers map[string]string) (int64, int, error) {
		return 1000, 200, nil
	})
}

func (s *TaskMgrTestSuite) TearDownSuite(c *check.C) {
	s.contentLengthStub.Reset()
	s.mockCtl.Finish()
}

func (s *TaskMgrTestSuite) TestCheckTaskStatus(c *check.C) {
	s.taskManager.taskStore = dutil.NewStore()
	req := &types.TaskCreateRequest{
		CID:        "cid",
		CallSystem: "foo",
		Dfdaemon:   true,
		Path:       "/peer/file/foo",
		RawURL:     "http://aa.bb.com",
		PeerID:     "fooPeerID",
	}
	resp, err := s.taskManager.Register(context.Background(), req)
	c.Check(err, check.IsNil)

	isSuccess, err := s.taskManager.CheckTaskStatus(context.Background(), resp.ID)
	c.Check(err, check.IsNil)
	c.Check(isSuccess, check.Equals, false)

	isSuccess, err = s.taskManager.CheckTaskStatus(context.Background(), "foo")
	c.Check(errors.IsDataNotFound(err), check.Equals, true)
	c.Check(isSuccess, check.Equals, false)

	task, err := s.taskManager.Get(context.Background(), resp.ID)
	c.Check(err, check.IsNil)
	task.CdnStatus = types.TaskInfoCdnStatusSUCCESS
	isSuccess, err = s.taskManager.CheckTaskStatus(context.Background(), resp.ID)
	c.Check(err, check.IsNil)
	c.Check(isSuccess, check.Equals, true)
}

func (s *TaskMgrTestSuite) TestUpdateTaskInfo(c *check.C) {
	s.taskManager.taskStore = dutil.NewStore()
	req := &types.TaskCreateRequest{
		CID:        "cid",
		CallSystem: "foo",
		Dfdaemon:   true,
		Path:       "/peer/file/foo",
		RawURL:     "http://aa.bb.com",
		PeerID:     "fooPeerID",
	}
	resp, err := s.taskManager.Register(context.Background(), req)
	c.Check(err, check.IsNil)

	// return error when taskInfo equals nil
	err = s.taskManager.Update(context.Background(), resp.ID, nil)
	c.Check(errors.IsEmptyValue(err), check.Equals, true)

	// return error when taskInfo.CDNStatus equals ""
	err = s.taskManager.Update(context.Background(), resp.ID, &types.TaskInfo{})
	c.Check(errors.IsEmptyValue(err), check.Equals, true)

	// only update the cdnStatus when CDNStatus is not success.
	err = s.taskManager.Update(context.Background(), resp.ID, &types.TaskInfo{
		CdnStatus:  types.TaskInfoCdnStatusFAILED,
		FileLength: 2000,
		Md5:        "fooMd5",
	})
	c.Check(err, check.IsNil)
	task, err := s.taskManager.Get(context.Background(), resp.ID)
	c.Check(err, check.IsNil)
	c.Check(task.CdnStatus, check.Equals, types.TaskInfoCdnStatusFAILED)
	c.Check(task.FileLength, check.Equals, int64(0))
	c.Check(task.Md5, check.Equals, "")

	// update the taskInfo when CDNStatus is success.
	err = s.taskManager.Update(context.Background(), resp.ID, &types.TaskInfo{
		CdnStatus:  types.TaskInfoCdnStatusSUCCESS,
		FileLength: 2000,
		Md5:        "fooMd5",
	})
	c.Check(err, check.IsNil)
	task, err = s.taskManager.Get(context.Background(), resp.ID)
	c.Check(err, check.IsNil)
	c.Check(task.CdnStatus, check.Equals, types.TaskInfoCdnStatusSUCCESS)
	c.Check(task.FileLength, check.Equals, int64(2000))

	// do not update if origin CDNStatus equals success
	err = s.taskManager.Update(context.Background(), resp.ID, &types.TaskInfo{
		CdnStatus:  types.TaskInfoCdnStatusFAILED,
		FileLength: 3000,
		Md5:        "fooMd5",
	})
	c.Check(err, check.IsNil)
	task, err = s.taskManager.Get(context.Background(), resp.ID)
	c.Check(err, check.IsNil)
	c.Check(task.CdnStatus, check.Equals, types.TaskInfoCdnStatusSUCCESS)
	c.Check(task.FileLength, check.Equals, int64(2000))

}
