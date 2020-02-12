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
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type DownloaderTestSuite struct {
}

func init() {
	check.Suite(&DownloaderTestSuite{})
}

func (s *DownloaderTestSuite) TestDoDownloadTimeout(c *check.C) {
	md := &MockDownloader{100}

	err := DoDownloadTimeout(md, 0*time.Millisecond)
	c.Assert(err, check.IsNil)

	err = DoDownloadTimeout(md, 50*time.Millisecond)
	c.Assert(err, check.NotNil)

	err = DoDownloadTimeout(md, 110*time.Millisecond)
	c.Assert(err, check.IsNil)
}

func (s *DownloaderTestSuite) TestMoveFile(c *check.C) {
	tmp, _ := ioutil.TempDir("/tmp", "dfget-TestMoveFile-")
	defer os.RemoveAll(tmp)

	src := filepath.Join(tmp, "a")
	dst := filepath.Join(tmp, "b")
	md5str := helper.CreateTestFileWithMD5(src, "hello")

	err := MoveFile(src, dst, "x")
	c.Assert(fileutils.PathExist(src), check.Equals, true)
	c.Assert(fileutils.PathExist(dst), check.Equals, false)
	c.Assert(err, check.NotNil)

	err = MoveFile(src, dst, md5str)
	c.Assert(fileutils.PathExist(src), check.Equals, false)
	c.Assert(fileutils.PathExist(dst), check.Equals, true)
	c.Assert(err, check.IsNil)
	content, _ := ioutil.ReadFile(dst)
	c.Assert(string(content), check.Equals, "hello")

	err = MoveFile(src, dst, "")
	c.Assert(err, check.NotNil)
}

// ----------------------------------------------------------------------------
// helper functions

type MockDownloader struct {
	Sleep int
}

func (md *MockDownloader) Run(ctx context.Context) error {
	time.Sleep(time.Duration(md.Sleep) * time.Millisecond)
	return nil
}

func (md *MockDownloader) RunStream(ctx context.Context) (io.Reader, error) {
	return nil, nil
}

func (md *MockDownloader) Cleanup() {
}
