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

package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/schema"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/constants"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/rangeutils"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
)

// RegisterResponseData is the data when registering supernode successfully.
type RegisterResponseData struct {
	TaskID     string `json:"taskId"`
	FileLength int64  `json:"fileLength"`
	PieceSize  int32  `json:"pieceSize"`
	CDNSource  string `json:"cdnSource"`

	// in seed pattern, if peer selected as seed, AsSeed sets true.
	AsSeed bool `json:"asSeed"`

	// in seed pattern, if as seed, SeedTaskID is the taskID of seed file.
	SeedTaskID string `json:"seedTaskID"`
}

// PullPieceTaskResponseContinueData is the data when successfully pulling piece task
// and the task is continuing.
type PullPieceTaskResponseContinueData struct {
	Range     string `json:"range"`
	PieceNum  int    `json:"pieceNum"`
	PieceSize int32  `json:"pieceSize"`
	PieceMd5  string `json:"pieceMd5"`
	Cid       string `json:"cid"`
	PeerIP    string `json:"peerIp"`
	PeerPort  int    `json:"peerPort"`
	Path      string `json:"path"`
	DownLink  int    `json:"downLink"`
}

var statusMap = map[string]string{
	"700": types.PiecePullRequestDfgetTaskStatusSTARTED,
	"701": types.PiecePullRequestDfgetTaskStatusRUNNING,
	"702": types.PiecePullRequestDfgetTaskStatusFINISHED,
}

var resultMap = map[string]string{
	"500": types.PiecePullRequestPieceResultFAILED,
	"501": types.PiecePullRequestPieceResultSUCCESS,
	"502": types.PiecePullRequestPieceResultINVALID,
	"503": types.PiecePullRequestPieceResultSEMISUC,
}

func (s *Server) registry(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	request := &types.TaskRegisterRequest{}

	// parse request.Body to the types.TaskRegisterRequest struct
	ct := req.Header.Get("Content-Type")
	if ct == "application/x-www-form-urlencoded" {
		if err := req.ParseForm(); err != nil {
			return errors.Wrapf(errortypes.ErrInvalidValue, "failed to parse the request body as a form: %v", err)
		}

		decoder := schema.NewDecoder()
		err = decoder.Decode(request, req.PostForm)
		if err != nil {
			return errors.Wrap(errortypes.ErrInvalidValue, err.Error())
		}
	} else {
		reader := req.Body
		if err := json.NewDecoder(reader).Decode(request); err != nil {
			return errors.Wrap(errortypes.ErrInvalidValue, err.Error())
		}
	}

	if err := request.Validate(strfmt.NewFormats()); err != nil {
		return errors.Wrap(errortypes.ErrInvalidValue, err.Error())
	}

	peerCreateRequest := &types.PeerCreateRequest{
		IP:       request.IP,
		HostName: strfmt.Hostname(request.HostName),
		Port:     request.Port,
		Version:  request.Version,
	}
	peerCreateResponse, err := s.PeerMgr.Register(ctx, peerCreateRequest)
	if err != nil {
		logrus.Errorf("failed to register peer %+v: %v", peerCreateRequest, err)
		return errors.Wrapf(errortypes.ErrSystemError, "failed to register peer: %v", err)
	}
	logrus.Infof("success to register peer %+v", peerCreateRequest)

	peerID := peerCreateResponse.ID
	taskCreateRequest := &types.TaskCreateRequest{
		CID:         request.CID,
		CallSystem:  request.CallSystem,
		Dfdaemon:    request.Dfdaemon,
		Headers:     netutils.ConvertHeaders(request.Headers),
		Identifier:  request.Identifier,
		Md5:         request.Md5,
		Path:        request.Path,
		PeerID:      peerID,
		RawURL:      request.RawURL,
		TaskURL:     request.TaskURL,
		SupernodeIP: request.SuperNodeIP,
	}
	s.originClient.RegisterTLSConfig(taskCreateRequest.RawURL, request.Insecure, request.RootCAs)
	resp, err := s.TaskMgr.Register(ctx, taskCreateRequest)
	if err != nil {
		logrus.Errorf("failed to register task %+v: %v", taskCreateRequest, err)
		return err
	}
	logrus.Debugf("success to register task %+v", taskCreateRequest)
	return EncodeResponse(rw, http.StatusOK, &types.ResultInfo{
		Code: constants.Success,
		Msg:  constants.GetMsgByCode(constants.Success),
		Data: &RegisterResponseData{
			TaskID:     resp.ID,
			FileLength: resp.FileLength,
			PieceSize:  resp.PieceSize,
			CDNSource:  string(resp.CdnSource),
		},
	})
}

func (s *Server) pullPieceTask(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	params := req.URL.Query()
	taskID := params.Get("taskId")
	srcCID := params.Get("srcCid")

	request := &types.PiecePullRequest{
		DfgetTaskStatus: statusMap[params.Get("status")],
		PieceRange:      params.Get("range"),
		PieceResult:     resultMap[params.Get("result")],
	}

	// try to get dstPID
	dstCID := params.Get("dstCid")
	if !stringutils.IsEmptyStr(dstCID) {
		dstDfgetTask, err := s.DfgetTaskMgr.Get(ctx, dstCID, taskID)
		if err != nil {
			logrus.Warnf("failed to get dfget task by dstCID(%s) and taskID(%s), and the srcCID is %s, err: %v",
				dstCID, taskID, srcCID, err)
		} else {
			request.DstPID = dstDfgetTask.PeerID
		}
	}

	isFinished, data, err := s.TaskMgr.GetPieces(ctx, taskID, srcCID, request)
	if err != nil {
		if errortypes.IsCDNFail(err) {
			logrus.Errorf("taskID:%s, failed to get pieces %+v: %v", taskID, request, err)
		}
		resultInfo := NewResultInfoWithError(err)
		return EncodeResponse(rw, http.StatusOK, &types.ResultInfo{
			Code: int32(resultInfo.code),
			Msg:  resultInfo.msg,
			Data: data,
		})
	}

	if isFinished {
		return EncodeResponse(rw, http.StatusOK, &types.ResultInfo{
			Code: constants.CodePeerFinish,
			Data: data,
		})
	}

	var datas []*PullPieceTaskResponseContinueData
	pieceInfos, ok := data.([]*types.PieceInfo)
	if !ok {
		return EncodeResponse(rw, http.StatusOK, &types.ResultInfo{
			Code: constants.CodeSystemError,
			Msg:  "failed to parse PullPieceTaskResponseContinueData",
		})
	}

	for _, v := range pieceInfos {
		cid, err := s.DfgetTaskMgr.GetCIDByPeerIDAndTaskID(ctx, v.PID, taskID)
		if err != nil {
			continue
		}
		datas = append(datas, &PullPieceTaskResponseContinueData{
			Range:     v.PieceRange,
			PieceNum:  rangeutils.CalculatePieceNum(v.PieceRange),
			PieceSize: v.PieceSize,
			PieceMd5:  v.PieceMD5,
			Cid:       cid,
			PeerIP:    v.PeerIP,
			PeerPort:  int(v.PeerPort),
			Path:      v.Path,
		})
	}
	return EncodeResponse(rw, http.StatusOK, &types.ResultInfo{
		Code: constants.CodePeerContinue,
		Data: datas,
	})
}

func (s *Server) reportPiece(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	params := req.URL.Query()
	taskID := params.Get("taskId")
	srcCID := params.Get("cid")
	dstCID := params.Get("dstCid")
	pieceRange := params.Get("pieceRange")

	dstDfgetTask, err := s.DfgetTaskMgr.Get(ctx, dstCID, taskID)
	if err != nil {
		return err
	}

	// If piece is downloaded from supernode, add metrics.
	if s.Config.IsSuperCID(dstCID) {
		m.pieceDownloadedBytes.WithLabelValues().Add(float64(rangeutils.CalculatePieceSize(pieceRange)))
	}

	request := &types.PieceUpdateRequest{
		ClientID:    srcCID,
		DstPID:      dstDfgetTask.PeerID,
		PieceStatus: types.PieceUpdateRequestPieceStatusSUCCESS,
	}

	if err := s.TaskMgr.UpdatePieceStatus(ctx, taskID, pieceRange, request); err != nil {
		logrus.Errorf("failed to update pieces status %+v: %v", request, err)
		return err
	}

	return EncodeResponse(rw, http.StatusOK, &types.ResultInfo{
		Code: constants.CodeGetPieceReport,
	})
}

func (s *Server) reportServiceDown(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	params := req.URL.Query()
	taskID := params.Get("taskId")
	cID := params.Get("cid")

	// get peerID according to the CID and taskID
	dfgetTask, err := s.DfgetTaskMgr.Get(ctx, cID, taskID)
	if err != nil {
		return err
	}
	if err := s.ProgressMgr.UpdatePeerServiceDown(ctx, dfgetTask.PeerID); err != nil {
		return err
	}

	return EncodeResponse(rw, http.StatusOK, &types.ResultInfo{
		Code: constants.CodeGetPeerDown,
	})
}

func (s *Server) reportPieceError(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	logrus.Warnf("get report piece error request %v", req)

	params := req.URL.Query()
	taskID := params.Get("taskId")
	pieceRange := params.Get("range")
	srcCid := params.Get("srcCid")
	dstCid := params.Get("dstCid")
	dstIP := params.Get("dstIp")
	realMd5 := params.Get("realMd5")
	expectedMd5 := params.Get("expectedMd5")
	errorType := params.Get("errorType")

	// get peerID according to the CID and taskID
	dstDfgetTask, err := s.DfgetTaskMgr.Get(ctx, dstCid, taskID)
	if err != nil {
		return nil
	}

	request := &types.PieceErrorRequest{
		DstIP:       dstIP,
		DstPid:      dstDfgetTask.PeerID,
		ErrorType:   errorType,
		Range:       pieceRange,
		ExpectedMd5: expectedMd5,
		RealMd5:     realMd5,
		SrcCid:      srcCid,
		TaskID:      taskID,
	}

	if stringutils.IsEmptyStr(request.DstPid) {
		return errors.Wrap(errortypes.ErrEmptyValue, "dstPid")
	}

	if err := s.PieceErrorMgr.HandlePieceError(ctx, request); err != nil {
		return err
	}

	rw.WriteHeader(http.StatusOK)
	return nil
}

func (s *Server) fetchP2PNetworkInfo(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	return EncodeResponse(rw, http.StatusOK, &types.NetworkInfoFetchResponse{})
}

func (s *Server) reportPeerHealth(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	return EncodeResponse(rw, http.StatusOK, &types.HeartBeatResponse{})
}
