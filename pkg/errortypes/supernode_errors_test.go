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

package errortypes

import (
	"github.com/go-check/check"
)

type SupernodeErrorTestSuite struct{}

func init() {
	check.Suite(&SupernodeErrorTestSuite{})
}

func (suite *SupernodeErrorTestSuite) TestIsSystemError(c *check.C) {
	err1 := New(6, "system error")
	err2 := New(0, "test")
	c.Assert(IsSystemError(*err1), check.Equals, true)
	c.Assert(IsSystemError(*err2), check.Equals, false)
}
