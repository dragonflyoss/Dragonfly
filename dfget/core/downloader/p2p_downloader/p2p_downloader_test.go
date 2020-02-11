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

package downloader

import (
	"testing"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type P2PDownloaderTestSuite struct {
}

func init() {
	check.Suite(&P2PDownloaderTestSuite{})
}

func (s *P2PDownloaderTestSuite) TestWipeOutOfRange(c *check.C) {
	var cases = []struct {
		pieceRange string
		maxLength  int64
		expected   string
	}{
		{
			pieceRange: "0-5",
			maxLength:  10,
			expected:   "0-5",
		},
		{
			pieceRange: "0-5",
			maxLength:  5,
			expected:   "0-4",
		},
		{
			pieceRange: "0-5",
			maxLength:  3,
			expected:   "0-2",
		},
	}

	for _, v := range cases {
		result := wipeOutOfRange(v.pieceRange, v.maxLength)
		c.Assert(result, check.Equals, v.expected)
	}
}
