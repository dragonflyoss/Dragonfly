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
	"crypto/md5"
	"encoding/binary"

	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"

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

// TODO: add more unit tests

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
