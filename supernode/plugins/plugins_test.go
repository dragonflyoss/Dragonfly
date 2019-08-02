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

package plugins

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/dragonflyoss/Dragonfly/supernode/config"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&PluginsTestSuite{})
}

type PluginsTestSuite struct {
	mgr Manager
}

func (s *PluginsTestSuite) SetUpSuite(c *check.C) {
	s.mgr = mgr
}

func (s *PluginsTestSuite) TearDownSuite(c *check.C) {
	mgr = s.mgr
}

func (s *PluginsTestSuite) TearDownTest(c *check.C) {
	mgr = s.mgr
}

func (s *PluginsTestSuite) TestSetManager(c *check.C) {
	tmp := &managerIml{}
	SetManager(tmp)
	c.Assert(mgr, check.Equals, tmp)
}

// -----------------------------------------------------------------------------

func (s *PluginsTestSuite) TestInitialize(c *check.C) {
	var testCase = func(cfg *config.Config, b Builder,
		pt config.PluginType, name string, hasPlugin bool, errMsg string) {
		SetManager(NewManager())
		RegisterPlugin(pt, name, b)
		err := Initialize(cfg)
		plugin := GetPlugin(pt, name)

		if errMsg != "" {
			c.Assert(err, check.NotNil)
			c.Assert(err, check.ErrorMatches, ".*"+errMsg+".*")
			c.Assert(plugin, check.IsNil)
		} else {
			c.Assert(err, check.IsNil)
			if hasPlugin {
				c.Assert(plugin.Type(), check.Equals, pt)
				c.Assert(plugin.Name(), check.Equals, name)
			} else {
				c.Assert(plugin, check.IsNil)
			}
		}
	}
	var testFunc = func(pt config.PluginType) {
		errMsg := "build error"
		name := "test"
		var createBuilder = func(err bool) Builder {
			return func(conf string) (plugin Plugin, e error) {
				if err {
					return nil, fmt.Errorf(errMsg)
				}
				return &mockPlugin{pt, name}, nil
			}
		}
		var createConf = func(enabled bool) *config.Config {
			plugins := make(map[config.PluginType][]*config.PluginProperties)
			plugins[pt] = []*config.PluginProperties{{Name: name, Enabled: enabled}}
			return &config.Config{Plugins: plugins}
		}
		testCase(createConf(false), createBuilder(false),
			pt, name, false, "")
		testCase(createConf(true), nil,
			pt, name, false, "cannot find builder")
		testCase(createConf(true), createBuilder(true),
			pt, name, false, errMsg)
		testCase(createConf(true), createBuilder(false),
			pt, name, true, "")
	}

	for _, pt := range config.PluginTypes {
		testFunc(pt)
	}
}

func (s *PluginsTestSuite) TestManagerIml_Builder(c *check.C) {
	var builder Builder = func(conf string) (plugin Plugin, e error) {
		return nil, nil
	}
	manager := NewManager()

	var testFunc = func(pt config.PluginType, name string, b Builder, result bool) {
		manager.AddBuilder(pt, name, b)
		obj := manager.GetBuilder(pt, name)
		if result {
			c.Assert(obj, check.NotNil)
			objVal := reflect.ValueOf(obj)
			bVal := reflect.ValueOf(b)
			c.Assert(objVal.Pointer(), check.Equals, bVal.Pointer())
			manager.DeleteBuilder(pt, name)
		} else {
			c.Assert(obj, check.IsNil)
		}
	}

	testFunc(config.PluginType("test"), "test", builder, false)
	for _, pt := range config.PluginTypes {
		testFunc(pt, "test", builder, true)
		testFunc(pt, "", nil, false)
		testFunc(pt, "", builder, false)
		testFunc(pt, "test", nil, false)
	}
}

func (s *PluginsTestSuite) TestManagerIml_Plugin(c *check.C) {
	manager := NewManager()

	var testFunc = func(p Plugin, result bool) {
		manager.AddPlugin(p)
		obj := manager.GetPlugin(p.Type(), p.Name())
		if result {
			c.Assert(obj, check.NotNil)
			c.Assert(obj, check.DeepEquals, p)
			manager.DeletePlugin(p.Type(), p.Name())
		} else {
			c.Assert(obj, check.IsNil)
		}
	}

	testFunc(&mockPlugin{"test", "test"}, false)
	for _, pt := range config.PluginTypes {
		testFunc(&mockPlugin{pt, "test"}, true)
		testFunc(&mockPlugin{pt, ""}, false)
	}
}

func (s *PluginsTestSuite) TestRepositoryIml(c *check.C) {
	type testCase struct {
		pt        config.PluginType
		name      string
		data      interface{}
		addResult bool
	}
	var createCase = func(validPlugin bool, name string, data interface{}, result bool) testCase {
		pt := config.StoragePlugin
		if !validPlugin {
			pt = config.PluginType("test-validPlugin")
		}
		return testCase{
			pt:        pt,
			name:      name,
			data:      data,
			addResult: result,
		}
	}
	var tc = func(valid bool, name string, data interface{}) testCase {
		return createCase(valid, name, data, true)
	}
	var fc = func(valid bool, name string, data interface{}) testCase {
		return createCase(valid, name, data, false)
	}
	var cases = []testCase{
		fc(true, "test", nil),
		fc(true, "", "data"),
		fc(false, "test", "data"),
		tc(true, "test", "data"),
	}

	repo := NewRepository()
	for _, v := range cases {
		repo.Add(v.pt, v.name, v.data)
		data := repo.Get(v.pt, v.name)
		if v.addResult {
			c.Assert(data, check.NotNil)
			c.Assert(data, check.DeepEquals, v.data)
			repo.Delete(v.pt, v.name)
			data = repo.Get(v.pt, v.name)
			c.Assert(data, check.IsNil)
		} else {
			c.Assert(data, check.IsNil)
		}
	}
}

func (s *PluginsTestSuite) TestValidate(c *check.C) {
	type testCase struct {
		pt       config.PluginType
		name     string
		expected bool
	}
	var cases = []testCase{
		{config.PluginType("test"), "", false},
		{config.PluginType("test"), "test", false},
	}
	for _, pt := range config.PluginTypes {
		cases = append(cases,
			testCase{pt, "", false},
			testCase{pt, "test", true},
		)
	}
	for _, v := range cases {
		c.Assert(validate(v.pt, v.name), check.Equals, v.expected,
			check.Commentf("pluginType:%v name:%s", v.pt, v.name))
	}
}

// -----------------------------------------------------------------------------

type mockPlugin struct {
	pt   config.PluginType
	name string
}

func (m *mockPlugin) Type() config.PluginType {
	return m.pt
}

func (m *mockPlugin) Name() string {
	return m.name
}
