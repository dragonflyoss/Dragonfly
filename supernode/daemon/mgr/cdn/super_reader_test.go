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

package cdn

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"

	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/go-check/check"
)

type SuperReaderTestSuite struct {
	workHome string
	config   string
	writer   *superWriter
}

func init() {
	check.Suite(&SuperReaderTestSuite{})
}

func (s *SuperReaderTestSuite) TestReadFile(c *check.C) {
	testStr1 := []byte("hello ")
	testStr2 := []byte("dragonfly")
	testPiece1 := append(append([]byte{0, 0, 0, 6}, testStr1...), 0x7f)
	testPiece2 := append(append([]byte{0, 0, 0, 9}, testStr2...), 0x7f)
	testStr := append(testPiece1, testPiece2...)

	contentBuf := &bytes.Buffer{}
	binary.Write(contentBuf, binary.BigEndian, testStr)

	cacheReader := newSuperReader()
	result, err := cacheReader.readFile(context.Background(), contentBuf, true, true)

	c.Check(err, check.IsNil)
	c.Check(int64(len(testStr)), check.Equals, result.fileLength)
	c.Check(2, check.Equals, result.pieceCount)
	md5Init := md5.New()
	md5Init.Write(testPiece1)
	c.Check(fmt.Sprintf("%s:%d", fileutils.GetMd5Sum(md5Init, nil), len(testStr1)+config.PieceWrapSize), check.Equals, result.pieceMd5s[0])
	md5Init.Reset()
	md5Init.Write(testPiece2)
	c.Check(fmt.Sprintf("%s:%d", fileutils.GetMd5Sum(md5Init, nil), len(testStr2)+config.PieceWrapSize), check.Equals, result.pieceMd5s[1])
	md5Init.Reset()
	md5Init.Write(testStr1)
	md5Init.Write(testStr2)
	c.Check(fileutils.GetMd5Sum(md5Init, nil), check.Equals, fileutils.GetMd5Sum(result.fileMd5, nil))
}

func (s *SuperReaderTestSuite) TestGetMD5ByReadFile(c *check.C) {
	testStr := []byte("hello dragonfly")

	for i := 0; i < len(testStr); i++ {
		contentBuf := &bytes.Buffer{}
		binary.Write(contentBuf, binary.BigEndian, testStr[:i])
		md5Init := md5.New()
		md5Init.Write([]byte(testStr[:i]))
		expectedMD5 := fileutils.GetMd5Sum(md5Init, nil)
		realMD5, err := getMD5ByReadFile(contentBuf, int32(i))
		c.Check(err, check.IsNil)
		c.Check(expectedMD5, check.Equals, realMD5)
	}
}
