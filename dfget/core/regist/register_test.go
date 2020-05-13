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

package regist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	. "github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/locator"
	"github.com/dragonflyoss/Dragonfly/pkg/constants"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&RegistTestSuite{})
}

type RegistTestSuite struct {
	workHome string
}

func (s *RegistTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "dfget-CoreTestSuite-")
}

func (s *RegistTestSuite) TearDownSuite(c *check.C) {
	if s.workHome != "" {
		if err := os.RemoveAll(s.workHome); err != nil {
			fmt.Printf("remove path:%s error", s.workHome)
		}
	}
}

func (s *RegistTestSuite) TestNewRegisterResult(c *check.C) {
	result := NewRegisterResult("node", "url", "taskID",
		10, 1, "supernode")
	c.Assert(result.Node, check.Equals, "node")
	c.Assert(result.URL, check.Equals, "url")
	c.Assert(result.TaskID, check.Equals, "taskID")
	c.Assert(result.FileLength, check.Equals, int64(10))
	c.Assert(result.PieceSize, check.Equals, int32(1))
	c.Assert(result.CDNSource, check.Equals, types.CdnSource("supernode"))

	str, _ := json.Marshal(result)
	c.Assert(result.String(), check.Equals, string(str))
}

func (s *RegistTestSuite) TestSupernodeRegister_Register(c *check.C) {
	buf := &bytes.Buffer{}
	cfg := s.createConfig(buf)
	m := new(MockSupernodeAPI)
	m.RegisterFunc = CreateRegisterFunc()

	nodeStr := "127.0.0.1:8002"
	snLocator, _ := locator.NewStaticLocatorFromStr("test", []string{nodeStr})
	var f = func(ec int, msg string, data *RegisterResult) {
		register := NewSupernodeRegister(cfg, m, snLocator)
		resp, e := register.Register(0)
		if msg == "" {
			c.Assert(e, check.IsNil)
			c.Assert(resp, check.NotNil)
			c.Assert(resp, check.DeepEquals, data)
		} else {
			c.Assert(e, check.NotNil)
			c.Assert(e.Msg, check.Equals, msg)
			c.Assert(resp, check.IsNil)
		}
	}

	snLocator.Next()
	f(constants.HTTPError, "empty response, unknown error", nil)

	snLocator.Refresh()
	f(501, "invalid source url", nil)

	snLocator.Refresh()
	cfg.URL = "http://taobao.com"
	f(constants.CodeNeedAuth, "need auth", nil)

	snLocator.Refresh()
	cfg.URL = "http://github.com"
	f(constants.CodeWaitAuth, "wait auth", nil)

	snLocator.Refresh()
	cfg.URL = "http://lowzj.com"
	f(constants.Success, "", &RegisterResult{
		Node: nodeStr, URL: cfg.URL, TaskID: "a",
		FileLength: 100, PieceSize: 10})

	f(constants.HTTPError, "empty response, unknown error", nil)
}

func (s *RegistTestSuite) TestSupernodeRegister_constructRegisterRequest(c *check.C) {
	buf := &bytes.Buffer{}
	cfg := s.createConfig(buf)
	register := &supernodeRegister{nil, nil, cfg, nil}

	cfg.Identifier = "id"
	req := register.constructRegisterRequest(0)
	c.Assert(req.Identifier, check.Equals, cfg.Identifier)
	c.Assert(req.Md5, check.Equals, "")

	cfg.Md5 = "md5"
	req = register.constructRegisterRequest(0)
	c.Assert(req.Identifier, check.Equals, "")
	c.Assert(req.Md5, check.Equals, cfg.Md5)
}

// ----------------------------------------------------------------------------
// helper functions

func (s *RegistTestSuite) createConfig(writer io.Writer) *config.Config {
	return CreateConfig(writer, s.workHome)
}
