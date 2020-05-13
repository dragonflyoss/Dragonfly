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

package api

import (
	"github.com/go-check/check"
)

type DownloadAPITestSuite struct {
}

func init() {
	check.Suite(&DownloadAPITestSuite{})
}

// ----------------------------------------------------------------------------
// unit tests for DownloadAPI

func (s *DownloadAPITestSuite) TestGetRealRange(c *check.C) {
	cases := []struct {
		pieceRange string
		rangeValue string
		expected   string
	}{
		{"", "0-1", ""},
		{"0-1", "", "0-1"},
		{"0-1", "1-100", "1-2"},
		{"0-100", "1-100", "1-100"},
		{"100-100", "1-100", "101-100"},
		{"100-200", "1-100", "101-100"},
	}

	for _, v := range cases {
		res := getRealRange(v.pieceRange, "bytes="+v.rangeValue)
		c.Assert(res, check.Equals, v.expected,
			check.Commentf("%v", v))
	}
}
