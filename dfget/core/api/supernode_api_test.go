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

	"github.com/alibaba/Dragonfly/dfget/types"
	"github.com/alibaba/Dragonfly/dfget/util"
	"github.com/go-check/check"
)

type SupernodeAPITestSuite struct {
	mock   *mockHTTPClient
	origin util.SimpleHTTPClient
}

func (s *SupernodeAPITestSuite) SetUpSuite(c *check.C) {
	s.origin = util.DefaultHTTPClient

	s.mock = &mockHTTPClient{}
	util.DefaultHTTPClient = s.mock
}

func (s *SupernodeAPITestSuite) TearDownSuite(c *check.C) {
	util.DefaultHTTPClient = s.origin
}

func init() {
	check.Suite(&SupernodeAPITestSuite{})
}

// ----------------------------------------------------------------------------
// unit tests for SupernodeAPI

func (s *SupernodeAPITestSuite) TestSupernodeAPI_Register(c *check.C) {
	api := NewSupernodeAPI()
	api.ServicePort = 8080
	r, e := api.Register("localhost", createRegisterRequest())
	fmt.Printf("res: %v\n", r)
	fmt.Printf("err: %v\n", e)
}

// ----------------------------------------------------------------------------
// helper functions

func createRegisterRequest() (req *types.RegisterRequest) {
	req = &types.RegisterRequest{}
	return req
}
