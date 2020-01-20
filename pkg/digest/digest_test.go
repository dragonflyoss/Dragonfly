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

package digest

import (
	"testing"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type DigestUtilSuite struct{}

func init() {
	check.Suite(&DigestUtilSuite{})
}

func (suite *DigestUtilSuite) TestSha256(c *check.C) {
	result := Sha256("test")
	c.Check(result, check.Equals, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08")
}

func (suite *DigestUtilSuite) TestSha1(c *check.C) {
	result := Sha1([]string{"test1", "test2"})
	c.Check(result, check.Equals, "dff964f6e3c1761b6288f5c75c319d36fb09b2b9")
}
