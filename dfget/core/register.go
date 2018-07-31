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
	"fmt"

	"github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/dfget/util"
)

func register(ctx *config.Context) error {
	if ctx.Pattern == config.PatternSource {
		ctx.BackSourceReason = config.BackSourceReasonByUser
		return fmt.Errorf("not register, pattern:%s", ctx.Pattern)
	}
	if len(ctx.Node) == 0 {
		ctx.BackSourceReason = config.BackSourceReasonNodeEmpty
		return fmt.Errorf("register fail, no available supernodes")
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
		util.Shuffle(nodesLen, func(i, j int) {
			nodes[i], nodes[j] = nodes[j], nodes[i]
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
