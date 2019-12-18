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

	api_types "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/pkg/constants"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"

	"github.com/sirupsen/logrus"
)

/* the url paths of supernode APIs*/
const (
	peerRegisterPath      = "/peer/registry"
	peerPullPieceTaskPath = "/peer/task"
	peerReportPiecePath   = "/peer/piece/suc"
	peerClientErrorPath   = "/peer/piece/error"
	peerServiceDownPath   = "/peer/service/down"
	metricsReportPath     = "/task/metrics"
)

// NewSupernodeAPI creates a new instance of SupernodeAPI with default value.
func NewSupernodeAPI() SupernodeAPI {
	return &supernodeAPI{
		Scheme:     "http",
		Timeout:    10 * time.Second,
		HTTPClient: httputils.DefaultHTTPClient,
	}
}

// SupernodeAPI defines the communication methods between supernode and dfget.
type SupernodeAPI interface {
	Register(node string, req *types.RegisterRequest) (resp *types.RegisterResponse, e error)
	PullPieceTask(node string, req *types.PullPieceTaskRequest) (resp *types.PullPieceTaskResponse, e error)
	ReportPiece(node string, req *types.ReportPieceRequest) (resp *types.BaseResponse, e error)
	ServiceDown(node string, taskID string, cid string) (resp *types.BaseResponse, e error)
	ReportClientError(node string, req *types.ClientErrorRequest) (resp *types.BaseResponse, e error)
	ReportMetrics(node string, req *api_types.TaskMetricsRequest) (resp *types.BaseResponse, e error)
}

type supernodeAPI struct {
	Scheme     string
	Timeout    time.Duration
	HTTPClient httputils.SimpleHTTPClient
}

var _ SupernodeAPI = &supernodeAPI{}

// Register sends a request to the supernode to register itself as a peer
// and create downloading task.
func (api *supernodeAPI) Register(node string, req *types.RegisterRequest) (
	resp *types.RegisterResponse, e error) {
	var (
		code int
		body []byte
	)
	url := fmt.Sprintf("%s://%s%s",
		api.Scheme, node, peerRegisterPath)
	if code, body, e = api.HTTPClient.PostJSON(url, req, api.Timeout); e != nil {
		return nil, e
	}
	if !httputils.HTTPStatusOk(code) {
		return nil, fmt.Errorf("%d:%s", code, body)
	}
	resp = new(types.RegisterResponse)
	if e = json.Unmarshal(body, resp); e != nil {
		return nil, e
	}
	return resp, e
}

// PullPieceTask pull a piece downloading task from supernode, and get a
// response that describes from which peer to download.
func (api *supernodeAPI) PullPieceTask(node string, req *types.PullPieceTaskRequest) (
	resp *types.PullPieceTaskResponse, e error) {

	url := fmt.Sprintf("%s://%s%s?%s",
		api.Scheme, node, peerPullPieceTaskPath, httputils.ParseQuery(req))

	logrus.Debugf("start to Pull PieceTask taskId:%s, req: %s", req.TaskID, url)
	for i := 0; i < 3; i++ {
		resp = new(types.PullPieceTaskResponse)
		if e = api.get(url, resp); e != nil {
			continue
		} else {
			return
		}
	}
	return
}

// ReportPiece reports the status of piece downloading task to supernode.
func (api *supernodeAPI) ReportPiece(node string, req *types.ReportPieceRequest) (
	resp *types.BaseResponse, e error) {

	url := fmt.Sprintf("%s://%s%s?%s",
		api.Scheme, node, peerReportPiecePath, httputils.ParseQuery(req))

	resp = new(types.BaseResponse)
	if e = api.get(url, resp); e != nil {
		logrus.Errorf("failed to report piece{taskid:%s,range:%s},err: %v", req.TaskID, req.PieceRange, e)
		return nil, e
	}
	if resp.Code != constants.CodeGetPieceReport {
		logrus.Errorf("failed to report piece{taskid:%s,range:%s} to supernode: api response code is %d not equal to %d", req.TaskID, req.PieceRange, resp.Code, constants.CodeGetPieceReport)
	}
	return
}

// ServiceDown reports the status of the local peer to supernode.
func (api *supernodeAPI) ServiceDown(node string, taskID string, cid string) (
	resp *types.BaseResponse, e error) {

	url := fmt.Sprintf("%s://%s%s?taskId=%s&cid=%s",
		api.Scheme, node, peerServiceDownPath, taskID, cid)

	resp = new(types.BaseResponse)
	if e = api.get(url, resp); e != nil {
		logrus.Errorf("failed to send service down,err: %v", e)
		return nil, e
	}
	if resp.Code != constants.CodeGetPeerDown {
		logrus.Errorf("failed to send service down to supernode: api response code is %d not equal to %d", resp.Code, constants.CodeGetPeerDown)
	}
	return
}

// ReportClientError reports the client error when downloading piece to supernode.
func (api *supernodeAPI) ReportClientError(node string, req *types.ClientErrorRequest) (
	resp *types.BaseResponse, e error) {

	url := fmt.Sprintf("%s://%s%s?%s",
		api.Scheme, node, peerClientErrorPath, httputils.ParseQuery(req))

	resp = new(types.BaseResponse)
	e = api.get(url, resp)
	return
}

func (api *supernodeAPI) ReportMetrics(node string, req *api_types.TaskMetricsRequest) (resp *types.BaseResponse, err error) {
	var (
		code int
		body []byte
	)
	url := fmt.Sprintf("%s://%s%s",
		api.Scheme, node, metricsReportPath)
	if code, body, err = api.HTTPClient.PostJSON(url, req, api.Timeout); err != nil {
		return nil, err
	}
	if !httputils.HTTPStatusOk(code) {
		return nil, fmt.Errorf("%d:%s", code, body)
	}
	resp = new(types.BaseResponse)
	if err = json.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, err
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
	if !httputils.HTTPStatusOk(code) {
		return fmt.Errorf("%d:%s", code, body)
	}
	return json.Unmarshal(body, resp)
}
