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

package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alibaba/Dragonfly/dfget/types"
	"github.com/alibaba/Dragonfly/dfget/util"
)

/* the url paths of supernode APIs*/
const (
	PeerRegisterPath      = "/peer/registry"
	PeerPullPieceTaskPath = "/peer/task"
	PeerReportPiecePath   = "/peer/piece/suc"
	PeerServiceDownPath   = "/peer/service/down"
)

// NewSupernodeAPI creates a new instance of SupernodeAPI with default value.
func NewSupernodeAPI() *SupernodeAPI {
	return &SupernodeAPI{
		Scheme:      "http",
		ServicePort: 8002,
		Timeout:     5 * time.Second,
	}
}

// SupernodeAPI implements the communication between supernode and dfget.
type SupernodeAPI struct {
	Scheme      string
	ServicePort int
	Timeout     time.Duration
}

// Register sends a request to the supernode to register itself as a peer
// and create downloading task.
func (api *SupernodeAPI) Register(ip string, req *types.RegisterRequest) (
	resp *types.RegisterResponse, e error) {
	var (
		code int
		body []byte
	)
	url := fmt.Sprintf("%s://%s:%d%s",
		api.Scheme, ip, api.ServicePort, PeerRegisterPath)
	if code, body, e = util.PostJSON(url, req, api.Timeout); e != nil {
		return nil, e
	}
	if !util.HTTPStatusOk(code) {
		return nil, fmt.Errorf("%d:%s", code, body)
	}
	resp = new(types.RegisterResponse)
	e = json.Unmarshal(body, resp)
	return resp, e
}

// PullPieceTask pull a piece downloading task from supernode, and get a
// response that describes from which peer to download.
func (api *SupernodeAPI) PullPieceTask(ip string, req *types.PullPieceTaskRequest) (
	resp *types.PullPieceTaskResponse, e error) {
	return
}

// ReportPiece reports the status of piece downloading task to supernode.
func (api *SupernodeAPI) ReportPiece(ip string, req *types.ReportPieceRequest) (
	resp *types.BaseResponse, e error) {
	return
}

// ServiceDown reports the status of the local peer to supernode.
func (api *SupernodeAPI) ServiceDown(ip string, taskID string, cid string) (
	resp *types.BaseResponse, e error) {
	return
}
