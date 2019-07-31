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
	"github.com/dragonflyoss/Dragonfly/pkg/atomiccount"
	"github.com/dragonflyoss/Dragonfly/supernode/config"

	"github.com/go-check/check"
	"github.com/willf/bitset"
)

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
			expectedBitSet: bitset.New(8).Set(1),
		},
		{
			desc:        "update piece status to PieceSEMISUC",
			pieceNum:    1,
			pieceStatus: config.PieceSEMISUC,
			pieceBitSet: bitset.New(16),

			updateResult:   true,
			expectedBitSet: bitset.New(16).Set(9),
		},
		{
			desc:        "update piece status to PieceWAITING",
			pieceNum:    1,
			pieceStatus: config.PieceWAITING,
			pieceBitSet: bitset.New(16).Set(8),

			updateResult:   true,
			expectedBitSet: bitset.New(16),
		},
		{
			desc: `try to update piece status from PieceSUCCESS to PieceFAILED,
		    and the result should be false`,
			pieceNum:    1,
			pieceStatus: config.PieceFAILED,
			pieceBitSet: bitset.New(16).Set(9),

			updateResult:   false,
			expectedBitSet: bitset.New(16).Set(9),
		},
	}

	for _, v := range cases {
		result := updatePieceBitSet(v.pieceBitSet, v.pieceNum, v.pieceStatus)
		c.Check(result, check.Equals, v.updateResult)
		c.Check(v.pieceBitSet, check.DeepEquals, v.expectedBitSet)
	}
}

func (s *ProgressUtilTestSuite) TestUpdateBlackInfo(c *check.C) {
	pm, _ := NewManager(nil)

	updateAndCheckBlackInfo(pm, "src0", "dst0", 1, c)

	updateAndCheckBlackInfo(pm, "src0", "dst0", 2, c)

	updateAndCheckBlackInfo(pm, "src0", "dst1", 1, c)

	updateAndCheckBlackInfo(pm, "src1", "dst1", 1, c)
}

func updateAndCheckBlackInfo(pm *Manager, srcPID, dstPID string, expected int32, c *check.C) {
	err := pm.updateBlackInfo(srcPID, dstPID)
	c.Check(err, check.IsNil)
	dstPIDMap, err := pm.clientBlackInfo.GetAsMap(srcPID)
	c.Check(err, check.IsNil)
	count, err := dstPIDMap.GetAsAtomicInt(dstPID)
	c.Check(err, check.IsNil)
	c.Check(count.Get(), check.Equals, atomiccount.NewAtomicInt(expected).Get())
}
