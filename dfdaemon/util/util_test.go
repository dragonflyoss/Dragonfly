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

package util

import (
	"runtime"
	"testing"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type UtilSuite struct{}

func init() {
	check.Suite(&UtilSuite{})
}

func (suite *UtilSuite) TestExtractHost(c *check.C) {
	host := ExtractHost("1:0")
	c.Assert(host, check.Equals, "1")
}

func (suite *UtilSuite) TestNetLimit(c *check.C) {
	speed := NetLimit()
	if runtime.NumCPU() < 24 {
		c.Assert(speed, check.Equals, "20M")
	}
}
