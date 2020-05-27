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

package locator

import (
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/queue"

	"github.com/go-check/check"
)

type hashCirclerLocatorTestSuite struct {
}

func init() {
	check.Suite(&hashCirclerLocatorTestSuite{})
}

var testGroupName1 = "test-group1"

func (s *hashCirclerLocatorTestSuite) TestHashCirclerLocator(c *check.C) {
	evQ := queue.NewQueue(0)
	nodes := []string{"1.1.1.1:8002", "2.2.2.2:8002", "3.3.3.3:8002"}
	hl, err := NewHashCirclerLocator(testGroupName1, nodes, evQ)
	c.Assert(err, check.IsNil)

	c.Assert(hl.Get(), check.IsNil)
	c.Assert(hl.Next(), check.IsNil)

	groups := hl.All()
	c.Assert(len(groups), check.Equals, 1)
	c.Assert(len(groups[0].Nodes), check.Equals, 3)
	c.Assert(groups[0].Nodes[0].String(), check.Equals, nodes[0])
	c.Assert(groups[0].Nodes[1].String(), check.Equals, nodes[1])
	c.Assert(groups[0].Nodes[2].String(), check.Equals, nodes[2])

	keys := []string{"x", "y", "z", "a", "b", "c", "m", "n", "p", "q", "j", "k", "i", "e", "f", "g"}
	originSp := make([]string, len(keys))

	for i, k := range keys {
		sp := hl.Select(k)
		c.Assert(sp, check.NotNil)
		originSp[i] = sp.String()
	}

	// select again, the supernode should be equal
	for i, k := range keys {
		sp := hl.Select(k)
		c.Assert(sp, check.NotNil)
		c.Assert(originSp[i], check.Equals, sp.String())
	}

	// disable nodes[0]
	evQ.Put(NewDisableEvent(nodes[0]))
	time.Sleep(time.Second * 2)
	// select again, if originSp is not nodes[0], it should not be changed.
	for i, k := range keys {
		sp := hl.Select(k)
		c.Assert(sp, check.NotNil)
		if originSp[i] == nodes[0] {
			c.Assert(originSp[i], check.Not(check.Equals), sp.String())
			continue
		}

		c.Assert(originSp[i], check.Equals, sp.String())
	}

	// disable nodes[1]
	evQ.Put(NewDisableEvent(nodes[1]))
	time.Sleep(time.Second * 2)
	// select again, all select node should be nodes[2]
	for _, k := range keys {
		sp := hl.Select(k)
		c.Assert(sp, check.NotNil)
		c.Assert(nodes[2], check.Equals, sp.String())
	}

	// enable nodes[0], disable nodes[2]
	evQ.Put(NewDisableEvent(nodes[2]))
	evQ.Put(NewEnableEvent(nodes[0]))
	time.Sleep(time.Second * 2)
	for _, k := range keys {
		sp := hl.Select(k)
		c.Assert(sp, check.NotNil)
		c.Assert(nodes[0], check.Equals, sp.String())
	}

	// enable nodes[1]
	evQ.Put(NewEnableEvent(nodes[1]))
	time.Sleep(time.Second * 2)
	for i, k := range keys {
		sp := hl.Select(k)
		c.Assert(sp, check.NotNil)
		if originSp[i] == nodes[2] {
			c.Assert(originSp[i], check.Not(check.Equals), sp.String())
			continue
		}

		c.Assert(originSp[i], check.Equals, sp.String())
	}

	// enable nodes[2], select node should be equal with origin one
	evQ.Put(NewEnableEvent(nodes[2]))
	time.Sleep(time.Second * 2)
	for i, k := range keys {
		sp := hl.Select(k)
		c.Assert(sp, check.NotNil)
		c.Assert(originSp[i], check.Equals, sp.String())
	}
}
