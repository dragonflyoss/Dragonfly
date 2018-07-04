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
	Params = new(cliParameters)
}

func (suite *CliSuite) Test_initParameters_noArguments(c *check.C) {
	initParameters(nil)
	c.Assert(Params.LocalLimit, check.Equals, "20M")
	c.Assert(Params.Notbs, check.Equals, false)
	c.Assert(Params.DFDaemon, check.Equals, false)
	c.Assert(Params.Version, check.Equals, false)
	c.Assert(Params.ShowBar, check.Equals, false)
	c.Assert(Params.Console, check.Equals, false)
	c.Assert(Params.Verbose, check.Equals, false)
	c.Assert(Params.Help, check.Equals, false)

}

func (suite *CliSuite) Test_initParameters_withArguments(c *check.C) {
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
	initParameters(args)

	res := []struct {
		actual   interface{}
		expected interface{}
	}{
		{Params.URL, arguments["url"]},
		{Params.Output, arguments["output"]},
		{Params.LocalLimit, arguments["locallimit"]},
		{Params.TotalLimit, arguments["totallimit"]},
		{strconv.Itoa(Params.Timeout), arguments["timeout"]},
		{Params.Md5, arguments["md5"]},
		{Params.Identifier, arguments["identifier"]},
		{Params.CallSystem, arguments["callsystem"]},
		{Params.Filter, arguments["filter"]},
		{Params.Pattern, arguments["pattern"]},
		{strings.Join(Params.Header, ","), arguments["header"]},
		{strings.Join(Params.Node, ","), arguments["node"]},
		{Params.Notbs, arguments["notbs"] == "true"},
		{Params.Verbose, arguments["notbs"] == "true"},
		{Params.DFDaemon, false},
		{Params.Version, false},
		{Params.ShowBar, false},
		{Params.Console, false},
		{Params.Help, false},
	}

	for _, cc := range res {
		c.Assert(cc.actual, check.Equals, cc.expected)
	}
}

func (suite *CliSuite) TestCliParameters_String(c *check.C) {
	expected := "{\"url\":\"\",\"output\":\"\"}"
	c.Assert(Params.String(), check.Equals, expected)
	initParameters([]string{"-v"})
	expected = "{\"url\":\"\",\"output\":\"\",\"locallimit\":\"20M\",\"pattern\":\"p2p\",\"version\":true}"
	c.Assert(Params.String(), check.Equals, expected)
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
	initParameters(nil)
	Usage()
	output = buffer.String()
	c.Assert(strings.Contains(output, pflag.CommandLine.FlagUsages()), check.Equals, true)
}
