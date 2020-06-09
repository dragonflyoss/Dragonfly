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
	"math/rand"
	"strings"
	"testing"

	"github.com/go-check/check"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type StaticLocatorTestSuite struct {
}

func init() {
	check.Suite(&StaticLocatorTestSuite{})
}

var testGroupName = "test-group"

func (s *StaticLocatorTestSuite) Test_NewStaticLocator(c *check.C) {
	rand.Seed(0)
	l := NewStaticLocator(testGroupName, nil)
	c.Assert(l, check.NotNil)
	c.Assert(l.idx, check.Equals, int32(-1))
	c.Assert(l.Group, check.IsNil)

	l = NewStaticLocator(testGroupName, []*config.NodeWeight{})
	c.Assert(l, check.NotNil)
	c.Assert(l.idx, check.Equals, int32(-1))
	c.Assert(l.Group, check.IsNil)

	l = NewStaticLocator(testGroupName, []*config.NodeWeight{
		{Node: "a:80", Weight: 1},
		{Node: "a:81", Weight: 2},
	})
	c.Assert(l, check.NotNil)
	c.Assert(l.Group, check.DeepEquals, &SupernodeGroup{
		Name: testGroupName,
		Nodes: shuffleNodes([]*Supernode{
			create("a", 80, 1),
			create("a", 81, 2),
			create("a", 81, 2),
		}),
		Infos: nil,
	})
}

func (s *StaticLocatorTestSuite) Test_NewStaticLocatorFromString(c *check.C) {
	cases := []struct {
		nodes       string
		err         bool
		expectedLen int
	}{
		{":80=1", true, 0},
		{"a:80=1", false, 1},
		{"a:80=1,a:81=2", false, 3},
	}

	for _, v := range cases {
		l, err := NewStaticLocatorFromStr(testGroupName, strings.Split(v.nodes, ","))
		if v.err {
			c.Assert(err, check.NotNil)
			c.Assert(l, check.IsNil)
		} else {
			c.Assert(err, check.IsNil)
			c.Assert(l, check.NotNil)
			c.Assert(len(l.Group.Nodes), check.Equals, v.expectedLen)
			c.Assert(l.Size(), check.Equals, v.expectedLen)
		}
	}
}

func (s *StaticLocatorTestSuite) Test_Get(c *check.C) {
	cases := []struct {
		nodes    string
		expected *Supernode
	}{
		{"a:80=1", create("a", 80, 1)},
	}
	for _, v := range cases {
		l := createLocator(strings.Split(v.nodes, ",")...)
		sn := l.Get()
		c.Assert(sn, check.IsNil)
		l.Next()
		sn = l.Get()
		if v.expected == nil {
			c.Assert(sn, check.IsNil)
		} else {
			c.Assert(sn, check.NotNil)
			c.Assert(sn, check.DeepEquals, v.expected)
		}
	}
}

func (s *StaticLocatorTestSuite) Test_Next(c *check.C) {
	cases := []struct {
		nodes       string
		cnt         int
		expectedIdx int
	}{
		{"a:80=1", 0, -1},
		{"a:80=1", 1, 0},
		{"a:80=1,a:81=2", 2, 1},
		// the weight of a:81 is 2, it will be chosen twice
		{"a:80=1,a:81=2", 3, 2},
		// return nil because 4 is greater than the length
		{"a:80=1,a:81=2", 4, -1},
	}

	var sn *Supernode
	for _, v := range cases {
		l := createLocator(strings.Split(v.nodes, ",")...)
		for i := 0; i < v.cnt; i++ {
			sn = l.Next()
		}
		if v.expectedIdx < 0 {
			c.Assert(sn, check.IsNil)
		} else {
			c.Assert(sn, check.NotNil)
			c.Assert(sn, check.DeepEquals, l.Group.Nodes[v.expectedIdx])
		}
	}
}

func (s *StaticLocatorTestSuite) Test_GetGroup(c *check.C) {
	l := createLocator("a:80=1")
	group := l.GetGroup(testGroupName)
	c.Assert(group, check.NotNil)
	c.Assert(group.Nodes[0], check.DeepEquals, create("a", 80, 1))

	group = l.GetGroup("test")
	c.Assert(group, check.IsNil)
}

func (s *StaticLocatorTestSuite) Test_All(c *check.C) {
	l := createLocator("a:80=1")
	groups := l.All()
	c.Assert(groups, check.NotNil)
	c.Assert(len(groups), check.Equals, 1)
}

func (s *StaticLocatorTestSuite) Test_Refresh(c *check.C) {
	l := createLocator("a:80=1")
	_ = l.Next()
	c.Assert(l.load(), check.Equals, 0)

	l.Refresh()
	c.Assert(l.load(), check.Equals, -1)
}

func (s *StaticLocatorTestSuite) Test_String(c *check.C) {
	cases := []struct {
		locator    *StaticLocator
		isGroupNil bool
		increments int
		expected   string
	}{
		{
			locator:    createLocator("a:80=1"),
			isGroupNil: true,
			increments: 0,
			expected:   "empty",
		},
		{
			locator:    createLocator("a:80=1"),
			isGroupNil: false,
			increments: 2,
			expected:   "empty",
		},
		{
			locator:    createLocator("a:80=1"),
			isGroupNil: false,
			increments: 0,
			expected:   "test-group:[a:80=1]",
		},
	}

	for _, v := range cases {
		if v.isGroupNil {
			v.locator.Group = nil
		}
		for i := 0; i < v.increments; i++ {
			v.locator.inc()
		}

		c.Assert(v.locator.String(), check.Equals, v.expected)
	}
}

func create(ip string, port, weight int) *Supernode {
	return &Supernode{
		Schema:    config.DefaultSupernodeSchema,
		IP:        ip,
		Port:      port,
		Weight:    weight,
		GroupName: testGroupName,
	}
}
func createLocator(nodes ...string) *StaticLocator {
	l, _ := NewStaticLocatorFromStr(testGroupName, nodes)
	return l
}
