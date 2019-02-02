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
	"path"
	"strings"
	"testing"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	. "github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/core/regist"
	"github.com/dragonflyoss/Dragonfly/dfget/util"

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
	cfg.Output = path.Join(s.workHome, "test.output")

	err := prepare(cfg)
	fmt.Printf("%s\nerror:%v", buf.String(), err)
}

func (s *CoreTestSuite) TestRegisterToSupernode(c *check.C) {
	cfg := s.createConfig(&bytes.Buffer{})
	m := new(MockSupernodeAPI)
	m.RegisterFunc = CreateRegisterFunc()
	register := regist.NewSupernodeRegister(cfg, m)

	var f = func(bc int, errIsNil bool, data *regist.RegisterResult) {
		res, e := registerToSuperNode(cfg, register)
		c.Assert(res == nil, check.Equals, data == nil)
		c.Assert(e == nil, check.Equals, errIsNil)
		c.Assert(cfg.BackSourceReason, check.Equals, bc)
		if data != nil {
			c.Assert(res, check.DeepEquals, data)
		}
	}

	f(config.BackSourceReasonNodeEmpty, true, nil)

	cfg.Pattern = config.PatternSource
	f(config.BackSourceReasonUserSpecified, true, nil)

	cfg.Pattern = config.PatternP2P

	cfg.Node = []string{"x"}
	cfg.URL = "http://x.com"
	f(config.BackSourceReasonRegisterFail, true, nil)

	cfg.Node = []string{"x"}
	cfg.URL = "http://taobao.com"
	cfg.BackSourceReason = config.BackSourceReasonNone
	// f(config.BackSourceReasonNone, false, nil)

	cfg.Node = []string{"x"}
	cfg.URL = "http://lowzj.com"
	f(config.BackSourceReasonNone, true, &regist.RegisterResult{
		Node: "x", RemainderNodes: []string{}, URL: cfg.URL, TaskID: "a",
		FileLength: 100, PieceSize: 10})
}

func (s *CoreTestSuite) TestGetTaskURL(c *check.C) {
	var cases = []struct {
		u string
		f []string
		e string
	}{
		{"a?b=1", nil, "a?b=1"},
		{"a?b=1", []string{"b"}, "a"},
		{"a?b=1&b=1", []string{"b"}, "a"},
		{"a?b=1&c=1", []string{"b"}, "a?c=1"},
		{"a?b=1&c=1&c", []string{"b", "c"}, "a"},
		{"a?", nil, "a?"},
	}
	for _, v := range cases {
		c.Assert(getTaskURL(v.u, v.f), check.Equals, v.e)
	}
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
			c.Assert(util.ContainsString(nodes[:len(v)], n), check.Equals, true)
			c.Assert(util.ContainsString(nodes[len(v):], n), check.Equals, true)
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
	ip := checkConnectSupernode(nodes)
	c.Assert(ip, check.Equals, "127.0.0.1")

	buf.Reset()
	ip = checkConnectSupernode([]string{"127.0.0.2"})
	c.Assert(strings.Index(buf.String(), "Connect") > 0, check.Equals, true)
	c.Assert(ip, check.Equals, "")
}

// ----------------------------------------------------------------------------
// helper functions

func (s *CoreTestSuite) createConfig(writer io.Writer) *config.Config {
	return CreateConfig(writer, s.workHome)
}
