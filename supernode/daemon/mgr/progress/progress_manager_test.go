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

package progress

import (
	"testing"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/go-check/check"
	"github.com/willf/bitset"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&ProgressManagerTestSuite{})
}

type ProgressManagerTestSuite struct {
}

func (s *ProgressManagerTestSuite) TestGetSuccessfulPieces(c *check.C) {
	var cases = []struct {
		clientBitset *bitset.BitSet
		cdnBitset    *bitset.BitSet
		expected     []int
		errCheck     func(error) bool
	}{
		{
			clientBitset: bitset.New(16).Set(1).Set(9),
			cdnBitset:    bitset.New(16).Set(1).Set(9),
			expected:     []int{0, 1},
			errCheck:     errortypes.IsNilError,
		},
		{
			clientBitset: bitset.New(16).Set(9),
			cdnBitset:    bitset.New(16).Set(1).Set(9),
			expected:     []int{1},
			errCheck:     errortypes.IsNilError,
		},
		{
			clientBitset: bitset.New(16).Set(2).Set(9),
			cdnBitset:    bitset.New(16).Set(1).Set(9),
			expected:     []int{1},
			errCheck:     errortypes.IsNilError,
		},
	}

	for _, v := range cases {
		result, err := getSuccessfulPieces(v.clientBitset, v.cdnBitset)
		c.Check(v.errCheck(err), check.Equals, true)
		c.Check(result, check.DeepEquals, v.expected)
	}
}

func (s *ProgressManagerTestSuite) TestGetAvailablePieces(c *check.C) {
	var cases = []struct {
		clientBitset  *bitset.BitSet
		cdnBitset     *bitset.BitSet
		runningPieces []int
		expected      []int
		errCheck      func(error) bool
	}{
		{
			clientBitset:  bitset.New(24).Set(8),
			cdnBitset:     bitset.New(24).Set(1).Set(9),
			runningPieces: []int{1},
			expected:      []int{0},
			errCheck:      errortypes.IsNilError,
		},
		{
			clientBitset:  bitset.New(24).Set(8),
			cdnBitset:     bitset.New(24).Set(1).Set(9).Set(18),
			runningPieces: []int{1},
			expected:      nil,
			errCheck:      errortypes.IsCDNFail,
		},
	}

	for _, v := range cases {
		result, err := getAvailablePieces(v.clientBitset, v.cdnBitset, v.runningPieces)
		c.Check(v.errCheck(err), check.Equals, true)
		c.Check(result, check.DeepEquals, v.expected)
	}
}
