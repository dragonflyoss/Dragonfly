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
	"io"
	"sort"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/pkg/pool"

	"github.com/go-check/check"
)

type ClientStreamWriterTestSuite struct {
}

func init() {
	check.Suite(&ClientStreamWriterTestSuite{})
}

func (s *ClientStreamWriterTestSuite) SetUpSuite(*check.C) {
}

func (s *ClientStreamWriterTestSuite) TearDownSuite(*check.C) {
}

func (s *ClientStreamWriterTestSuite) TestWrite(c *check.C) {
	var cases = []struct {
		piece     *Piece
		noWrapper bool
		expected  string
	}{
		{
			piece: &Piece{
				PieceNum:  0,
				PieceSize: 6,
				Content:   pool.NewBufferString("000010"),
			},
			noWrapper: false,
			expected:  "1",
		},
		{
			piece: &Piece{
				PieceNum:  1,
				PieceSize: 6,
				Content:   pool.NewBufferString("000020"),
			},
			noWrapper: false,
			expected:  "2",
		},
		{
			piece: &Piece{
				PieceNum:  3,
				PieceSize: 6,
				Content:   pool.NewBufferString("000040"),
			},
			noWrapper: false,
			expected:  "4",
		},
		{
			piece: &Piece{
				PieceNum:  4,
				PieceSize: 6,
				Content:   pool.NewBufferString("000050"),
			},
			noWrapper: false,
			expected:  "5",
		},
		{
			piece: &Piece{
				PieceNum:  2,
				PieceSize: 6,
				Content:   pool.NewBufferString("000030"),
			},
			noWrapper: false,
			expected:  "3",
		},
	}

	cases2 := make([]struct {
		piece     *Piece
		noWrapper bool
		expected  string
	}, len(cases))
	copy(cases2, cases)

	cfg := &config.Config{}
	csw := NewClientStreamWriter(nil, nil, nil, cfg)
	go func() {
		for _, v := range cases2 {
			err := csw.writePieceToPipe(v.piece)
			c.Check(err, check.IsNil)
		}
	}()
	sort.Slice(cases, func(i, j int) bool {
		return cases[i].piece.PieceNum < cases[j].piece.PieceNum
	})
	for _, v := range cases {
		content := s.getString(csw, v.piece.RawContent(v.noWrapper).Len())
		c.Check(content, check.Equals, v.expected)
	}
}

func (s *ClientStreamWriterTestSuite) getString(reader io.Reader, length int) string {
	b := make([]byte, length)
	reader.Read(b)
	return string(b)
}
