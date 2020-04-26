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

package seed

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type SeedTestSuite struct {
	port     int
	host     string
	server   *helper.MockFileServer
	tmpDir   string
	cacheDir string
}

func init() {
	rand.Seed(time.Now().Unix())
	check.Suite(&SeedTestSuite{})
}

func (suite *SeedTestSuite) SetUpSuite(c *check.C) {
	suite.tmpDir = "./testdata"
	err := os.MkdirAll(suite.tmpDir, 0774)
	c.Assert(err, check.IsNil)

	suite.cacheDir = "./testcache"
	err = os.MkdirAll(suite.cacheDir, 0774)
	c.Assert(err, check.IsNil)

	suite.port = rand.Intn(1000) + 63000
	suite.host = fmt.Sprintf("127.0.0.1:%d", suite.port)

	suite.server = helper.NewMockFileServer()
	err = suite.server.StartServer(context.Background(), suite.port)
	c.Assert(err, check.IsNil)

	// 500KB
	err = suite.server.RegisterFile("fileA", 500*1024, "abcde0123456789")
	c.Assert(err, check.IsNil)
	// 1MB
	err = suite.server.RegisterFile("fileB", 1024*1024, "abcdefg")
	c.Assert(err, check.IsNil)
	// 1.5 MB
	err = suite.server.RegisterFile("fileC", 1500*1024, "abcdefg")
	c.Assert(err, check.IsNil)
	// 2 MB
	err = suite.server.RegisterFile("fileD", 2048*1024, "abcdefg")
	c.Assert(err, check.IsNil)
	// 9.5 MB
	err = suite.server.RegisterFile("fileE", 9500*1024, "abcdefg")
	c.Assert(err, check.IsNil)
	// 10 MB
	err = suite.server.RegisterFile("fileF", 10*1024*1024, "abcdefg")
	c.Assert(err, check.IsNil)
	// 1 G
	err = suite.server.RegisterFile("fileG", 1024*1024*1024, "1abcdefg")
	c.Assert(err, check.IsNil)

	// 100 M
	err = suite.server.RegisterFile("fileH", 100*1024*1024, "1abcdefg")
	c.Assert(err, check.IsNil)
}

func (suite *SeedTestSuite) TearDownSuite(c *check.C) {
	if suite.tmpDir != "" {
		os.RemoveAll(suite.tmpDir)
	}
	if suite.cacheDir != "" {
		os.RemoveAll(suite.cacheDir)
	}
}

func (suite *SeedTestSuite) readFromFileServer(path string, off int64, size int64) ([]byte, error) {
	url := fmt.Sprintf("http://%s/%s", suite.host, path)
	header := map[string]string{}

	if size > 0 {
		header["Range"] = fmt.Sprintf("bytes=%d-%d", off, off+size-1)
	}

	code, data, err := httputils.GetWithHeaders(url, header, 5*time.Second)
	if err != nil {
		return nil, err
	}

	if code >= 400 {
		return nil, fmt.Errorf("resp code %d", code)
	}

	return data, nil
}

func (suite *SeedTestSuite) checkDataWithFileServer(c *check.C, path string, off int64, size int64, obtained []byte) {
	expected, err := suite.readFromFileServer(path, off, size)
	c.Assert(err, check.IsNil)
	if string(obtained) != string(expected) {
		c.Errorf("path %s, range [%d-%d]: get %s, expect %s", path, off, off+size-1,
			string(obtained), string(expected))
	}

	c.Assert(string(obtained), check.Equals, string(expected))
}

func (suite *SeedTestSuite) checkFileWithSeed(c *check.C, path string, fileLength int64, sd Seed) {
	// download all
	rc, err := sd.Download(0, -1)
	c.Assert(err, check.IsNil)
	obtainedData, err := ioutil.ReadAll(rc)
	rc.Close()
	c.Assert(err, check.IsNil)
	suite.checkDataWithFileServer(c, path, 0, -1, obtainedData)

	// download {fileLength-100KB}- {fileLength}-1
	rc, err = sd.Download(fileLength-100*1024, 100*1024)
	c.Assert(err, check.IsNil)
	obtainedData, err = ioutil.ReadAll(rc)
	rc.Close()
	c.Assert(err, check.IsNil)
	suite.checkDataWithFileServer(c, path, fileLength-100*1024, 100*1024, obtainedData)

	// download 0-{100KB-1}
	rc, err = sd.Download(0, 100*1024)
	c.Assert(err, check.IsNil)
	obtainedData, err = ioutil.ReadAll(rc)
	rc.Close()
	c.Assert(err, check.IsNil)
	suite.checkDataWithFileServer(c, path, 0, 100*1024, obtainedData)

	start := int64(0)
	end := int64(0)
	rangeSize := int64(100 * 1024)

	for {
		end = start + rangeSize - 1
		if end >= fileLength {
			end = fileLength - 1
		}

		if start > end {
			break
		}

		rc, err = sd.Download(start, end-start+1)
		c.Assert(err, check.IsNil)
		obtainedData, err = ioutil.ReadAll(rc)
		rc.Close()
		c.Assert(err, check.IsNil)
		suite.checkDataWithFileServer(c, path, start, end-start+1, obtainedData)
		start = end + 1
	}

	start = 0
	end = 0
	rangeSize = 99 * 1023

	for {
		end = start + rangeSize - 1
		if end >= fileLength {
			end = fileLength - 1
		}

		if start > end {
			break
		}

		rc, err = sd.Download(start, end-start+1)
		c.Assert(err, check.IsNil)
		obtainedData, err = ioutil.ReadAll(rc)
		rc.Close()
		c.Assert(err, check.IsNil)
		suite.checkDataWithFileServer(c, path, start, end-start+1, obtainedData)
		start = end + 1
	}
}
