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

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/errors"
	"github.com/dragonflyoss/Dragonfly/dfget/util"

	"github.com/go-check/check"
	"github.com/sirupsen/logrus"
)

var cfg = NewConfig()

func Test(t *testing.T) {
	check.TestingT(t)
}

type ConfigSuite struct{}

func init() {
	check.Suite(&ConfigSuite{})
}

func (suite *ConfigSuite) SetUpTest(c *check.C) {

}

func (suite *ConfigSuite) TestConfig_String(c *check.C) {
	cfg := NewConfig()
	expected := "{\"url\":\"\",\"output\":\"\""
	c.Assert(strings.Contains(cfg.String(), expected), check.Equals, true)
	cfg.LocalLimit = 20971520
	cfg.Pattern = "p2p"
	cfg.Version = true
	expected = "\"url\":\"\",\"output\":\"\",\"localLimit\":20971520," +
		"\"pattern\":\"p2p\",\"version\":true"
	c.Assert(strings.Contains(cfg.String(), expected), check.Equals, true)
}

func (suite *ConfigSuite) TestNewConfig(c *check.C) {
	before := time.Now()
	time.Sleep(time.Millisecond)
	cfg := NewConfig()
	time.Sleep(time.Millisecond)
	after := time.Now()

	c.Assert(cfg.StartTime.After(before), check.Equals, true)
	c.Assert(cfg.StartTime.Before(after), check.Equals, true)

	beforeSign := fmt.Sprintf("%d-%.3f",
		os.Getpid(), float64(before.UnixNano())/float64(time.Second))
	afterSign := fmt.Sprintf("%d-%.3f",
		os.Getpid(), float64(after.UnixNano())/float64(time.Second))
	c.Assert(beforeSign < cfg.Sign, check.Equals, true)
	c.Assert(afterSign > cfg.Sign, check.Equals, true)

	if curUser, err := user.Current(); err != nil {
		c.Assert(cfg.User, check.Equals, curUser.Username)
		c.Assert(cfg.WorkHome, check.Equals, path.Join(curUser.HomeDir, ".small-dragonfly"))
	}
}

func (suite *ConfigSuite) TestAssertConfig(c *check.C) {
	var (
		clog = logrus.StandardLogger()
		buf  = &bytes.Buffer{}
	)
	clog.Out = buf

	var cases = []struct {
		clog      *logrus.Logger
		url       string
		output    string
		checkFunc func(err error) bool
	}{
		{clog: clog, checkFunc: errors.IsInvalidValue},
		{clog: clog, url: "http://a", checkFunc: errors.IsInvalidValue},
		{clog: clog, url: "http://a.b.com", output: "/tmp/output", checkFunc: errors.IsNilError},
		{clog: clog, url: "http://a.b.com", output: "/root", checkFunc: errors.IsInvalidValue},
	}

	var f = func() (err error) {
		return AssertConfig(cfg)
	}

	for _, v := range cases {
		cfg.URL = v.url
		cfg.Output = v.output
		actual := f()
		expected := v.checkFunc(actual)
		c.Assert(expected, check.Equals, true,
			check.Commentf("actual:[%s] expected:[%t]", actual, expected))
	}
}

func (suite *ConfigSuite) TestCheckURL(c *check.C) {
	var cases = map[string]bool{
		"":                     false,
		"abcdefg":              false,
		"////a//":              false,
		"a////a//":             false,
		"a.com////a//":         true,
		"a:b@a.com":            true,
		"a:b@127.0.0.1":        true,
		"a:b@127.0.0.1?a=b":    true,
		"127.0.0.1":            true,
		"127.0.0.1?a=b":        true,
		"127.0.0.1:":           true,
		"127.0.0.1:8080":       true,
		"127.0.0.1:8080/我":     true,
		"127.0.0.1:8080/我?x=1": true,
		"a.b":            true,
		"www.taobao.com": true,
		"https://github.com/dragonflyoss/Dragonfly/issues?" +
			"q=is%3Aissue+is%3Aclosed": true,
	}

	c.Assert(checkURL(cfg), check.NotNil)
	for k, v := range cases {
		for _, scheme := range []string{"http", "https", "HTTP", "HTTPS"} {
			cfg.URL = fmt.Sprintf("%s://%s", scheme, k)
			actual := fmt.Sprintf("%s:%v", k, checkURL(cfg))
			expected := fmt.Sprintf("%s:%s://%s", k, scheme, k)
			if v {
				expected = fmt.Sprintf("%s:<nil>", k)
			}
			c.Assert(actual, check.Equals, expected)
		}
	}
}

func (suite *ConfigSuite) TestCheckOutput(c *check.C) {
	type tester struct {
		url      string
		output   string
		expected string
	}
	curDir, _ := filepath.Abs(".")

	var j = func(p string) string { return filepath.Join(curDir, p) }
	var cases = []tester{
		{"http://www.taobao.com", "", j("www.taobao.com")},
		{"http://www.taobao.com", "/tmp/zj.test", "/tmp/zj.test"},
		{"www.taobao.com", "", ""},
		{"www.taobao.com", "/tmp/zj.test", "/tmp/zj.test"},
		{"", "/tmp/zj.test", "/tmp/zj.test"},
		{"", "zj.test", j("zj.test")},
		{"", "/tmp", ""},
		{"", "/tmp/a/b/c/d/e/zj.test", "/tmp/a/b/c/d/e/zj.test"},
	}

	if cfg.User != "root" {
		cases = append(cases, tester{url: "", output: "/root/zj.test", expected: ""})
	}
	for _, v := range cases {
		cfg.URL = v.url
		cfg.Output = v.output
		if util.IsEmptyStr(v.expected) {
			c.Assert(checkOutput(cfg), check.NotNil, check.Commentf("%v", v))
		} else {
			c.Assert(checkOutput(cfg), check.IsNil, check.Commentf("%v", v))
			c.Assert(cfg.Output, check.Equals, v.expected, check.Commentf("%v", v))
		}
	}
}

func (suite *ConfigSuite) TestProperties_Load(c *check.C) {
	dirName, _ := ioutil.TempDir("/tmp", "dfget-TestProperties_Load-")
	defer os.RemoveAll(dirName)

	var cases = []struct {
		create   bool
		ext      string
		content  string
		errMsg   string
		expected *Properties
	}{
		{create: false, ext: "x", errMsg: "extension of"},
		{create: false, ext: "yaml", errMsg: "read yaml config from", expected: nil},
		{create: true, ext: "yaml",
			content: "nodes:\n\t- 10.10.10.1", errMsg: "unmarshal yaml error", expected: nil},
		{create: true, ext: "yaml",
			content: "nodes:\n  - 10.10.10.1\n  - 10.10.10.2\n",
			errMsg:  "", expected: &Properties{Nodes: []string{"10.10.10.1", "10.10.10.2"}}},
		{create: true, ext: "yaml",
			content: "totalLimit: 10485760",
			errMsg:  "", expected: &Properties{TotalLimit: 10485760}},
		{create: false, ext: "ini", content: "[node]\naddress=1.1.1.1", errMsg: "read ini config"},
		{create: true, ext: "ini", content: "[node]\naddress=1.1.1.1",
			expected: &Properties{Nodes: []string{"1.1.1.1"}}},
		{create: true, ext: "conf", content: "[node]\naddress=1.1.1.1",
			expected: &Properties{Nodes: []string{"1.1.1.1"}}},
		{create: true, ext: "conf", content: "[node]\naddress=1.1.1.1,1.1.1.2",
			expected: &Properties{Nodes: []string{"1.1.1.1", "1.1.1.2"}}},
		{create: true, ext: "conf", content: "[node]\naddress=1.1.1.1\n[totalLimit]",
			expected: &Properties{Nodes: []string{"1.1.1.1"}}},
	}

	for idx, v := range cases {
		filename := filepath.Join(dirName, fmt.Sprintf("%d.%s", idx, v.ext))
		if v.create {
			ioutil.WriteFile(filename, []byte(v.content), os.ModePerm)
		}
		p := &Properties{}
		err := p.Load(filename)
		if v.expected != nil {
			c.Assert(err, check.IsNil)
			c.Assert(p, check.DeepEquals, v.expected)
		} else {
			c.Assert(err, check.NotNil)
			c.Assert(strings.Contains(err.Error(), v.errMsg), check.Equals, true,
				check.Commentf("error:%v expected:%s", err, v.errMsg))
		}
	}
}

func (suite *ConfigSuite) TestRuntimeVariable_String(c *check.C) {
	rv := RuntimeVariable{
		LocalIP: "127.0.0.1",
	}
	c.Assert(strings.Contains(rv.String(), "127.0.0.1"), check.Equals, true)

	jRv := &RuntimeVariable{}
	e := json.Unmarshal([]byte(rv.String()), jRv)
	c.Assert(e, check.IsNil)
	c.Assert(jRv.LocalIP, check.Equals, rv.LocalIP)
}
