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

package scheduler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/mock"

	"github.com/go-check/check"
	"github.com/golang/mock/gomock"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&SchedulerMgrTestSuite{})
}

type SchedulerMgrTestSuite struct {
	mockCtl         *gomock.Controller
	mockProgressMgr *mock.MockProgressMgr

	manager *Manager
}

func (s *SchedulerMgrTestSuite) SetUpSuite(c *check.C) {
	s.mockCtl = gomock.NewController(c)
	s.mockProgressMgr = mock.NewMockProgressMgr(s.mockCtl)
	s.mockProgressMgr.EXPECT().GetPeerIDsByPieceNum(gomock.Any(), gomock.Any(), gomock.Any()).Return([]string{"peerID"}, nil).AnyTimes()

	cfg := config.NewConfig()
	cfg.SetSuperPID("fooPid")
	s.manager, _ = NewManager(cfg, s.mockProgressMgr)
}

func (s *SchedulerMgrTestSuite) TearDownSuite(c *check.C) {
	s.mockCtl.Finish()
}

func (s *SchedulerMgrTestSuite) TestSortByPieceDistance(c *check.C) {
	var cases = []struct {
		pieceNums     []int
		centerNum     int
		pieceCountMap map[int]int
		expected      [][]int
	}{
		{
			pieceNums:     []int{1, 2, 4},
			centerNum:     3,
			pieceCountMap: map[int]int{1: 3, 2: 4, 4: 3},
			expected:      [][]int{{4, 1, 2}},
		},
		{
			pieceNums:     []int{1, 2, 5},
			centerNum:     3,
			pieceCountMap: map[int]int{1: 3, 2: 1, 5: 3},
			expected:      [][]int{{2, 1, 5}, {2, 5, 1}},
		},
	}

	for _, v := range cases {
		s.manager.sortExecutor(context.Background(), v.pieceNums, v.centerNum, v.pieceCountMap)
		fmt.Println(v.pieceNums)
		sortResult := false
		for _, e := range v.expected {
			if reflect.DeepEqual(e, v.pieceNums) {
				sortResult = true
			}
		}
		c.Check(sortResult, check.Equals, true)
	}
}

func (s *SchedulerMgrTestSuite) TestGetCenterNum(c *check.C) {
	var cases = []struct {
		runningPieces []int
		expected      int
	}{
		{
			runningPieces: []int{2, 4, 6},
			expected:      4,
		},
		{
			runningPieces: []int{},
			expected:      0,
		},
		{
			runningPieces: []int{0},
			expected:      0,
		},
		{
			runningPieces: []int{0, 4, 9},
			expected:      4,
		},
	}

	for _, v := range cases {
		result := getCenterNum(v.runningPieces)
		c.Check(result, check.Equals, v.expected)
	}
}

func (s *SchedulerMgrTestSuite) TestIsExistInMap(c *check.C) {
	mmap := syncmap.NewSyncMap()
	mmap.Add("a", "value")
	mmap.Add("b", "value")
	mmap.Add("c", "value")

	c.Check(isExistInMap(mmap, "a"), check.Equals, true)
	c.Check(isExistInMap(mmap, "d"), check.Equals, false)
}

func (s *SchedulerMgrTestSuite) BenchmarkGetPieceCountMap(c *check.C) {
	pieceNums := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		pieceNums[i] = i
	}

	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		s.manager.getPieceCountMap(context.TODO(), pieceNums, "foo")
	}
}
