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

func (suite *ErrorTestSuite) TestIsDataNotFound(c *check.C) {
	err1 := New(0, "data not found")
	err2 := New(11, "test")
	c.Assert(IsDataNotFound(*err1), check.Equals, true)
	c.Assert(IsDataNotFound(*err2), check.Equals, false)
}

func (suite *ErrorTestSuite) TestIsEmptyValue(c *check.C) {
	err1 := New(1, "empty value")
	err2 := New(11, "test")
	c.Assert(IsEmptyValue(*err1), check.Equals, true)
	c.Assert(IsEmptyValue(*err2), check.Equals, false)
}

func (suite *ErrorTestSuite) TestIsInvalidValue(c *check.C) {
	err1 := New(2, "invalid value")
	err2 := New(11, "test")
	c.Assert(IsInvalidValue(*err1), check.Equals, true)
	c.Assert(IsInvalidValue(*err2), check.Equals, false)
}

func (suite *ErrorTestSuite) TestIsNotInitialized(c *check.C) {
	err1 := New(3, "not initialized")
	err2 := New(11, "test")
	c.Assert(IsNotInitialized(*err1), check.Equals, true)
	c.Assert(IsNotInitialized(*err2), check.Equals, false)
}

func (suite *ErrorTestSuite) TestIsConvertFailed(c *check.C) {
	err1 := New(4, "convert failed")
	err2 := New(11, "test")
	c.Assert(IsConvertFailed(*err1), check.Equals, true)
	c.Assert(IsConvertFailed(*err2), check.Equals, false)
}

func (suite *ErrorTestSuite) TestIsRangeNotSatisfiable(c *check.C) {
	err1 := New(5, "range not satisfiable")
	err2 := New(11, "test")
	c.Assert(IsRangeNotSatisfiable(*err1), check.Equals, true)
	c.Assert(IsRangeNotSatisfiable(*err2), check.Equals, false)
}

func (suite *ErrorTestSuite) TestNewHTTPError(c *check.C) {
	err := NewHTTPError(1, "test")
	c.Assert(err.Code, check.Equals, 1)
	c.Assert(err.Msg, check.Equals, "test")
}

func (suite *ErrorTestSuite) TestHTTPError(c *check.C) {
	err := NewHTTPError(1, "test")
	c.Assert(err.Error(), check.Equals, "{\"Code\":1,\"Msg\":\"test\"}")
}

func (suite *ErrorTestSuite) TestHTTPCode(c *check.C) {
	err := NewHTTPError(1, "test")
	c.Assert(err.HTTPCode(), check.Equals, 1)
}
