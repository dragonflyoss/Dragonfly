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
	"testing"

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
	expected := "{\"url\":\"\",\"output\":\"\"}"
	c.Assert(Ctx.String(), check.Equals, expected)
	Ctx.LocalLimit = 20971520
	Ctx.Pattern = "p2p"
	Ctx.Version = true
	expected = "{\"url\":\"\",\"output\":\"\",\"localLimit\":20971520," +
		"\"pattern\":\"p2p\",\"version\":true}"
	c.Assert(Ctx.String(), check.Equals, expected)
}
