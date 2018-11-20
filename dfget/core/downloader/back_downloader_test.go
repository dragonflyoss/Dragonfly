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

package downloader

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
	"github.com/go-check/check"
)

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
	testFileMd5 := createTestFile(path.Join(s.workHome, "download.test"))
	dst := path.Join(s.workHome, "back.test")

	ctx := helper.CreateContext(nil, s.workHome)
	bd := &BackDownloader{
		Ctx:    ctx,
		URL:    "http://" + s.host + "/download.test",
		Target: dst,
	}

	ctx.Notbs = true
	c.Assert(bd.Run(), check.NotNil)

	ctx.Notbs = false
	bd.cleaned = false
	ctx.BackSourceReason = config.BackSourceReasonNoSpace
	c.Assert(bd.Run(), check.NotNil)

	ctx.BackSourceReason = 0
	bd.cleaned = false
	c.Assert(bd.Run(), check.IsNil)

	bd.cleaned = false
	bd.Md5 = testFileMd5
	md5sum := util.Md5Sum(dst)
	c.Assert(testFileMd5, check.Equals, md5sum)

	// test: realMd5 doesn't equal to expectedMd5
	bd.Md5 = "x"
	c.Assert(bd.Run(), check.NotNil)

	// test: realMd5 equals to expectedMd5
	bd.cleaned = false
	bd.Md5 = testFileMd5
	c.Assert(bd.Run(), check.IsNil)
}
