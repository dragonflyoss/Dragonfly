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

package global

import (
	"regexp"
	"testing"

	. "github.com/go-check/check"
)

var reg = "x"

func TestGlobal(t *testing.T) {
	TestingT(t)
}

type GlobalSuite struct{}

func init() {
	Suite(&GlobalSuite{})
}

func (s *GlobalSuite) SetUpTest(c *C) {
	DFPattern = make(map[string]*regexp.Regexp)
}

func (s *GlobalSuite) TestUpdateDFPattern(c *C) {
	c.Assert(len(DFPattern), Equals, 0)
	UpdateDFPattern("")
	c.Assert(len(DFPattern), Equals, 0)
	UpdateDFPattern(reg)
	c.Assert(len(DFPattern), Equals, 1)
	c.Assert(DFPattern[reg].String(), Equals, reg)
}

func (s *GlobalSuite) TestCopyDfPattern(c *C) {
	copied := CopyDfPattern()
	c.Assert(len(copied), Equals, 0)
	UpdateDFPattern(reg)
	copied = CopyDfPattern()
	c.Assert(len(copied), Equals, 1)
	c.Assert(copied[0], Equals, reg)
}

func (s *GlobalSuite) TestMatchDfPattern(c *C) {
	UpdateDFPattern(reg)
	c.Assert(MatchDfPattern(reg), Equals, true)
	c.Assert(MatchDfPattern("y"), Equals, false)
	c.Assert(MatchDfPattern("xy"), Equals, true)
}
