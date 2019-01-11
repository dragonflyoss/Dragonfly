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
	"bytes"
	"fmt"
	"io"
	"path"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/sirupsen/logrus"
)

// CreateConfig create a temporary config
func CreateConfig(writer io.Writer, workHome string) *config.Config {
	if writer == nil {
		writer = &bytes.Buffer{}
	}
	cfg := config.NewConfig()
	cfg.WorkHome = workHome
	cfg.RV.MetaPath = path.Join(cfg.WorkHome, "meta", "host.meta")
	cfg.RV.SystemDataDir = path.Join(cfg.WorkHome, "data")

	logrus.StandardLogger().Out = writer
	cfg.ClientLogger = logrus.StandardLogger()
	cfg.ServerLogger = logrus.StandardLogger()
	return cfg
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

// MockSupernodeAPI mock SupernodeAPI
type MockSupernodeAPI struct {
	RegisterFunc    RegisterFuncType
	PullFunc        PullFuncType
	ReportFunc      ReportFuncType
	ServiceDownFunc ServiceDownFuncType
}

// Register implements SupernodeAPI#Register
func (m *MockSupernodeAPI) Register(ip string, req *types.RegisterRequest) (
	*types.RegisterResponse, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ip, req)
	}
	return nil, nil
}

// PullPieceTask implements SupernodeAPI#PullPiece
func (m *MockSupernodeAPI) PullPieceTask(ip string, req *types.PullPieceTaskRequest) (
	*types.PullPieceTaskResponse, error) {
	if m.PullFunc != nil {
		return m.PullFunc(ip, req)
	}
	return nil, nil
}

// ReportPiece implements SupernodeAPI#ReportPiece
func (m *MockSupernodeAPI) ReportPiece(ip string, req *types.ReportPieceRequest) (
	*types.BaseResponse, error) {
	if m.ReportFunc != nil {
		return m.ReportFunc(ip, req)
	}
	return nil, nil
}

// ServiceDown implements SupernodeAPI#ServiceDown
func (m *MockSupernodeAPI) ServiceDown(ip string, taskID string, cid string) (
	*types.BaseResponse, error) {
	if m.ServiceDownFunc != nil {
		return m.ServiceDownFunc(ip, taskID, cid)
	}
	return nil, nil
}

// CreateRegisterFunc creates a mock register function
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
			return newResponse(config.TaskCodeNeedAuth, "need auth"), nil
		case "http://github.com":
			return newResponse(config.TaskCodeWaitAuth, "wait auth"), nil
		case "http://x.com":
			return newResponse(config.TaskCodeURLNotReachable, "not reachable"), nil
		case "http://lowzj.com":
			resp := newResponse(config.Success, "")
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
