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

package api

import (
	"fmt"
	"strings"
	"testing"

	api_types "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/pkg/constants"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"

	"github.com/go-check/check"
)

const localhost = "127.0.0.1"

func Test(t *testing.T) {
	check.TestingT(t)
}

type SupernodeAPITestSuite struct {
	mock *httputils.MockHTTPClient
	api  SupernodeAPI
}

func (s *SupernodeAPITestSuite) SetUpSuite(c *check.C) {
	s.mock = httputils.NewMockHTTPClient()
	s.api = NewSupernodeAPI()
	s.api.(*supernodeAPI).HTTPClient = s.mock
}

func (s *SupernodeAPITestSuite) TearDownTest(c *check.C) {
	s.mock.Reset()
}

func init() {
	check.Suite(&SupernodeAPITestSuite{})
}

// ----------------------------------------------------------------------------
// unit tests for SupernodeAPI

func (s *SupernodeAPITestSuite) TestSupernodeAPI_Register(c *check.C) {
	s.mock.PostJSONFunc = s.mock.CreatePostJSONFunc(0, nil, nil)
	r, e := s.api.Register(localhost, createRegisterRequest())
	c.Assert(r, check.IsNil)
	c.Assert(e.Error(), check.Equals, "0:")

	s.mock.PostJSONFunc = s.mock.CreatePostJSONFunc(0, nil,
		fmt.Errorf("test"))
	r, e = s.api.Register(localhost, createRegisterRequest())
	c.Assert(r, check.IsNil)
	c.Assert(e.Error(), check.Equals, "test")

	res := types.RegisterResponse{BaseResponse: &types.BaseResponse{}}
	s.mock.PostJSONFunc = s.mock.CreatePostJSONFunc(200, []byte(res.String()), nil)
	r, e = s.api.Register(localhost, createRegisterRequest())
	c.Assert(e, check.IsNil)
	c.Assert(r, check.NotNil)
	c.Assert(r.Code, check.Equals, 0)

	res.Code = constants.Success
	res.Data = &types.RegisterResponseData{FileLength: int64(32)}
	s.mock.PostJSONFunc = s.mock.CreatePostJSONFunc(200, []byte(res.String()), nil)
	r, e = s.api.Register(localhost, createRegisterRequest())
	c.Assert(e, check.IsNil)
	c.Assert(r, check.NotNil)
	c.Assert(r.Code, check.Equals, constants.Success)
	c.Assert(r.Data.FileLength, check.Equals, res.Data.FileLength)
}

func (s *SupernodeAPITestSuite) TestSupernodeAPI_PullPieceTask(c *check.C) {
	res := &types.PullPieceTaskResponse{BaseResponse: &types.BaseResponse{}}
	res.Code = constants.CodePeerFinish
	res.Data = []byte(`{"fileLength":2}`)
	s.mock.GetFunc = s.mock.CreateGetFunc(200, []byte(res.String()), nil)

	r, e := s.api.PullPieceTask(localhost, nil)

	c.Assert(e, check.IsNil)
	c.Assert(r.Code, check.Equals, res.Code)
	c.Assert(r.FinishData().FileLength, check.Equals, int64(2))
}

func (s *SupernodeAPITestSuite) TestSupernodeAPI_ReportPiece(c *check.C) {
	req := &types.ReportPieceRequest{
		TaskID:     "sssss",
		PieceRange: "0-11",
	}
	s.mock.GetFunc = s.mock.CreateGetFunc(200, []byte(`{"Code":611}`), nil)
	r, e := s.api.ReportPiece(localhost, req)
	c.Check(e, check.IsNil)
	c.Check(r.Code, check.Equals, 611)
}

func (s *SupernodeAPITestSuite) TestSupernodeAPI_ServiceDown(c *check.C) {
	s.mock.GetFunc = s.mock.CreateGetFunc(200, []byte(`{"Code":200}`), nil)
	r, e := s.api.ServiceDown(localhost, "", "")
	c.Check(e, check.IsNil)
	c.Check(r.Code, check.Equals, 200)
}

func (s *SupernodeAPITestSuite) TestSupernodeAPI_ReportClientError(c *check.C) {
	s.mock.GetFunc = s.mock.CreateGetFunc(200, []byte(`{"Code":700}`), nil)
	r, e := s.api.ReportClientError(localhost, nil)
	c.Check(e, check.IsNil)
	c.Check(r.Code, check.Equals, 700)
}

func (s *SupernodeAPITestSuite) TestSupernodeAPI_ReportMetrics(c *check.C) {
	s.mock.PostJSONFunc = s.mock.CreatePostJSONFunc(0, nil, nil)
	r, e := s.api.ReportMetrics(localhost, &api_types.TaskMetricsRequest{})
	c.Assert(r, check.IsNil)
	c.Assert(e.Error(), check.Equals, "0:")

	s.mock.PostJSONFunc = s.mock.CreatePostJSONFunc(0, nil,
		fmt.Errorf("test"))
	r, e = s.api.ReportMetrics(localhost, &api_types.TaskMetricsRequest{})
	c.Assert(r, check.IsNil)
	c.Assert(e.Error(), check.Equals, "test")

	res := types.RegisterResponse{BaseResponse: &types.BaseResponse{}}
	s.mock.PostJSONFunc = s.mock.CreatePostJSONFunc(200, []byte(res.String()), nil)
	r, e = s.api.ReportMetrics(localhost, &api_types.TaskMetricsRequest{})
	c.Assert(e, check.IsNil)
	c.Assert(r, check.NotNil)
	c.Assert(r.Code, check.Equals, 0)
}

func (s *SupernodeAPITestSuite) TestSupernodeAPI_get(c *check.C) {
	type testRes struct {
		A int
	}

	api := s.api.(*supernodeAPI)
	f := func(code int, res string, e error) (*testRes, string, error) {
		s.mock.GetFunc = s.mock.CreateGetFunc(code, []byte(res), e)
		msg := fmt.Sprintf("code:%d res:%s e:%v", code, res, e)
		resp := new(testRes)
		err := api.get("http://localhost", resp)
		return resp, msg, err
	}

	r, m, e := f(0, "test", nil)
	c.Assert(r.A, check.Equals, 0, check.Commentf(m))
	c.Assert(e.Error(), check.Equals, "0:test", check.Commentf(m))

	r, m, e = f(0, "x", fmt.Errorf("test error"))
	c.Assert(r.A, check.Equals, 0, check.Commentf(m))
	c.Assert(e.Error(), check.Equals, "test error", check.Commentf(m))

	r, m, e = f(200, "x", nil)
	c.Assert(r.A, check.Equals, 0, check.Commentf(m))
	c.Assert(strings.Contains(e.Error(), "invalid character"),
		check.Equals, true, check.Commentf(m))

	r, m, e = f(200, `{"A":1}`, nil)
	c.Assert(r.A, check.Equals, 1, check.Commentf(m))
	c.Assert(e, check.IsNil, check.Commentf(m))

	e = api.get("", nil)
	c.Assert(e.Error(), check.Equals, "invalid url")
}

// ----------------------------------------------------------------------------
// helper functions

func createRegisterRequest() (req *types.RegisterRequest) {
	req = &types.RegisterRequest{}
	return req
}
