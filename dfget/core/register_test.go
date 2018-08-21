/*
 * Copyright 1999-2018 Alibaba Group.
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

package core

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/dfget/types"
	"github.com/go-check/check"
)

func (s *CoreTestSuite) TestNewRegisterResult(c *check.C) {
	result := NewRegisterResult("node", []string{"1"}, "url", "taskID",
		10, 1)
	c.Assert(result.Node, check.Equals, "node")
	c.Assert(result.RemainderNodes, check.DeepEquals, []string{"1"})
	c.Assert(result.URL, check.Equals, "url")
	c.Assert(result.TaskID, check.Equals, "taskID")
	c.Assert(result.FileLength, check.Equals, int64(10))
	c.Assert(result.PieceSize, check.Equals, int32(1))

	str, _ := json.Marshal(result)
	c.Assert(result.String(), check.Equals, string(str))
}

func (s *CoreTestSuite) TestSupernodeRegister_Register(c *check.C) {
	buf := &bytes.Buffer{}
	ctx := s.createContext(buf)
	m := new(MockSupernodeAPI)
	m.RegisterFunc = createRegisterFunc()

	register := NewSupernodeRegister(ctx, m)

	var f = func(ec int, msg string, data *RegisterResult) {
		resp, e := register.Register(0)
		if msg == "" {
			c.Assert(e, check.IsNil)
			c.Assert(resp, check.NotNil)
			c.Assert(resp, check.DeepEquals, data)
		} else {
			c.Assert(e, check.NotNil)
			c.Assert(e.Msg, check.Equals, msg)
			c.Assert(resp, check.IsNil)
		}
	}

	ctx.Node = []string{""}
	f(config.HTTPError, "connection refused", nil)

	ctx.Node = []string{"x"}
	f(501, "invalid source url", nil)

	ctx.Node = []string{"x"}
	ctx.URL = "http://taobao.com"
	f(config.TaskCodeNeedAuth, "need auth", nil)

	ctx.Node = []string{"x"}
	ctx.URL = "http://github.com"
	f(config.TaskCodeWaitAuth, "wait auth", nil)

	ctx.Node = []string{"x"}
	ctx.URL = "http://lowzj.com"
	f(config.Success, "", &RegisterResult{
		Node: "x", RemainderNodes: []string{}, URL: ctx.URL, TaskID: "a",
		FileLength: 100, PieceSize: 10})

	f(config.HTTPError, "empty response, unknown error", nil)
}

func (s *CoreTestSuite) TestSupernodeRegister_constructRegisterRequest(c *check.C) {
	buf := &bytes.Buffer{}
	ctx := s.createContext(buf)
	register := &supernodeRegister{nil, ctx}

	ctx.Identifier = "id"
	req := register.constructRegisterRequest(0)
	c.Assert(req.Identifier, check.Equals, ctx.Identifier)
	c.Assert(req.Md5, check.Equals, "")

	ctx.Md5 = "md5"
	req = register.constructRegisterRequest(0)
	c.Assert(req.Identifier, check.Equals, "")
	c.Assert(req.Md5, check.Equals, ctx.Md5)
}

// ----------------------------------------------------------------------------
// helper functions

func createRegisterFunc() RegisterFuncType {
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

// ----------------------------------------------------------------------------
// MockSupernodeAPI

type RegisterFuncType func(ip string, req *types.RegisterRequest) (*types.RegisterResponse, error)
type PullFuncType func(ip string, req *types.PullPieceTaskRequest) (*types.PullPieceTaskResponse, error)
type ReportFuncType func(ip string, req *types.ReportPieceRequest) (*types.BaseResponse, error)
type ServiceDownFuncType func(ip string, taskID string, cid string) (*types.BaseResponse, error)

type MockSupernodeAPI struct {
	RegisterFunc    RegisterFuncType
	PullFunc        PullFuncType
	ReportFunc      ReportFuncType
	ServiceDownFunc ServiceDownFuncType
}

func (m *MockSupernodeAPI) Register(ip string, req *types.RegisterRequest) (
	*types.RegisterResponse, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ip, req)
	}
	return nil, nil
}

func (m *MockSupernodeAPI) PullPieceTask(ip string, req *types.PullPieceTaskRequest) (
	*types.PullPieceTaskResponse, error) {
	if m.PullFunc != nil {
		return m.PullFunc(ip, req)
	}
	return nil, nil
}
func (m *MockSupernodeAPI) ReportPiece(ip string, req *types.ReportPieceRequest) (
	*types.BaseResponse, error) {
	if m.ReportFunc != nil {
		return m.ReportFunc(ip, req)
	}
	return nil, nil
}
func (m *MockSupernodeAPI) ServiceDown(ip string, taskID string, cid string) (
	*types.BaseResponse, error) {
	if m.ServiceDownFunc != nil {
		return m.ServiceDownFunc(ip, taskID, cid)
	}
	return nil, nil
}
