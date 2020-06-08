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

package seed

import "github.com/go-check/check"

func (suite *seedSuite) TestRequestManager(c *check.C) {
	rm := newRequestManager()
	c.Assert(rm.addRequest("url1"), check.IsNil)
	urls := rm.getRecentRequest(10)
	c.Assert(len(urls), check.Equals, 1)
	c.Assert(urls[0], check.Equals, "url1")

	c.Assert(rm.addRequest("url2"), check.IsNil)
	c.Assert(rm.addRequest("url3"), check.IsNil)
	c.Assert(rm.addRequest("url4"), check.IsNil)
	c.Assert(rm.addRequest("url5"), check.IsNil)
	c.Assert(rm.addRequest("url2"), check.IsNil)

	urls = rm.getRecentRequest(10)
	c.Assert(len(urls), check.Equals, 5)
	c.Assert(urls[0], check.Equals, "url2")
	c.Assert(urls[1], check.Equals, "url5")
	c.Assert(urls[2], check.Equals, "url4")
	c.Assert(urls[3], check.Equals, "url3")
	c.Assert(urls[4], check.Equals, "url1")

	urls = rm.getRecentRequest(3)
	c.Assert(len(urls), check.Equals, 3)
	c.Assert(urls[0], check.Equals, "url2")
	c.Assert(urls[1], check.Equals, "url5")
	c.Assert(urls[2], check.Equals, "url4")

	c.Assert(rm.addRequest("url1"), check.IsNil)
	c.Assert(rm.addRequest("url1"), check.IsNil)
	c.Assert(rm.addRequest("url5"), check.IsNil)

	urls = rm.getRecentRequest(3)
	c.Assert(len(urls), check.Equals, 3)
	c.Assert(urls[0], check.Equals, "url5")
	c.Assert(urls[1], check.Equals, "url1")
	c.Assert(urls[2], check.Equals, "url2")
}
