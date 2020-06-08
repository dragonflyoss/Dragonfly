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
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/pkg/algorithm"
	"github.com/dragonflyoss/Dragonfly/pkg/hashcircler"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"

	"github.com/sirupsen/logrus"
)

const (
	addEv    = "add"
	deleteEv = "delete"
)

type SuperNodeEvent struct {
	evType string
	node   string
}

func NewEnableEvent(node string) *SuperNodeEvent {
	return &SuperNodeEvent{
		evType: addEv,
		node:   node,
	}
}

func NewDisableEvent(node string) *SuperNodeEvent {
	return &SuperNodeEvent{
		evType: deleteEv,
		node:   node,
	}
}

// hashCirclerLocator is an implementation of SupernodeLocator. And it provides ability to select a supernode
// by input key. It allows some supernodes disabled, on this condition the disable supernode will not be selected.
type hashCirclerLocator struct {
	hc        hashcircler.HashCircler
	nodes     []string
	groupName string
	group     *SupernodeGroup

	// evQueue will puts/polls SuperNodeEvent to disable/enable supernode.
	evQueue queue.Queue
}

func NewHashCirclerLocator(groupName string, nodes []string, eventQueue queue.Queue) (SupernodeLocator, error) {
	nodes = algorithm.DedupStringArr(nodes)
	if len(nodes) == 0 {
		return nil, fmt.Errorf("nodes should not be nil")
	}

	sort.Strings(nodes)

	group := &SupernodeGroup{
		Name:  groupName,
		Nodes: []*Supernode{},
		Infos: make(map[string]string),
	}
	keys := []string{}
	for _, node := range nodes {
		ip, port := netutils.GetIPAndPortFromNode(node, config.DefaultSupernodePort)
		if ip == "" {
			continue
		}
		supernode := &Supernode{
			Schema:    config.DefaultSupernodeSchema,
			IP:        ip,
			Port:      port,
			GroupName: groupName,
		}

		group.Nodes = append(group.Nodes, supernode)
		keys = append(keys, supernode.String())
	}

	hc, err := hashcircler.NewConsistentHashCircler(keys, nil)
	if err != nil {
		return nil, err
	}

	h := &hashCirclerLocator{
		hc:        hc,
		evQueue:   eventQueue,
		groupName: groupName,
		group:     group,
	}

	go h.eventLoop(context.Background())

	return h, nil
}

func (h *hashCirclerLocator) Get() *Supernode {
	// not implementation
	return nil
}

func (h *hashCirclerLocator) Next() *Supernode {
	// not implementation
	return nil
}

func (h *hashCirclerLocator) Select(key interface{}) *Supernode {
	s, err := h.hc.Hash(key.(string))
	if err != nil {
		logrus.Errorf("failed to get supernode: %v", err)
		return nil
	}

	for _, sp := range h.group.Nodes {
		if s == sp.String() {
			return sp
		}
	}

	return nil
}

func (h *hashCirclerLocator) GetGroup(name string) *SupernodeGroup {
	if h.group == nil || h.group.Name != name {
		return nil
	}

	return h.group
}

func (h *hashCirclerLocator) All() []*SupernodeGroup {
	return []*SupernodeGroup{h.group}
}

func (h *hashCirclerLocator) Size() int {
	return len(h.group.Nodes)
}

func (h *hashCirclerLocator) Report(node string, metrics *SupernodeMetrics) {
	return
}

func (h *hashCirclerLocator) Refresh() bool {
	return true
}

func (h *hashCirclerLocator) eventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if ev, ok := h.evQueue.PollTimeout(time.Second); ok {
			h.handleEvent(ev.(*SuperNodeEvent))
		}
	}
}

func (h *hashCirclerLocator) handleEvent(ev *SuperNodeEvent) {
	switch ev.evType {
	case addEv:
		h.hc.Add(ev.node)
	case deleteEv:
		h.hc.Delete(ev.node)
	default:
	}

	return
}
