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
	"fmt"
)

// SupernodeLocator defines the way how to get available supernodes.
// Developers can implement their own locator more flexibly , not just get the
// supernode list from configuration or CLI.
type SupernodeLocator interface {
	// Get returns the current selected supernode, it should be idempotent.
	// It should return nil before first calling the Next method.
	Get() *Supernode

	// Next chooses the next available supernode for retrying or other
	// purpose. The current supernode should be set as this result.
	Next() *Supernode

	// Select chooses a supernode based on the giving key.
	// It should not affect the result of method 'Get()'.
	Select(key interface{}) *Supernode

	// GetGroup returns the group with the giving name.
	GetGroup(name string) *SupernodeGroup

	// All returns all the supernodes.
	All() []*SupernodeGroup

	// Size returns the number of all supernodes.
	Size() int

	// Report records the metrics of the current supernode in order to choose a
	// more appropriate supernode for the next time if necessary.
	Report(node string, metrics *SupernodeMetrics)

	// Refresh refreshes all the supernodes.
	Refresh() bool
}

// SupernodeGroup groups supernodes which have same attributes.
// For example, we can group supernodes by region, business, version and so on.
// The implementation of SupernodeLocator can select a supernode based on the
// group.
type SupernodeGroup struct {
	Name  string
	Nodes []*Supernode

	// Infos stores other information that user can customized.
	Infos map[string]string
}

// GetNode return the node with the giving index.
func (sg *SupernodeGroup) GetNode(idx int) *Supernode {
	if idx < 0 || idx >= len(sg.Nodes) {
		return nil
	}
	return sg.Nodes[idx]
}

// Supernode holds the basic information of supernodes.
type Supernode struct {
	Schema    string
	IP        string
	Port      int
	Weight    int
	GroupName string
	Metrics   *SupernodeMetrics
}

func (s *Supernode) String() string {
	return fmt.Sprintf("%s:%d", s.IP, s.Port)
}

// SupernodeMetrics holds metrics used for the locator to choose supernode.
type SupernodeMetrics struct {
	Metrics map[string]interface{}
}
