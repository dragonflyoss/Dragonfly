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
