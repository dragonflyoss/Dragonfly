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

package cli

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Sirupsen/logrus"
	cfg "github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/dfget/util"
	"github.com/go-check/check"
	"github.com/spf13/pflag"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type CliSuite struct{}

func init() {
	check.Suite(&CliSuite{})
}

func (suite *CliSuite) SetUpTest(c *check.C) {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	cfg.Reset()
}

func (suite *CliSuite) Test_setupFlags_noArguments(c *check.C) {
	setupFlags(nil)
	c.Assert(cfg.Ctx.LocalLimit, check.Equals, 20971520)
	c.Assert(cfg.Ctx.Notbs, check.Equals, false)
	c.Assert(cfg.Ctx.DFDaemon, check.Equals, false)
	c.Assert(cfg.Ctx.Version, check.Equals, false)
	c.Assert(cfg.Ctx.ShowBar, check.Equals, false)
	c.Assert(cfg.Ctx.Console, check.Equals, false)
	c.Assert(cfg.Ctx.Verbose, check.Equals, false)
	c.Assert(cfg.Ctx.Help, check.Equals, false)
}

func (suite *CliSuite) Test_setupFlags_withArguments(c *check.C) {
	arguments := map[string]string{
		"url":        "http://www.taobao.com",
		"output":     "/tmp/" + os.Args[0] + ".test",
		"locallimit": "30M",
		"totallimit": "50M",
		"timeout":    "10",
		"md5":        "123",
		"identifier": "456",
		"callsystem": "unit-test",
		"filter":     "x&y",
		"pattern":    "cdn",
		"header":     "a:0,b:1,c:2",
		"node":       "1,2",
		"notbs":      "true",
		"verbose":    "true",
	}
	var args []string
	for k, v := range arguments {
		args = append(args, "--"+k, v)
	}
	setupFlags(args)

	res := []struct {
		actual   interface{}
		expected interface{}
	}{
		{cfg.Ctx.URL, arguments["url"]},
		{cfg.Ctx.Output, arguments["output"]},
		{strconv.Itoa(cfg.Ctx.LocalLimit/1024/1024) + "M",
			arguments["locallimit"]},
		{strconv.Itoa(cfg.Ctx.TotalLimit/1024/1024) + "M",
			arguments["totallimit"]},
		{strconv.Itoa(cfg.Ctx.Timeout), arguments["timeout"]},
		{cfg.Ctx.Md5, arguments["md5"]},
		{cfg.Ctx.Identifier, arguments["identifier"]},
		{cfg.Ctx.CallSystem, arguments["callsystem"]},
		{strings.Join(cfg.Ctx.Filter, "&"), arguments["filter"]},
		{cfg.Ctx.Pattern, arguments["pattern"]},
		{strings.Join(cfg.Ctx.Header, ","), arguments["header"]},
		{strings.Join(cfg.Ctx.Node, ","), arguments["node"]},
		{cfg.Ctx.Notbs, arguments["notbs"] == "true"},
		{cfg.Ctx.Verbose, arguments["notbs"] == "true"},
		{cfg.Ctx.DFDaemon, false},
		{cfg.Ctx.Version, false},
		{cfg.Ctx.ShowBar, false},
		{cfg.Ctx.Console, false},
		{cfg.Ctx.Help, false},
	}

	for _, cc := range res {
		c.Assert(cc.actual, check.Equals, cc.expected)
	}
}

func (suite *CliSuite) TestUsage(c *check.C) {
	var buffer bytes.Buffer
	cliOut = &buffer
	Usage()
	output := buffer.String()
	c.Assert(output, check.NotNil)
	c.Assert(strings.Contains(output, "Dragonfly"), check.Equals, true)
	c.Assert(strings.Contains(output, os.Args[0]), check.Equals, true)

	buffer.Reset()
	setupFlags(nil)
	Usage()
	output = buffer.String()
	c.Assert(strings.Contains(output, pflag.CommandLine.FlagUsages()), check.Equals, true)
}

func (suite *CliSuite) Test_transLimit(c *check.C) {
	var cases = map[string]struct {
		i   int
		err string
	}{
		"20M":   {20971520, ""},
		"20m":   {20971520, ""},
		"10k":   {10240, ""},
		"10K":   {10240, ""},
		"10x":   {0, "invalid unit 'x' of '10x', 'KkMm' are supported"},
		"10.0x": {0, "invalid syntax"},
		"ab":    {0, "invalid syntax"},
		"abM":   {0, "invalid syntax"},
	}

	for k, v := range cases {
		i, e := transLimit(k)
		c.Assert(i, check.Equals, v.i)
		if util.IsEmptyStr(v.err) {
			c.Assert(e, check.IsNil)
		} else {
			c.Assert(e, check.NotNil)
			c.Assert(strings.Contains(e.Error(), v.err), check.Equals, true)
		}
	}
	logrus.Info("hello")
}
