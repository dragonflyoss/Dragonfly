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

var content = `
base:
  homeDir: /tmp
storages:
  meta.driver:
plugins:
  storage:
    - name: local
      enabled: true
      config: |
        baseDir: /tmp/supernode/repo
`

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&SupernodeConfigTestSuite{})
}

type SupernodeConfigTestSuite struct {
	workHome string
}

func (s *SupernodeConfigTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "supernode-SupernodeConfigTestSuite-")
}

func (s *SupernodeConfigTestSuite) TearDownSuite(c *check.C) {
	if s.workHome != "" {
		err := os.RemoveAll(s.workHome)
		c.Assert(err, check.IsNil)
	}
}

func (s *SupernodeConfigTestSuite) TestConfig_Load(c *check.C) {
	confPath := filepath.Join(s.workHome, "supernode.yml")
	conf := NewConfig()
	exp := &Config{}

	err := ioutil.WriteFile(confPath, []byte(conf.String()), os.ModePerm)
	c.Assert(err, check.IsNil)
	err = exp.Load(confPath)
	c.Assert(err, check.IsNil)
	c.Assert(conf.BaseProperties, check.DeepEquals, exp.BaseProperties)

	conf = &Config{}
	err = ioutil.WriteFile(confPath, []byte(content), os.ModePerm)
	c.Assert(err, check.IsNil)
	err = conf.Load(confPath)
	c.Assert(err, check.IsNil)
	c.Assert(conf.HomeDir, check.Equals, "/tmp")
	c.Assert(conf.Plugins[StoragePlugin], check.NotNil)
	c.Assert(len(conf.Plugins[StoragePlugin]), check.Equals, 1)

	p := &PluginProperties{Name: "local", Enabled: true, Config: "baseDir: /tmp/supernode/repo\n"}
	c.Assert(conf.Plugins[StoragePlugin][0], check.DeepEquals, p)
}

func (s *SupernodeConfigTestSuite) TestGetSuperCID(c *check.C) {
	conf := Config{
		BaseProperties: &BaseProperties{cIDPrefix: "CIDPrefix"},
	}

	c.Assert(conf.GetSuperCID("taskID"), check.DeepEquals, "CIDPrefixtaskID")
}

func (s *SupernodeConfigTestSuite) TestIsSuperCID(c *check.C) {
	conf := Config{
		BaseProperties: &BaseProperties{cIDPrefix: "CIDPrefix"},
	}

	c.Assert(conf.IsSuperCID("CIDPrefixTest"), check.DeepEquals, true)
	c.Assert(conf.IsSuperCID("Test"), check.DeepEquals, false)
}

func (s *SupernodeConfigTestSuite) TestGetSuperPID(c *check.C) {
	conf := Config{
		BaseProperties: &BaseProperties{superNodePID: "superNodePID"},
	}

	c.Assert(conf.GetSuperPID(), check.DeepEquals, "superNodePID")

	conf.SetSuperPID("Test")
	c.Assert(conf.GetSuperPID(), check.DeepEquals, "Test")
}

func (s *SupernodeConfigTestSuite) TestIsSuperPID(c *check.C) {
	conf := Config{
		BaseProperties: &BaseProperties{superNodePID: "superNodePID"},
	}

	c.Assert(conf.IsSuperPID("superNodePID"), check.DeepEquals, true)
	c.Assert(conf.IsSuperPID("Test"), check.DeepEquals, false)
}
