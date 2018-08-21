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
	"math/rand"

	"github.com/go-check/check"
)

func init() {
	check.Suite(&AlgorithmTestSuite{})
}

type AlgorithmTestSuite struct{}

func (s *AlgorithmTestSuite) TestContainsString(c *check.C) {
	c.Assert(ContainsString(nil, "x"), check.Equals, false)
	c.Assert(ContainsString([]string{"x", "y"}, "x"), check.Equals, true)
	c.Assert(ContainsString([]string{"x", "y"}, "xx"), check.Equals, false)
}

func (s *AlgorithmTestSuite) TestShuffle(c *check.C) {
	// Check that Shuffle allows n=0 and n=1, but that swap is never called for them.
	rand.Seed(1)
	for n := 0; n <= 1; n++ {
		Shuffle(n, func(i, j int) { c.Fatalf("swap called, n=%d i=%d j=%d", n, i, j) })
	}

	// Check that Shuffle calls swap n-1 times when n >= 2.
	for n := 2; n <= 100; n++ {
		isRun := 0
		Shuffle(n, func(i, j int) { isRun++ })
		c.Assert(isRun, check.Equals, n-1)
	}
}
