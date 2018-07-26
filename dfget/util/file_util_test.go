/*
 * Copyright 1999-2018 Alibaba Group.
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

package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/go-check/check"
)

type FileUtilTestSuite struct {
	tmpDir string
}

func init() {
	check.Suite(&FileUtilTestSuite{})
}

func (s *FileUtilTestSuite) SetUpSuite(c *check.C) {
	s.tmpDir, _ = ioutil.TempDir("/tmp", "dfget-FileUtilTestSuite-")
}

func (s *FileUtilTestSuite) TearDownSuite(c *check.C) {
	if s.tmpDir != "" {
		if err := os.RemoveAll(s.tmpDir); err != nil {
			fmt.Printf("remove path:%s error", s.tmpDir)
		}
	}
}

func (s *FileUtilTestSuite) TestCreateDirectory(c *check.C) {
	dirPath := path.Join(s.tmpDir, "TestCreateDirectory")
	err := CreateDirectory(dirPath)
	c.Assert(err, check.IsNil)

	f, _ := os.Create(path.Join(dirPath, "createFile"))
	err = CreateDirectory(f.Name())
	c.Assert(err, check.NotNil)

	os.Chmod(dirPath, 0555)
	defer os.Chmod(dirPath, 0755)
	err = CreateDirectory(path.Join(dirPath, "1"))
	c.Assert(err, check.NotNil)
}

func (s *FileUtilTestSuite) TestPathExists(c *check.C) {
	pathStr := path.Join(s.tmpDir, "TestPathExists")
	c.Assert(PathExist(pathStr), check.Equals, false)

	os.Create(pathStr)
	c.Assert(PathExist(pathStr), check.Equals, true)
}

func (s *FileUtilTestSuite) TestIsDir(c *check.C) {
	pathStr := path.Join(s.tmpDir, "TestIsDir")
	c.Assert(IsDir(pathStr), check.Equals, false)

	os.Create(pathStr)
	c.Assert(IsDir(pathStr), check.Equals, false)
	os.Remove(pathStr)

	os.Mkdir(pathStr, 0000)
	c.Assert(IsDir(pathStr), check.Equals, true)
}
