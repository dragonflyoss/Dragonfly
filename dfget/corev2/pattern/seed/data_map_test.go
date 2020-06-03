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

import (
	"strings"
	"testing"

	"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/config"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type seedSuite struct{}

func init() {
	check.Suite(&seedSuite{})
}

func (suite *seedSuite) TestDataMapWithTaskState(c *check.C) {
	d := newDataMap()
	ts1 := &taskState{}

	err := d.add("ts1", ts1)
	c.Assert(err, check.IsNil)

	ret, err := d.getAsTaskState("ts1")
	c.Assert(err, check.IsNil)
	c.Assert(ret, check.Equals, ts1)

	err = d.remove("ts1")
	c.Assert(err, check.IsNil)

	_, err = d.getAsTaskState("ts1")
	c.Assert(strings.Contains(err.Error(), "data not found"), check.Equals, true)

	err = d.add("ts2", "xxx")
	c.Assert(err, check.IsNil)

	_, err = d.getAsTaskState("ts2")
	c.Assert(strings.Contains(err.Error(), "convert failed"), check.Equals, true)

	err = d.remove("ts2")
	c.Assert(err, check.IsNil)
}

func (suite *seedSuite) TestDataMapWithNode(c *check.C) {
	d := newDataMap()
	node1 := &config.Node{}

	err := d.add("node1", node1)
	c.Assert(err, check.IsNil)

	ret, err := d.getAsNode("node1")
	c.Assert(err, check.IsNil)
	c.Assert(ret, check.Equals, node1)

	err = d.remove("node1")
	c.Assert(err, check.IsNil)

	_, err = d.getAsNode("node1")
	c.Assert(strings.Contains(err.Error(), "data not found"), check.Equals, true)

	err = d.add("node2", "xxx")
	c.Assert(err, check.IsNil)

	_, err = d.getAsNode("node2")
	c.Assert(strings.Contains(err.Error(), "convert failed"), check.Equals, true)

	err = d.remove("node2")
	c.Assert(err, check.IsNil)
}

func (suite *seedSuite) TestDataMapWithLocalTaskState(c *check.C) {
	d := newDataMap()
	lts1 := &localTaskState{}

	err := d.add("lts1", lts1)
	c.Assert(err, check.IsNil)

	ret, err := d.getAsLocalTaskState("lts1")
	c.Assert(err, check.IsNil)
	c.Assert(ret, check.Equals, lts1)

	err = d.remove("lts1")
	c.Assert(err, check.IsNil)

	_, err = d.getAsLocalTaskState("lts1")
	c.Assert(strings.Contains(err.Error(), "data not found"), check.Equals, true)

	err = d.add("lts2", "xxx")
	c.Assert(err, check.IsNil)

	_, err = d.getAsLocalTaskState("lts2")
	c.Assert(strings.Contains(err.Error(), "convert failed"), check.Equals, true)

	err = d.remove("lts2")
	c.Assert(err, check.IsNil)
}
