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
	"testing"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type DFGetUtilSuite struct{}

func init() {
	check.Suite(&DFGetUtilSuite{})
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
