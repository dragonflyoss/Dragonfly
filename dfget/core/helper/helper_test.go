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

package helper

import (
	"context"
	"fmt"
	"io/ioutil"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"

	"github.com/go-check/check"
)

// ----------------------------------------------------------------------------
// initialize

func Test(t *testing.T) {
	check.TestingT(t)
}

type HelperTestSuite struct {
}

func init() {
	check.Suite(&HelperTestSuite{})
}

// ----------------------------------------------------------------------------
// Tests

func (s *HelperTestSuite) TestDownloadPattern(c *check.C) {
	var cases = []struct {
		f        func(string) bool
		pattern  string
		expected bool
	}{
		{IsP2P, config.PatternP2P, true},
		{IsP2P, strings.ToUpper(config.PatternP2P), true},
		{IsP2P, config.PatternCDN, false},
		{IsP2P, config.PatternSource, false},

		{IsCDN, config.PatternCDN, true},
		{IsCDN, strings.ToUpper(config.PatternCDN), true},
		{IsCDN, config.PatternP2P, false},
		{IsCDN, config.PatternSource, false},

		{IsSource, config.PatternSource, true},
		{IsSource, strings.ToUpper(config.PatternSource), true},
		{IsSource, config.PatternCDN, false},
		{IsSource, config.PatternP2P, false},
	}

	var name = func(f interface{}) string {
		return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	}

	for _, v := range cases {
		c.Assert(v.f(v.pattern), check.Equals, v.expected,
			check.Commentf("f:%v pattern:%s", name(v.f), v.pattern))
	}
}

func (s *HelperTestSuite) readFromFileServer(port int, path string, off int64, size int64) ([]byte, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/%s", port, path)
	header := map[string]string{}

	if size > 0 {
		header["Range"] = fmt.Sprintf("bytes=%d-%d", off, off+size-1)
	}

	resp, err := httputils.HTTPGet(url, header)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("resp code %d", resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
}

func (s *HelperTestSuite) getRepeatStr(data []byte, size int64) []byte {
	for {
		if int64(len(data)) >= size {
			break
		}

		newData := make([]byte, len(data)*2)
		copy(newData, data)
		copy(newData[len(data):], data)
		data = newData
	}

	return data[:size]
}

func (s *HelperTestSuite) TestMockFileServer(c *check.C) {
	var (
		data []byte
		err  error
	)

	// run server
	mfs := NewMockFileServer()
	err = mfs.StartServer(context.Background(), 10011)
	c.Assert(err, check.IsNil)

	// register fileA
	err = mfs.RegisterFile("fileA", 100, "a")
	c.Assert(err, check.IsNil)

	// test fileA
	// read 0-9
	data, err = s.readFromFileServer(mfs.Port, "fileA", 0, 10)
	c.Assert(err, check.IsNil)
	c.Assert(string(data), check.Equals, string(s.getRepeatStr([]byte("a"), 10)))

	// read 1-20
	data, err = s.readFromFileServer(mfs.Port, "fileA", 1, 20)
	c.Assert(err, check.IsNil)
	c.Assert(string(data), check.Equals, string(s.getRepeatStr([]byte("a"), 20)))

	// read 1-100
	data, err = s.readFromFileServer(mfs.Port, "fileA", 1, 99)
	c.Assert(err, check.IsNil)
	c.Assert(string(data), check.Equals, string(s.getRepeatStr([]byte("a"), 99)))

	// read all
	data, err = s.readFromFileServer(mfs.Port, "fileA", 0, -1)
	c.Assert(err, check.IsNil)
	c.Assert(string(data), check.Equals, string(s.getRepeatStr([]byte("a"), 100)))

	// read 0-99
	data, err = s.readFromFileServer(mfs.Port, "fileA", 0, 100)
	c.Assert(err, check.IsNil)
	c.Assert(string(data), check.Equals, string(s.getRepeatStr([]byte("a"), 100)))

	// register fileB
	err = mfs.RegisterFile("fileB", 10000, "abcde")
	c.Assert(err, check.IsNil)

	// read 0-9
	data, err = s.readFromFileServer(mfs.Port, "fileB", 0, 10)
	c.Assert(err, check.IsNil)
	c.Assert(string(data), check.Equals, string(s.getRepeatStr([]byte("abcde"), 10)))

	// read 8-100
	data, err = s.readFromFileServer(mfs.Port, "fileB", 8, 93)
	c.Assert(err, check.IsNil)
	expectStr := s.getRepeatStr([]byte("abcde"), 101)
	c.Assert(string(data), check.Equals, string(expectStr[8:]))

	// read 1000-9999
	data, err = s.readFromFileServer(mfs.Port, "fileB", 1000, 9000)
	c.Assert(err, check.IsNil)
	expectStr = s.getRepeatStr([]byte("abcde"), 10000)
	c.Assert(string(data), check.Equals, string(expectStr[1000:]))

	// read 1001-9000
	data, err = s.readFromFileServer(mfs.Port, "fileB", 1001, 8000)
	c.Assert(err, check.IsNil)
	expectStr = s.getRepeatStr([]byte("abcde"), 9001)
	c.Assert(string(data), check.Equals, string(expectStr[1001:]))

	// read all
	data, err = s.readFromFileServer(mfs.Port, "fileB", 0, -1)
	c.Assert(err, check.IsNil)
	expectStr = s.getRepeatStr([]byte("abcde"), 10000)
	c.Assert(string(data), check.Equals, string(expectStr))
}
