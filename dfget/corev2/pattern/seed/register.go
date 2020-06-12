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

package seed

import (
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/basic"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/config"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
)

type RegisterHandler struct {
	supernode    string
	req          *types.RegisterRequest
	supernodeAPI api.SupernodeAPI
}

func NewRegister(req *types.RegisterRequest, supernodeAPI api.SupernodeAPI) *RegisterHandler {
	return &RegisterHandler{
		req:          req,
		supernodeAPI: supernodeAPI,
	}
}

// Register apply for seed node.
func (regist *RegisterHandler) Register(peerPort int) (basic.Response, error) {
	if peerPort > 0 {
		regist.req.Port = peerPort
	}
	regist.req.AsSeed = true
	res, err := regist.supernodeAPI.ApplyForSeedNode(regist.supernode, regist.req)
	if err != nil {
		return nil, err
	}

	return &config.RegisterResponse{
		Code:       res.Code,
		AsSeed:     res.Data.AsSeed,
		SeedTaskID: res.Data.SeedTaskID,
	}, nil
}
