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

package uploader

import (
	"sort"
	"strconv"
	"strings"

	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"

	"github.com/dragonflyoss/Dragonfly/dfget/core/mock"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/go-check/check"
	"github.com/golang/mock/gomock"
)

type FIFOCacheTestSuite struct {
}

func init() {
	check.Suite(&FIFOCacheTestSuite{})
}

func (t *FIFOCacheTestSuite) TestCacheRegister(c *check.C) {
	// initialization
	cm := newFIFOCacheManager()
	var cases = []struct {
		req       *registerStreamTaskRequest
		expectErr error
	}{
		{
			req: &registerStreamTaskRequest{
				taskID: "1",
			},
			expectErr: nil,
		},
		{
			req: &registerStreamTaskRequest{
				taskID: "1",
			},
			expectErr: errortypes.ErrInvalidValue,
		},
	}

	// testing
	for _, ca := range cases {
		err := cm.register(ca.req)
		c.Check(err, check.Equals, ca.expectErr)
	}
}

func (t *FIFOCacheTestSuite) TestCacheStore(c *check.C) {
	// initialization
	testController := gomock.NewController(c)
	defer testController.Finish()
	taskID := "123"
	content := []byte{1}
	cm, supernodeAPI := prepare(testController, taskID)

	// construct the test cases
	var cases = []struct {
		taskID          string
		expectErr       error
		expectPieceNums []int
		expectWindow    window
	}{
		{
			// the task has not been registered yet
			taskID:    "1",
			expectErr: errortypes.ErrInvalidValue,
		},
		{
			// the window would be full after the insertion
			taskID:          taskID,
			expectErr:       nil,
			expectPieceNums: []int{0, 1, 2},
			expectWindow: window{
				una:   3,
				start: 0,
				wnd:   3,
			},
		},
		{
			// the fifo cache would pop the first piece from the current window segment
			taskID:          taskID,
			expectErr:       nil,
			expectPieceNums: []int{1, 2, 3},
			expectWindow: window{
				una:   4,
				start: 1,
				wnd:   3,
			},
		},
	}

	// set the expect behavior of the supernode API
	supernodeAPI.EXPECT().DeleteStreamCache(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

	// testing
	for i := 0; i < len(cases); i++ {
		err := cm.store(cases[i].taskID, content)
		if err != nil {
			c.Assert(cases[i].expectErr, check.NotNil)
			continue
		}

		pieceNums, err := getPieceNums(cases[i].taskID, cm.pieceContent)
		c.Assert(err, check.IsNil)
		c.Check(pieceNums, check.DeepEquals, cases[i].expectPieceNums)
		window, err := getAsWindowState(cm.windowSet, taskID)
		c.Assert(err, check.IsNil)

		// check the window
		c.Check(window.start, check.Equals, cases[i].expectWindow.start)
		c.Check(window.una, check.Equals, cases[i].expectWindow.una)
		c.Check(window.wnd, check.Equals, cases[i].expectWindow.wnd)
	}
}

func (t *FIFOCacheTestSuite) TestCacheLoad(c *check.C) {
	// initialization
	testController := gomock.NewController(c)
	defer testController.Finish()
	taskID := "123"
	cm, _ := prepare(testController, taskID)

	// construct the test cases
	var cases = []struct {
		taskID        string
		up            *uploadParam
		expectContent []byte
		expectLength  int64
		expectErr     error
	}{
		{
			taskID:        taskID,
			up:            &uploadParam{pieceNum: 1},
			expectContent: []byte{1},
			expectLength:  1,
			expectErr:     nil,
		},
		{
			taskID:        taskID,
			up:            &uploadParam{pieceNum: 2},
			expectContent: nil,
			expectLength:  0,
			expectErr:     errortypes.ErrInvalidValue,
		},
	}

	// testing
	for _, ca := range cases {
		content, length, err := cm.load(taskID, ca.up)
		if err != nil {
			c.Assert(ca.expectErr, check.NotNil)
			continue
		}

		c.Check(content, check.DeepEquals, ca.expectContent)
		c.Check(length, check.Equals, ca.expectLength)
	}
}

// ----------------------------------------------------------------------------
// helper functions

// prepare returns a FIFO cache manager, which stores the task with taskID [taskID], windowSize [3], start [0], una [2].
// The pieces have the content of its own pieceNum index.
func prepare(testController *gomock.Controller, taskID string) (*fifoCacheManager, *mock.MockSupernodeAPI) {
	// initialization
	cm := newFIFOCacheManager()
	supernodeAPI := mock.NewMockSupernodeAPI(testController)

	// set the status of the cache manager
	cm.supernodeAPI = supernodeAPI
	cm.register(&registerStreamTaskRequest{
		taskID:     taskID,
		windowSize: 3,
	})

	// insert the pieces
	for i := 0; i < 2; i++ {
		cm.store(taskID, []byte{byte(i)})
	}

	return cm, supernodeAPI
}

// getPiecesNum returns all the pieces number of the task [taskID] in sorted order.
func getPieceNums(taskID string, m *syncmap.SyncMap) (result []int, err error) {
	var (
		pieceNum int
	)

	pieceInfos := m.ListKeyAsStringSlice()
	for _, pieceInfo := range pieceInfos {
		key := strings.Split(pieceInfo, ":")
		if key[0] != taskID {
			return nil, errortypes.ErrInvalidValue
		}

		pieceNum, err = strconv.Atoi(key[1])
		if err != nil {
			return
		}

		result = append(result, pieceNum)
	}

	sort.Ints(result)
	return
}
