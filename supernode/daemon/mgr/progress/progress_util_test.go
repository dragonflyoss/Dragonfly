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

	"github.com/dragonflyoss/Dragonfly/supernode/config"

	"github.com/go-check/check"
	"github.com/willf/bitset"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&ProgressUtilTestSuite{})
}

type ProgressUtilTestSuite struct {
}

func (s *ProgressUtilTestSuite) TestUpdatePieceBitset(c *check.C) {
	var cases = []struct {
		desc        string
		pieceNum    int
		pieceStatus int
		pieceBitSet *bitset.BitSet

		updateResult   bool
		expectedBitSet *bitset.BitSet
	}{
		{
			desc:        "update piece status to PieceSUCCESS",
			pieceNum:    0,
			pieceStatus: config.PieceSUCCESS,
			pieceBitSet: bitset.New(8),

			updateResult:   true,
			expectedBitSet: bitset.New(8).SetTo(1, true),
		},
		{
			desc:        "update piece status to PieceSEMISUC",
			pieceNum:    1,
			pieceStatus: config.PieceSEMISUC,
			pieceBitSet: bitset.New(16),

			updateResult:   true,
			expectedBitSet: bitset.New(16).SetTo(9, true),
		},
		{
			desc:        "update piece status to PieceWAITING",
			pieceNum:    1,
			pieceStatus: config.PieceWAITING,
			pieceBitSet: bitset.New(16).SetTo(8, true),

			updateResult:   true,
			expectedBitSet: bitset.New(16),
		},
		{
			desc: `try to update piece status from PieceSUCCESS to PieceFAILED,
		    and the result should be false`,
			pieceNum:    1,
			pieceStatus: config.PieceFAILED,
			pieceBitSet: bitset.New(16).SetTo(9, true),

			updateResult:   false,
			expectedBitSet: bitset.New(16).SetTo(9, true),
		},
	}

	for _, v := range cases {
		result := updatePieceBitSet(v.pieceBitSet, v.pieceNum, v.pieceStatus)
		c.Check(result, check.Equals, v.updateResult)
		c.Check(v.pieceBitSet, check.DeepEquals, v.expectedBitSet)
	}
}
