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

package queue

import (
	"github.com/go-check/check"
)

func (suite *DFGetUtilSuite) TestLRUQueue(c *check.C) {
	q := NewLRUQueue(5)

	q.Put("key1", 1)

	v1, err := q.GetItemByKey("key1")
	c.Assert(err, check.IsNil)
	c.Assert(v1.(int), check.Equals, 1)

	items := q.GetFront(1)
	c.Assert(len(items), check.Equals, 1)
	c.Assert(items[0], check.Equals, 1)

	q.Put("key2", 2)
	q.Put("key1", 3)

	v1, err = q.GetItemByKey("key1")
	c.Assert(err, check.IsNil)
	c.Assert(v1.(int), check.Equals, 3)

	items = q.GetFront(10)
	c.Assert(len(items), check.Equals, 2)
	c.Assert(items[0], check.Equals, 3)
	c.Assert(items[1], check.Equals, 2)

	items = q.GetFront(1)
	c.Assert(len(items), check.Equals, 1)
	c.Assert(items[0], check.Equals, 3)

	_, err = q.GetItemByKey("key3")
	c.Assert(err, check.NotNil)

	obsoleteKey, _ := q.Put("key3", "data3")
	c.Assert(obsoleteKey, check.Equals, "")
	obsoleteKey, _ = q.Put("key4", "data4")
	c.Assert(obsoleteKey, check.Equals, "")
	obsoleteKey, _ = q.Put("key5", "data5")
	c.Assert(obsoleteKey, check.Equals, "")

	items = q.GetFront(10)
	c.Assert(len(items), check.Equals, 5)
	c.Assert(items[0], check.Equals, "data5")
	c.Assert(items[1], check.Equals, "data4")
	c.Assert(items[2], check.Equals, "data3")
	c.Assert(items[3], check.Equals, 3)
	c.Assert(items[4], check.Equals, 2)

	obsoleteKey, obsoleteData := q.Put("key6", "data6")
	c.Assert(obsoleteKey, check.Equals, "key2")
	c.Assert(obsoleteData.(int), check.Equals, 2)
	_, err = q.GetItemByKey("key2")
	c.Assert(err, check.NotNil)

	items = q.GetFront(5)
	c.Assert(len(items), check.Equals, 5)
	c.Assert(items[0], check.Equals, "data6")
	c.Assert(items[1], check.Equals, "data5")
	c.Assert(items[2], check.Equals, "data4")
	c.Assert(items[3], check.Equals, "data3")
	c.Assert(items[4], check.Equals, 3)

	v1, err = q.Get("key5")
	c.Assert(err, check.IsNil)
	c.Assert(v1.(string), check.Equals, "data5")

	items = q.GetFront(5)
	c.Assert(len(items), check.Equals, 5)
	c.Assert(items[0], check.Equals, "data5")
	c.Assert(items[1], check.Equals, "data6")
	c.Assert(items[2], check.Equals, "data4")
	c.Assert(items[3], check.Equals, "data3")
	c.Assert(items[4], check.Equals, 3)

	v1 = q.Delete("key3")
	c.Assert(v1, check.Equals, "data3")

	items = q.GetFront(5)
	c.Assert(len(items), check.Equals, 4)
	c.Assert(items[0], check.Equals, "data5")
	c.Assert(items[1], check.Equals, "data6")
	c.Assert(items[2], check.Equals, "data4")
	c.Assert(items[3], check.Equals, 3)

	v1 = q.Delete("key3")
	c.Assert(v1, check.IsNil)
}
