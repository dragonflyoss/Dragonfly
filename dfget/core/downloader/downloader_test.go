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
	"os"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/util"
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
	c.Assert(err, check.NotNil)

	err = DoDownloadTimeout(md, 50*time.Millisecond)
	c.Assert(err, check.NotNil)

	err = DoDownloadTimeout(md, 110*time.Millisecond)
	c.Assert(err, check.IsNil)
}

func (s *DownloaderTestSuite) TestConvertHeaders(c *check.C) {
	cases := []struct {
		h []string
		e map[string]string
	}{
		{
			h: []string{"a:1", "a:2", "b:", "b", "c:3"},
			e: map[string]string{"a": "1,2", "c": "3"},
		},
		{
			h: []string{},
			e: nil,
		},
	}
	for _, v := range cases {
		headers := convertHeaders(v.h)
		c.Assert(headers, check.DeepEquals, v.e)
	}
}

// ----------------------------------------------------------------------------
// helper functions

type MockDownloader struct {
	Sleep int
}

func (md *MockDownloader) Run() error {
	time.Sleep(time.Duration(md.Sleep) * time.Millisecond)
	return nil
}

func (md *MockDownloader) Cleanup() {
}

func createTestFile(name string) string {
	f, err := os.Create(name)
	if err != nil {
		return ""
	}
	defer f.Close()
	f.WriteString("test downloader")
	return util.Md5Sum(f.Name())
}
