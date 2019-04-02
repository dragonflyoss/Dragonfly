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

package errors

import (
	"testing"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type ErrorTestSuite struct{}

func init() {
	check.Suite(&ErrorTestSuite{})
}

func (suite *ErrorTestSuite) TestNew(c *check.C) {
	err := New(1, "test")
	c.Assert(err.Code, check.Equals, 1)
	c.Assert(err.Msg, check.Equals, "test")
}

func (suite *ErrorTestSuite) TestNewf(c *check.C) {
	err := Newf(1, "test-%d", 2)
	c.Assert(err.Code, check.Equals, 1)
	c.Assert(err.Msg, check.Equals, "test-2")
}

func (suite *ErrorTestSuite) TestError(c *check.C) {
	err := New(1, "test")
	c.Assert(err.Error(), check.Equals, "{\"Code\":1,\"Msg\":\"test\"}")
}
