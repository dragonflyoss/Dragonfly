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

package core

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	. "github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/core/regist"
	"github.com/dragonflyoss/Dragonfly/dfget/core/uploader"
	"github.com/dragonflyoss/Dragonfly/dfget/locator"
	"github.com/dragonflyoss/Dragonfly/pkg/algorithm"

	"github.com/go-check/check"
	"github.com/valyala/fasthttp"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type CoreTestSuite struct {
	workHome string
}

func init() {
	check.Suite(&CoreTestSuite{})
}

func (s *CoreTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "dfget-CoreTestSuite-")
}

func (s *CoreTestSuite) TearDownSuite(c *check.C) {
	if s.workHome != "" {
		if err := os.RemoveAll(s.workHome); err != nil {
			fmt.Printf("remove path:%s error", s.workHome)
		}
	}
}

func (s *CoreTestSuite) TestPrepare(c *check.C) {
	buf := &bytes.Buffer{}
	cfg := s.createConfig(buf)
	cfg.Output = filepath.Join(s.workHome, "test.output")

	err := prepare(cfg, nil)
	fmt.Printf("%s\nerror:%v", buf.String(), err)
}

func (s *CoreTestSuite) TestRegisterToSupernode(c *check.C) {
	cfg := s.createConfig(&bytes.Buffer{})
	m := new(MockSupernodeAPI)
	m.RegisterFunc = CreateRegisterFunc()
	nodeStr := "127.0.0.1:8002"
	snLocator, _ := locator.NewStaticLocatorFromStr("test", []string{nodeStr})
	register := regist.NewSupernodeRegister(cfg, m, snLocator)

	var f = func(bc int, errIsNil bool, data *regist.RegisterResult) {
		res, e := registerToSuperNode(cfg, register, snLocator)
		c.Assert(res == nil, check.Equals, data == nil)
		c.Assert(e == nil, check.Equals, errIsNil)
		c.Assert(cfg.BackSourceReason, check.Equals, bc)
		if data != nil {
			c.Assert(res, check.DeepEquals, data)
		}
	}

	tmpGroup := snLocator.Group
	snLocator.Group = nil
	f(config.BackSourceReasonNodeEmpty, true, nil)

	cfg.Pattern = config.PatternSource
	f(config.BackSourceReasonUserSpecified, true, nil)

	uploader.SetupPeerServerExecutor(nil)
	cfg.Pattern = config.PatternP2P
	snLocator.Group = tmpGroup
	snLocator.Refresh()
	cfg.URL = "http://x.com"
	f(config.BackSourceReasonRegisterFail, true, nil)

	snLocator.Refresh()
	cfg.URL = "http://taobao.com"
	cfg.BackSourceReason = config.BackSourceReasonNone
}

func (s *CoreTestSuite) TestAdjustSupernodeList(c *check.C) {
	var cases = [][]string{
		{},
		{"1"},
		{"1", "2", "3"},
	}

	for _, v := range cases {
		nodes := adjustSupernodeList(v)
		for _, n := range v {
			c.Assert(algorithm.ContainsString(nodes[:len(v)], n), check.Equals, true)
			c.Assert(algorithm.ContainsString(nodes[len(v):], n), check.Equals, true)
		}
	}
}

func (s *CoreTestSuite) TestCheckConnectSupernode(c *check.C) {
	port := rand.Intn(1000) + 63000
	host := fmt.Sprintf("127.0.0.1:%d", port)
	ln, _ := net.Listen("tcp", host)
	defer ln.Close()
	go fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {})

	buf := &bytes.Buffer{}
	s.createConfig(buf)

	nodes := []string{host}
	l, _ := locator.NewStaticLocatorFromStr("test", nodes)
	ip := checkConnectSupernode(l)
	c.Assert(ip, check.Equals, "127.0.0.1")

	buf.Reset()
	l, _ = locator.NewStaticLocatorFromStr("test", []string{"127.0.0.2"})
	ip = checkConnectSupernode(l)
	c.Assert(strings.Index(buf.String(), "Connect") > 0, check.Equals, true)
	c.Assert(ip, check.Equals, "")
}

func (s *CoreTestSuite) TestCalculateTimeout(c *check.C) {
	cfg := s.createConfig(&bytes.Buffer{})

	timeout := calculateTimeout(nil)
	c.Assert(timeout, check.Equals, config.DefaultDownloadTimeout)

	timeout = calculateTimeout(cfg)
	c.Assert(timeout, check.Equals, config.DefaultDownloadTimeout)

	cfg.RV.FileLength = 1000
	timeout = calculateTimeout(cfg)
	c.Assert(timeout, check.Equals, 10*time.Second)

	cfg.Timeout = time.Minute
	timeout = calculateTimeout(cfg)
	c.Assert(timeout, check.Equals, time.Minute)
}

// ----------------------------------------------------------------------------
// helper functions

func (s *CoreTestSuite) createConfig(writer io.Writer) *config.Config {
	return CreateConfig(writer, s.workHome)
}
