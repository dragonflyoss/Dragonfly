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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	. "path/filepath"
	"time"

	"github.com/dragonflyoss/Dragonfly/test/command"
	"github.com/dragonflyoss/Dragonfly/test/environment"

	"github.com/go-check/check"
)

func init() {
	check.Suite(&DFGetP2PTestSuite{})
}

type DFGetP2PTestSuite struct {
	starter *command.Starter
}

func (s *DFGetP2PTestSuite) SetUpSuite(c *check.C) {
	s.starter = command.NewStarter("DFGetP2PTestSuite")
	if _, err := s.starter.Supernode(0); err != nil {
		panic(fmt.Sprintf("start supernode failed:%v", err))
	}
	if _, err := s.starter.DFGetServer(0, "--ip", "localhost"); err != nil {
		panic(fmt.Sprintf("start dfget server failed:%v", err))
	}
}

func (s *DFGetP2PTestSuite) TearDownSuite(c *check.C) {
	s.starter.Clean()
}

func (s *DFGetP2PTestSuite) TestDownloadFile(c *check.C) {
	var cases = []struct {
		filePath    string
		fileContent []byte
		targetPath  string
		createFile  bool
		execSuccess bool
		timeout     time.Duration
	}{
		{
			filePath:    "normal.txt",
			fileContent: []byte("hello Dragonfly"),
			targetPath:  Join(s.starter.Home, "normal.test"),
			createFile:  true,
			execSuccess: true,
			timeout:     5,
		},
		{
			filePath:    "empty.txt",
			fileContent: []byte(""),
			targetPath:  Join(s.starter.Home, "empty.test"),
			createFile:  true,
			execSuccess: true,
			timeout:     5,
		},
		{
			filePath:    "notExist.txt",
			fileContent: []byte(""),
			targetPath:  Join(s.starter.Home, "notExist.test"),
			createFile:  false,
			execSuccess: false,
			timeout:     5,
		},
	}

	for _, ca := range cases {
		if ca.createFile {
			err := s.starter.WriteSupernodeFileServer(ca.filePath, ca.fileContent, os.ModePerm)
			c.Assert(err, check.IsNil)
		}
		cmd, err := s.starter.DFGet(ca.timeout*time.Second,
			"-u", fmt.Sprintf("http://127.0.0.1:%d/%s", environment.SupernodeDownloadPort, ca.filePath),
			"-o", ca.targetPath,
			"--node", fmt.Sprintf("127.0.0.1:%d", environment.SupernodeListenPort),
			"--notbs")
		cmd.Wait()

		c.Assert(err, check.IsNil)
		execResult := cmd.ProcessState.Success()
		if ca.execSuccess {
			c.Assert(execResult, check.Equals, true)

			if execResult {
				// check the downloaded file content
				data, err := ioutil.ReadFile(ca.targetPath)
				c.Assert(err, check.IsNil)
				c.Assert(data, check.DeepEquals, ca.fileContent)
			}
		} else {
			c.Assert(execResult, check.Equals, false)
		}
	}
}
