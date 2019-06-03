package task

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	errorType "github.com/dragonflyoss/Dragonfly/common/errors"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// addOrUpdateTask adds a new task or update the exist task to taskStore.
func (tm *Manager) addOrUpdateTask(ctx context.Context, req *types.TaskCreateRequest) (*types.TaskInfo, error) {
	taskURL := req.TaskURL
	if cutil.IsEmptyStr(req.TaskURL) {
		taskURL = cutil.FilterURLParam(req.RawURL, req.Filter)
	}
	taskID := generateTaskID(taskURL, req.Md5, req.Identifier)

	// using the existing task if it already exists corresponding to taskID
	var task *types.TaskInfo
	newTask := &types.TaskInfo{
		ID:         taskID,
		CallSystem: req.CallSystem,
		Dfdaemon:   req.Dfdaemon,
		Headers:    req.Headers,
		Identifier: req.Identifier,
		Md5:        req.Md5,
		RawURL:     req.RawURL,
		TaskURL:    taskURL,
		CdnStatus:  types.TaskInfoCdnStatusWAITING,
		PieceTotal: -1,
	}

	if v, err := tm.taskStore.Get(taskID); err == nil {
		task = v.(*types.TaskInfo)
		if !equalsTask(task, newTask) {
			return nil, errors.Wrapf(errorType.ErrTaskIDDuplicate, "%s", taskID)
		}
	} else {
		task = newTask
	}

	tm.taskLocker.GetLock(taskID, false)
	defer tm.taskLocker.ReleaseLock(taskID, false)

	if task.FileLength != 0 {
		return task, nil
	}

	// get fileLength with req.Headers
	fileLength, err := getHTTPFileLength(taskID, task.TaskURL, req.Headers)
	if err != nil {
		logrus.Errorf("failed to get file length from http client for taskID(%s): %v", taskID, err)
	}
	task.HTTPFileLength = fileLength
	logrus.Infof("get file length %d from http client for taskID(%s)", fileLength, taskID)

	// if success to get the information successfully with the req.Headers,
	// and then update the task.Headers to req.Headers.
	if !cutil.IsNil(req.Headers) {
		task.Headers = req.Headers
	}

	// caculate piece size and update the PieceSize and PieceTotal
	pieceSize := computePieceSize(fileLength)
	task.PieceSize = pieceSize
	task.PieceTotal = int32((fileLength + (int64(pieceSize) - 1)) / int64(pieceSize))

	tm.taskStore.Put(taskID, task)
	return task, nil
}

// getTask returns the taskInfo according to the specified taskID.
func (tm *Manager) getTask(taskID string) (*types.TaskInfo, error) {
	if cutil.IsEmptyStr(taskID) {
		return nil, errors.Wrap(errorType.ErrEmptyValue, "taskID")
	}

	v, err := tm.taskStore.Get(taskID)
	if err != nil {
		return nil, err
	}

	// type assertion
	if info, ok := v.(*types.TaskInfo); ok {
		return info, nil
	}
	return nil, errors.Wrapf(errorType.ErrConvertFailed, "taskID %s: %v", taskID, v)
}

func (tm *Manager) updateTask(taskID string, updateTaskInfo *types.TaskInfo) error {
	if cutil.IsEmptyStr(taskID) {
		return errors.Wrap(errorType.ErrEmptyValue, "taskID")
	}

	if cutil.IsNil(updateTaskInfo) {
		return errors.Wrap(errorType.ErrEmptyValue, "Update TaskInfo")
	}

	// the expected new CDNStatus is not nil
	if cutil.IsEmptyStr(updateTaskInfo.CdnStatus) {
		return errors.Wrapf(errorType.ErrEmptyValue, "CDNStatus of TaskInfo: %+v", updateTaskInfo)
	}

	tm.taskLocker.GetLock(taskID, false)
	defer tm.taskLocker.ReleaseLock(taskID, false)

	task, err := tm.getTask(taskID)
	if err != nil {
		return err
	}

	if !isSuccessCDN(updateTaskInfo.CdnStatus) {
		// when the origin CDNStatus equals success, do not update it to unsuccessful
		if isSuccessCDN(task.CdnStatus) {
			return nil
		}

		// only update the task CdnStatus when the new CDNStatus and
		// the origin CDNStatus both not equals success
		task.CdnStatus = updateTaskInfo.CdnStatus
		return nil
	}

	// only update the task info when the new CDNStatus equals success
	// and the origin CDNStatus not equals success.
	if updateTaskInfo.FileLength != 0 {
		task.FileLength = updateTaskInfo.FileLength
	}

	if !cutil.IsEmptyStr(updateTaskInfo.RealMd5) {
		task.RealMd5 = updateTaskInfo.RealMd5
	}

	var pieceTotal int32
	if updateTaskInfo.FileLength > 0 {
		pieceTotal = int32((updateTaskInfo.FileLength + int64(task.PieceSize-1)) / int64(task.PieceSize))
	}
	if pieceTotal != 0 {
		task.PieceTotal = pieceTotal
	}
	task.CdnStatus = updateTaskInfo.CdnStatus

	return nil
}

func (tm *Manager) addDfgetTask(ctx context.Context, req *types.TaskCreateRequest, task *types.TaskInfo) (*types.DfGetTask, error) {
	dfgetTask := &types.DfGetTask{
		CID:       req.CID,
		Path:      req.Path,
		PieceSize: task.PieceSize,
		Status:    types.DfGetTaskStatusWAITING,
		TaskID:    task.ID,
		PeerID:    req.PeerID,
	}

	if err := tm.dfgetTaskMgr.Add(ctx, dfgetTask); err != nil {
		return dfgetTask, err
	}

	return dfgetTask, nil
}

func (tm *Manager) triggerCdnSyncAction(ctx context.Context, task *types.TaskInfo) error {
	if !isFrozen(task.CdnStatus) {
		logrus.Infof("CDN(%s) is running or has been downloaded successfully for taskID: %s", task.CdnStatus, task.ID)
		return nil
	}

	if isWait(task.CdnStatus) {
		if err := tm.initCdnNode(ctx, task); err != nil {
			logrus.Errorf("failed to init cdn node for taskID %s: %v", task.ID, err)
			return err
		}
		logrus.Infof("success to init cdn node or taskID %s", task.ID)
	}
	if err := tm.updateTask(task.ID, &types.TaskInfo{
		CdnStatus: types.TaskInfoCdnStatusRUNNING,
	}); err != nil {
		return err
	}

	go func() {
		updateTaskInfo, err := tm.cdnMgr.TriggerCDN(ctx, task)
		if err != nil {
			logrus.Errorf("trigger cdn get error: %v", err)
		}
		tm.updateTask(task.ID, updateTaskInfo)
		logrus.Infof("success to update task cdn %+v", updateTaskInfo)
	}()
	logrus.Infof("success to start cdn trigger for taskID: %s", task.ID)
	return nil
}

func (tm *Manager) initCdnNode(ctx context.Context, task *types.TaskInfo) error {
	var cid = tm.cfg.GetSuperCID(task.ID)
	var pid = tm.cfg.GetSuperPID()
	path, err := tm.cdnMgr.GetHTTPPath(ctx, task.ID)
	if err != nil {
		return err
	}

	if err := tm.dfgetTaskMgr.Add(ctx, &types.DfGetTask{
		CID:       cid,
		Path:      path,
		PeerID:    pid,
		PieceSize: task.PieceSize,
		Status:    types.DfGetTaskStatusWAITING,
		TaskID:    task.ID,
	}); err != nil {
		return errors.Wrapf(err, "failed to add cdn dfgetTask for taskID %s", task.ID)
	}

	return tm.progressMgr.InitProgress(ctx, task.ID, pid, cid)
}

func (tm *Manager) processTaskStart(ctx context.Context, srcCID string, task *types.TaskInfo, dfgetTask *types.DfGetTask) (bool, interface{}, error) {
	if err := tm.dfgetTaskMgr.UpdateStatus(ctx, srcCID, task.ID, types.DfGetTaskStatusRUNNING); err != nil {
		return false, nil, err
	}
	logrus.Infof("success update dfgetTask status to RUNNING with taskID: %s clientID: %s", task.ID, srcCID)

	return tm.parseAvaliablePeers(ctx, srcCID, task, dfgetTask)
}

// req.DstPID, req.PieceRange, req.PieceResult, req.DfgetTaskStatus
func (tm *Manager) processTaskRunning(ctx context.Context, srcCID, srcPID string, task *types.TaskInfo, req *types.PiecePullRequest,
	dfgetTask *types.DfGetTask) (bool, interface{}, error) {
	pieceNum := util.CalculatePieceNum(req.PieceRange)
	if pieceNum == -1 {
		return false, nil, errors.Wrapf(errorType.ErrInvalidValue, "pieceRange: %s", req.PieceRange)
	}
	pieceStatus := convertToPeerPieceStatus(req.PieceResult, req.DfgetTaskStatus)
	if pieceStatus == -1 {
		return false, nil, errors.Wrapf(errorType.ErrInvalidValue, "failed to convert result: %s and status %s to pieceStatus", req.PieceResult, req.DfgetTaskStatus)
	}

	logrus.Infof("start to update progress taskID (%s) srcCID (%s) srcPID (%s) dstPID (%s) pieceNum (%d) pieceStatus (%d)",
		task.ID, srcCID, srcPID, req.DstPID, pieceNum, pieceStatus)
	if err := tm.progressMgr.UpdateProgress(ctx, task.ID, srcCID, srcPID, req.DstPID, pieceNum, pieceStatus); err != nil {
		return false, nil, errors.Wrap(err, "failed to update progress")
	}

	return tm.parseAvaliablePeers(ctx, srcCID, task, dfgetTask)
}

func (tm *Manager) processTaskFinish(ctx context.Context, taskID, clientID, dfgetTaskStatus string) error {
	if err := tm.dfgetTaskMgr.UpdateStatus(ctx, clientID, taskID, dfgetTaskStatus); err != nil {
		return fmt.Errorf("failed to update dfget task status with taskID(%s) clientID(%s) status(%s): %v", taskID, clientID, dfgetTaskStatus, err)
	}

	return nil
}

func (tm *Manager) parseAvaliablePeers(ctx context.Context, clientID string, task *types.TaskInfo, dfgetTask *types.DfGetTask) (bool, interface{}, error) {
	// Step1. validate
	if cutil.IsEmptyStr(clientID) {
		return false, nil, errors.Wrapf(errorType.ErrEmptyValue, "clientID")
	}

	// Step2. validate cdn status
	if task.CdnStatus == types.TaskInfoCdnStatusFAILED {
		return false, nil, errors.Wrapf(errorType.ErrCDNFail, "taskID: %s", task.ID)
	}
	if task.CdnStatus == types.TaskInfoCdnStatusWAITING {
		return false, nil, errors.Wrapf(errorType.ErrPeerWait, "taskID: %s cdn status is waiting", task.ID)
	}

	// Step3. whether success
	cdnSuccess := (task.CdnStatus == types.TaskInfoCdnStatusSUCCESS)
	pieceSuccess, _ := tm.progressMgr.GetPieceProgressByCID(ctx, task.ID, clientID, "success")
	logrus.Infof("get successful pieces: %v", pieceSuccess)
	if cdnSuccess && (int32(len(pieceSuccess)) == task.PieceTotal) {
		finishInfo := make(map[string]interface{})
		finishInfo["md5"] = task.Md5
		finishInfo["fileLength"] = task.FileLength
		return true, finishInfo, nil
	}

	// get scheduler pieceResult
	logrus.Infof("start scheduler for taskID: %s clientID: %s", task.ID, clientID)
	pieceResult, err := tm.schedulerMgr.Schedule(ctx, task.ID, clientID, dfgetTask.PeerID)
	if err != nil {
		return false, nil, err
	}
	logrus.Infof("get scheduler result length(%d) with taskID(%s) and clientID(%s)", len(pieceResult), task.ID, clientID)

	var pieceInfos []*types.PieceInfo
	for _, v := range pieceResult {
		logrus.Debugf("get scheduler result item: %+v with taskID(%s) and clientID(%s)", v, task.ID, clientID)
		pieceInfo, err := tm.pieceResultToPieceInfo(ctx, v, task.PieceSize)
		if err != nil {
			return false, nil, err
		}

		pieceInfos = append(pieceInfos, pieceInfo)
	}

	return false, pieceInfos, nil
}

func (tm *Manager) pieceResultToPieceInfo(ctx context.Context, pr *mgr.PieceResult, pieceSize int32) (*types.PieceInfo, error) {
	cid, err := tm.dfgetTaskMgr.GetCIDByPeerIDAndTaskID(ctx, pr.DstPID, pr.TaskID)
	if err != nil {
		return nil, err
	}
	dfgetTask, err := tm.dfgetTaskMgr.Get(ctx, cid, pr.TaskID)
	if err != nil {
		return nil, err
	}

	peer, err := tm.peerMgr.Get(ctx, pr.DstPID)
	if err != nil {
		return nil, err
	}

	return &types.PieceInfo{
		PID:        pr.DstPID,
		Path:       dfgetTask.Path,
		PeerIP:     peer.IP.String(),
		PeerPort:   peer.Port,
		PieceRange: util.CalculatePieceRange(pr.PieceNum, pieceSize),
		PieceSize:  pieceSize,
	}, nil
}

// convertToPeerPieceStatus convert piece result and dfgetTask status to dfgetTask status code.
// And it should return "" if failed to convert.
func convertToDfgetTaskStatus(result, status string) string {
	if status == types.PiecePullRequestDfgetTaskStatusSTARTED {
		return types.DfGetTaskStatusWAITING
	}

	if status == types.PiecePullRequestDfgetTaskStatusRUNNING {
		return types.DfGetTaskStatusRUNNING
	}

	if status == types.PiecePullRequestDfgetTaskStatusFINISHED {
		if result == types.PiecePullRequestPieceResultSUCCESS {
			return types.DfGetTaskStatusSUCCESS
		}
		return types.DfGetTaskStatusFAILED
	}

	return ""
}

// convertToPeerPieceStatus convert piece result and dfgetTask status to piece status code.
// And it should return -1 if failed to convert.
func convertToPeerPieceStatus(result, status string) int {
	if status == types.PiecePullRequestDfgetTaskStatusSTARTED {
		return config.PieceWAITING
	}

	if status == types.PiecePullRequestDfgetTaskStatusRUNNING {
		if result == types.PiecePullRequestPieceResultSUCCESS {
			return config.PieceSUCCESS
		}
		if result == types.PiecePullRequestPieceResultFAILED {
			return config.PieceFAILED
		}
		if result == types.PiecePullRequestPieceResultSEMISUC {
			return config.PieceSEMISUC
		}
	}

	return -1
}

// equalsTask determines that whether the two task objects are the same.
//
// The result is based only on whether the attributes used to generate taskID are the same
// which including taskURL, md5, identifier.
func equalsTask(existTask, newTask *types.TaskInfo) bool {
	if existTask.TaskURL != newTask.TaskURL {
		return false
	}

	if !cutil.IsEmptyStr(existTask.Md5) {
		return existTask.Md5 == newTask.Md5
	}

	return existTask.Identifier == newTask.Identifier
}

// validateParams validates the params of TaskCreateRequest.
func validateParams(req *types.TaskCreateRequest) error {
	if !cutil.IsValidURL(req.RawURL) {
		return errors.Wrapf(errorType.ErrInvalidValue, "raw url: %s", req.RawURL)
	}

	if cutil.IsEmptyStr(req.Path) {
		return errors.Wrapf(errorType.ErrEmptyValue, "path")
	}

	if cutil.IsEmptyStr(req.CID) {
		return errors.Wrapf(errorType.ErrEmptyValue, "cID")
	}

	if cutil.IsEmptyStr(req.PeerID) {
		return errors.Wrapf(errorType.ErrEmptyValue, "peerID")
	}

	return nil
}

// generateTaskID generates taskID with taskURL,md5 and identifier
// and returns the SHA-256 checksum of the data.
func generateTaskID(taskURL, md5, identifier string) string {
	sign := ""
	if !cutil.IsEmptyStr(md5) {
		sign = md5
	} else if !cutil.IsEmptyStr(identifier) {
		sign = identifier
	}
	id := fmt.Sprintf("%s%s%s%s", key, taskURL, sign, key)

	return cutil.Sha256(id)
}

// computePieceSize computes the piece size with specified fileLength.
//
// If the fileLength<=0, which means failed to get fileLength
// and then use the DefaultPieceSize.
func computePieceSize(length int64) int32 {
	if length <= 0 || length <= 200*1024*1024 {
		return config.DefaultPieceSize
	}

	gapCount := length / int64(100*1024*1024)
	tmpSize := (gapCount-2)*1024*1024 + config.DefaultPieceSize
	if tmpSize > config.DefaultPieceSizeLimit {
		return config.DefaultPieceSizeLimit
	}
	return int32(tmpSize)
}

// isSuccessCDN determines that whether the CDNStatus is success.
func isSuccessCDN(CDNStatus string) bool {
	return CDNStatus == types.TaskInfoCdnStatusSUCCESS
}

func isFrozen(CDNStatus string) bool {
	return CDNStatus == types.TaskInfoCdnStatusFAILED ||
		CDNStatus == types.TaskInfoCdnStatusWAITING ||
		CDNStatus == types.TaskInfoCdnStatusSOURCEERROR
}

func isWait(CDNStatus string) bool {
	return CDNStatus == types.TaskInfoCdnStatusWAITING
}

func getHTTPFileLength(taskID, url string, headers map[string]string) (int64, error) {
	fileLength, code, err := getContentLength(url, headers)
	if err != nil {
		return -1, err
	}

	if code == http.StatusUnauthorized || code == http.StatusProxyAuthRequired {
		return -1, errors.Wrapf(errorType.ErrAuthenticationRequired, "taskID: %s,code: %d", taskID, code)
	}
	if code != http.StatusOK {
		logrus.Warnf("failed to get http file length with unexpected code: %d", code)
		return -1, nil
	}

	return fileLength, nil
}
