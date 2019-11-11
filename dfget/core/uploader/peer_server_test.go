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
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"time"

	"github.com/go-check/check"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/version"
)

func init() {
	check.Suite(&PeerServerTestSuite{})
}

var (
	commonFile        = "commonFile"
	commonFileContent = "hello File"

	file2000        = "file2000"
	file2000Content = helper.CreateRandomString(2000)

	emptyFile = "emptyFile"
)

type PeerServerTestSuite struct {
	workHome string
	srv      *peerServer
}

func (s *PeerServerTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "dfget-PeerServerTestSuite-")
	s.srv = newTestPeerServer(s.workHome)
	initHelper(s.srv, commonFile, s.workHome, commonFileContent)
	initHelper(s.srv, file2000, s.workHome, file2000Content)
	initHelper(s.srv, emptyFile, s.workHome, "")
}

func (s *PeerServerTestSuite) TearDownSuite(c *check.C) {
	if s.workHome != "" {
		if err := os.RemoveAll(s.workHome); err != nil {
			fmt.Printf("remove path:%s error", s.workHome)
		}
	}
}

func (s *PeerServerTestSuite) TestGetTaskFile(c *check.C) {
	// normal test
	f, _, err := s.srv.getTaskFile(commonFile)
	// check get file correctly
	c.Assert(err, check.IsNil)
	c.Assert(f, check.NotNil)
	defer f.Close()
	// check read file correctly
	result, err := ioutil.ReadAll(f)
	c.Assert(err, check.IsNil)
	c.Assert(string(result), check.Equals, commonFileContent)
}

func (s *PeerServerTestSuite) TestUploadPiece(c *check.C) {
	f, size, _ := s.srv.getTaskFile(commonFile)
	defer f.Close()

	var up = func(start, length int64, pad bool) *uploadParam {
		up := &uploadParam{
			start:     start,
			length:    length,
			pieceNum:  0,
			pieceSize: defaultPieceSize,
		}
		amendRange(size, pad, up)
		return up
	}

	var cases = []struct {
		start    int64
		end      int64
		pad      bool
		expected string
	}{
		// normal test when start offset equals zero
		{0, 10, false, commonFileContent},
		// normal test when start offset not equals zero
		{1, 5, false, commonFileContent[1:6]},
		// range length more than file data test
		{0, 20, false, commonFileContent},
		// range length less than file data test
		{0, 5, false, commonFileContent[:6]},
		// with piece meta data
		{0, 5, true, commonFileContent[:1]},
		{0, 4, true, ""},
		{0, 15, true, commonFileContent},
	}

	for _, v := range cases {
		rr := httptest.NewRecorder()
		p := up(v.start, v.end-v.start+1, v.pad)
		err := s.srv.uploadPiece(f, rr, p)
		c.Check(err, check.IsNil)
		cmt := check.Commentf("content:'%s' start:%d end:%d pad:%v",
			commonFileContent, v.start, v.end, v.pad)
		if v.pad {
			c.Check(rr.Body.String(), check.DeepEquals,
				pieceContent(p.pieceSize, v.expected), cmt)
		} else {
			c.Check(rr.Body.String(), check.Equals, v.expected, cmt)
		}
	}
}

func (s *PeerServerTestSuite) TestAmendRange(c *check.C) {
	var p = func() *uploadParamBuilder {
		return &uploadParamBuilder{
			up: uploadParam{
				padSize:  config.PieceMetaSize,
				start:    0,
				length:   5,
				pieceNum: 0,
			},
		}
	}
	var f = func() *uploadParamBuilder { return p().padSize(0) }

	var cases = []struct {
		size        int64
		needPad     bool
		up          *uploadParam
		expected    *uploadParam
		expectedErr bool
	}{
		{
			size:     10,
			up:       p().build(),
			expected: f().build(),
		},
		{
			size:     10,
			up:       p().length(11).build(),
			expected: f().length(10).build(),
		},
		{
			size:        10,
			up:          p().length(-1).build(),
			expectedErr: true,
		},
		{
			size:        10,
			up:          p().start(-1).build(),
			expectedErr: true,
		},

		{
			size:     10,
			needPad:  true,
			up:       p().build(),
			expected: p().build(),
		},
		{
			size:     10,
			needPad:  true,
			up:       p().start(5).pieceNum(1).build(),
			expected: p().pieceNum(1).build(),
		},
		{
			size:        10,
			needPad:     true,
			up:          p().pieceNum(1).build(),
			expectedErr: true,
		},
		{
			size:     0,
			needPad:  true,
			up:       p().build(),
			expected: p().build(),
		},
	}

	for _, v := range cases {
		err := amendRange(v.size, v.needPad, v.up)
		if v.expectedErr {
			c.Assert(err, check.Equals, errortypes.ErrRangeNotSatisfiable)
		} else {
			c.Assert(err, check.IsNil)
			c.Assert(v.up, check.DeepEquals, v.expected)
		}
	}
}

func (s *PeerServerTestSuite) TestParseParams(c *check.C) {
	uh := defaultUploadHeader

	tests := []struct {
		name    string
		header  uploadHeader
		want    *uploadParam
		wantErr bool
	}{
		{
			name:   "normalTest",
			header: uh.newRange("0-65575"),
			want: &uploadParam{
				start:     0,
				length:    65576,
				pieceSize: defaultPieceSize,
			},
			wantErr: false,
		},
		{
			name:    "MultiDashesTest",
			header:  uh.newRange("0-65-575"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "NotIntTest",
			header:  uh.newRange("0-hello"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "EndLessStartTest",
			header:  uh.newRange("65575-0"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "NegativeStartTest",
			header:  uh.newRange("-1-8"),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		h := tt.header
		got, err := parseParams(h.rangeStr, h.num, h.size)
		if tt.wantErr {
			c.Check(err, check.NotNil)
		} else {
			c.Check(err, check.IsNil)
		}
		c.Check(got, check.DeepEquals, tt.want,
			check.Commentf("%s:%v", tt.name, tt.header))
	}
}

func (s *PeerServerTestSuite) TestIsFinished(c *check.C) {
	ps := &peerServer{}
	c.Assert(ps.isFinished(), check.Equals, true)

	ps.finished = make(chan struct{})
	c.Assert(ps.isFinished(), check.Equals, false)

	close(ps.finished)
	c.Assert(ps.isFinished(), check.Equals, true)
}

func (s *PeerServerTestSuite) TestSetFinished(c *check.C) {
	ps := &peerServer{finished: make(chan struct{})}

	ps.setFinished()
	c.Assert(ps.isFinished(), check.Equals, true)

	// close a closed channel
	defer c.Assert(recover(), check.IsNil,
		check.Commentf("close a closed channel"))
	ps.setFinished()
}

func (s *PeerServerTestSuite) TestShutdown(c *check.C) {
	cfg := createConfig(s.workHome, 0)
	updateServicePortInMeta(cfg.RV.MetaPath, 1)
	c.Assert(getPortFromMeta(cfg.RV.MetaPath), check.Equals, 1)

	taskName := fmt.Sprintf("%d", rand.Int63())
	tmpFile := helper.GetServiceFile(taskName, cfg.RV.SystemDataDir)
	ioutil.WriteFile(tmpFile, []byte("hello"), os.ModePerm)

	ps := newPeerServer(cfg, 0)
	ps.syncTaskMap.Store(taskName, &taskConfig{
		cid:       "x",
		superNode: "localhost",
		taskID:    "b",
		dataDir:   cfg.RV.SystemDataDir,
	})
	ps.api = &helper.MockSupernodeAPI{
		ServiceDownFunc: func(ip string, taskID string, cid string) (*types.BaseResponse, error) {
			c.Assert(ip, check.Equals, "localhost")
			c.Assert(taskID, check.Equals, "b")
			c.Assert(cid, check.Equals, "x")
			return nil, nil
		},
	}

	ps.shutdown()
	c.Assert(fileutils.PathExist(tmpFile), check.Equals, false)
	c.Assert(ps.isFinished(), check.Equals, true)
	c.Assert(getPortFromMeta(cfg.RV.MetaPath), check.Equals, 0)
}

func (s *PeerServerTestSuite) TestDeleteExpiredFile(c *check.C) {
	cfg := createConfig(s.workHome, 0)
	mark := make(map[string]bool)

	var f = func() string {
		taskName := fmt.Sprintf("TestDeleteExpiredFile-%d", rand.Int63())
		tmpFile := helper.GetServiceFile(taskName, cfg.RV.SystemDataDir)
		ioutil.WriteFile(tmpFile, []byte{}, os.ModePerm)
		return taskName
	}
	var t = func(f bool) *taskConfig {
		return &taskConfig{
			taskID:     fmt.Sprintf("%d", rand.Int63()),
			finished:   f,
			dataDir:    cfg.RV.SystemDataDir,
			accessTime: time.Now(),
		}
	}

	var cases = []struct {
		name    string
		task    *taskConfig
		expire  time.Duration
		deleted bool
	}{
		// delete finished and expired task file
		{name: f(), task: t(true), expire: 0, deleted: true},
		// don't delete finished but not expired task file
		{name: f(), task: t(true), expire: time.Minute, deleted: false},
		// don't delete unfinished task file
		{name: f(), task: t(false), expire: 0, deleted: false},
		// delete a non-task file
		{name: f(), task: nil, expire: time.Minute, deleted: true},
	}

	ps := newPeerServer(cfg, 0)
	ps.api = &helper.MockSupernodeAPI{
		ServiceDownFunc: func(ip string, taskID string, cid string) (*types.BaseResponse, error) {
			mark[taskID] = true
			return nil, nil
		},
	}
	for _, v := range cases {
		filePath := helper.GetServiceFile(v.name, cfg.RV.SystemDataDir)
		finished := "<nil>"
		if v.task != nil {
			ps.syncTaskMap.Store(v.name, v.task)
			finished = fmt.Sprintf("%v", v.task.finished)
		}
		info, _ := os.Stat(filePath)
		deleted := ps.deleteExpiredFile(filePath, info, v.expire)
		cmt := check.Commentf("task:%v expire:%v deleted:%v",
			finished, v.expire, v.deleted)

		c.Assert(deleted, check.Equals, v.deleted, cmt)
		c.Assert(fileutils.PathExist(filePath), check.Equals, !v.deleted, cmt)
		if v.task != nil {
			c.Assert(mark[v.task.taskID], check.Equals, v.deleted, cmt)
			_, ok := ps.syncTaskMap.Load(v.name)
			c.Assert(ok, check.Equals, !v.deleted, cmt)
		}
	}
}

// -----------------------------------------------------------------------------
// test peerServer handlers

func (s *PeerServerTestSuite) TestUploadHandler(c *check.C) {
	headers := make(map[string]string)
	headers[config.StrPieceSize] = defaultPieceSizeStr
	headers[config.StrPieceNum] = "0"

	// normal test
	headers["range"] = "bytes=0-1999"
	if rr, err := testHandlerHelper(s.srv, &HandlerHelper{
		method:  http.MethodGet,
		url:     config.PeerHTTPPathPrefix + file2000,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusPartialContent)
		c.Check(rr.Body.String(), check.Equals, pc(file2000Content[:1995]))

		// TODO: limit read check
	}

	// RangeNotSatisfiable
	headers["range"] = "bytes=0-1"
	if rr, err := testHandlerHelper(s.srv, &HandlerHelper{
		method:  http.MethodGet,
		url:     config.PeerHTTPPathPrefix + emptyFile,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusRequestedRangeNotSatisfiable)
	}

	// not found test
	if rr, err := testHandlerHelper(s.srv, &HandlerHelper{
		method:  http.MethodGet,
		url:     config.PeerHTTPPathPrefix + "foo",
		body:    nil,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusInternalServerError)
	}

	// bad request test
	headers["range"] = "bytes=0-x"
	if rr, err := testHandlerHelper(s.srv, &HandlerHelper{
		method:  http.MethodGet,
		url:     config.PeerHTTPPathPrefix + file2000,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusBadRequest)
	}
}

func (s *PeerServerTestSuite) TestParseRateHandler(c *check.C) {
	headers := make(map[string]string)

	// normal test
	testRateLimit := 1000
	headers["rateLimit"] = strconv.Itoa(testRateLimit)
	if rr, err := testHandlerHelper(s.srv, &HandlerHelper{
		method:  http.MethodGet,
		url:     config.LocalHTTPPathRate + file2000,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusOK)
		limit := s.srv.calculateRateLimit(testRateLimit)
		c.Check(rr.Body.String(), check.Equals, strconv.Itoa(limit))
	}

	// totalLimitRate zero test
	s.srv.totalLimitRate = 0
	if rr, err := testHandlerHelper(s.srv, &HandlerHelper{
		method:  http.MethodGet,
		url:     config.LocalHTTPPathRate + file2000,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusOK)
		c.Check(rr.Body.String(), check.Equals, strconv.Itoa(testRateLimit))
	}
	s.srv.totalLimitRate = 1000

	// wrong rateLimit test
	headers["rateLimit"] = "foo"
	if rr, err := testHandlerHelper(s.srv, &HandlerHelper{
		method:  http.MethodGet,
		url:     config.LocalHTTPPathRate + file2000,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusBadRequest)
	}
}

func (s *PeerServerTestSuite) TestCheckHandler(c *check.C) {
	headers := make(map[string]string)
	srv := newTestPeerServer(s.workHome)
	taskFile := "a"

	// normal test
	headers[config.StrDataDir] = srv.cfg.RV.SystemDataDir
	headers[config.StrTotalLimit] = "1000"
	if r, e := testHandlerHelper(srv, &HandlerHelper{
		method:  http.MethodGet,
		url:     config.LocalHTTPPathCheck + taskFile,
		headers: headers,
	}); e == nil {
		c.Assert(r.Code, check.Equals, http.StatusOK)
		c.Assert(r.Body.String(), check.Equals, taskFile+"@"+version.DFGetVersion)
	}
}

func (s *PeerServerTestSuite) TestOneFinishHandler(c *check.C) {
	var r = func() *api.FinishTaskRequest {
		return &api.FinishTaskRequest{
			TaskFileName: fmt.Sprintf("TestOneFinishHandler%d", rand.Int()),
			TaskID:       fmt.Sprintf("%d", rand.Int()),
			ClientID:     fmt.Sprintf("%d", rand.Int()),
			Node:         "127.0.0.1",
		}
	}
	exist := r()
	srv := newTestPeerServer(s.workHome)
	srv.syncTaskMap.Store(exist.TaskFileName, &taskConfig{
		taskID: exist.TaskID,
	})

	var cases = []struct {
		req  *api.FinishTaskRequest
		code int
	}{
		{req: nil, code: http.StatusBadRequest},
		{req: r(), code: http.StatusOK},
		{req: exist, code: http.StatusOK},
	}

	for _, v := range cases {
		res, err := testHandlerHelper(srv, &HandlerHelper{
			method: http.MethodGet,
			url:    config.LocalHTTPPathClient + "finish?" + httputils.ParseQuery(v.req),
		})
		c.Assert(err, check.IsNil)
		c.Assert(res.Code, check.Equals, v.code)

		if v.req == nil {
			continue
		}

		t, ok := srv.syncTaskMap.Load(v.req.TaskFileName)
		c.Assert(ok, check.Equals, v.code == http.StatusOK)
		if ok {
			tt, ok := t.(*taskConfig)
			c.Assert(ok, check.Equals, true)
			c.Assert(tt.rateLimit, check.Equals, 0)
			c.Assert(tt.finished, check.Equals, true)
		}
	}
}

// -----------------------------------------------------------------------------
// helper functions

func testHandlerHelper(srv *peerServer, hh *HandlerHelper) (*httptest.ResponseRecorder, error) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest(hh.method, hh.url, hh.body)
	if err != nil {
		return nil, err
	}

	// Set request headers.
	for k, v := range hh.headers {
		req.Header.Set(k, v)
	}

	// We create a ResponseRecorder
	// (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Init a router.
	r := srv.initRouter()
	r.ServeHTTP(rr, req)

	return rr, nil
}
