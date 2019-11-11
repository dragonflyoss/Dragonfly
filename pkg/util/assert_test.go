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

package util

import (
	"testing"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type AssertSuite struct{}

func init() {
	check.Suite(&AssertSuite{})
}

func (suite *AssertSuite) TestMax(c *check.C) {
	c.Assert(Max(1, 2), check.Not(check.Equals), 2)
	c.Assert(Max(1, 2), check.Equals, int64(2))
	c.Assert(Max(1, 1), check.Equals, int64(1))
	c.Assert(Max(3, 2), check.Equals, int64(3))
}

func (suite *AssertSuite) TestMin(c *check.C) {
	c.Assert(Min(1, 2), check.Not(check.Equals), 1)
	c.Assert(Min(1, 2), check.Equals, int64(1))
	c.Assert(Min(1, 1), check.Equals, int64(1))
	c.Assert(Min(3, 2), check.Equals, int64(2))
}

func (suite *AssertSuite) TestIsNil(c *check.C) {
	c.Assert(IsNil(nil), check.Equals, true)
	c.Assert(IsNil(suite), check.Equals, false)

	var temp *AssertSuite
	c.Assert(IsNil(temp), check.Equals, true)
}

func (suite *AssertSuite) TestIsTrue(c *check.C) {
	c.Assert(IsTrue(true), check.Equals, true)
	c.Assert(IsTrue(false), check.Equals, false)
}

func (suite *AssertSuite) TestIsPositive(c *check.C) {
	c.Assert(IsPositive(0), check.Equals, false)
	c.Assert(IsPositive(1), check.Equals, true)
	c.Assert(IsPositive(-1), check.Equals, false)
}

func (suite *AssertSuite) TestIsNatural(c *check.C) {
	c.Assert(IsNatural("0"), check.Equals, true)
	c.Assert(IsNatural("1"), check.Equals, true)
	c.Assert(IsNatural("-1"), check.Equals, false)
}

func (suite *AssertSuite) TestIsNumeric(c *check.C) {
	c.Assert(IsNumeric("0"), check.Equals, true)
	c.Assert(IsNumeric("1"), check.Equals, true)
	c.Assert(IsNumeric("-1"), check.Equals, true)
	c.Assert(IsNumeric("1 "), check.Equals, false)
	c.Assert(IsNumeric("aaa"), check.Equals, false)
}

func (suite *AssertSuite) TestJsonString(c *check.C) {
	type T1 struct {
		A int
	}
	v1 := &T1{A: 1}
	c.Assert(JSONString(v1), check.Equals, `{"A":1}`)

	type T2 struct {
		F func()
	}
	v2 := &T2{nil}
	c.Assert(JSONString(v2), check.Equals, ``)
}
