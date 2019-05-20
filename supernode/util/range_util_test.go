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

type RangeUtilSuite struct{}

func init() {
	check.Suite(&RangeUtilSuite{})
}

func (suite *RangeUtilSuite) TestCalculatePieceNum(c *check.C) {
	var cases = []struct {
		rangeStr string
		expected int
	}{
		{
			rangeStr: "foo",
			expected: -1,
		},
		{
			rangeStr: "aaa-bbb",
			expected: -1,
		},
		{
			rangeStr: "3-2",
			expected: -1,
		},
		{
			rangeStr: "1 -3",
			expected: -1,
		},
		{
			rangeStr: "0-0",
			expected: 0,
		},
		{
			rangeStr: "3-3",
			expected: 3,
		},
		{
			rangeStr: "6-8",
			expected: 2,
		},
	}

	for _, v := range cases {
		result := CalculatePieceNum(v.rangeStr)
		c.Assert(result, check.Equals, v.expected)
	}
}

func (suite *RangeUtilSuite) TestCalculateBreakRange(c *check.C) {
	var cases = []struct {
		startPieceNum int
		pieceContSize int
		rangeLength   int64
		expected      string
		errOccured    bool
	}{
		{
			startPieceNum: 3,
			pieceContSize: 2,
			rangeLength:   50,
			expected:      "6-49",
			errOccured:    false,
		},
		{
			expected:   "",
			errOccured: true,
		},
		{
			startPieceNum: 1,
			expected:      "",
			errOccured:    true,
		},
		{
			startPieceNum: 3,
			pieceContSize: 2,
			rangeLength:   5,
			expected:      "",
			errOccured:    true,
		},
	}

	for _, v := range cases {
		result, err := CalculateBreakRange(v.startPieceNum, v.pieceContSize, v.rangeLength)
		c.Assert(result, check.Equals, v.expected)
		if v.errOccured {
			c.Assert(err, check.NotNil)
		} else {
			c.Assert(err, check.IsNil)
		}
	}
}

func (suite *RangeUtilSuite) TestCalculatePieceRange(c *check.C) {
	var cases = []struct {
		startPieceNum int
		pieceContSize int
		expected      string
	}{
		{
			startPieceNum: 3,
			pieceContSize: 2,
			expected:      "6-7",
		},
	}

	for _, v := range cases {
		result := CalculatePieceRange(v.startPieceNum, v.pieceContSize)
		c.Assert(result, check.Equals, v.expected)
	}
}
