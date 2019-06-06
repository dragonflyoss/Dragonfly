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

package util

import (
	"github.com/go-check/check"
)

type UtilSuite struct{}

func init() {
	check.Suite(&UtilSuite{})
}

func (suite *UtilSuite) TestExtractHost(c *check.C) {
	host := ExtractHost("1:0")
	c.Assert(host, check.Equals, "1")
}

func (suite *UtilSuite) TestGetIPAndPortFromNode(c *check.C) {
	var cases = []struct {
		node         string
		defaultPort  int
		expectedIP   string
		expectedPort int
	}{
		{"127.0.0.1", 8002, "127.0.0.1", 8002},
		{"127.0.0.1:8001", 8002, "127.0.0.1", 8001},
		{"127.0.0.1:abcd", 8002, "127.0.0.1", 8002},
	}

	for _, v := range cases {
		ip, port := GetIPAndPortFromNode(v.node, v.defaultPort)
		c.Check(ip, check.Equals, v.expectedIP)
		c.Check(port, check.Equals, v.expectedPort)
	}
}
