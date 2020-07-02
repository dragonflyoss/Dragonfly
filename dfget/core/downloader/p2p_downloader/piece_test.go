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
	"bytes"

	"github.com/go-check/check"

	"github.com/dragonflyoss/Dragonfly/pkg/pool"
)

type PieceTestSuite struct {
}

func init() {
	check.Suite(&PieceTestSuite{})
}

func (s *PieceTestSuite) TestRawContent(c *check.C) {
	var cases = []struct {
		piece     *Piece
		noWrapper bool
		expected  *bytes.Buffer
	}{
		{piece: &Piece{Content: pool.NewBufferString("")}, noWrapper: false, expected: nil},
		{piece: &Piece{Content: pool.NewBufferString("000010")}, noWrapper: false, expected: bytes.NewBufferString("1")},
		{piece: &Piece{Content: pool.NewBufferString("000020")}, noWrapper: true, expected: bytes.NewBufferString("000020")},
	}

	for _, v := range cases {
		result := v.piece.RawContent(v.noWrapper)
		c.Assert(result, check.DeepEquals, v.expected)
	}
}

func (s *PieceTestSuite) TestString(c *check.C) {
	var cases = []struct {
		piece    *Piece
		expected string
	}{
		{piece: &Piece{}, expected: `{"taskID":"","superNode":"","dstCid":"","range":"","result":0,"status":0,"pieceSize":0,"pieceNum":0}`},
	}

	for _, v := range cases {
		result := v.piece.String()
		c.Assert(result, check.Equals, v.expected)
	}
}
