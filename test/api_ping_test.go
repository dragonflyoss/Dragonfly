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

package main

import (
	"fmt"

	"github.com/dragonflyoss/Dragonfly/test/command"
	"github.com/dragonflyoss/Dragonfly/test/request"

	"github.com/go-check/check"
)

// APIPingSuite is the test suite for info related API.
type APIPingSuite struct {
	starter *command.Starter
}

func init() {
	check.Suite(&APIPingSuite{})
}

// SetUpSuite does common setup in the beginning of each test.
func (s *APIPingSuite) SetUpSuite(c *check.C) {
	s.starter = command.NewStarter("SupernodeAPITestSuite")
	if _, err := s.starter.Supernode(0); err != nil {
		panic(fmt.Sprintf("start supernode failed:%v", err))
	}
}

func (s *APIPingSuite) TearDownSuite(c *check.C) {
	s.starter.Clean()
}

// TestPing tests /_ping API.
func (s *APIPingSuite) TestPing(c *check.C) {
	resp, err := request.Get("/_ping")
	c.Assert(err, check.IsNil)
	defer resp.Body.Close()

	CheckRespStatus(c, resp, 200)
}
