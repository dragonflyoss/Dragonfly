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
	"time"

	"github.com/go-check/check"
)

type FileLockTestSuite struct {
	tmpDir string
	flockA *FileLock
	flockB *FileLock
}

func init() {
	check.Suite(&FileLockTestSuite{})
}

func (s *FileLockTestSuite) SetUpSuite(c *check.C) {
	tmpDir, _ := ioutil.TempDir("/tmp", "dfget-FileLockTestSuite-")
	os.Create(tmpDir)
	s.tmpDir = tmpDir
	s.flockA = NewFileLock(tmpDir)
	s.flockB = NewFileLock(tmpDir)
}

func (s *FileLockTestSuite) TearDownSuite(c *check.C) {
	if s.tmpDir != "" {
		if err := os.RemoveAll(s.tmpDir); err != nil {
			fmt.Printf("remove path:%s error", s.tmpDir)
		}
	}
}

func (s *FileLockTestSuite) TestFileLock(c *check.C) {
	err := s.flockA.Lock()
	c.Assert(err, check.IsNil)

	err = s.flockA.Lock()
	c.Assert(err, check.NotNil)

	err = s.flockA.Unlock()
	c.Assert(err, check.IsNil)

	err = s.flockA.Unlock()
	c.Assert(err, check.NotNil)

	s.flockA.Lock()
	start := time.Now()
	go func() {
		time.Sleep(time.Second)
		s.flockA.Unlock()
	}()
	s.flockB.Lock()
	c.Assert(time.Since(start) >= time.Second, check.Equals, true)
	s.flockB.Unlock()
}
