package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dragonflyoss/Dragonfly/dfget/core/api"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/common/constants"
	errTypes "github.com/dragonflyoss/Dragonfly/common/errors"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	dfType "github.com/dragonflyoss/Dragonfly/dfget/types"
	sutil "github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RegisterResponseData is the data when registering supernode successfully.
type RegisterResponseData struct {
	TaskID     string `json:"taskId"`
	FileLength int64  `json:"fileLength"`
	PieceSize  int32  `json:"pieceSize"`
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

//TODO(yunfeiyangbuaa):all the --copy func is just a test

func sendRequestCopy(req *types.TaskRegisterRequest, nodes string) {
	var API api.SupernodeAPI
	reqcopy := &dfType.RegisterRequest{
		RawURL:   req.RawURL,
		TaskURL:  req.TaskURL,
		Cid:      req.CID,
		IP:       req.IP.String(),
		HostName: req.HostName,
		Port:     int(req.Port),
		Path:     req.Path,
		Version:  req.Version,
		//CallSystem: cfg.CallSystem,
		Headers:  req.Headers,
		Dfdaemon: req.Dfdaemon,
	}
	API = api.NewSupernodeAPI()
	API.Register(nodes, reqcopy)
}
func (s *Server) registry(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	reader := req.Body
	request := &types.TaskRegisterRequest{}
	if err := json.NewDecoder(reader).Decode(request); err != nil {
		return errors.Wrap(errTypes.ErrInvalidValue, err.Error())
	}

	if err := request.Validate(strfmt.NewFormats()); err != nil {
		return errors.Wrap(errTypes.ErrInvalidValue, err.Error())
	}
	fmt.Println("register-req:", request)
	if s.HaMgr.GetSupernodeStatus() == constants.SupernodeUseHaActive {
		sendRequestCopy(request, "127.0.0.1:8003")
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
		return errors.Wrapf(errTypes.ErrSystemError, "failed to register peer: %v", err)
	}
	logrus.Infof("success to register peer %+v", peerCreateRequest)

	peerID := peerCreateResponse.ID
	taskCreateRequest := &types.TaskCreateRequest{
		CID:        request.CID,
		Dfdaemon:   request.Dfdaemon,
		Headers:    cutil.ConvertHeaders(request.Headers),
		Identifier: request.Identifier,
		Md5:        request.Md5,
		Path:       request.Path,
		PeerID:     peerID,
		RawURL:     request.RawURL,
		TaskURL:    request.TaskURL,
	}
	resp, err := s.TaskMgr.Register(ctx, taskCreateRequest)
	if err != nil {
		logrus.Errorf("failed to register task %+v: %v", taskCreateRequest, err)
		return err
	}
	logrus.Infof("success to register task %+v", taskCreateRequest)
	return EncodeResponse(rw, http.StatusOK, &types.ResultInfo{
		Code: constants.Success,
		Msg:  constants.GetMsgByCode(constants.Success),
		Data: &RegisterResponseData{
			TaskID:     resp.ID,
			FileLength: resp.FileLength,
			PieceSize:  resp.PieceSize,
		},
	})
}

func sendPullPieceTaskCopy(req *http.Request, nodes string) {
	var API api.SupernodeAPI
	params := req.URL.Query()
	result, _ := strconv.Atoi(params.Get("result"))
	status, _ := strconv.Atoi(params.Get("status"))
	reqcopy := &dfType.PullPieceTaskRequest{
		SrcCid: params.Get("srcCid"),
		DstCid: params.Get("dstCid"),
		Range:  params.Get("range"),
		Result: result,
		Status: status,
		TaskID: params.Get("taskId"),
	}
	API = api.NewSupernodeAPI()
	API.PullPieceTask(nodes, reqcopy)
}

func (s *Server) pullPieceTask(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	params := req.URL.Query()
	taskID := params.Get("taskId")
	srcCID := params.Get("srcCid")

	fmt.Println("pullpiece-req:", params)
	request := &types.PiecePullRequest{
		DfgetTaskStatus: statusMap[params.Get("status")],
		PieceRange:      params.Get("range"),
		PieceResult:     resultMap[params.Get("result")],
	}
	if s.HaMgr.GetSupernodeStatus() == constants.SupernodeUseHaActive {
		sendPullPieceTaskCopy(req, "127.0.0.1:8003")
	}
	// try to get dstPID
	dstCID := params.Get("dstCid")
	if !cutil.IsEmptyStr(dstCID) {
		dstDfgetTask, err := s.DfgetTaskMgr.Get(ctx, dstCID, taskID)
		if err != nil {
			return err
		}
		request.DstPID = dstDfgetTask.PeerID
	}

	isFinished, data, err := s.TaskMgr.GetPieces(ctx, taskID, srcCID, request)
	if err != nil {
		if errTypes.IsCDNFail(err) {
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
			PieceNum:  sutil.CalculatePieceNum(v.PieceRange),
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

func copyReportPieceRequestCopy(req *http.Request, nodes string) {
	var API api.SupernodeAPI
	params := req.URL.Query()
	reqcopy := &dfType.ReportPieceRequest{
		Cid:        params.Get("cid"),
		DstCid:     params.Get("dstCid"),
		PieceRange: params.Get("pieceRange"),
		TaskID:     params.Get("taskId"),
	}
	API = api.NewSupernodeAPI()
	API.ReportPiece(nodes, reqcopy)
}

func (s *Server) reportPiece(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	params := req.URL.Query()
	taskID := params.Get("taskId")
	srcCID := params.Get("cid")
	dstCID := params.Get("dstCid")
	pieceRange := params.Get("pieceRange")
	fmt.Println("report-req:", params)
	dstDfgetTask, err := s.DfgetTaskMgr.Get(ctx, dstCID, taskID)
	if err != nil {
		return err
	}
	if s.HaMgr.GetSupernodeStatus() == constants.SupernodeUseHaActive {
		copyReportPieceRequestCopy(req, "127.0.0.1:8003")
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

	rw.WriteHeader(http.StatusOK)
	return nil
}

func reportServiceDownCopy(req *http.Request, nodes string) {
	var API api.SupernodeAPI
	params := req.URL.Query()
	API = api.NewSupernodeAPI()
	API.ServiceDown(nodes, params.Get("taskId"), params.Get("cid"))
}

func (s *Server) reportServiceDown(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	params := req.URL.Query()
	taskID := params.Get("taskId")
	cID := params.Get("cid")
	if s.HaMgr.GetSupernodeStatus() == constants.SupernodeUseHaActive {
		reportServiceDownCopy(req, "127.0.0.1:8003")
	}
	fmt.Println("servicedown-req:", params)
	dfgetTask, err := s.DfgetTaskMgr.Get(ctx, cID, taskID)
	if err != nil {
		return err
	}

	if err := s.ProgressMgr.DeletePieceProgressByCID(ctx, taskID, cID); err != nil {
		return err
	}

	if err := s.ProgressMgr.DeletePeerStateByPeerID(ctx, dfgetTask.PeerID); err != nil {
		return err
	}

	if err := s.PeerMgr.DeRegister(ctx, dfgetTask.PeerID); err != nil {
		return err
	}

	if err := s.DfgetTaskMgr.Delete(ctx, cID, taskID); err != nil {
		return err
	}
	rw.WriteHeader(http.StatusOK)
	return nil
}
