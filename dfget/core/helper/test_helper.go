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

package helper

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	api_types "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/pkg/constants"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"

	"github.com/sirupsen/logrus"
)

// CreateConfig creates a temporary config.
func CreateConfig(writer io.Writer, workHome string) *config.Config {
	if writer == nil {
		writer = ioutil.Discard
	}
	cfg := config.NewConfig()
	cfg.WorkHome = workHome
	cfg.RV.MetaPath = filepath.Join(cfg.WorkHome, "meta", "host.meta")
	cfg.RV.SystemDataDir = filepath.Join(cfg.WorkHome, "data")
	fileutils.CreateDirectory(filepath.Dir(cfg.RV.MetaPath))
	fileutils.CreateDirectory(cfg.RV.SystemDataDir)

	logrus.StandardLogger().Out = writer
	return cfg
}

// CreateTestFile creates a temp file and write a string.
func CreateTestFile(path string, content string) error {
	f, err := createFile(path, content)
	if f != nil {
		f.Close()
	}
	return err
}

// CreateTestFileWithMD5 creates a temp file and write a string
// and return the md5 of the file.
func CreateTestFileWithMD5(path string, content string) string {
	f, err := createFile(path, content)
	if err != nil {
		return ""
	}
	defer f.Close()
	return fileutils.Md5Sum(f.Name())
}

func createFile(path string, content string) (*os.File, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if content != "" {
		f.WriteString(content)
	}
	return f, nil
}

// CreateRandomString creates a random string of specified length.
func CreateRandomString(cap int) string {
	var letterBytes = "abcdefghijklmnopqrstuvwxyz"
	var length = len(letterBytes)

	b := make([]byte, cap)
	for i := range b {
		b[i] = letterBytes[rand.Intn(length)]
	}
	return string(b)
}

// ----------------------------------------------------------------------------
// MockSupernodeAPI

// RegisterFuncType function type of SupernodeAPI#Register
type RegisterFuncType func(ip string, req *types.RegisterRequest) (*types.RegisterResponse, error)

// PullFuncType function type of SupernodeAPI#PullPiece
type PullFuncType func(ip string, req *types.PullPieceTaskRequest) (*types.PullPieceTaskResponse, error)

// ReportFuncType function type of SupernodeAPI#ReportPiece
type ReportFuncType func(ip string, req *types.ReportPieceRequest) (*types.BaseResponse, error)

// ServiceDownFuncType function type of SupernodeAPI#ServiceDown
type ServiceDownFuncType func(ip string, taskID string, cid string) (*types.BaseResponse, error)

// ClientErrorFuncType function type of SupernodeAPI#ReportClientError
type ClientErrorFuncType func(ip string, req *types.ClientErrorRequest) (*types.BaseResponse, error)

// ClientErrorFuncType function type of SupernodeAPI#ReportMetricsType
type ReportMetricsFuncType func(node string, req *api_types.TaskMetricsRequest) (*types.BaseResponse, error)

// MockSupernodeAPI mocks the SupernodeAPI.
type MockSupernodeAPI struct {
	RegisterFunc      RegisterFuncType
	PullFunc          PullFuncType
	ReportFunc        ReportFuncType
	ServiceDownFunc   ServiceDownFuncType
	ClientErrorFunc   ClientErrorFuncType
	ReportMetricsFunc ReportMetricsFuncType
}

var _ api.SupernodeAPI = &MockSupernodeAPI{}

// Register implements SupernodeAPI#Register.
func (m *MockSupernodeAPI) Register(ip string, req *types.RegisterRequest) (
	*types.RegisterResponse, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ip, req)
	}
	return nil, nil
}

// PullPieceTask implements SupernodeAPI#PullPiece.
func (m *MockSupernodeAPI) PullPieceTask(ip string, req *types.PullPieceTaskRequest) (
	*types.PullPieceTaskResponse, error) {
	if m.PullFunc != nil {
		return m.PullFunc(ip, req)
	}
	return nil, nil
}

// ReportPiece implements SupernodeAPI#ReportPiece.
func (m *MockSupernodeAPI) ReportPiece(ip string, req *types.ReportPieceRequest) (
	*types.BaseResponse, error) {
	if m.ReportFunc != nil {
		return m.ReportFunc(ip, req)
	}
	return nil, nil
}

// ServiceDown implements SupernodeAPI#ServiceDown.
func (m *MockSupernodeAPI) ServiceDown(ip string, taskID string, cid string) (
	*types.BaseResponse, error) {
	if m.ServiceDownFunc != nil {
		return m.ServiceDownFunc(ip, taskID, cid)
	}
	return nil, nil
}

// ReportClientError implements SupernodeAPI#ReportClientError.
func (m *MockSupernodeAPI) ReportClientError(ip string, req *types.ClientErrorRequest) (resp *types.BaseResponse, e error) {
	if m.ClientErrorFunc != nil {
		return m.ClientErrorFunc(ip, req)
	}
	return nil, nil
}

func (m *MockSupernodeAPI) ReportMetrics(ip string, req *api_types.TaskMetricsRequest) (resp *types.BaseResponse, e error) {
	if m.ClientErrorFunc != nil {
		return m.ReportMetricsFunc(ip, req)
	}
	return nil, nil
}

func (m *MockSupernodeAPI) HeartBeat(node string, req *api_types.HeartBeatRequest) (resp *types.HeartBeatResponse, err error) {
	return nil, nil
}
func (m *MockSupernodeAPI) FetchP2PNetworkInfo(node string, start int, limit int, req *api_types.NetworkInfoFetchRequest) (resp *api_types.NetworkInfoFetchResponse, e error) {
	return nil, nil
}
func (m *MockSupernodeAPI) ReportResource(node string, req *types.RegisterRequest) (resp *types.RegisterResponse, err error) {
	return nil, nil
}
func (m *MockSupernodeAPI) ApplyForSeedNode(node string, req *types.RegisterRequest) (resp *types.RegisterResponse, err error) {
	return nil, nil
}
func (m *MockSupernodeAPI) ReportResourceDeleted(node string, taskID string, cid string) (resp *types.BaseResponse, err error) {
	return nil, nil
}

// CreateRegisterFunc creates a mock register function.
func CreateRegisterFunc() RegisterFuncType {
	var newResponse = func(code int, msg string) *types.RegisterResponse {
		return &types.RegisterResponse{
			BaseResponse: &types.BaseResponse{Code: code, Msg: msg},
		}
	}

	return func(ip string, req *types.RegisterRequest) (*types.RegisterResponse, error) {
		if ip == "" {
			return nil, fmt.Errorf("connection refused")
		}
		switch req.RawURL {
		case "":
			return newResponse(501, "invalid source url"), nil
		case "http://taobao.com":
			return newResponse(constants.CodeNeedAuth, "need auth"), nil
		case "http://github.com":
			return newResponse(constants.CodeWaitAuth, "wait auth"), nil
		case "http://x.com":
			return newResponse(constants.CodeURLNotReachable, "not reachable"), nil
		case "http://lowzj.com":
			resp := newResponse(constants.Success, "")
			resp.Data = &types.RegisterResponseData{
				TaskID:     "a",
				FileLength: 100,
				PieceSize:  10,
			}
			return resp, nil
		}
		return nil, nil
	}
}

// MockFileServer mocks the file server.
type MockFileServer struct {
	sync.Mutex
	Port    int
	fileMap map[string]*mockFile
	sr      *http.Server
}

func NewMockFileServer() *MockFileServer {
	return &MockFileServer{
		fileMap: make(map[string]*mockFile),
	}
}

// StartServer asynchronously starts the server, it will not be blocked.
func (fs *MockFileServer) StartServer(ctx context.Context, port int) error {
	addr, err := net.ResolveTCPAddr("", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	fs.Port = addr.Port
	sr := &http.Server{}
	sr.Handler = fs
	fs.sr = sr

	go func() {
		if err := fs.sr.Serve(l); err != nil {
			panic(err)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				fs.sr.Close()
				return
			}
		}
	}()

	return nil
}

func (fs *MockFileServer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var (
		err      error
		reqRange *httputils.RangeStruct
	)

	if req.Method != http.MethodGet {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	path := req.URL.Path
	path = strings.Trim(path, "/")

	fs.Lock()
	mf, exist := fs.fileMap[path]
	fs.Unlock()

	if !exist {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	rangeSt := []*httputils.RangeStruct{}
	rangeStr := req.Header.Get("Range")

	if rangeStr != "" {
		rangeSt, err = httputils.GetRangeSE(rangeStr, math.MaxInt64)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	if len(rangeSt) > 0 {
		reqRange = rangeSt[0]
	}

	fs.MockResp(resp, mf, reqRange)
}

func (fs *MockFileServer) RegisterFile(path string, size int64, repeatStr string) error {
	fs.Lock()
	defer fs.Unlock()

	path = strings.Trim(path, "/")
	_, exist := fs.fileMap[path]
	if exist {
		return os.ErrExist
	}

	data := []byte(repeatStr)
	if len(data) < 1024 {
		for {
			newData := make([]byte, len(data)*2)
			copy(newData, data)
			copy(newData[len(data):], data)
			data = newData

			if len(data) >= 1024 {
				break
			}
		}
	}

	fs.fileMap[path] = &mockFile{
		path:      path,
		size:      size,
		repeatStr: data,
	}

	return nil
}

func (fs *MockFileServer) UnRegisterFile(path string) {
	fs.Lock()
	defer fs.Unlock()

	delete(fs.fileMap, strings.Trim(path, "/"))
}

func (fs *MockFileServer) MockResp(resp http.ResponseWriter, mf *mockFile, rangeSt *httputils.RangeStruct) {
	var (
		respCode int
		start    int64
		end      = mf.size - 1
	)
	if rangeSt != nil {
		start = rangeSt.StartIndex
		if rangeSt.EndIndex < end {
			end = rangeSt.EndIndex
		}
		respCode = http.StatusPartialContent
	} else {
		respCode = http.StatusOK
	}

	resp.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
	resp.WriteHeader(respCode)

	repeatStrLen := int64(len(mf.repeatStr))
	strIndex := start % int64(repeatStrLen)

	for {
		if start > end {
			break
		}

		copyDataLen := repeatStrLen - strIndex
		if copyDataLen > (end - start + 1) {
			copyDataLen = end - start + 1
		}

		resp.Write(mf.repeatStr[strIndex : copyDataLen+strIndex])
		strIndex = 0
		start += copyDataLen
	}

	return
}

type mockFile struct {
	path      string
	size      int64
	repeatStr []byte
}
