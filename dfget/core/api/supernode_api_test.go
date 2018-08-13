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

package api

import (
	"fmt"

	"github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/dfget/types"
	"github.com/go-check/check"
)

type SupernodeAPITestSuite struct {
	mock *mockHTTPClient
	api  SupernodeAPI
}

func (s *SupernodeAPITestSuite) SetUpSuite(c *check.C) {
	s.mock = &mockHTTPClient{}
	s.api = NewSupernodeAPI()
	s.api.(*supernodeAPI).HTTPClient = s.mock
}

func (s *SupernodeAPITestSuite) TearDownTest(c *check.C) {
	s.mock.reset()
}

func init() {
	check.Suite(&SupernodeAPITestSuite{})
}

// ----------------------------------------------------------------------------
// unit tests for SupernodeAPI

func (s *SupernodeAPITestSuite) TestSupernodeAPI_Register(c *check.C) {
	ip := "127.0.0.1"

	s.mock.postJSON = s.mock.createPostJSONFunc(0, nil, nil)
	r, e := s.api.Register(ip, createRegisterRequest())
	c.Assert(r, check.IsNil)
	c.Assert(e.Error(), check.Equals, "0:")

	s.mock.postJSON = s.mock.createPostJSONFunc(0, nil,
		fmt.Errorf("test"))
	r, e = s.api.Register(ip, createRegisterRequest())
	c.Assert(r, check.IsNil)
	c.Assert(e.Error(), check.Equals, "test")

	res := types.RegisterResponse{BaseResponse: &types.BaseResponse{}}
	s.mock.postJSON = s.mock.createPostJSONFunc(200, []byte(res.String()), nil)
	r, e = s.api.Register(ip, createRegisterRequest())
	c.Assert(r, check.NotNil)
	c.Assert(r.Code, check.Equals, 0)

	res.Code = config.HTTPSuccess
	res.Data = &types.RegisterResponseData{FileLength: int64(32)}
	s.mock.postJSON = s.mock.createPostJSONFunc(200, []byte(res.String()), nil)
	r, e = s.api.Register(ip, createRegisterRequest())
	c.Assert(r, check.NotNil)
	c.Assert(r.Code, check.Equals, config.HTTPSuccess)
	c.Assert(r.Data.FileLength, check.Equals, res.Data.FileLength)
}

// ----------------------------------------------------------------------------
// helper functions

func createRegisterRequest() (req *types.RegisterRequest) {
	req = &types.RegisterRequest{}
	return req
}
