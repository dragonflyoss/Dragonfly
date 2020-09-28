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
	"github.com/dragonflyoss/Dragonfly/dfget/config"
)

func (s *LocatorTestSuite) Test_CreateLocator() {
	cases := []struct {
		cfg      *config.Config
		expected SupernodeLocator
	}{
		{
			cfg: nil,
			expected: &StaticLocator{
				idx: -1,
				Group: &SupernodeGroup{
					Name: "default",
					Nodes: []*Supernode{
						{
							Schema:    "http",
							IP:        "127.0.0.1",
							Port:      8002,
							Weight:    1,
							GroupName: "default",
						},
					},
				},
			},
		},
		{
			cfg: &config.Config{},
			expected: &StaticLocator{
				idx: -1,
				Group: &SupernodeGroup{
					Name: "default",
					Nodes: []*Supernode{
						{
							Schema:    "http",
							IP:        "127.0.0.1",
							Port:      8002,
							Weight:    1,
							GroupName: "default",
						},
					},
				},
			},
		},
		{
			cfg: &config.Config{
				Nodes: []string{"localhost"},
			},
			expected: &StaticLocator{
				idx: -1,
				Group: &SupernodeGroup{
					Name: "config",
					Nodes: []*Supernode{
						{
							Schema:    "http",
							IP:        "localhost",
							Port:      8002,
							Weight:    1,
							GroupName: "config",
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		got := CreateLocator(c.cfg)
		s.Equal(c.expected, got)
	}
}
