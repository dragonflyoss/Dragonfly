/*
 * Copyright 1999-2018 Alibaba Group.
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

	"github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/dfget/util"
	"github.com/go-check/check"
	"github.com/sirupsen/logrus"
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
	ctx := s.createContext(buf)
	ctx.Output = path.Join(s.workHome, "test.output")

	err := prepare(ctx)
	fmt.Printf("%s\nerror:%v", buf.String(), err)
}

func (s *CoreTestSuite) TestRegisterToSupernode(c *check.C) {
	ctx := s.createContext(&bytes.Buffer{})
	m := new(MockSupernodeAPI)
	m.RegisterFunc = createRegisterFunc()
	register := NewSupernodeRegister(ctx, m)

	var f = func(bc int, errIsNil bool, data *RegisterResult) {
		res, e := registerToSuperNode(ctx, register)
		c.Assert(res == nil, check.Equals, data == nil)
		c.Assert(e == nil, check.Equals, errIsNil)
		c.Assert(ctx.BackSourceReason, check.Equals, bc)
		if data != nil {
			c.Assert(res, check.DeepEquals, data)
		}
	}

	f(config.BackSourceReasonNodeEmpty, true, nil)

	ctx.Pattern = config.PatternSource
	f(config.BackSourceReasonUserSpecified, true, nil)

	ctx.Pattern = config.PatternP2P

	ctx.Node = []string{"x"}
	ctx.URL = "http://x.com"
	f(config.BackSourceReasonRegisterFail, true, nil)

	ctx.Node = []string{"x"}
	ctx.URL = "http://taobao.com"
	ctx.BackSourceReason = config.BackSourceReasonNone
	f(config.BackSourceReasonNone, false, nil)

	ctx.Node = []string{"x"}
	ctx.URL = "http://lowzj.com"
	f(config.BackSourceReasonNone, true, &RegisterResult{
		Node: "x", RemainderNodes: []string{}, URL: ctx.URL, TaskID: "a",
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

func (s *CoreTestSuite) TestGetTaskPath(c *check.C) {
	c.Assert(getTaskPath("a"), check.Equals, config.PeerHTTPPathPrefix+"a")
	c.Assert(getTaskPath(""), check.Equals, "")
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
	config.Ctx = s.createContext(buf)

	nodes := []string{host}
	ip := checkConnectSupernode(nodes)
	c.Assert(ip, check.Equals, "127.0.0.1")

	buf.Reset()
	ip = checkConnectSupernode([]string{"127.0.0.2"})
	c.Assert(strings.Index(buf.String(), "connect") > 0, check.Equals, true)
	c.Assert(ip, check.Equals, "")
}

// ----------------------------------------------------------------------------
// helper functions

func (s *CoreTestSuite) createContext(writer io.Writer) *config.Context {
	if writer == nil {
		writer = &bytes.Buffer{}
	}
	ctx := config.NewContext()
	ctx.WorkHome = s.workHome
	ctx.RV.MetaPath = path.Join(ctx.WorkHome, "meta", "host.meta")
	ctx.RV.SystemDataDir = path.Join(ctx.WorkHome, "data")

	logrus.StandardLogger().Out = writer
	ctx.ClientLogger = logrus.StandardLogger()
	ctx.ServerLogger = logrus.StandardLogger()
	return ctx
}
