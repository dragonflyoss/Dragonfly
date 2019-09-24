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

package fileutils

import (
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/go-check/check"
	"gopkg.in/yaml.v2"
)

type FsizeTestSuite struct {
	tmpDir   string
	username string
}

func init() {
	check.Suite(&FsizeTestSuite{})
}

func (suite *FsizeTestSuite) TestFsizeToString(c *check.C) {
	var cases = []struct {
		fsize    Fsize
		fsizeStr string
	}{
		{B, "1B"},
		{1024 * B, "1KB"},
		{3 * MB, "3MB"},
		{0 * GB, "0B"},
	}

	for _, ca := range cases {
		result := FsizeToString(ca.fsize)
		c.Assert(result, check.Equals, ca.fsizeStr)
	}
}

func (suite *FsizeTestSuite) TestStringToFSize(c *check.C) {
	var cases = []struct {
		fsizeStr      string
		fsize         Fsize
		errAssertFunc errortypes.ErrAssertFunc
	}{
		{"0B", 0 * B, errortypes.IsNilError},
		{"1B", B, errortypes.IsNilError},
		{"10G", 10 * GB, errortypes.IsNilError},
		{"1024", 1 * KB, errortypes.IsNilError},
		{"-1", 0, errortypes.IsInvalidValue},
		{"10b", 0, errortypes.IsInvalidValue},
	}

	for _, ca := range cases {
		result, err := StringToFSize(ca.fsizeStr)
		c.Assert(ca.errAssertFunc(err), check.Equals, true)
		c.Assert(result, check.DeepEquals, ca.fsize)
	}
}

func (suite *FsizeTestSuite) TestMarshalYAML(c *check.C) {
	var cases = []struct {
		input  Fsize
		output string
	}{
		{5 * MB, "5MB\n"},
		{1 * GB, "1GB\n"},
		{1 * KB, "1KB\n"},
		{1 * B, "1B\n"},
		{0, "0B\n"},
	}

	for _, ca := range cases {
		output, err := yaml.Marshal(ca.input)
		c.Check(err, check.IsNil)
		c.Check(string(output), check.Equals, ca.output)
	}
}

func (suite *FsizeTestSuite) TestUnMarshalYAML(c *check.C) {
	var cases = []struct {
		input  string
		output Fsize
	}{
		{"5M\n", 5 * MB},
		{"1G\n", 1 * GB},
		{"1B\n", 1 * B},
		{"1\n", 1 * B},
		{"1024\n", 1 * KB},
		{"1K\n", 1 * KB},
	}

	for _, ca := range cases {
		var output Fsize
		err := yaml.Unmarshal([]byte(ca.input), &output)
		c.Check(err, check.IsNil)
		c.Check(output, check.Equals, ca.output)
	}
}
