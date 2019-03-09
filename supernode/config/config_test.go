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
	"path"
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
		os.RemoveAll(s.workHome)
	}
}

func (s *SupernodeConfigTestSuite) TestConfig_Load(c *check.C) {
	confPath := path.Join(s.workHome, "supernode.yml")
	conf := NewConfig()
	exp := &Config{}

	ioutil.WriteFile(confPath, []byte(conf.String()), os.ModePerm)
	exp.Load(confPath)
	c.Assert(conf.BaseProperties, check.DeepEquals, exp.BaseProperties)

	conf = &Config{}
	ioutil.WriteFile(confPath, []byte(content), os.ModePerm)
	err := conf.Load(confPath)
	c.Assert(err, check.IsNil)
	c.Assert(conf.HomeDir, check.Equals, "/tmp")
	c.Assert(conf.Plugins[StoragePlugin], check.NotNil)
	c.Assert(len(conf.Plugins[StoragePlugin]), check.Equals, 1)

	p := &PluginProperties{Name: "local", Enabled: true, Config: "baseDir: /tmp/supernode/repo\n"}
	c.Assert(conf.Plugins[StoragePlugin][0], check.DeepEquals, p)
}
