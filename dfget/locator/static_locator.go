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
	"sync/atomic"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/pkg/algorithm"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
)

const staticLocatorGroupName = "config"

var _ SupernodeLocator = &StaticLocator{}

// StaticLocator uses the nodes passed from configuration or CLI.
type StaticLocator struct {
	idx   int32
	Group *SupernodeGroup
}

// ----------------------------------------------------------------------------
// constructors

// NewStaticLocator constructs StaticLocator which uses the nodes passed from
// configuration or CLI.
func NewStaticLocator(nodes []*config.NodeWeight) *StaticLocator {
	locator := &StaticLocator{
		idx: -1,
	}
	if len(nodes) == 0 {
		return locator
	}
	group := &SupernodeGroup{
		Name: staticLocatorGroupName,
	}
	for _, node := range nodes {
		ip, port := netutils.GetIPAndPortFromNode(node.Node, config.DefaultSupernodePort)
		if ip == "" {
			continue
		}
		supernode := &Supernode{
			Schema:    config.DefaultSupernodeSchema,
			IP:        ip,
			Port:      port,
			Weight:    node.Weight,
			GroupName: staticLocatorGroupName,
		}
		for i := 0; i < supernode.Weight; i++ {
			group.Nodes = append(group.Nodes, supernode)
		}
	}
	shuffleNodes(group.Nodes)
	locator.Group = group
	return locator
}

// NewStaticLocatorFromStr constructs StaticLocator from string list.
// The format of nodes is: ip:port=weight
func NewStaticLocatorFromStr(nodes []string) (*StaticLocator, error) {
	nodeWeight, err := config.ParseNodesSlice(nodes)
	if err != nil {
		return nil, err
	}
	return NewStaticLocator(nodeWeight), nil
}

// ----------------------------------------------------------------------------
// implement api methods

func (s *StaticLocator) Get() *Supernode {
	if s.Group == nil {
		return nil
	}
	return s.Group.GetNode(s.load())
}

func (s *StaticLocator) Next() *Supernode {
	if s.Group == nil || s.load() >= len(s.Group.Nodes) {
		return nil
	}
	return s.Group.GetNode(s.inc())
}

func (s *StaticLocator) GetGroup(name string) *SupernodeGroup {
	if s.Group == nil || s.Group.Name != name {
		return nil
	}
	return s.Group
}

func (s *StaticLocator) All() []*SupernodeGroup {
	if s.Group == nil {
		return nil
	}
	return []*SupernodeGroup{s.Group}
}

func (s *StaticLocator) Size() int {
	if s.Group == nil {
		return 0
	}
	return len(s.Group.Nodes)
}

func (s *StaticLocator) Report(node string, metrics *SupernodeMetrics) {
	// unnecessary to implement this method
	return
}

func (s *StaticLocator) Refresh() bool {
	atomic.StoreInt32(&s.idx, -1)
	return true
}

// ----------------------------------------------------------------------------
// private methods of StaticLocator

func (s *StaticLocator) load() int {
	return int(atomic.LoadInt32(&s.idx))
}

func (s *StaticLocator) inc() int {
	return int(atomic.AddInt32(&s.idx, 1))
}

// ----------------------------------------------------------------------------
// helper functions

func shuffleNodes(nodes []*Supernode) []*Supernode {
	if length := len(nodes); length > 1 {
		algorithm.Shuffle(length, func(i, j int) {
			nodes[i], nodes[j] = nodes[j], nodes[i]
		})
	}
	return nodes
}
