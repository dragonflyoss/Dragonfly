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

package stringutils

import (
	"testing"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type StringUtilSuite struct{}

func init() {
	check.Suite(&StringUtilSuite{})
}

func (suite *StringUtilSuite) TestSubString(c *check.C) {
	var cases = []struct {
		str      string
		start    int
		end      int
		expected string
	}{
		{"abcdef", 0, 3, "abc"},
		{"abcdef", 0, 6, "abcdef"},
		{"abcdef", -1, 3, ""},
		{"abcdef", 0, 7, ""},
		{"abcdef", 3, 1, ""},
	}

	for _, v := range cases {
		c.Check(SubString(v.str, v.start, v.end), check.Equals, v.expected)
	}
}

func (suite *StringUtilSuite) TestIsEmptyStr(c *check.C) {
	c.Assert(IsEmptyStr(""), check.Equals, true)
	c.Assert(IsEmptyStr("  "), check.Equals, true)
	c.Assert(IsEmptyStr("\n  "), check.Equals, true)
	c.Assert(IsEmptyStr("x"), check.Equals, false)
}
