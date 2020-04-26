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
	"bytes"
	"context"
	"fmt"

	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"

	"github.com/go-check/check"
)

type mockBufferWriterAt struct {
	buf *bytes.Buffer
}

func newMockBufferWriterAt() *mockBufferWriterAt {
	return &mockBufferWriterAt{
		buf: bytes.NewBuffer(nil),
	}
}

func (mb *mockBufferWriterAt) WriteAt(p []byte, off int64) (n int, err error) {
	if off != int64(mb.buf.Len()) {
		return 0, fmt.Errorf("failed to seek to %d", off)
	}

	return mb.buf.Write(p)
}

func (mb *mockBufferWriterAt) Bytes() []byte {
	return mb.buf.Bytes()
}

func (suite *SeedTestSuite) checkLocalDownloadDataFromFileServer(c *check.C, path string, off int64, size int64) {
	buf := newMockBufferWriterAt()

	ld := newLocalDownloader(fmt.Sprintf("http://%s/%s", suite.host, path), nil, ratelimiter.NewRateLimiter(0, 0), false)

	length, err := ld.DownloadToWriterAt(context.Background(), httputils.RangeStruct{StartIndex: off, EndIndex: off + size - 1}, 0, 0, buf, true)
	c.Check(err, check.IsNil)
	c.Check(size, check.Equals, length)

	expectData, err := suite.readFromFileServer(path, off, size)
	c.Check(err, check.IsNil)
	c.Check(string(buf.Bytes()), check.Equals, string(expectData))

	buf2 := newMockBufferWriterAt()
	ld2 := newLocalDownloader(fmt.Sprintf("http://%s/%s", suite.host, path), nil, ratelimiter.NewRateLimiter(0, 0), true)

	length2, err := ld2.DownloadToWriterAt(context.Background(), httputils.RangeStruct{StartIndex: off, EndIndex: off + size - 1}, 0, 0, buf2, false)
	c.Check(err, check.IsNil)
	c.Check(size, check.Equals, length2)
	c.Check(string(buf2.Bytes()), check.Equals, string(expectData))
}

func (suite *SeedTestSuite) TestLocalDownload(c *check.C) {
	// test read fileA
	suite.checkLocalDownloadDataFromFileServer(c, "fileA", 0, 500*1024)
	suite.checkLocalDownloadDataFromFileServer(c, "fileA", 0, 100*1024)
	for i := 0; i < 5; i++ {
		suite.checkLocalDownloadDataFromFileServer(c, "fileA", int64(i*100*1024), 100*1024)
	}

	// test read fileB
	suite.checkLocalDownloadDataFromFileServer(c, "fileB", 0, 1024*1024)
	suite.checkLocalDownloadDataFromFileServer(c, "fileB", 0, 100*1024)
	for i := 0; i < 20; i++ {
		suite.checkLocalDownloadDataFromFileServer(c, "fileB", int64(i*50*1024), 50*1024)
	}
	suite.checkLocalDownloadDataFromFileServer(c, "fileB", 1000*1024, 24*1024)

	// test read fileC
	suite.checkLocalDownloadDataFromFileServer(c, "fileC", 0, 1500*1024)
	suite.checkLocalDownloadDataFromFileServer(c, "fileC", 0, 100*1024)
	suite.checkLocalDownloadDataFromFileServer(c, "fileC", 1400*1024, 100*1024)
	for i := 0; i < 75; i++ {
		suite.checkLocalDownloadDataFromFileServer(c, "fileC", int64(i*20*1024), 20*1024)
	}

	//test read fileF
	for i := 0; i < 50; i++ {
		suite.checkLocalDownloadDataFromFileServer(c, "fileF", int64(i*20000), 20000)
	}
}
