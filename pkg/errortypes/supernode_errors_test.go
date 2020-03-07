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

package errortypes

import (
	"github.com/go-check/check"
)

type SupernodeErrorTestSuite struct{}

func init() {
	check.Suite(&SupernodeErrorTestSuite{})
}

func (suite *SupernodeErrorTestSuite) TestIsSystemError(c *check.C) {
	err1 := New(6, "system error")
	err2 := New(0, "test")
	c.Assert(IsSystemError(*err1), check.Equals, true)
	c.Assert(IsSystemError(*err2), check.Equals, false)
}

func (suite *SupernodeErrorTestSuite) TestIsCDNFail(c *check.C) {
	err1 := New(7, "cdn status is fail")
	err2 := New(0, "test")
	c.Assert(IsCDNFail(*err1), check.Equals, true)
	c.Assert(IsCDNFail(*err2), check.Equals, false)
}

func (suite *SupernodeErrorTestSuite) TestIsCDNWait(c *check.C) {
	err1 := New(8, "cdn status is wait")
	err2 := New(0, "test")
	c.Assert(IsCDNWait(*err1), check.Equals, true)
	c.Assert(IsCDNWait(*err2), check.Equals, false)
}

func (suite *SupernodeErrorTestSuite) TestIsPeerWait(c *check.C) {
	err1 := New(9, "peer should wait")
	err2 := New(0, "test")
	c.Assert(IsPeerWait(*err1), check.Equals, true)
	c.Assert(IsPeerWait(*err2), check.Equals, false)
}

func (suite *SupernodeErrorTestSuite) TestIsUnknowError(c *check.C) {
	err1 := New(10, "unknown error")
	err2 := New(0, "test")
	c.Assert(IsUnknowError(*err1), check.Equals, true)
	c.Assert(IsUnknowError(*err2), check.Equals, false)
}

func (suite *SupernodeErrorTestSuite) TestIsPeerContinue(c *check.C) {
	err1 := New(11, "peer continue")
	err2 := New(0, "test")
	c.Assert(IsPeerContinue(*err1), check.Equals, true)
	c.Assert(IsPeerContinue(*err2), check.Equals, false)
}

func (suite *SupernodeErrorTestSuite) TestIsURLNotReachable(c *check.C) {
	err1 := New(12, "url not reachable")
	err2 := New(0, "test")
	c.Assert(IsURLNotReachable(*err1), check.Equals, true)
	c.Assert(IsURLNotReachable(*err2), check.Equals, false)
}

func (suite *SupernodeErrorTestSuite) TestIsTaskIDDuplicate(c *check.C) {
	err1 := New(13, "taskId conflict")
	err2 := New(0, "test")
	c.Assert(IsTaskIDDuplicate(*err1), check.Equals, true)
	c.Assert(IsTaskIDDuplicate(*err2), check.Equals, false)
}

func (suite *SupernodeErrorTestSuite) TestIsAuthenticationRequired(c *check.C) {
	err1 := New(14, "authentication required")
	err2 := New(0, "test")
	c.Assert(IsAuthenticationRequired(*err1), check.Equals, true)
	c.Assert(IsAuthenticationRequired(*err2), check.Equals, false)
}
