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

package config

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestSuite(t *testing.T) {
	suite.Run(t, new(SupernodeValueSuite))
}

type SupernodeValueSuite struct {
	suite.Suite
}

func (suit *SupernodeValueSuite) TestHandleNodes() {
	var cases = []struct {
		nodeWithWeightList []string
		expectedNodes      []*NodeWight
		gotError           bool
	}{
		{
			nodeWithWeightList: []string{"127.0.0.1", "127.0.0.2"},
			expectedNodes: []*NodeWight{
				{"127.0.0.1:8002", 1},
				{"127.0.0.2:8002", 1},
			},
		},
		{
			nodeWithWeightList: []string{"127.0.0.1=2", "127.0.0.2"},
			expectedNodes: []*NodeWight{
				{"127.0.0.1:8002", 2},
				{"127.0.0.2:8002", 1},
			},
		},
		{
			nodeWithWeightList: []string{"127.0.0.1=20", "127.0.0.2=20"},
			expectedNodes: []*NodeWight{
				{"127.0.0.1:8002", 1},
				{"127.0.0.2:8002", 1}},
		},
		{
			nodeWithWeightList: []string{"127.0.0.1=2", "127.0.0.2=4"},
			expectedNodes: []*NodeWight{
				{"127.0.0.1:8002", 1},
				{"127.0.0.2:8002", 2}},
		},
		{
			nodeWithWeightList: []string{"127.0.0.1:8002=1", "127.0.0.2:8001=2"},
			expectedNodes: []*NodeWight{
				{"127.0.0.1:8002", 1},
				{"127.0.0.2:8001", 2}},
		},
		{
			nodeWithWeightList: []string{"127.0.0.1:=2"},
			gotError:           true,
		},
		{
			nodeWithWeightList: []string{"127.0.0.1==1"},
			expectedNodes:      nil,
			gotError:           true,
		},
		{
			nodeWithWeightList: []string{"==2"},
			expectedNodes:      nil,
			gotError:           true,
		},
		{
			nodeWithWeightList: []string{"127.0.0.1==2"},
			expectedNodes:      nil,
			gotError:           true,
		},
	}

	for _, v := range cases {
		nodes, err := ParseNodesSlice(v.nodeWithWeightList)
		if v.gotError {
			suit.NotNil(err)
		} else {
			suit.Equal(v.expectedNodes, nodes)
		}
	}
}
