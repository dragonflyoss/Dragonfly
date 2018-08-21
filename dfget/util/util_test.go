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

package util

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type DFGetUtilSuite struct{}

func init() {
	check.Suite(&DFGetUtilSuite{})
}

func (suite *DFGetUtilSuite) SetUpTest(c *check.C) {
	rand.Seed(time.Now().UnixNano())
}

func (suite *DFGetUtilSuite) TestMax(c *check.C) {
	c.Assert(Max(1, 2), check.Not(check.Equals), 2)
	c.Assert(Max(1, 2), check.Equals, int32(2))
	c.Assert(Max(1, 1), check.Equals, int32(1))
	c.Assert(Max(3, 2), check.Equals, int32(3))
}

func (suite *DFGetUtilSuite) TestMin(c *check.C) {
	c.Assert(Min(1, 2), check.Not(check.Equals), 1)
	c.Assert(Min(1, 2), check.Equals, int32(1))
	c.Assert(Min(1, 1), check.Equals, int32(1))
	c.Assert(Min(3, 2), check.Equals, int32(2))
}

func (suite *DFGetUtilSuite) TestIsEmptyStr(c *check.C) {
	c.Assert(IsEmptyStr(""), check.Equals, true)
	c.Assert(IsEmptyStr("x"), check.Equals, false)
}

func (suite *DFGetUtilSuite) TestIsNil(c *check.C) {
	c.Assert(IsNil(nil), check.Equals, true)
	c.Assert(IsNil(suite), check.Equals, false)

	var temp *DFGetUtilSuite
	c.Assert(IsNil(temp), check.Equals, true)
}

func (suite *DFGetUtilSuite) TestPanicIfNil(c *check.C) {
	var f = func(v interface{}, msg string) (retMsg string) {
		defer func() {
			if r := recover(); r != nil {
				retMsg = r.(error).Error()
			}
		}()

		PanicIfNil(v, msg)
		return ""
	}

	c.Assert(f(1, "int"), check.Equals, "")
	c.Assert(f("", "string"), check.Equals, "")
	c.Assert(f(nil, "nil"), check.Equals, "nil")
	c.Assert(f(suite, "*DFGetUtilSuite"), check.Equals, "")
}

func (suite *DFGetUtilSuite) TestPanicIfError(c *check.C) {
	var f = func(v error, msg string) (retMsg string) {
		defer func() {
			if r := recover(); r != nil {
				retMsg = r.(error).Error()
			}
		}()

		PanicIfError(v, msg)
		return ""
	}
	c.Assert(f(nil, ""), check.Equals, "")
	c.Assert(f(fmt.Errorf("test"), "error"), check.Equals, "error: test")
}

func (suite *DFGetUtilSuite) TestJsonString(c *check.C) {
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
