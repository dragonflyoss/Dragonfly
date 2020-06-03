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
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/go-check/check"
)

func (suite *SeedTestSuite) TestFileCacheBufferWithNoFile(c *check.C) {
	testDir := suite.tmpDir

	cb, err := newFileCacheBuffer(filepath.Join(testDir, "fileA"), 30, true, false, 0)
	c.Assert(err, check.IsNil)

	data := []byte("0123456789")
	// write data
	n, err := cb.WriteAt(data, 0)
	c.Assert(int(n), check.Equals, len(data))
	c.Assert(err, check.IsNil)

	// write data
	n, err = cb.WriteAt(data, 10)
	c.Assert(int(n), check.Equals, len(data))
	c.Assert(err, check.IsNil)

	// read stream
	rc, err := cb.ReadStream(0, 10)
	c.Assert(err, check.IsNil)
	data0, err := ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	c.Assert(len(data0), check.Equals, 10)
	expectAllData := []byte("0123456789")
	c.Assert(string(data0), check.Equals, string(expectAllData))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	rc, err = cb.ReadStream(0, 20)
	c.Assert(err, check.IsNil)
	data0, err = ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	c.Assert(len(data0), check.Equals, 20)
	expectAllData = []byte("01234567890123456789")
	c.Assert(string(data0), check.Equals, string(expectAllData))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	// write data
	n, err = cb.WriteAt(data, 20)
	c.Assert(int(n), check.Equals, len(data))
	c.Assert(err, check.IsNil)
	err = cb.Sync()
	c.Assert(err, check.IsNil)

	// close
	err = cb.Close()
	c.Assert(err, check.IsNil)

	// read all
	rc, err = cb.ReadStream(0, -1)
	c.Assert(err, check.IsNil)
	data1, err := ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	c.Assert(len(data1), check.Equals, len(data)*3)
	expectAllData = []byte("012345678901234567890123456789")
	c.Assert(string(data1), check.Equals, string(expectAllData))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	// read 10-
	rc, err = cb.ReadStream(10, -1)
	c.Assert(err, check.IsNil)
	data2, err := ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	c.Assert(len(data2), check.Equals, 20)
	expectData2 := []byte("01234567890123456789")
	c.Assert(string(data2), check.Equals, string(expectData2))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	// read 5-14
	rc, err = cb.ReadStream(5, 10)
	c.Assert(err, check.IsNil)
	data3, err := ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	c.Assert(len(data3), check.Equals, 10)
	expectData3 := []byte("5678901234")
	c.Assert(string(data3), check.Equals, string(expectData3))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	// read 20-30, expect failed
	_, err = cb.ReadStream(20, 11)
	httpErr, ok := err.(*errortypes.HTTPError)
	c.Assert(ok, check.Equals, true)
	c.Assert(httpErr.HTTPCode(), check.Equals, http.StatusRequestedRangeNotSatisfiable)

	// remove cache
	err = cb.Remove()
	c.Assert(err, check.IsNil)

	// read again
	_, err = cb.ReadStream(20, 5)
	c.Assert(err, check.NotNil)
}

func (suite *SeedTestSuite) TestFileCacheBufferWithExistFile(c *check.C) {
	testDir := suite.tmpDir

	// create cb
	cb, err := newFileCacheBuffer(filepath.Join(testDir, "fileB"), 35, true, false, 0)
	c.Assert(err, check.IsNil)

	inputData1 := []byte("0123456789")
	inputData2 := []byte("abcde")

	// write data inputData1 * 3
	n, err := cb.WriteAt(inputData1, 0)
	c.Assert(int(n), check.Equals, len(inputData1))
	c.Assert(err, check.IsNil)

	n, err = cb.WriteAt(inputData1, 10)
	c.Assert(int(n), check.Equals, len(inputData1))
	c.Assert(err, check.IsNil)

	n, err = cb.WriteAt(inputData1, 20)
	c.Assert(int(n), check.Equals, len(inputData1))
	c.Assert(err, check.IsNil)

	err = cb.Close()
	c.Assert(err, check.IsNil)

	// reopen again
	cb, err = newFileCacheBuffer(filepath.Join(testDir, "fileB"), 35, false, false, 0)
	c.Assert(err, check.IsNil)

	rc, err := cb.ReadStream(0, 30)
	c.Assert(err, check.IsNil)
	data0, err := ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	c.Assert(len(data0), check.Equals, 30)
	expectAllData := []byte("012345678901234567890123456789")
	c.Assert(string(data0), check.Equals, string(expectAllData))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	// write  data inputData2
	n, err = cb.WriteAt(inputData2, 30)
	c.Assert(int(n), check.Equals, len(inputData2))
	c.Assert(err, check.IsNil)

	// close
	err = cb.Close()
	c.Assert(err, check.IsNil)

	// read all
	rc, err = cb.ReadStream(0, -1)
	c.Assert(err, check.IsNil)
	data1, err := ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	c.Assert(len(data1), check.Equals, 35)
	expectAllData = []byte("012345678901234567890123456789abcde")
	c.Assert(string(data1), check.Equals, string(expectAllData))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	// read 5-29
	rc, err = cb.ReadStream(5, 25)
	c.Assert(err, check.IsNil)
	data2, err := ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	c.Assert(len(data2), check.Equals, 25)
	expectData2 := []byte("5678901234567890123456789")
	c.Assert(string(data2), check.Equals, string(expectData2))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	// read 10-34
	rc, err = cb.ReadStream(10, 25)
	c.Assert(err, check.IsNil)
	data3, err := ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	c.Assert(len(data3), check.Equals, 25)
	expectData3 := []byte("01234567890123456789abcde")
	c.Assert(string(data3), check.Equals, string(expectData3))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	// read 10-35. expect failed
	_, err = cb.ReadStream(10, 26)
	httpErr, ok := err.(*errortypes.HTTPError)
	c.Assert(ok, check.Equals, true)
	c.Assert(httpErr.HTTPCode(), check.Equals, http.StatusRequestedRangeNotSatisfiable)

	// remove cache
	err = cb.Remove()
	c.Assert(err, check.IsNil)

	// read again
	_, err = cb.ReadStream(20, 5)
	c.Assert(err, check.NotNil)
}

func (suite *SeedTestSuite) TestCacheMemoryMode(c *check.C) {
	testDir := suite.tmpDir
	cb1, err := newFileCacheBuffer(filepath.Join(testDir, "TestCacheMemoryModeFileB"), 64, true, true, 4)
	c.Assert(err, check.IsNil)

	inputData1 := []byte("0123456789abcdef")
	inputData2 := []byte("fedcba9876543210")

	data1 := make([]byte, 16)
	copy(data1, inputData1)
	n, err := cb1.WriteAt(data1, 0)
	c.Assert(err, check.IsNil)
	c.Assert(n, check.Equals, 16)

	rc, err := cb1.ReadStream(0, 10)
	c.Assert(err, check.IsNil)
	rcData, err := ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	expectData := []byte("0123456789")
	c.Assert(string(rcData), check.Equals, string(expectData))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	data1 = make([]byte, 16)
	copy(data1, inputData2)
	n, err = cb1.WriteAt(data1, 32)
	c.Assert(err, check.IsNil)
	c.Assert(n, check.Equals, 16)

	rc, err = cb1.ReadStream(32, 10)
	c.Assert(err, check.IsNil)
	rcData, err = ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	expectData = []byte("fedcba9876")
	c.Assert(string(rcData), check.Equals, string(expectData))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	// sync
	err = cb1.Sync()
	c.Check(err, check.IsNil)

	data1 = make([]byte, 16)
	copy(data1, inputData2)
	n, err = cb1.WriteAt(data1, 16)
	c.Assert(err, check.IsNil)
	c.Assert(n, check.Equals, 16)

	rc, err = cb1.ReadStream(10, 20)
	c.Assert(err, check.IsNil)
	rcData, err = ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	expectData = []byte("abcdeffedcba98765432")
	c.Assert(string(rcData), check.Equals, string(expectData))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	rc, err = cb1.ReadStream(10, 20)
	c.Assert(err, check.IsNil)
	rcData, err = ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	expectData = []byte("abcdeffedcba98765432")
	c.Assert(string(rcData), check.Equals, string(expectData))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	rc, err = cb1.ReadStream(15, 20)
	c.Assert(err, check.IsNil)
	mrc, ok := rc.(*multiReadCloser)
	c.Assert(ok, check.Equals, true)
	c.Assert(len(mrc.rds), check.Equals, 3)
	_, ok = mrc.rds[0].(*io.SectionReader)
	c.Assert(ok, check.Equals, true)
	_, ok = mrc.rds[1].(*bytes.Reader)
	c.Assert(ok, check.Equals, true)
	_, ok = mrc.rds[2].(*io.SectionReader)
	c.Assert(ok, check.Equals, true)
	rcData, err = ioutil.ReadAll(rc)
	c.Assert(err, check.IsNil)
	expectData = []byte("ffedcba9876543210fed")
	c.Assert(string(rcData), check.Equals, string(expectData))
	err = rc.Close()
	c.Assert(err, check.IsNil)

	// sync
	err = cb1.Sync()
	c.Assert(err, check.IsNil)
	rc, err = cb1.ReadStream(15, 20)
	c.Assert(err, check.IsNil)
	_, ok = rc.(*fileReadCloser)
	c.Assert(ok, check.Equals, true)
	expectData = []byte("ffedcba9876543210fed")
	c.Assert(string(rcData), check.Equals, string(expectData))
	err = rc.Close()
	c.Assert(err, check.IsNil)
}
