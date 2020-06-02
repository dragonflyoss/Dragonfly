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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	apiTypes "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/dragonflyoss/Dragonfly/pkg/pool"

	"github.com/go-check/check"
)

type ClientWriterTestSuite struct {
	workHome    string
	serviceFile *os.File
}

func init() {
	check.Suite(&ClientWriterTestSuite{})
}

func (s *ClientWriterTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "dfget-ClientWriterTestSuite-")
	serviceFilePath := filepath.Join(s.workHome, "cwtest.service")
	s.serviceFile, _ = fileutils.OpenFile(serviceFilePath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0755)
}

func (s *ClientWriterTestSuite) TearDownSuite(c *check.C) {
	s.serviceFile.Close()
	if s.workHome != "" {
		if err := os.RemoveAll(s.workHome); err != nil {
			fmt.Printf("remove path:%s error", s.workHome)
		}
	}
}

func (s *ClientWriterTestSuite) TestWrite(c *check.C) {
	var cases = []struct {
		piece     *Piece
		cdnSource apiTypes.CdnSource
		expected  string
	}{
		{
			piece: &Piece{
				PieceNum:  0,
				PieceSize: 6,
				Content:   pool.NewBufferString("000010"),
			},
			cdnSource: apiTypes.CdnSourceSupernode,
			expected:  "1",
		},
		{
			piece: &Piece{
				PieceNum:  1,
				PieceSize: 6,
				Content:   pool.NewBufferString("000020"),
			},
			cdnSource: apiTypes.CdnSourceSupernode,
			expected:  "2",
		},
		{
			piece: &Piece{
				PieceNum:  1,
				PieceSize: 6,
				Content:   pool.NewBufferString("000030"),
			},
			cdnSource: apiTypes.CdnSourceSource,
			expected:  "000030",
		},
	}

	for _, v := range cases {
		err := writePieceToFile(v.piece, s.serviceFile, v.cdnSource)
		c.Assert(err, check.IsNil)

		var pieceHeaderLength = 5
		if v.cdnSource == apiTypes.CdnSourceSource {
			pieceHeaderLength = 0
		}
		start := int64(v.piece.PieceNum) * (int64(v.piece.PieceSize) - int64(pieceHeaderLength))
		content := s.getString(start, int(v.piece.PieceSize)-pieceHeaderLength)
		c.Check(content, check.Equals, v.expected)
	}
}

func (s *ClientWriterTestSuite) getString(start int64, length int) string {
	s.serviceFile.Seek(start, 0)
	b := make([]byte, length)
	s.serviceFile.Read(b)
	return string(b)
}
