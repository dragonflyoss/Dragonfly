/*
 * Copyright 1999-2018 Alibaba Group.
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

package core

import (
	"math/rand"

	"github.com/alibaba/Dragonfly/dfget/config"
)

func register(ctx *config.Context) error {
	if len(ctx.Node) == 0 {
		ctx.BackSourceReason = config.BackSourceReasonNodeEmpty
	}
	return nil
}

func adjustSupernodeList(nodes []string) []string {
	switch nodesLen := len(nodes); nodesLen {
	case 0:
		return nodes
	case 1:
		return append(nodes, nodes[0])
	default:
		rand.Shuffle(nodesLen, func(i, j int) {
			tmp := nodes[i]
			nodes[i] = nodes[j]
			nodes[j] = tmp
		})
		return append(nodes, nodes...)
	}
}

func checkConnectSupernode(nodes []string) (localIP string) {
	return
}

func launchPeerServer() (port string) {
	return
}

/*
  def parse_super():
    nodes_len = len(nodes) if nodes else 0
    while nodes_len > 0:
        node = nodes.pop(0)
        nodes_len -= 1
        if node:
            addr_fields = node.split(":")
            if len(addr_fields) == 1:
                addr_fields.append(8002)
            local_ip = netutil.check_connect(addr_fields[0], int(addr_fields[1]), timeout=2)
            if local_ip:
                nodes.insert(0, node)
                return local_ip
    return None
*/
