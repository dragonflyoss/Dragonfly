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
	. "path/filepath"
	"time"

	"github.com/go-check/check"

	"github.com/dragonflyoss/Dragonfly/test/command"
	"github.com/dragonflyoss/Dragonfly/test/environment"
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

func (s *DFGetP2PTestSuite) TestDownload(c *check.C) {
	cmd, err := s.starter.DFGet(5*time.Second,
		"-u", "https://lowzj.com",
		"-o", Join(s.starter.Home, "a.test"),
		"--node", fmt.Sprintf("127.0.0.1:%d", environment.SupernodeListenPort),
		"--notbs")
	cmd.Wait()

	c.Assert(err, check.IsNil)
	c.Assert(cmd.ProcessState.Success(), check.Equals, true)
}
