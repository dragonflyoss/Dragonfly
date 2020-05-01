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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/rate"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"

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
	cfg.LocalLimit = 20 * rate.MB
	cfg.MinRate = 64 * rate.KB
	cfg.Pattern = "p2p"
	expected = "\"url\":\"\",\"output\":\"\",\"pattern\":\"p2p\"," +
		"\"localLimit\":\"20MB\",\"minRate\":\"64KB\""
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
		c.Assert(cfg.WorkHome, check.Equals, filepath.Join(curUser.HomeDir, ".small-dragonfly"))
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
		{clog: clog, checkFunc: errortypes.IsInvalidValue},
		{clog: clog, url: "htt://a", checkFunc: errortypes.IsInvalidValue},
		{clog: clog, url: "htt://a.b.com", checkFunc: errortypes.IsInvalidValue},
		{clog: clog, url: "http://a.b.com", output: "/tmp/output", checkFunc: errortypes.IsNilError},
		{clog: clog, url: "http://a.b.com", output: "./root", checkFunc: errortypes.IsNilError},
		{clog: clog, url: "http://a.b.com", output: "/root", checkFunc: errortypes.IsInvalidValue},
		{clog: clog, url: "http://a.b.com", output: "/", checkFunc: errortypes.IsInvalidValue},
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
		{"", "/", ""},
	}

	if cfg.User != "root" {
		cases = append(cases, tester{url: "", output: "/root/zj.test", expected: ""})
	}
	for _, v := range cases {
		cfg.URL = v.url
		cfg.Output = v.output
		if stringutils.IsEmptyStr(v.expected) {
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
		{create: false, ext: "yaml", errMsg: "no such file or directory", expected: nil},
		{create: true, ext: "yaml",
			content: "nodes:\n\t- 10.10.10.1", errMsg: "yaml", expected: nil},
		{create: true, ext: "yaml",
			content: "nodes:\n  - 10.10.10.1\n  - 10.10.10.2\n",
			errMsg:  "", expected: &Properties{Supernodes: []*NodeWeight{
				{"10.10.10.1:8002", 1},
				{"10.10.10.2:8002", 1},
			}}},
		{create: true, ext: "yaml",
			content: "totalLimit: 10M",
			errMsg:  "", expected: &Properties{TotalLimit: 10 * rate.MB}},
		{create: false, ext: "ini", content: "[node]\naddress=1.1.1.1", errMsg: "read ini config"},
		{create: true, ext: "ini", content: "[node]\naddress=1.1.1.1",
			expected: &Properties{Supernodes: []*NodeWeight{
				{"1.1.1.1:8002", 1},
			}}},
		{create: true, ext: "conf", content: "[node]\naddress=1.1.1.1",
			expected: &Properties{Supernodes: []*NodeWeight{
				{"1.1.1.1:8002", 1},
			}}},
		{create: true, ext: "conf", content: "[node]\naddress=1.1.1.1,1.1.1.2",
			expected: &Properties{Supernodes: []*NodeWeight{
				{"1.1.1.1:8002", 1},
				{"1.1.1.2:8002", 1},
			}}},
		{create: true, ext: "conf", content: "[node]\naddress=1.1.1.1\n[totalLimit]",
			expected: &Properties{Supernodes: []*NodeWeight{
				{"1.1.1.1:8002", 1},
			}}},
	}

	for idx, v := range cases {
		filename := filepath.Join(dirName, fmt.Sprintf("%d.%s", idx, v.ext))
		if v.create {
			err := ioutil.WriteFile(filename, []byte(v.content), os.ModePerm)
			c.Assert(err, check.IsNil)
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

func (suite *ConfigSuite) TestProperties_String(c *check.C) {
	p := NewProperties()
	str := p.String()

	actual := &Properties{}
	e := json.Unmarshal([]byte(str), actual)
	c.Assert(e, check.IsNil)
	c.Assert(actual, check.DeepEquals, p)
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
