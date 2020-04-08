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

package task

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/digest"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/rangeutils"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/pkg/timeutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// addOrUpdateTask adds a new task or update the exist task to taskStore.
func (tm *Manager) addOrUpdateTask(ctx context.Context, req *types.TaskCreateRequest, failAccessInterval time.Duration) (*types.TaskInfo, error) {
	taskURL := req.TaskURL
	if stringutils.IsEmptyStr(req.TaskURL) {
		taskURL = netutils.FilterURLParam(req.RawURL, req.Filter)
	}
	taskID := generateTaskID(taskURL, req.Md5, req.Identifier, req.Headers)

	util.GetLock(taskID, true)
	defer util.ReleaseLock(taskID, true)

	if key, err := tm.taskURLUnReachableStore.Get(taskID); err == nil {
		if unReachableStartTime, ok := key.(time.Time); ok &&
			time.Since(unReachableStartTime) < failAccessInterval {
			return nil, errors.Wrapf(errortypes.ErrURLNotReachable, "cache taskID: %s, url: %s", taskID, req.RawURL)
		}

		tm.taskURLUnReachableStore.Delete(taskID)
	}

	// using the existing task if it already exists corresponding to taskID
	var task *types.TaskInfo
	newTask := &types.TaskInfo{
		ID:         taskID,
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
			return nil, errors.Wrapf(errortypes.ErrTaskIDDuplicate, "%s", taskID)
		}
	} else {
		task = newTask
	}

	if task.HTTPFileLength != 0 {
		return task, nil
	}

	// get fileLength with req.Headers
	fileLength, err := tm.getHTTPFileLength(taskID, task.RawURL, req.Headers)
	if err != nil {
		logrus.Errorf("failed to get file length from http client for taskID(%s): %v", taskID, err)

		if errortypes.IsURLNotReachable(err) {
			tm.taskURLUnReachableStore.Add(taskID, time.Now())
			return nil, err
		}
		if errortypes.IsAuthenticationRequired(err) {
			return nil, err
		}
	}
	if tm.cfg.CDNPattern == config.CDNPatternSource {
		if fileLength <= 0 {
			return nil, fmt.Errorf("failed to get file length and it is required in source CDN pattern")
		}

		supportRange, err := tm.originClient.IsSupportRange(task.TaskURL, task.Headers)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check whether the task(%s) supports partial requests", task.ID)
		}
		if !supportRange {
			return nil, fmt.Errorf("the task URL should support range request in source CDN pattern: %s", taskID)
		}
	}
	task.HTTPFileLength = fileLength
	logrus.Infof("get file length %d from http client for taskID(%s)", fileLength, taskID)

	// if success to get the information successfully with the req.Headers,
	// and then update the task.Headers to req.Headers.
	if req.Headers != nil {
		task.Headers = req.Headers
	}

	// calculate piece size and update the PieceSize and PieceTotal
	pieceSize := computePieceSize(fileLength)
	task.PieceSize = pieceSize
	task.PieceTotal = int32((fileLength + (int64(pieceSize) - 1)) / int64(pieceSize))

	tm.taskStore.Put(taskID, task)
	tm.metrics.tasks.WithLabelValues(task.CdnStatus).Inc()
	return task, nil
}

// getTask returns the taskInfo according to the specified taskID.
func (tm *Manager) getTask(taskID string) (*types.TaskInfo, error) {
	if stringutils.IsEmptyStr(taskID) {
		return nil, errors.Wrap(errortypes.ErrEmptyValue, "taskID")
	}

	v, err := tm.taskStore.Get(taskID)
	if err != nil {
		return nil, err
	}

	// type assertion
	if info, ok := v.(*types.TaskInfo); ok {
		return info, nil
	}
	return nil, errors.Wrapf(errortypes.ErrConvertFailed, "taskID %s: %v", taskID, v)
}

func (tm *Manager) updateTask(taskID string, updateTaskInfo *types.TaskInfo) error {
	if stringutils.IsEmptyStr(taskID) {
		return errors.Wrap(errortypes.ErrEmptyValue, "taskID")
	}

	if updateTaskInfo == nil {
		return errors.Wrap(errortypes.ErrEmptyValue, "Update TaskInfo")
	}

	// the expected new CDNStatus is not nil
	if stringutils.IsEmptyStr(updateTaskInfo.CdnStatus) {
		return errors.Wrapf(errortypes.ErrEmptyValue, "CDNStatus of TaskInfo: %+v", updateTaskInfo)
	}

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
		tm.metrics.tasks.WithLabelValues(task.CdnStatus).Dec()
		tm.metrics.tasks.WithLabelValues(updateTaskInfo.CdnStatus).Inc()
		task.CdnStatus = updateTaskInfo.CdnStatus
		return nil
	}

	// only update the task info when the new CDNStatus equals success
	// and the origin CDNStatus not equals success.
	if updateTaskInfo.FileLength != 0 {
		task.FileLength = updateTaskInfo.FileLength
	}

	if !stringutils.IsEmptyStr(updateTaskInfo.RealMd5) {
		task.RealMd5 = updateTaskInfo.RealMd5
	}

	var pieceTotal int32
	if updateTaskInfo.FileLength > 0 {
		pieceTotal = int32((updateTaskInfo.FileLength + int64(task.PieceSize-1)) / int64(task.PieceSize))
	}
	if pieceTotal != 0 {
		task.PieceTotal = pieceTotal
	}
	tm.metrics.tasks.WithLabelValues(task.CdnStatus).Dec()
	tm.metrics.tasks.WithLabelValues(updateTaskInfo.CdnStatus).Inc()
	task.CdnStatus = updateTaskInfo.CdnStatus

	return nil
}

func (tm *Manager) addDfgetTask(ctx context.Context, req *types.TaskCreateRequest, task *types.TaskInfo) (*types.DfGetTask, error) {
	dfgetTask := &types.DfGetTask{
		CID:         req.CID,
		CallSystem:  req.CallSystem,
		Dfdaemon:    req.Dfdaemon,
		Path:        req.Path,
		PieceSize:   task.PieceSize,
		Status:      types.DfGetTaskStatusWAITING,
		TaskID:      task.ID,
		PeerID:      req.PeerID,
		SupernodeIP: req.SupernodeIP,
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
		tm.metrics.triggerCdnCount.WithLabelValues().Inc()
		if err != nil {
			tm.metrics.triggerCdnFailCount.WithLabelValues().Inc()
			logrus.Errorf("taskID(%s) trigger cdn get error: %v", task.ID, err)
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
	path, err := tm.cdnMgr.GetHTTPPath(ctx, task)
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

	return tm.parseAvailablePeers(ctx, srcCID, task, dfgetTask)
}

// req.DstPID, req.PieceRange, req.PieceResult, req.DfgetTaskStatus
func (tm *Manager) processTaskRunning(ctx context.Context, srcCID, srcPID string, task *types.TaskInfo, req *types.PiecePullRequest,
	dfgetTask *types.DfGetTask) (bool, interface{}, error) {
	pieceNum := rangeutils.CalculatePieceNum(req.PieceRange)
	if pieceNum == -1 {
		return false, nil, errors.Wrapf(errortypes.ErrInvalidValue, "pieceRange: %s", req.PieceRange)
	}
	pieceStatus, success := convertToPeerPieceStatus(req.PieceResult, req.DfgetTaskStatus)
	if !success {
		return false, nil, errors.Wrapf(errortypes.ErrInvalidValue, "failed to convert result: %s and status %s to pieceStatus", req.PieceResult, req.DfgetTaskStatus)
	}

	logrus.Debugf("start to update progress taskID (%s) srcCID (%s) srcPID (%s) dstPID (%s) pieceNum (%d) pieceStatus (%d)",
		task.ID, srcCID, srcPID, req.DstPID, pieceNum, pieceStatus)
	if err := tm.progressMgr.UpdateProgress(ctx, task.ID, srcCID, srcPID, req.DstPID, pieceNum, pieceStatus); err != nil {
		return false, nil, errors.Wrap(err, "failed to update progress")
	}

	return tm.parseAvailablePeers(ctx, srcCID, task, dfgetTask)
}

func (tm *Manager) processTaskFinish(ctx context.Context, taskID, clientID, dfgetTaskStatus string) error {
	if err := tm.dfgetTaskMgr.UpdateStatus(ctx, clientID, taskID, dfgetTaskStatus); err != nil {
		return fmt.Errorf("failed to update dfget task status with taskID(%s) clientID(%s) status(%s): %v", taskID, clientID, dfgetTaskStatus, err)
	}

	return nil
}

func (tm *Manager) parseAvailablePeers(ctx context.Context, clientID string, task *types.TaskInfo, dfgetTask *types.DfGetTask) (bool, interface{}, error) {
	// Step1. validate
	if stringutils.IsEmptyStr(clientID) {
		return false, nil, errors.Wrapf(errortypes.ErrEmptyValue, "clientID")
	}

	// Step2. validate cdn status
	if task.CdnStatus == types.TaskInfoCdnStatusFAILED {
		return false, nil, errors.Wrapf(errortypes.ErrCDNFail, "taskID: %s", task.ID)
	}
	if task.CdnStatus == types.TaskInfoCdnStatusWAITING {
		return false, nil, errors.Wrapf(errortypes.ErrPeerWait, "taskID: %s cdn status is waiting", task.ID)
	}

	// Step3. whether success
	cdnSuccess := task.CdnStatus == types.TaskInfoCdnStatusSUCCESS
	pieceSuccess, _ := tm.progressMgr.GetPieceProgressByCID(ctx, task.ID, clientID, "success")
	logrus.Debugf("taskID(%s) clientID(%s) get successful pieces: %v", task.ID, clientID, pieceSuccess)
	if cdnSuccess && (task.PieceTotal != 0 && (int32(len(pieceSuccess)) == task.PieceTotal)) {
		// update dfget task status to success
		if err := tm.dfgetTaskMgr.UpdateStatus(ctx, clientID, task.ID, types.DfGetTaskStatusSUCCESS); err != nil {
			logrus.Errorf("failed to update dfget task status with "+
				"taskID(%s) clientID(%s) status(%s): %v", task.ID, clientID, types.DfGetTaskStatusSUCCESS, err)
		}
		finishInfo := make(map[string]interface{})
		finishInfo["md5"] = task.RealMd5
		finishInfo["fileLength"] = task.FileLength
		return true, finishInfo, nil
	}

	// get scheduler pieceResult
	logrus.Debugf("start scheduler for taskID: %s clientID: %s", task.ID, clientID)
	startTime := time.Now()
	pieceResult, err := tm.schedulerMgr.Schedule(ctx, task.ID, clientID, dfgetTask.PeerID)
	if err != nil {
		return false, nil, err
	}
	timeCost := timeutils.SinceInMilliseconds(startTime)
	// Get peerName to represent peer in metrics.
	if peer, err := tm.peerMgr.Get(context.Background(), dfgetTask.PeerID); err == nil {
		tm.metrics.scheduleDurationMilliSeconds.WithLabelValues(peer.IP.String()).Observe(timeCost)
	} else {
		logrus.Warnf("failed to get peer with peerId(%s) taskId(%s): %v",
			dfgetTask.PeerID, task.ID, err)
	}
	logrus.Debugf("get scheduler result length(%d) with taskID(%s) and clientID(%s)", len(pieceResult), task.ID, clientID)

	if len(pieceResult) == 0 {
		return false, nil, errortypes.ErrPeerWait
	}
	var pieceInfos []*types.PieceInfo
	for _, v := range pieceResult {
		logrus.Debugf("get scheduler result item: %+v with taskID(%s) and clientID(%s)", v, task.ID, clientID)
		pieceInfo, err := tm.pieceResultToPieceInfo(ctx, v, task.PieceSize)
		if err != nil {
			return false, nil, err
		}

		// get supernode IP according to the cid dynamically
		if tm.cfg.IsSuperPID(pieceInfo.PID) {
			pieceInfo.PeerIP = dfgetTask.SupernodeIP
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

	pieceMD5, err := tm.cdnMgr.GetPieceMD5(ctx, pr.TaskID, pr.PieceNum, "", "default")
	if err != nil {
		logrus.Warnf("failed to get piece MD5 taskID(%s) pieceNum(%d): %v", pr.TaskID, pr.PieceNum, err)
		pieceMD5 = ""
	}
	return &types.PieceInfo{
		PID:        pr.DstPID,
		Path:       dfgetTask.Path,
		PeerIP:     peer.IP.String(),
		PeerPort:   peer.Port,
		PieceMD5:   pieceMD5,
		PieceRange: rangeutils.CalculatePieceRange(pr.PieceNum, pieceSize),
		PieceSize:  pieceSize,
	}, nil
}

// convertToPeerPieceStatus converts piece result and dfgetTask status to dfgetTask status code.
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

// convertToPeerPieceStatus converts piece result and dfgetTask status to piece status code.
// And it should return -1 if failed to convert.
func convertToPeerPieceStatus(result, status string) (int, bool) {
	if status == types.PiecePullRequestDfgetTaskStatusSTARTED {
		return config.PieceWAITING, true
	}

	if status == types.PiecePullRequestDfgetTaskStatusRUNNING {
		if result == types.PiecePullRequestPieceResultSUCCESS {
			return config.PieceSUCCESS, true
		}
		if result == types.PiecePullRequestPieceResultFAILED {
			return config.PieceFAILED, true
		}
		if result == types.PiecePullRequestPieceResultSEMISUC {
			return config.PieceSEMISUC, true
		}
	}

	return -1, false
}

// equalsTask determines that whether the two task objects are the same.
//
// The result is based only on whether the attributes used to generate taskID are the same
// which including taskURL, md5, identifier.
func equalsTask(existTask, newTask *types.TaskInfo) bool {
	if existTask.TaskURL != newTask.TaskURL {
		return false
	}

	if !stringutils.IsEmptyStr(existTask.Md5) {
		return existTask.Md5 == newTask.Md5
	}

	return existTask.Identifier == newTask.Identifier
}

// validateParams validates the params of TaskCreateRequest.
func validateParams(req *types.TaskCreateRequest) error {
	if !netutils.IsValidURL(req.RawURL) {
		return errors.Wrapf(errortypes.ErrInvalidValue, "raw url: %s", req.RawURL)
	}

	if stringutils.IsEmptyStr(req.Path) {
		return errors.Wrapf(errortypes.ErrEmptyValue, "path")
	}

	if stringutils.IsEmptyStr(req.CID) {
		return errors.Wrapf(errortypes.ErrEmptyValue, "cID")
	}

	if stringutils.IsEmptyStr(req.PeerID) {
		return errors.Wrapf(errortypes.ErrEmptyValue, "peerID")
	}

	return nil
}

// generateTaskID generates taskID with taskURL,md5 and identifier
// and returns the SHA-256 checksum of the data.
func generateTaskID(taskURL, md5, identifier string, header map[string]string) string {
	sign := ""
	if !stringutils.IsEmptyStr(md5) {
		sign = md5
	} else if !stringutils.IsEmptyStr(identifier) {
		sign = identifier
	}
	var id string
	if r, ok := header["Range"]; ok {
		id = fmt.Sprintf("%s%s%s%s%s", key, taskURL, sign, r, key)
	} else {
		id = fmt.Sprintf("%s%s%s%s", key, taskURL, sign, key)
	}
	return digest.Sha256(id)
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

func (tm *Manager) getHTTPFileLength(taskID, url string, headers map[string]string) (int64, error) {
	fileLength, code, err := tm.originClient.GetContentLength(url, headers)
	if err != nil {
		return -1, errors.Wrapf(errortypes.ErrUnknownError, "failed to get http file Length: %v", err)
	}

	if code == http.StatusUnauthorized || code == http.StatusProxyAuthRequired {
		return -1, errors.Wrapf(errortypes.ErrAuthenticationRequired, "taskID: %s,code: %d", taskID, code)
	}
	if code != http.StatusOK && code != http.StatusPartialContent {
		logrus.Warnf("failed to get http file length with unexpected code: %d", code)
		if code == http.StatusNotFound {
			return -1, errors.Wrapf(errortypes.ErrURLNotReachable, "taskID: %s, url: %s", taskID, url)
		}
		return -1, nil
	}

	return fileLength, nil
}
