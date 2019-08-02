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
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&UploaderTestSuite{})
	check.Suite(&UploaderLaunchTestSuite{})
}

// -----------------------------------------------------------------------------
// LaunchPeerServer() test, separating from other tests to avoid data race
// caused by tests' codes

type UploaderLaunchTestSuite struct {
	workHome string
}

func (s *UploaderLaunchTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "dfget-UploaderLaunchTestSuite-")
}

func (s *UploaderLaunchTestSuite) TearDownSuite(c *check.C) {
	if s.workHome != "" {
		os.RemoveAll(s.workHome)
	}
}

func (s *UploaderLaunchTestSuite) TestLaunchPeerServer(c *check.C) {
	flag := false
	var f http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		if flag = !flag; flag {
			sendSuccess(w)
		} else {
			sendHeader(w, http.StatusBadGateway)
		}
	}
	_, port, server := startTestServer(f)
	defer stopTestServer(server)

	var cases = []struct {
		port         int
		expectedPort int
		expectedErr  string
	}{
		// auto-generate a port
		{0, generatePort(0), ""},

		// invalid port
		{70000, 0, "invalid port"},

		// use already existing peer server
		{port, port, ""},
		// port bind by other process and retry failed
		{port, 0, "start peer server error"},
	}

	cfg := createConfig(s.workHome, 0)
	for _, v := range cases {
		cfg.RV.PeerPort = v.port
		port, err := LaunchPeerServer(cfg)

		c.Assert(port, check.Equals, v.expectedPort)
		if v.expectedErr != "" {
			cmt := check.Commentf("real(%d, err:%s) deleted(%d err:%s)",
				port, err, v.expectedPort, v.expectedErr)
			c.Assert(err, check.NotNil, cmt)
			c.Assert(strings.Contains(err.Error(), v.expectedErr),
				check.Equals, true, cmt)
		} else {
			c.Assert(err, check.IsNil)
		}
		if port > 0 {
			p2p.shutdown()
		}
	}
}

// -----------------------------------------------------------------------------
// other tests

type UploaderTestSuite struct {
	workHome string
}

func (s *UploaderTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "dfget-UploaderTestSuite-")
}

func (s *UploaderTestSuite) TearDownSuite(c *check.C) {
	if s.workHome != "" {
		os.RemoveAll(s.workHome)
	}
}

func (s *UploaderTestSuite) TestWaitForShutdown(c *check.C) {
	var cases = []struct {
		server   *peerServer
		interval time.Duration
		expected bool
	}{
		// immediately shutdown when p2p is nil
		{nil, 50, false},

		// immediately shutdown when p2p.finished is nil
		{&peerServer{}, 50, false},

		// shutdown when p2p.finished is closed  after 50ms
		{&peerServer{finished: make(chan struct{})}, 50, true},
	}

	for _, v := range cases {
		p2p = v.server
		d := v.interval * time.Millisecond

		t := time.AfterFunc(d, func() {
			if p2p != nil {
				p2p.setFinished()
			}
		})

		start := time.Now()
		WaitForShutdown()
		c.Assert(time.Since(start) > d, check.Equals, v.expected,
			check.Commentf("real cost(%v) > interval(%v) is not deleted(%v)",
				time.Since(start), d, v.expected))
		t.Stop()
	}
}

func (s *UploaderTestSuite) TestWaitForStartup(c *check.C) {
	var (
		ptr unsafe.Pointer
		res = make(chan error)
	)

	e := waitForStartup(res, &ptr)
	c.Assert(e, check.NotNil)
	c.Assert(strings.Contains(e.Error(), "initialize"), check.Equals, true)

	storeSrvPtr(&ptr, &peerServer{finished: make(chan struct{})})
	e = waitForStartup(res, &ptr)
	c.Assert(e, check.NotNil)
	c.Assert(strings.Contains(e.Error(), "ping"), check.Equals, true)

	time.AfterFunc(50*time.Millisecond, func() { res <- nil })
	e = waitForStartup(res, &ptr)
	c.Assert(e, check.IsNil)

	time.AfterFunc(50*time.Millisecond, func() { res <- fmt.Errorf("test") })
	e = waitForStartup(res, &ptr)
	c.Assert(e, check.NotNil)
	c.Assert(e.Error(), check.Equals, "test")
}

func (s *UploaderTestSuite) TestServerGC(c *check.C) {
	cfg := createConfig(s.workHome, 0)
	cfg.RV.DataExpireTime = time.Minute
	p2p = &peerServer{
		cfg:      cfg,
		api:      &helper.MockSupernodeAPI{},
		finished: make(chan struct{}),
	}

	// create test directories and files
	dirName, _ := ioutil.TempDir(cfg.RV.SystemDataDir, "TestServerGC-")
	var cases = []struct {
		name     string
		finished bool
		expire   int
		store    bool
		deleted  bool
	}{
		{finished: true, expire: -2, store: true, deleted: true},
		{finished: true, expire: 2, store: true, deleted: false},
		{finished: false, expire: -2, store: true, deleted: false},
		{finished: false, expire: 2, store: true, deleted: false},

		{finished: true, expire: -2, store: false, deleted: true},
		{finished: true, expire: 2, store: false, deleted: true},
		{finished: false, expire: -2, store: false, deleted: true},
		{finished: false, expire: 2, store: false, deleted: true},
	}
	for i := 0; i < len(cases); i++ {
		v := &cases[i]
		expire := time.Duration(v.expire) * time.Minute
		v.name = createTestFile(p2p, v.store, v.finished, expire)
	}

	time.AfterFunc(500*time.Millisecond, func() {
		c.Assert(fileutils.PathExist(dirName), check.Equals, false)
		for _, v := range cases {
			dir := cfg.RV.SystemDataDir
			cmt := check.Commentf("%v", v)
			srvFile := helper.GetServiceFile(v.name, dir)
			taskFile := helper.GetTaskFile(v.name, dir)
			c.Assert(!fileutils.PathExist(srvFile), check.Equals, v.deleted, cmt)
			c.Assert(!fileutils.PathExist(taskFile), check.Equals, v.deleted, cmt)
		}

		p2p.setFinished()
	})

	serverGC(cfg, time.Second)
	p2p = nil
}

func (s *UploaderTestSuite) TestIsRunning(c *check.C) {
	p2p = nil
	c.Assert(isRunning(), check.Equals, false)

	p2p = &peerServer{finished: make(chan struct{})}
	c.Assert(isRunning(), check.Equals, true)
	close(p2p.finished)
	c.Assert(isRunning(), check.Equals, false)
	p2p = nil
}

// -----------------------------------------------------------------------------
// helper functions

func createConfig(workHome string, port int) *config.Config {
	cfg := helper.CreateConfig(nil, workHome)
	cfg.RV.LocalIP = "127.0.0.1"
	cfg.RV.PeerPort = port
	return cfg
}

func createTestFile(srv *peerServer, store bool, finished bool, expire time.Duration) string {
	name := fmt.Sprintf("TestServerGC-%d", rand.Int())
	dataDir := srv.cfg.RV.SystemDataDir
	serviceFile := helper.GetServiceFile(name, dataDir)
	taskFile := helper.GetTaskFile(name, dataDir)

	helper.CreateTestFile(serviceFile, "")
	helper.CreateTestFile(taskFile, "")

	expireTime := time.Now().Add(expire)
	os.Chtimes(serviceFile, expireTime, expireTime)
	os.Chtimes(taskFile, expireTime, expireTime)

	if store {
		srv.syncTaskMap.Store(name, &taskConfig{
			finished:   finished,
			accessTime: expireTime,
		})
	}
	return name
}
