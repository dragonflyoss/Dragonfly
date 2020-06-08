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

package fileutils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type FileUtilTestSuite struct {
	tmpDir   string
	username string
}

func init() {
	check.Suite(&FileUtilTestSuite{})
}

func (s *FileUtilTestSuite) SetUpSuite(c *check.C) {
	s.tmpDir, _ = ioutil.TempDir("/tmp", "dfget-FileUtilTestSuite-")
	if u, e := user.Current(); e == nil {
		s.username = u.Username
	}
}

func (s *FileUtilTestSuite) TearDownSuite(c *check.C) {
	if s.tmpDir != "" {
		if err := os.RemoveAll(s.tmpDir); err != nil {
			fmt.Printf("remove path:%s error", s.tmpDir)
		}
	}
}

func (s *FileUtilTestSuite) TestCreateDirectory(c *check.C) {
	dirPath := filepath.Join(s.tmpDir, "TestCreateDirectory")
	err := CreateDirectory(dirPath)
	c.Assert(err, check.IsNil)

	f, _ := os.Create(filepath.Join(dirPath, "createFile"))
	err = CreateDirectory(f.Name())
	c.Assert(err, check.NotNil)

	os.Chmod(dirPath, 0555)
	defer os.Chmod(dirPath, 0755)
	err = CreateDirectory(filepath.Join(dirPath, "1"))
	if s.username != "root" {
		c.Assert(err, check.NotNil)
	} else {
		c.Assert(err, check.IsNil)
	}
}

func (s *FileUtilTestSuite) TestPathExists(c *check.C) {
	pathStr := filepath.Join(s.tmpDir, "TestPathExists")
	c.Assert(PathExist(pathStr), check.Equals, false)

	os.Create(pathStr)
	c.Assert(PathExist(pathStr), check.Equals, true)
}

func (s *FileUtilTestSuite) TestIsDir(c *check.C) {
	pathStr := filepath.Join(s.tmpDir, "TestIsDir")
	c.Assert(IsDir(pathStr), check.Equals, false)

	os.Create(pathStr)
	c.Assert(IsDir(pathStr), check.Equals, false)
	os.Remove(pathStr)

	os.Mkdir(pathStr, 0000)
	c.Assert(IsDir(pathStr), check.Equals, true)
}

func (s *FileUtilTestSuite) TestDeleteFile(c *check.C) {
	pathStr := filepath.Join(s.tmpDir, "TestDeleteFile")
	os.Create(pathStr)
	err := DeleteFile(pathStr)
	c.Assert(err, check.IsNil)

	dirStr := filepath.Join(s.tmpDir, "test_delete_file")
	os.Mkdir(dirStr, 0000)
	err = DeleteFile(dirStr)
	c.Assert(err, check.NotNil)

	f := filepath.Join(s.tmpDir, "test", "empty", "file")
	err = DeleteFile(f)
	c.Assert(err, check.NotNil)
}

func (s *FileUtilTestSuite) TestDeleteFiles(c *check.C) {
	f1 := filepath.Join(s.tmpDir, "TestDeleteFile001")
	f2 := filepath.Join(s.tmpDir, "TestDeleteFile002")
	os.Create(f1)
	DeleteFiles(f1, f2)
	c.Assert(PathExist(f1) || PathExist(f2), check.Equals, false)
}

func (s *FileUtilTestSuite) TestMoveFile(c *check.C) {

	f1 := filepath.Join(s.tmpDir, "TestMovefileSrc01")
	f2 := filepath.Join(s.tmpDir, "TestMovefileDstExist")
	os.Create(f1)
	os.Create(f2)
	ioutil.WriteFile(f1, []byte("Test move file src"), 0755)
	f1Md5 := Md5Sum(f1)
	err := MoveFile(f1, f2)
	c.Assert(err, check.IsNil)

	f2Md5 := Md5Sum(f2)
	c.Assert(f1Md5, check.Equals, f2Md5)

	f3 := filepath.Join(s.tmpDir, "TestMovefileSrc02")
	f4 := filepath.Join(s.tmpDir, "TestMovefileDstNonExist")
	os.Create(f3)
	ioutil.WriteFile(f3, []byte("Test move file src when dst not exist"), 0755)
	f3Md5 := Md5Sum(f3)
	err = MoveFile(f3, f4)
	c.Assert(err, check.IsNil)
	f4Md5 := Md5Sum(f4)
	c.Assert(f3Md5, check.Equals, f4Md5)

	f1 = filepath.Join(s.tmpDir, "TestMovefileSrcDir")
	os.Mkdir(f1, 0755)
	err = MoveFile(f1, f2)
	c.Assert(err, check.NotNil)
}

func (s *FileUtilTestSuite) TestOpenFile(c *check.C) {
	f1 := filepath.Join(s.tmpDir, "dir1", "TestOpenFile")
	_, err := OpenFile(f1, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)
	c.Assert(err, check.IsNil)

	f2 := filepath.Join(s.tmpDir, "TestOpenFile")
	os.Create(f2)
	_, err = OpenFile(f2, os.O_RDONLY, 0666)
	c.Assert(err, check.IsNil)
}

func (s *FileUtilTestSuite) TestLink(c *check.C) {
	pathStr := filepath.Join(s.tmpDir, "TestLinkFile")
	os.Create(pathStr)
	linkStr := filepath.Join(s.tmpDir, "TestLinkName")

	err := Link(pathStr, linkStr)
	c.Assert(err, check.IsNil)
	c.Assert(PathExist(linkStr), check.Equals, true)

	linkStr = filepath.Join(s.tmpDir, "TestLinkNameExist")
	os.Create(linkStr)
	err = Link(pathStr, linkStr)
	c.Assert(err, check.IsNil)
	c.Assert(PathExist(linkStr), check.Equals, true)

	linkStr = filepath.Join(s.tmpDir, "testLinkNonExistDir")
	os.Mkdir(linkStr, 0755)
	err = Link(pathStr, linkStr)
	c.Assert(err, check.NotNil)
}

func (s *FileUtilTestSuite) TestSymbolicLink(c *check.C) {
	pathStr := filepath.Join(s.tmpDir, "TestSymLinkFileNonExist")
	linkStr := filepath.Join(s.tmpDir, "TestSymLinkNameFileNonExist")
	err := SymbolicLink(pathStr, linkStr)
	c.Assert(err, check.NotNil)
	c.Assert(PathExist(linkStr), check.Equals, false)

	pathStr = filepath.Join(s.tmpDir, "TestSymLinkDir")
	os.Mkdir(pathStr, 0755)
	linkStr = filepath.Join(s.tmpDir, "TestSymLinkNameDir")
	err = SymbolicLink(pathStr, linkStr)
	c.Assert(err, check.IsNil)
	c.Assert(PathExist(linkStr), check.Equals, true)

	pathStr = filepath.Join(s.tmpDir, "TestSymLinkFile")
	os.Create(pathStr)
	linkStr = filepath.Join(s.tmpDir, "TestSymLinkNameFile")
	err = SymbolicLink(pathStr, linkStr)
	c.Assert(err, check.IsNil)
	c.Assert(PathExist(linkStr), check.Equals, true)

	linkStr = filepath.Join(s.tmpDir, "TestSymLinkNameDirExist")
	os.Mkdir(linkStr, 0755)
	err = SymbolicLink(pathStr, linkStr)
	c.Assert(err, check.NotNil)

	linkStr = filepath.Join(s.tmpDir, "TestSymLinkNameFileExist")
	os.Create(linkStr)
	err = SymbolicLink(pathStr, linkStr)
	c.Assert(err, check.IsNil)
}

func (s *FileUtilTestSuite) TestCopyFile(c *check.C) {
	srcPath := filepath.Join(s.tmpDir, "TestCopyFileSrc")
	dstPath := filepath.Join(s.tmpDir, "TestCopyFileDst")
	err := CopyFile(srcPath, dstPath)
	c.Assert(err, check.NotNil)

	os.Create(srcPath)
	os.Create(dstPath)
	ioutil.WriteFile(srcPath, []byte("Test copy file"), 0755)
	err = CopyFile(srcPath, dstPath)
	c.Assert(err, check.NotNil)

	tmpPath := filepath.Join(s.tmpDir, "TestCopyFileTmp")
	err = CopyFile(srcPath, tmpPath)
	c.Assert(err, check.IsNil)
}

func (s *FileUtilTestSuite) TestMoveFileAfterCheckMd5(c *check.C) {
	srcPath := filepath.Join(s.tmpDir, "TestMoveFileAfterCheckMd5Src")
	dstPath := filepath.Join(s.tmpDir, "TestMoveFileAfterCheckMd5Dst")
	os.Create(srcPath)
	ioutil.WriteFile(srcPath, []byte("Test move file after check md5"), 0755)
	srcPathMd5 := Md5Sum(srcPath)
	err := MoveFileAfterCheckMd5(srcPath, dstPath, srcPathMd5)
	c.Assert(err, check.IsNil)
	dstPathMd5 := Md5Sum(dstPath)
	c.Assert(srcPathMd5, check.Equals, dstPathMd5)

	ioutil.WriteFile(srcPath, []byte("Test move file afte md5, change content"), 0755)
	err = MoveFileAfterCheckMd5(srcPath, dstPath, srcPathMd5)
	c.Assert(err, check.NotNil)

	srcPath = filepath.Join(s.tmpDir, "TestMoveFileAfterCheckMd5Dir")
	os.Mkdir(srcPath, 0755)
	err = MoveFileAfterCheckMd5(srcPath, dstPath, srcPathMd5)
	c.Assert(err, check.NotNil)
}

func (s *FileUtilTestSuite) TestMd5sum(c *check.C) {
	pathStr := filepath.Join(s.tmpDir, "TestMd5Sum")
	_, _ = OpenFile(pathStr, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0000)
	pathStrMd5 := Md5Sum(pathStr)
	if s.username != "root" {
		c.Assert(pathStrMd5, check.Equals, "")
	} else {
		c.Assert(pathStrMd5, check.Equals, "d41d8cd98f00b204e9800998ecf8427e")
	}

	pathStr = filepath.Join(s.tmpDir, "TestMd5SumDir")
	os.Mkdir(pathStr, 0755)
	pathStrMd5 = Md5Sum(pathStr)
	c.Assert(pathStrMd5, check.Equals, "")
}

func (s *FileUtilTestSuite) TestLoadYaml(c *check.C) {
	type T struct {
		A int    `yaml:"a"`
		B string `yaml:"b"`
	}
	var cases = []struct {
		create   bool
		content  string
		errMsg   string
		expected *T
	}{
		{create: false, content: "", errMsg: ".*no such file or directory", expected: nil},
		{create: true, content: "a: x",
			errMsg: ".*yaml: unmarshal.*(\n.*)*", expected: nil},
		{create: true, content: "a: 1", errMsg: "", expected: &T{1, ""}},
		{
			create:   true,
			content:  "a: 1\nb: x",
			errMsg:   "",
			expected: &T{1, "x"},
		},
	}

	for idx, v := range cases {
		filename := filepath.Join(s.tmpDir, fmt.Sprintf("test-%d", idx))
		if v.create {
			ioutil.WriteFile(filename, []byte(v.content), os.ModePerm)
		}
		var t T
		err := LoadYaml(filename, &t)
		if v.expected == nil {
			c.Assert(err, check.NotNil)
			c.Assert(err, check.ErrorMatches, v.errMsg,
				check.Commentf("err:%v expected:%s", err, v.errMsg))
		} else {
			c.Assert(err, check.IsNil)
			c.Assert(&t, check.DeepEquals, v.expected)
		}

	}
}

func (s *FileUtilTestSuite) TestIsRegularFile(c *check.C) {
	pathStr := filepath.Join(s.tmpDir, "TestIsRegularFile")
	c.Assert(IsRegularFile(pathStr), check.Equals, false)

	os.Create(pathStr)
	c.Assert(IsRegularFile(pathStr), check.Equals, true)
	os.Remove(pathStr)

	// Don't set mode to create a non-regular file
	os.OpenFile(pathStr, 0, 0666)
	c.Assert(IsRegularFile(pathStr), check.Equals, false)
	os.Remove(pathStr)
}

func (s *FileUtilTestSuite) TestIsEmptyDir(c *check.C) {
	pathStr := filepath.Join(s.tmpDir, "TestIsEmptyDir")

	// not exist
	empty, err := IsEmptyDir(pathStr)
	c.Assert(empty, check.Equals, false)
	c.Assert(err, check.NotNil)

	// not a directory
	_, _ = os.Create(pathStr)
	empty, err = IsEmptyDir(pathStr)
	c.Assert(empty, check.Equals, false)
	c.Assert(err, check.NotNil)
	_ = os.Remove(pathStr)

	// empty
	_ = os.Mkdir(pathStr, 0755)
	empty, err = IsEmptyDir(pathStr)
	c.Assert(empty, check.Equals, true)
	c.Assert(err, check.IsNil)

	// not empty
	childPath := filepath.Join(pathStr, "child")
	_ = os.Mkdir(childPath, 0755)
	empty, err = IsEmptyDir(pathStr)
	c.Assert(empty, check.Equals, false)
	c.Assert(err, check.IsNil)
	_ = os.Remove(pathStr)
}
