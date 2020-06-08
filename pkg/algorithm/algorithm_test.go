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

package algorithm

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestSuite(t *testing.T) {
	suite.Run(t, new(AlgorithmSuite))
}

type AlgorithmSuite struct {
	suite.Suite
}

func (suit *AlgorithmSuite) TestGCD() {
	var cases = []struct {
		x      int
		y      int
		result int
	}{
		{2, 2, 2},
		{4, 8, 4},
		{2, 5, 1},
		{10, 15, 5},
	}

	for _, v := range cases {
		result := GCD(v.x, v.y)
		suit.Equal(v.result, result)
	}
}

func (suit *AlgorithmSuite) TestGCDSlice() {
	var cases = []struct {
		slice  []int
		result int
	}{
		{nil, 1},
		{[]int{2}, 2},
		{[]int{2, 2, 4}, 2},
		{[]int{5, 10, 25}, 5},
		{[]int{1, 3, 5}, 1},
		{[]int{66, 22, 33}, 11},
	}

	for _, v := range cases {
		result := GCDSlice(v.slice)
		suit.Equal(v.result, result)
	}
}

func (suit *AlgorithmSuite) TestContainsString() {
	suit.Equal(ContainsString(nil, "x"), false)
	suit.Equal(ContainsString([]string{"x", "y"}, "x"), true)
	suit.Equal(ContainsString([]string{"x", "y"}, "xx"), false)
}

func (suit *AlgorithmSuite) TestShuffle() {
	// Check that Shuffle allows n=0 and n=1, but that swap is never called for them.
	rand.Seed(1)
	for n := 0; n <= 1; n++ {
		Shuffle(n, func(i, j int) {
			suit.Failf("swap called", "n=%d i=%d j=%d", n, i, j)
		})
	}

	// Check that Shuffle calls swap n-1 times when n >= 2.
	for n := 2; n <= 100; n++ {
		isRun := 0
		Shuffle(n, func(i, j int) { isRun++ })
		suit.Equal(isRun, n-1)
	}
}

func (suit *AlgorithmSuite) TestDedup() {
	cases := []struct {
		input  []string
		expect []string
	}{
		{
			input:  []string{},
			expect: []string{},
		},
		{
			input: []string{
				"abc", "bbc", "abc",
			},
			expect: []string{
				"abc", "bbc",
			},
		},
		{
			input: []string{
				"abc", "bbc", "abc", "bbc", "ddc", "abc",
			},
			expect: []string{
				"abc", "bbc", "ddc",
			},
		},
	}

	for _, t := range cases {
		out := DedupStringArr(t.input)
		sort.Strings(out)
		sort.Strings(t.expect)
		suit.Equal(t.expect, out)
	}
}
