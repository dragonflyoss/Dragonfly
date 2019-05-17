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
	"io/ioutil"
	"os"
	"path/filepath"
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
