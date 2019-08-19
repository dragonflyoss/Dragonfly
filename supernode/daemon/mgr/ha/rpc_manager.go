package ha

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/rpc"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"

	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RPCManager is the struct of rpc task
type RPCManager struct {
	cfg          *config.Config
	CdnMgr       mgr.CDNMgr
	DfgetTaskMgr mgr.DfgetTaskMgr
	ProgressMgr  mgr.ProgressMgr
	TaskMgr      mgr.TaskMgr
	PeerMgr      mgr.PeerMgr
}

// RPCReportPieceRequest is the report piece rpc request
type RPCReportPieceRequest struct {
	TaskID      string
	PieceNum    int
	Md5         string
	PieceStatus int
	CID         string
	SrcPID      string
	DstCID      string
}

// RPCUpdateTaskInfoRequest is the dateTaskInfo rpc request
type RPCUpdateTaskInfoRequest struct {
	CdnStatus  string
	FileLength int64
	RealMd5    string
	TaskID     string
	CDNPeerID  string
}

// RPCGetPieceRequest is the getpiece rpc request
type RPCGetPieceRequest struct {
	DfgetTaskStatus string
	PieceRange      string
	PieceResult     string
	TaskID          string
	Cid             string
	DstCID          string
}

// RPCGetPieceResponse is the get piece rpc response
type RPCGetPieceResponse struct {
	IsFinished bool
	Data       []*types.PieceInfo
	ErrCode    int
	ErrMsg     string
}

// RPCServerDownRequest is the server down rpc request
type RPCServerDownRequest struct {
	TaskID string
	CID    string
}

// RPCAddSupernodeWatchRequest is the supernode task info update change add notify rpc request
type RPCAddSupernodeWatchRequest struct {
	TaskID       string
	SupernodePID string
}

// NewRPCMgr produces a RPCManager
func NewRPCMgr(cfg *config.Config, CdnMgr mgr.CDNMgr, DfgetTaskMgr mgr.DfgetTaskMgr,
	ProgressMgr mgr.ProgressMgr, TaskMgr mgr.TaskMgr, PeerMgr mgr.PeerMgr) *RPCManager {
	rpcMgr := &RPCManager{
		CdnMgr:       CdnMgr,
		ProgressMgr:  ProgressMgr,
		DfgetTaskMgr: DfgetTaskMgr,
		TaskMgr:      TaskMgr,
		PeerMgr:      PeerMgr,
		cfg:          cfg,
	}
	return rpcMgr
}

// StartRPCServer starts rpc server
func StartRPCServer(cfg *config.Config, CdnMgr mgr.CDNMgr, DfgetTaskMgr mgr.DfgetTaskMgr, ProgressMgr mgr.ProgressMgr,
	TaskMgr mgr.TaskMgr, PeerMgr mgr.PeerMgr) error {
	rpc.Register(NewRPCMgr(cfg, CdnMgr, DfgetTaskMgr, ProgressMgr, TaskMgr, PeerMgr))
	rpc.HandleHTTP()
	rpcAddress := fmt.Sprintf("%s:%d", cfg.AdvertiseIP, cfg.HARpcPort)
	lis, err := net.Listen("tcp", rpcAddress)
	if err != nil {
		logrus.Errorf("failed to start a rpc server,err %v", err)
		return err
	}
	go http.Serve(lis, nil)
	return nil
}

// RPCUpdateProgress updates progress
func (rpc *RPCManager) RPCUpdateProgress(req RPCReportPieceRequest, res *bool) error {
	dstDfgetTask, err := rpc.DfgetTaskMgr.Get(context.TODO(), req.DstCID, req.TaskID)
	if err != nil {
		logrus.Errorf("failed to get dfgetTask by cid(%s) and taskID(%s) from rpc request,err: %v", req.DstCID, req.TaskID, err)
		return err
	}
	return rpc.ProgressMgr.UpdateProgress(context.TODO(), req.TaskID, req.CID, req.SrcPID, dstDfgetTask.PeerID, req.PieceNum, req.PieceStatus)
}

// RPCGetTaskInfo get task info according a req
func (rpc *RPCManager) RPCGetTaskInfo(task string, resp *RPCUpdateTaskInfoRequest) error {
	taskInfo, err := rpc.TaskMgr.Get(context.TODO(), task)
	if err != nil {
		logrus.Errorf("failed to get Task by task(%s) from rpc request,err: %v", task, err)
		return err
	}
	resp.CdnStatus = taskInfo.CdnStatus
	resp.FileLength = taskInfo.FileLength
	resp.RealMd5 = taskInfo.RealMd5
	resp.CDNPeerID = taskInfo.CDNPeerID
	resp.TaskID = taskInfo.ID
	return nil
}

// RPCGetPiece gets pieces according req
func (rpc *RPCManager) RPCGetPiece(req RPCGetPieceRequest, resp *RPCGetPieceResponse) error {
	piecePullRequest := &types.PiecePullRequest{
		DfgetTaskStatus: req.DfgetTaskStatus,
		PieceRange:      req.PieceRange,
		PieceResult:     req.PieceResult,
		DstCid:          req.DstCID,
	}
	if !stringutils.IsEmptyStr(req.DstCID) {
		dstDfgetTask, err := rpc.DfgetTaskMgr.Get(context.TODO(), req.Cid, req.TaskID)
		if err != nil {
			logrus.Warnf("failed to get dfget task by dstCID(%s) and taskID(%s) from rpc request, and the srcCID is %s, err: %v",
				req.DstCID, req.TaskID, req.Cid, err)
		} else {
			piecePullRequest.DstPID = dstDfgetTask.PeerID
		}
	}
	isFinished, data, err := rpc.TaskMgr.GetPieces(context.TODO(), req.TaskID, req.Cid, piecePullRequest)
	if err != nil {
		e, ok := errors.Cause(err).(errortypes.DfError)
		if ok {
			resp.ErrCode = e.Code
			resp.ErrMsg = e.Msg
		}
	} else {
		pieceInfos, _ := data.([]*types.PieceInfo)
		resp.Data = pieceInfos
		resp.IsFinished = isFinished
	}
	return nil
}

// RPCDfgetServerDown report dfserver if off
func (rpc *RPCManager) RPCDfgetServerDown(request RPCServerDownRequest, resp *bool) error {
	dfgetTask, err := rpc.DfgetTaskMgr.Get(context.TODO(), request.CID, request.TaskID)
	if err != nil {
		logrus.Errorf("failed to get dfgetTask by cid(%s) and taskID(%s) from rpc request,err: %v", request.CID, request.TaskID, err)
		return err
	}
	if err := rpc.ProgressMgr.UpdatePeerServiceDown(context.TODO(), dfgetTask.PeerID); err != nil {
		return err
	}
	return nil
}

// RPCOnlyTriggerCDNDownload trigger a cdn download
func (rpc *RPCManager) RPCOnlyTriggerCDNDownload(req types.TaskRegisterRequest, resp *bool) error {
	if err := req.Validate(strfmt.NewFormats()); err != nil {
		return errors.Wrap(errortypes.ErrInvalidValue, err.Error())
	}
	taskCreateRequest := &types.TaskCreateRequest{
		CID:         req.CID,
		CallSystem:  req.CallSystem,
		Dfdaemon:    req.Dfdaemon,
		Headers:     netutils.ConvertHeaders(req.Headers),
		Identifier:  req.Identifier,
		Md5:         req.Md5,
		Path:        req.Path,
		PeerID:      "",
		RawURL:      req.RawURL,
		TaskURL:     req.TaskURL,
		SupernodeIP: req.SuperNodeIP,
	}
	if err := rpc.TaskMgr.OnlyTriggerDownload(context.TODO(), taskCreateRequest, &req); err != nil {
		logrus.Errorf("failed to trigger CDN download by rpc,err: %v", err)
		return err
	}
	return nil
}

// RPCUpdateTaskInfo uodate supernode's task info via rpc
func (rpc *RPCManager) RPCUpdateTaskInfo(req RPCUpdateTaskInfoRequest, resp *bool) error {
	var (
		updateTask *types.TaskInfo
		err        error
	)
	updateTask = &types.TaskInfo{
		CdnStatus:  req.CdnStatus,
		FileLength: req.FileLength,
		RealMd5:    req.RealMd5,
	}

	if err = rpc.TaskMgr.Update(context.TODO(), req.TaskID, updateTask); err != nil {
		logrus.Errorf("failed to update task %v via rpc,err: %v", updateTask, req)
		return err
	}
	logrus.Debugf("success to update task cdn via rpc %+v", updateTask)
	return nil
}

// RPCAddSupernodeWatch register a watch to other supernode,if other supernode's task update,it can be notified
func (rpc *RPCManager) RPCAddSupernodeWatch(req RPCAddSupernodeWatchRequest, resp *bool) error {
	task, err := rpc.TaskMgr.Get(context.TODO(), req.TaskID)
	if err != nil {
		logrus.Errorf("failed to get Task by task(%s) from rpc request,err: %v", req.TaskID, err)
		return err
	}
	task.NotifySupernodesPID = append(task.NotifySupernodesPID, req.SupernodePID)
	return nil
}
