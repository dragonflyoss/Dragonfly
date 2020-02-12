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
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type BackDownloaderTestSuite struct {
	workHome string
	host     string
	ln       net.Listener
}

func init() {
	check.Suite(&BackDownloaderTestSuite{})
}

func (s *BackDownloaderTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "dfget-BackDownloaderTestSuite-")
	s.host = fmt.Sprintf("127.0.0.1:%d", rand.Intn(1000)+63000)
	s.ln, _ = net.Listen("tcp", s.host)
	go http.Serve(s.ln, http.FileServer(http.Dir(s.workHome)))
}

func (s *BackDownloaderTestSuite) TearDownSuite(c *check.C) {
	if s.workHome != "" {
		if err := os.RemoveAll(s.workHome); err != nil {
			fmt.Printf("remove path:%s error", s.workHome)
		}
	}
	s.ln.Close()
}

func (s *BackDownloaderTestSuite) TestBackDownloader_Run(c *check.C) {
	testFileMd5 := helper.CreateTestFileWithMD5(filepath.Join(s.workHome, "download.test"), "test downloader")
	dst := filepath.Join(s.workHome, "back.test")

	cfg := helper.CreateConfig(nil, s.workHome)
	bd := &BackDownloader{
		cfg:    cfg,
		URL:    "http://" + s.host + "/download.test",
		Target: dst,
	}

	cfg.Notbs = true
	c.Assert(bd.Run(context.TODO()), check.NotNil)

	cfg.Notbs = false
	bd.cleaned = false
	cfg.BackSourceReason = config.BackSourceReasonNoSpace
	c.Assert(bd.Run(context.TODO()), check.NotNil)

	cfg.BackSourceReason = 0
	bd.cleaned = false
	c.Assert(bd.Run(context.TODO()), check.IsNil)

	bd.cleaned = false
	bd.Md5 = testFileMd5
	md5sum := fileutils.Md5Sum(dst)
	c.Assert(testFileMd5, check.Equals, md5sum)

	// test: realMd5 doesn't equal to expectedMd5
	bd.Md5 = "x"
	c.Assert(bd.Run(context.TODO()), check.NotNil)

	// test: realMd5 equals to expectedMd5
	bd.cleaned = false
	bd.Md5 = testFileMd5
	c.Assert(bd.Run(context.TODO()), check.IsNil)
}

func (s *BackDownloaderTestSuite) TestBackDownloader_RunStream(c *check.C) {
	testFileMd5 := helper.CreateTestFileWithMD5(filepath.Join(s.workHome, "download.test"), "test downloader")
	dst := filepath.Join(s.workHome, "back.test")

	cfg := helper.CreateConfig(nil, s.workHome)
	bd := &BackDownloader{
		cfg:    cfg,
		URL:    "http://" + s.host + "/download.test",
		Target: dst,
	}

	var reader io.Reader
	var err error
	cfg.Notbs = true
	_, err = bd.RunStream(context.TODO())
	c.Assert(err, check.NotNil)

	cfg.Notbs = false
	bd.cleaned = false
	cfg.BackSourceReason = config.BackSourceReasonNoSpace
	reader, err = bd.RunStream(context.TODO())
	c.Assert(reader, check.IsNil)
	c.Assert(err, check.NotNil)

	// test: realMd5 doesn't equal to expectedMd5
	bd.Md5 = "x"
	reader, err = bd.RunStream(context.TODO())

	c.Assert(reader, check.NotNil)
	if reader != nil {
		_, err = ioutil.ReadAll(reader)
	}
	c.Assert(err, check.NotNil)

	// test: realMd5 equals to expectedMd5
	bd.cleaned = false
	bd.Md5 = testFileMd5
	reader, err = bd.RunStream(context.TODO())

	c.Assert(reader, check.NotNil)
	if reader != nil {
		_, err = ioutil.ReadAll(reader)
	}
	c.Assert(err, check.IsNil)
}

func (s *BackDownloaderTestSuite) TestBackDownloader_Run_NotExist(c *check.C) {
	dst := filepath.Join(s.workHome, "back.test")

	cfg := helper.CreateConfig(nil, s.workHome)
	bd := &BackDownloader{
		cfg:    cfg,
		URL:    "http://" + s.host + "/download1.test",
		Target: dst,
	}

	err := bd.Run(context.TODO())
	c.Check(err, check.NotNil)
	c.Check(err, check.ErrorMatches, ".*404")
}
