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
	"github.com/dragonflyoss/Dragonfly/dfget/types"

	//"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/api"

	"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/config"
	"github.com/dragonflyoss/Dragonfly/dfget/locator"
)

type Reporter struct {
	req *types.RegisterRequest
	//mReq          *config.ReportRequest
	supernodeAPI api.SupernodeAPI
}

func NewReporter(req *types.RegisterRequest, supernodeAPI api.SupernodeAPI) *Reporter {
	return &Reporter{
		req:          req,
		supernodeAPI: supernodeAPI,
	}
}

// Report reports local seed task to supernode.
func (rp *Reporter) Report(supernode *locator.Supernode) (basic.Response, error) {
	res, err := rp.supernodeAPI.ReportResource(supernode.String(), rp.req)
	if err != nil {
		return nil, err
	}
	return &config.ReportResponse{
		Code:   res.Code,
		AsSeed: res.Data.AsSeed,
	}, nil
}
