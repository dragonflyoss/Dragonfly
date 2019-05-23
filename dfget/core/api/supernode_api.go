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

package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"strings"
	"strconv"
)

/* the url paths of supernode APIs*/
const (
	peerRegisterPath      = "/peer/registry"
	peerPullPieceTaskPath = "/peer/task"
	peerReportPiecePath   = "/peer/piece/suc"
	peerServiceDownPath   = "/peer/service/down"
)

// NewSupernodeAPI creates a new instance of SupernodeAPI with default value.
func NewSupernodeAPI() SupernodeAPI {
	return &supernodeAPI{
		Scheme:      "http",
		ServicePort: 8002,
		Timeout:     5 * time.Second,
		HTTPClient:  util.DefaultHTTPClient,
	}
}

// SupernodeAPI defines the communication methods between supernode and dfget.
type SupernodeAPI interface {
	Register(node string, req *types.RegisterRequest) (resp *types.RegisterResponse, e error)
	PullPieceTask(node string, req *types.PullPieceTaskRequest) (resp *types.PullPieceTaskResponse, e error)
	ReportPiece(node string, req *types.ReportPieceRequest) (resp *types.BaseResponse, e error)
	ServiceDown(node string, taskID string, cid string) (resp *types.BaseResponse, e error)
}

type supernodeAPI struct {
	Scheme      string
	ServicePort int
	Timeout     time.Duration
	HTTPClient  util.SimpleHTTPClient
}

var _ SupernodeAPI = &supernodeAPI{}

//dfget --node ends with ":port" or not
func (api *supernodeAPI) nodeToIPAndPort( node string) (ip string, port int ){

	strs := strings.Split(node, ":")
	switch len(strs) {
	case 0:
		return node, api.ServicePort
	case 1:
		return strs[0], api.ServicePort
	default:
		p , err := strconv.Atoi(strs[1])
		if err != nil {
			return strs[0], api.ServicePort
		}
		return strs[0], p
	}

}

// Register sends a request to the supernode to register itself as a peer
// and create downloading task.
func (api *supernodeAPI) Register(node string, req *types.RegisterRequest) (
	resp *types.RegisterResponse, e error) {
	var (
		code int
		body []byte
	)
	ip, port := api.nodeToIPAndPort(node)
	url := fmt.Sprintf("%s://%s:%d%s",
		api.Scheme, ip, port, peerRegisterPath)
	if code, body, e = api.HTTPClient.PostJSON(url, req, api.Timeout); e != nil {
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
func (api *supernodeAPI) PullPieceTask(node string, req *types.PullPieceTaskRequest) (
	resp *types.PullPieceTaskResponse, e error) {

	ip, port := api.nodeToIPAndPort(node)

	url := fmt.Sprintf("%s://%s:%d%s?%s",
		api.Scheme, ip, port, peerPullPieceTaskPath, util.ParseQuery(req))

	resp = new(types.PullPieceTaskResponse)
	e = api.get(url, resp)
	return
}

// ReportPiece reports the status of piece downloading task to supernode.
func (api *supernodeAPI) ReportPiece(node string, req *types.ReportPieceRequest) (
	resp *types.BaseResponse, e error) {

	ip, port := api.nodeToIPAndPort(node)
	url := fmt.Sprintf("%s://%s:%d%s?%s",
		api.Scheme, ip, port, peerReportPiecePath, util.ParseQuery(req))

	resp = new(types.BaseResponse)
	e = api.get(url, resp)
	return
}

// ServiceDown reports the status of the local peer to supernode.
func (api *supernodeAPI) ServiceDown(node string, taskID string, cid string) (
	resp *types.BaseResponse, e error) {

	ip, port := api.nodeToIPAndPort(node)
	url := fmt.Sprintf("%s://%s:%d%s?taskId=%s&cid=%s",
		api.Scheme, ip, port, peerServiceDownPath, taskID, cid)

	resp = new(types.BaseResponse)
	e = api.get(url, resp)
	return
}

func (api *supernodeAPI) get(url string, resp interface{}) error {
	var (
		code int
		body []byte
		e    error
	)
	if url == "" {
		return fmt.Errorf("invalid url")
	}
	if code, body, e = api.HTTPClient.Get(url, api.Timeout); e != nil {
		return e
	}
	if !util.HTTPStatusOk(code) {
		return fmt.Errorf("%d:%s", code, body)
	}
	e = json.Unmarshal(body, resp)
	return e
}
