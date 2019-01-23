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

package helper

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/go-check/check"
)

// ----------------------------------------------------------------------------
// initialize

func Test(t *testing.T) {
	check.TestingT(t)
}

type HelperTestSuite struct {
}

func init() {
	check.Suite(&HelperTestSuite{})
}

// ----------------------------------------------------------------------------
// Tests

func (s *HelperTestSuite) TestDownloadPattern(c *check.C) {
	var cases = []struct {
		f        func(string) bool
		pattern  string
		expected bool
	}{
		{IsP2P, config.PatternP2P, true},
		{IsP2P, strings.ToUpper(config.PatternP2P), true},
		{IsP2P, config.PatternCDN, false},
		{IsP2P, config.PatternSource, false},

		{IsCDN, config.PatternCDN, true},
		{IsCDN, strings.ToUpper(config.PatternCDN), true},
		{IsCDN, config.PatternP2P, false},
		{IsCDN, config.PatternSource, false},

		{IsSource, config.PatternSource, true},
		{IsSource, strings.ToUpper(config.PatternSource), true},
		{IsSource, config.PatternCDN, false},
		{IsSource, config.PatternP2P, false},
	}

	var name = func(f interface{}) string {
		return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	}

	for _, v := range cases {
		c.Assert(v.f(v.pattern), check.Equals, v.expected,
			check.Commentf("f:%v pattern:%s", name(v.f), v.pattern))
	}
}
