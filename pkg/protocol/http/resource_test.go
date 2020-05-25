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

package http

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"

	"github.com/go-check/check"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func (suite *HTTPSuite) SetUpTest(c *check.C) {
	suite.port = rand.Intn(1000) + 63000 + 15
	suite.host = fmt.Sprintf("127.0.0.1:%d", suite.port)

	suite.server = helper.NewMockFileServer()
	err := suite.server.StartServer(context.Background(), suite.port)
	c.Assert(err, check.IsNil)

	// 500KB
	err = suite.server.RegisterFile("fileA", 500*1024, "abcde0123456789")
	c.Assert(err, check.IsNil)
}

func (suite *HTTPSuite) readFromFileServer(path string, off int64, size int64) ([]byte, error) {
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

func (suite *HTTPSuite) checkDataWithFileServer(c *check.C, path string, off int64, size int64, obtainedRc io.ReadCloser) {
	obtained, err := ioutil.ReadAll(obtainedRc)
	c.Assert(err, check.IsNil)
	defer obtainedRc.Close()

	expected, err := suite.readFromFileServer(path, off, size)
	c.Assert(err, check.IsNil)
	if string(obtained) != string(expected) {
		c.Errorf("path %s, range [%d-%d]: get %s, expect %s", path, off, off+size-1,
			string(obtained), string(expected))
	}

	c.Assert(string(obtained), check.Equals, string(expected))
}

func (suite *HTTPSuite) TestResource(c *check.C) {
	fileName := "fileA"
	fileLen := int64(500 * 1024)

	res := DefaultClient.GetResource(fmt.Sprintf("http://%s/%s", suite.host, fileName), nil)
	length, err := res.Length(context.Background())
	c.Assert(err, check.IsNil)
	c.Assert(length, check.Equals, fileLen)

	rc, err := res.Read(context.Background(), 0, 100)
	c.Assert(err, check.IsNil)
	suite.checkDataWithFileServer(c, fileName, 0, 100, rc)

	rc, err = res.Read(context.Background(), 1000, 100)
	c.Assert(err, check.IsNil)
	suite.checkDataWithFileServer(c, fileName, 1000, 100, rc)

	// all bytes
	rc, err = res.Read(context.Background(), 0, 0)
	c.Assert(err, check.IsNil)
	suite.checkDataWithFileServer(c, fileName, 0, 0, rc)

	_, err = res.Read(context.Background(), 1000, 0)
	c.Assert(err, check.NotNil)

	rc, err = res.Read(context.Background(), 1000, length-1000+1)
	c.Assert(err, check.IsNil)
	suite.checkDataWithFileServer(c, fileName, 1000, length-1000+1, rc)
}
