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
	"github.com/pkg/errors"

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
	fetchP2PNetworkPath   = "/peer/network"
	peerHeartBeatPath     = "/peer/heartbeat"
)

// NewSupernodeAPI creates a new instance of SupernodeAPI with default value.
func NewSupernodeAPI() SupernodeAPI {
	return &supernodeAPI{
		Scheme:     "http",
		Timeout:    5 * time.Second,
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
	HeartBeat(node string, req *api_types.HeartBeatRequest) (resp *types.HeartBeatResponse, err error)
	FetchP2PNetworkInfo(node string, start int, limit int, req *api_types.NetworkInfoFetchRequest) (resp *api_types.NetworkInfoFetchResponse, e error)
	ReportResource(node string, req *types.RegisterRequest) (resp *types.RegisterResponse, err error)
	ApplyForSeedNode(node string, req *types.RegisterRequest) (resp *types.RegisterResponse, err error)
	ReportResourceDeleted(node string, taskID string, cid string) (resp *types.BaseResponse, err error)
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

	resp = new(types.PullPieceTaskResponse)
	if e = api.get(url, resp); e != nil {
		return nil, e
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
		return nil, errors.Wrapf(e, "failed to report piece{taskid:%s,range:%s}", req.TaskID, req.PieceRange)
	}
	if resp.Code != constants.CodeGetPieceReport {
		logrus.Errorf("failed to report piece{taskid:%s,range:%s} to supernode: api response code is %d not equal to %d", req.TaskID, req.PieceRange, resp.Code, constants.CodeGetPieceReport)
		return nil, errors.Wrapf(e, "failed to report piece{taskid:%s,range:%s} to supernode: api response code is %d not equal to %d", req.TaskID, req.PieceRange, resp.Code, constants.CodeGetPieceReport)
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

// report resource
func (api *supernodeAPI) ReportResource(node string, req *types.RegisterRequest) (resp *types.RegisterResponse, err error) {
	var (
		code int
		body []byte
	)
	url := fmt.Sprintf("%s://%s%s",
		api.Scheme, node, peerRegisterPath)
	header := map[string]string{
		"X-report-resource": "true",
	}
	if code, body, err = api.HTTPClient.PostJSONWithHeaders(url, header, req, api.Timeout); err != nil {
		return nil, err
	}

	logrus.Infof("ReportResource, url: %s, header: %v, req: %v, "+
		"code: %d, body: %s", url, header, req, code, string(body))

	if !httputils.HTTPStatusOk(code) {
		return nil, fmt.Errorf("%d:%s", code, body)
	}
	resp = new(types.RegisterResponse)
	if err = json.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, err
}

func (api *supernodeAPI) ReportResourceDeleted(node string, taskID string, cid string) (resp *types.BaseResponse, err error) {
	url := fmt.Sprintf("%s://%s%s?taskId=%s&cid=%s",
		api.Scheme, node, peerServiceDownPath, taskID, cid)

	header := map[string]string{
		"X-report-resource": "true",
	}

	logrus.Infof("Call ReportResourceDeleted, node: %s, taskID: %s, cid: %s, "+
		"url: %s, header: %v", node, taskID, cid, url, header)

	resp = new(types.BaseResponse)
	resp.Code = constants.Success

	if err = api.get(url, resp); err != nil {
		logrus.Errorf("failed to send service down,err: %v", err)
		return nil, err
	}
	if resp.Code != constants.CodeGetPeerDown {
		logrus.Errorf("failed to send service down to supernode: api response code is %d not equal to %d", resp.Code, constants.CodeGetPeerDown)
	}

	return
}

// apply for seed node to supernode, if selected as seed, the resp.AsSeed will set true.
func (api *supernodeAPI) ApplyForSeedNode(node string, req *types.RegisterRequest) (resp *types.RegisterResponse, err error) {
	var (
		code int
		body []byte
	)
	url := fmt.Sprintf("%s://%s%s",
		api.Scheme, node, peerRegisterPath)
	header := map[string]string{
		"X-report-resource": "true",
	}
	if code, body, err = api.HTTPClient.PostJSONWithHeaders(url, header, req, api.Timeout); err != nil {
		return nil, err
	}

	logrus.Infof("ReportResource, url: %s, header: %v, req: %v, "+
		"code: %d, body: %s", url, header, req, code, string(body))

	if !httputils.HTTPStatusOk(code) {
		return nil, fmt.Errorf("%d:%s", code, body)
	}
	resp = new(types.RegisterResponse)
	if err = json.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, err
}

// FetchP2PNetworkInfo fetch the p2p network info from supernode.
// @parameter
// start: the start index for array of result
// limit: the limit size of array of result, if -1 means no paging
func (api *supernodeAPI) FetchP2PNetworkInfo(node string, start int, limit int, req *api_types.NetworkInfoFetchRequest) (resp *api_types.NetworkInfoFetchResponse, err error) {
	var (
		code int
		body []byte
	)

	if start < 0 {
		start = 0
	}

	if limit < 0 {
		limit = -1
	}

	if limit == 0 {
		//todo: the page default limit should be configuration item of dfdaemon
		limit = 500
	}

	url := fmt.Sprintf("%s://%s%s?start=%d&limit=%d",
		api.Scheme, node, fetchP2PNetworkPath, start, limit)
	if code, body, err = api.HTTPClient.PostJSON(url, req, api.Timeout); err != nil {
		return nil, err
	}

	logrus.Debugf("in FetchP2PNetworkInfo, req url: %s, timeout: %v, body: %v", url, api.Timeout, req)
	logrus.Debugf("in FetchP2PNetworkInfo, resp code: %d, body: %s", code, string(body))

	if !httputils.HTTPStatusOk(code) {
		return nil, fmt.Errorf("%d:%s", code, body)
	}
	rr := new(types.FetchP2PNetworkInfoResponse)
	if err = json.Unmarshal(body, rr); err != nil {
		return nil, err
	}

	if rr.Code != constants.Success {
		return nil, fmt.Errorf("%d:%s", code, rr.Msg)
	}

	return rr.Data, nil
}

func (api *supernodeAPI) HeartBeat(node string, req *api_types.HeartBeatRequest) (resp *types.HeartBeatResponse, err error) {
	var (
		code int
		body []byte
	)

	url := fmt.Sprintf("%s://%s%s",
		api.Scheme, node, peerHeartBeatPath)

	if code, body, err = api.HTTPClient.PostJSON(url, req, api.Timeout); err != nil {
		return nil, err
	}

	if !httputils.HTTPStatusOk(code) {
		logrus.Errorf("failed to heart beat, code %d, body: %s", code, string(body))
		return nil, fmt.Errorf("%d:%s", code, string(body))
	}

	logrus.Debugf("heart beat resp: %s", string(body))

	resp = new(types.HeartBeatResponse)
	if err = json.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, err
}
