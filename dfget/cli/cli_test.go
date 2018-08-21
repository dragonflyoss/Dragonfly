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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	cfg "github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/dfget/errors"
	"github.com/alibaba/Dragonfly/dfget/util"
	"github.com/alibaba/Dragonfly/version"
	"github.com/go-check/check"
	"github.com/sirupsen/logrus"
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
	reset()
}

func reset() {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	cfg.Reset()
}

func (suite *CliSuite) Test_setupFlags_noArguments(c *check.C) {
	setupFlags(nil)
	c.Assert(cfg.Ctx.Node, check.IsNil)
	c.Assert(cfg.Ctx.LocalLimit, check.Equals, 0)
	c.Assert(cfg.Ctx.TotalLimit, check.Equals, 0)
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
}

func (suite *CliSuite) TestInitProperties(c *check.C) {
	cfg.Ctx.ConfigFiles = nil
	dirName, _ := ioutil.TempDir("/tmp", "dfget-TestInitProperties-")
	defer os.RemoveAll(dirName)

	iniFile := filepath.Join(dirName, "dragonfly.ini")
	yamlFile := filepath.Join(dirName, "dragonfly.yaml")
	iniContent := []byte("[node]\naddress=1.1.1.1")
	yamlContent := []byte("nodes:\n  - 1.1.1.2\nlocalLimit: 1024000")
	ioutil.WriteFile(iniFile, iniContent, os.ModePerm)
	ioutil.WriteFile(yamlFile, yamlContent, os.ModePerm)

	var buf = &bytes.Buffer{}
	logrus.StandardLogger().Out = buf

	var cases = []struct {
		configs  []string
		expected *cfg.Properties
	}{
		{configs: nil,
			expected: cfg.NewProperties()},
		{configs: []string{iniFile, yamlFile},
			expected: newProp(0, 0, 0, "1.1.1.1")},
		{configs: []string{yamlFile, iniFile},
			expected: newProp(1024000, 0, 0, "1.1.1.2")},
		{configs: []string{filepath.Join(dirName, "x"), yamlFile},
			expected: newProp(1024000, 0, 0, "1.1.1.2")},
	}

	for _, v := range cases {
		cfg.Reset()
		buf.Reset()
		cfg.Ctx.ClientLogger = logrus.StandardLogger()
		cfg.Ctx.ConfigFiles = v.configs

		initProperties()
		c.Assert(cfg.Ctx.Node, check.DeepEquals, v.expected.Nodes)
		c.Assert(cfg.Ctx.LocalLimit, check.Equals, v.expected.LocalLimit)
		c.Assert(cfg.Ctx.TotalLimit, check.Equals, v.expected.TotalLimit)
		c.Assert(cfg.Ctx.ClientQueueSize, check.Equals, v.expected.ClientQueueSize)
	}
}

func newProp(local int, total int, size int, nodes ...string) *cfg.Properties {
	p := cfg.NewProperties()
	if nodes != nil {
		p.Nodes = nodes
	}
	if local != 0 {
		p.LocalLimit = local
	}
	if total != 0 {
		p.TotalLimit = total
	}
	if size != 0 {
		p.ClientQueueSize = size
	}
	return p
}

func (suite *CliSuite) TestInitialize(c *check.C) {
	const workHomeEnv = "DFGET_TEST_WORK_HOME"
	if hasTestEnv() {
		args := testArgs()
		cfg.Ctx.WorkHome = os.Getenv(workHomeEnv)
		os.Args = args
		cliOut = os.Stdout
		initialize()
		return
	}

	tmpDir, _ := ioutil.TempDir("/tmp", "dfget-cli-test-")
	defer os.RemoveAll(tmpDir)

	var cases = []struct {
		args   []string
		exMsg  string
		exCode int
	}{
		{args: []string{}, exMsg: "please use", exCode: 0},
		{args: []string{"-h"}, exMsg: "Usage", exCode: 0},
		{args: []string{"-v"}, exMsg: version.DFGetVersion, exCode: 0},
		{args: []string{"-u"}, exMsg: "flag needs an argument", exCode: 2},
		{args: []string{"-u", "http://www.taobao.com"}, exMsg: "", exCode: 0},
	}

	for _, v := range cases {
		cmd := helperCommand("TestInitialize", v.args...)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", workHomeEnv, tmpDir))
		bs, err := cmd.Output()
		c.Assert(strings.Contains(string(bs), v.exMsg), check.Equals, true,
			check.Commentf("args:%v out:%s", v.args, bs))
		if e, ok := err.(*exec.ExitError); ok {
			c.Assert(e.Success(), check.Equals, v.exCode == 0, check.Commentf(
				"args:%v err:%v", v.args, err))
		}
	}
}

func (suite *CliSuite) TestResultMsg(c *check.C) {
	ctx := cfg.NewContext()
	end := ctx.StartTime.Add(100 * time.Millisecond)

	msg := resultMsg(ctx, end, nil)
	c.Assert(msg, check.Equals, "download SUCCESS(0) cost:0.100s length:0 reason:0")

	ctx.BackSourceReason = cfg.BackSourceReasonRegisterFail
	msg = resultMsg(ctx, end, errors.New(1, "TestFail"))
	c.Assert(msg, check.Equals, "download FAIL(1) cost:0.100s length:0 reason:1 error:"+
		`{"Code":1,"Msg":"TestFail"}`)
}

// ----------------------------------------------------------------------------
// helper functions

func helperCommand(name string, args ...string) (cmd *exec.Cmd) {
	cs := []string{"-check.f", fmt.Sprintf("^%s$", name), "--"}
	cs = append(cs, args...)
	cmd = exec.Command(os.Args[0], cs...)
	cmd.Env = testEnv()
	return cmd
}

func testArgs() []string {
	args := []string{os.Args[0]}
	if len(os.Args) > 4 {
		args = append(args, os.Args[4:]...)
	}
	return args
}

func testEnv() []string {
	return []string{"DFGET_TEST_HELPER_PROCESS=1"}
}

func hasTestEnv() bool {
	return os.Getenv("DFGET_TEST_HELPER_PROCESS") == "1"
}
