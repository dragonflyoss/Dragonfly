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

package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/go-check/check"

	"github.com/dragonflyoss/Dragonfly/supernode/config"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&SupernodeAppTest{})
}

type SupernodeAppTest struct {
	workHome string
	confPath string
}

func (s *SupernodeAppTest) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "supernode-AppTest-")
	s.confPath = path.Join(s.workHome, "supernode.yml")
}

func (s *SupernodeAppTest) TearDownSuite(c *check.C) {
	if s.workHome != "" {
		if err := os.RemoveAll(s.workHome); err != nil {
			fmt.Printf("remove path:%s error", s.workHome)
		}
	}
}

func (s *SupernodeAppTest) SetUpTest(c *check.C) {
	configFilePath = config.DefaultSupernodeConfigFilePath
	cfg = config.NewConfig()
	options = NewOptions()
}

func (s *SupernodeAppTest) TestInitLog(c *check.C) {

}

func (s *SupernodeAppTest) TestInitConfig(c *check.C) {
	err := initConfig()
	c.Assert(err, check.IsNil)

	configFilePath = s.confPath
	c.Assert(initConfig(), check.NotNil)

	content := "base:\n  listenPort: 1000"
	ioutil.WriteFile(s.confPath, []byte(content), os.ModePerm)
	configFilePath = s.confPath
	c.Assert(initConfig(), check.IsNil)
	c.Assert(cfg.ListenPort, check.Equals, 1000)

	os.Args = []string{os.Args[0], "--port", "1100"}
	configFilePath = s.confPath
	c.Assert(initConfig(), check.IsNil)
	c.Assert(cfg.ListenPort, check.Equals, 1100)
}

func (s *SupernodeAppTest) TestNewOptions(c *check.C) {
	opt := NewOptions()
	expected := config.NewBaseProperties()

	c.Assert(opt.BaseProperties, check.NotNil)
	c.Assert(opt.BaseProperties, check.DeepEquals, expected)
}
