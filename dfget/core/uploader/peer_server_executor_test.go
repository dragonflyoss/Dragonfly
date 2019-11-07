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

package uploader

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/version"
	"github.com/go-check/check"
)

func init() {
	check.Suite(&PeerServerExecutorTestSuite{})
}

type PeerServerExecutorTestSuite struct {
	workHome string
	script   string

	ip     string
	port   int
	server *http.Server
}

func (s *PeerServerExecutorTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "dfget-PeerServerTestSuite-")
	s.script = filepath.Join(s.workHome, "script.sh")
	s.writeScript("")
	s.start()
}

func (s *PeerServerExecutorTestSuite) TearDownSuite(c *check.C) {
	stopTestServer(s.server)
	if s.workHome != "" {
		os.RemoveAll(s.workHome)
	}
}

// ---------------------------------------------------------------------------
// tests

func (s *PeerServerExecutorTestSuite) TestSetupAndGetPeerServerExecutor(c *check.C) {
	pe := GetPeerServerExecutor()
	c.Assert(pe, check.NotNil)
	_, ok := pe.(*peerServerExecutor)
	c.Assert(ok, check.Equals, true)

	SetupPeerServerExecutor(nil)
	c.Assert(GetPeerServerExecutor(), check.IsNil)
}

func (s *PeerServerExecutorTestSuite) TestStartPeerServerProcess(c *check.C) {
	cfg := helper.CreateConfig(nil, s.workHome)
	cfg.RV.LocalIP = s.ip
	os.Args[0] = s.script

	SetupPeerServerExecutor(nil)
	port, e := StartPeerServerProcess(cfg)
	c.Assert(port, check.Equals, 0)
	c.Assert(e, check.NotNil)

	SetupPeerServerExecutor(&peerServerExecutor{})

	// read pipe EOF
	port, e = StartPeerServerProcess(cfg)
	c.Assert(port, check.Equals, 0)
	c.Assert(e, check.Equals, io.EOF)

	// wait for reading pipe timeout
	s.writeScript("sleep 2")
	port, e = StartPeerServerProcess(cfg)
	c.Assert(port, check.Equals, 0)
	c.Assert(strings.Contains(e.Error(), "timeout"), check.Equals, true)

	// invalid server
	s.writeScript("echo 65555")
	port, e = StartPeerServerProcess(cfg)
	c.Assert(port, check.Equals, 0)
	c.Assert(strings.Contains(e.Error(), "invalid server"), check.Equals, true)

	// start a new server successfully
	s.writeScript(fmt.Sprintf("echo %d", s.port))
	port, e = StartPeerServerProcess(cfg)
	c.Assert(port, check.Equals, s.port)
	c.Assert(e, check.IsNil)

	// use an existing server
	s.writeScript("")
	updateServicePortInMeta(cfg.RV.MetaPath, s.port)
	port, e = StartPeerServerProcess(cfg)
	c.Assert(port, check.Equals, s.port)
	c.Assert(e, check.IsNil)
}

func (s *PeerServerExecutorTestSuite) TestReadPort(c *check.C) {
	port := 39480
	reader := strings.NewReader("dfget uploader server port is " + strconv.Itoa(port) + "\n")
	result, err := readPort(reader)
	c.Check(err, check.IsNil)
	c.Check(result, check.Equals, port)
}

// ---------------------------------------------------------------------------
// helper functions

func (s *PeerServerExecutorTestSuite) start() {
	var f http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		sendSuccess(w)
		fmt.Fprint(w, "@"+version.DFGetVersion)
	}
	s.ip, s.port, s.server = startTestServer(f)
}

func (s *PeerServerExecutorTestSuite) writeScript(content string) {
	buf := &bytes.Buffer{}
	buf.WriteString("#!/bin/bash\n")
	buf.WriteString(content)
	buf.WriteString("\n")
	ioutil.WriteFile(s.script, buf.Bytes(), os.ModePerm)
}
