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

package config

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type ConfigSuite struct{}

func init() {
	check.Suite(&ConfigSuite{})
}

func (suite *ConfigSuite) SetUpTest(c *check.C) {
	Reset()
}

func (suite *ConfigSuite) TestContext_String(c *check.C) {
	expected := "{\"url\":\"\",\"output\":\"\""
	c.Assert(strings.Contains(Ctx.String(), expected), check.Equals, true)
	Ctx.LocalLimit = 20971520
	Ctx.Pattern = "p2p"
	Ctx.Version = true
	expected = "\"url\":\"\",\"output\":\"\",\"localLimit\":20971520," +
		"\"pattern\":\"p2p\",\"version\":true"
	c.Assert(strings.Contains(Ctx.String(), expected), check.Equals, true)
}

func (suite *ConfigSuite) TestNewContext(c *check.C) {
	before := time.Now()
	time.Sleep(time.Millisecond)
	Ctx = NewContext()
	time.Sleep(time.Millisecond)
	after := time.Now()

	c.Assert(Ctx.StartTime.After(before), check.Equals, true)
	c.Assert(Ctx.StartTime.Before(after), check.Equals, true)

	beforeSign := fmt.Sprintf("%d-%.3f",
		os.Getpid(), float64(before.UnixNano())/float64(time.Second))
	afterSign := fmt.Sprintf("%d-%.3f",
		os.Getpid(), float64(after.UnixNano())/float64(time.Second))
	c.Assert(beforeSign < Ctx.Sign, check.Equals, true)
	c.Assert(afterSign > Ctx.Sign, check.Equals, true)

	if curUser, err := user.Current(); err != nil {
		c.Assert(Ctx.User, check.Equals, curUser.Username)
		c.Assert(Ctx.WorkHome, check.Equals, path.Join(curUser.HomeDir, ".small-dragonfly"))
	}
}
