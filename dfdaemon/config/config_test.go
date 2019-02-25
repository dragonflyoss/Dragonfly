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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&ConfigTestSuite{})
}

var testCrt = `-----BEGIN CERTIFICATE-----
MIICKzCCAZQCCQDZrCsm2rX81DANBgkqhkiG9w0BAQUFADBaMQswCQYDVQQGEwJD
TjERMA8GA1UECAwIWmhlamlhbmcxETAPBgNVBAcMCEhhbmd6aG91MQ4wDAYDVQQK
DAVsb3d6ajEVMBMGA1UEAwwMZGZkYWVtb24uY29tMB4XDTE5MDIyNTAyNDYwN1oX
DTE5MDMyNzAyNDYwN1owWjELMAkGA1UEBhMCQ04xETAPBgNVBAgMCFpoZWppYW5n
MREwDwYDVQQHDAhIYW5nemhvdTEOMAwGA1UECgwFbG93emoxFTATBgNVBAMMDGRm
ZGFlbW9uLmNvbTCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAtX1VzZRg1tgF
D0AFkUW2FpakkrhRzFuukWepoN0LfFSS/rNf8v1823de1SkpXBHsm2pMf94BIdmY
NDWH1tk27i4V5xydjNqxbdjjNjGHedBAM2tRQWWQuJAEo12sWUVYwDyN7RbL6wnz
7Egeac023FA9JhfMxaDvJHqJHVuKW3kCAwEAATANBgkqhkiG9w0BAQUFAAOBgQCT
VrDbo4m3QkcUT8ohuAUD8OHjTwJAuoxqVdHm+SpgjBYMLQgqXAPwaTGsIvx+32h2
J88xU3xXABE5QsNNbqLcMgQoXeMmqk1WuUhxXzTXT5h5gdW53faxV5M5Cb3zI8My
PPpBF5Cw+khgkJcY/ezKjHIvyABJwdzW8aAqwDBFAQ==
-----END CERTIFICATE-----`

type ConfigTestSuite struct {
	workHome string
	crtPath  string
}

func (s *ConfigTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "dfget-ConfigTestSuite-")
	s.crtPath = filepath.Join(s.workHome, "server.crt")
	ioutil.WriteFile(s.crtPath, []byte(testCrt), os.ModePerm)
}

func (s *ConfigTestSuite) TearDownSuite(c *check.C) {
	if s.workHome != "" {
		os.RemoveAll(s.workHome)
	}
}

func (s *ConfigTestSuite) TestProperties_Load(c *check.C) {
	var f = func(schema ...string) *Properties {
		var tmp []*Registry
		for _, s := range schema {
			r := &Registry{Schema: s}
			r.init()
			tmp = append(tmp, r)
		}
		return &Properties{Registries: tmp}
	}
	var cases = []struct {
		create   bool
		content  string
		errMsg   string
		expected *Properties
	}{
		{create: false, content: "", errMsg: "read yaml", expected: nil},
		{create: true, content: "-", errMsg: "unmarshal yaml", expected: nil},
		{create: true, content: "registries:\n - regx: '^['",
			errMsg: "missing closing", expected: nil},
		{create: true, content: "registries:\n  -", errMsg: "", expected: f()},
		{
			create:   true,
			content:  "registries:\n  - schema: http",
			errMsg:   "",
			expected: f("http"),
		},
	}
	for idx, v := range cases {
		filename := filepath.Join(s.workHome, fmt.Sprintf("test-%d", idx))
		if v.create {
			ioutil.WriteFile(filename, []byte(v.content), os.ModePerm)
		}
		p := &Properties{}
		err := p.Load(filename)
		if v.expected != nil {
			c.Assert(err, check.IsNil)
			c.Assert(len(p.Registries), check.Equals, len(v.expected.Registries))
			for i := 0; i < len(v.expected.Registries); i++ {
				c.Assert(p.Registries[i], check.DeepEquals, v.expected.Registries[i])
			}
		} else {
			c.Assert(err, check.NotNil)
			c.Assert(strings.Contains(err.Error(), v.errMsg), check.Equals, true,
				check.Commentf("error:%v expected:%s", err, v.errMsg))
		}
	}
}

func (s *ConfigTestSuite) TestRegistry_Match(c *check.C) {
	var cases = []struct {
		regx      string
		str       string
		errNotNil bool
		matched   bool
	}{
		{regx: "[a.com", str: "a.com", errNotNil: true, matched: false},
		{regx: "a.com", str: "a.com", matched: true},
		{regx: "a.com", str: "ba.com", matched: true},
		{regx: "a.com", str: "a.comm", matched: true},
		{regx: "^a.com", str: "ba.com", matched: false},
		{regx: "^a.com$", str: "ba.com", matched: false},
		{regx: "^a.com$", str: "a.comm", matched: false},
		{regx: "^a.com$", str: "a.com", matched: true},
		{regx: "", str: "a.com", matched: true},
	}

	for _, v := range cases {
		reg, err := NewRegistry("", "", v.regx, nil)
		if v.errNotNil {
			c.Assert(err, check.NotNil)
		} else {
			c.Assert(err, check.IsNil)
			c.Assert(reg.Match(v.str), check.Equals, v.matched,
				check.Commentf("%v", v))
		}
	}
}

func (s *ConfigTestSuite) TestRegistry_validate(c *check.C) {
	var cases = []struct {
		schema string
		err    string
	}{
		{schema: "http", err: ""},
		{schema: "https", err: ""},
		{schema: "", err: ""},
		{schema: "x", err: "invalid schema.*"},
	}

	for _, v := range cases {
		reg := Registry{Schema: v.schema}
		err := reg.init()
		if v.err == "" {
			c.Assert(err, check.IsNil)
		} else {
			c.Assert(err, check.NotNil)
			c.Assert(err, check.ErrorMatches, v.err)
		}
	}
}

func (s *ConfigTestSuite) TestRegistry_initTLSConfig(c *check.C) {
	invalidCrt := filepath.Join(s.workHome, "invalid.crt")
	ioutil.WriteFile(invalidCrt, nil, os.ModePerm)

	var cases = []struct {
		crt string
		err string
	}{
		{s.workHome, "read.*"},
		{s.crtPath, ""},
		{invalidCrt, "invalid cert.*"},
	}

	for _, v := range cases {
		reg := Registry{Certs: []string{v.crt}}
		err := reg.initTLSConfig()
		if v.err == "" {
			c.Assert(err, check.IsNil)
			c.Assert(reg.TLSConfig(), check.NotNil)
		} else {
			c.Assert(reg.TLSConfig(), check.IsNil)
			c.Assert(err, check.NotNil)
			c.Assert(err, check.ErrorMatches, v.err)
		}
	}
}
