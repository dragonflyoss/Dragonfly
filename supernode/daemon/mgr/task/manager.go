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
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/metricsutils"
	"github.com/dragonflyoss/Dragonfly/pkg/rangeutils"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	dutil "github.com/dragonflyoss/Dragonfly/supernode/daemon/util"
	"github.com/dragonflyoss/Dragonfly/supernode/httpclient"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	key = ">I$pg-~AS~sP'rqu_`Oh&lz#9]\"=;nE%"
)

var _ mgr.TaskMgr = &Manager{}

type metrics struct {
	tasks                        *prometheus.GaugeVec
	tasksRegisterCount           *prometheus.CounterVec
	triggerCdnCount              *prometheus.CounterVec
	triggerCdnFailCount          *prometheus.CounterVec
	scheduleDurationMilliSeconds *prometheus.HistogramVec
}

func newMetrics(register prometheus.Registerer) *metrics {
	return &metrics{
		tasks: metricsutils.NewGauge(config.SubsystemSupernode, "tasks",
			"Current status of Supernode tasks", []string{"cdnstatus"}, register),

		tasksRegisterCount: metricsutils.NewCounter(config.SubsystemSupernode, "tasks_registered_total",
			"Total times of registering tasks", []string{}, register),

		triggerCdnCount: metricsutils.NewCounter(config.SubsystemSupernode, "cdn_trigger_total",
			"Total times of triggering cdn", []string{}, register),

		triggerCdnFailCount: metricsutils.NewCounter(config.SubsystemSupernode, "cdn_trigger_failed_total",
			"Total failure times of triggering cdn", []string{}, register),

		scheduleDurationMilliSeconds: metricsutils.NewHistogram(config.SubsystemSupernode, "schedule_duration_milliseconds",
			"Duration for task scheduling in milliseconds", []string{"peer"},
			prometheus.ExponentialBuckets(0.02, 2, 6), register),
	}
}

// Manager is an implementation of the interface of TaskMgr.
type Manager struct {
	cfg          *config.Config
	metrics      *metrics
	originClient httpclient.OriginHTTPClient

	// store object
	taskStore               *dutil.Store
	accessTimeMap           *syncmap.SyncMap
	taskURLUnReachableStore *syncmap.SyncMap

	// mgr object
	peerMgr      mgr.PeerMgr
	dfgetTaskMgr mgr.DfgetTaskMgr
	progressMgr  mgr.ProgressMgr
	cdnMgr       mgr.CDNMgr
	schedulerMgr mgr.SchedulerMgr
}

// NewManager returns a new Manager Object.
func NewManager(cfg *config.Config, peerMgr mgr.PeerMgr, dfgetTaskMgr mgr.DfgetTaskMgr,
	progressMgr mgr.ProgressMgr, cdnMgr mgr.CDNMgr, schedulerMgr mgr.SchedulerMgr,
	originClient httpclient.OriginHTTPClient, register prometheus.Registerer) (*Manager, error) {
	return &Manager{
		cfg:                     cfg,
		taskStore:               dutil.NewStore(),
		peerMgr:                 peerMgr,
		dfgetTaskMgr:            dfgetTaskMgr,
		progressMgr:             progressMgr,
		cdnMgr:                  cdnMgr,
		schedulerMgr:            schedulerMgr,
		accessTimeMap:           syncmap.NewSyncMap(),
		taskURLUnReachableStore: syncmap.NewSyncMap(),
		originClient:            originClient,
		metrics:                 newMetrics(register),
	}, nil
}

// Register will not only register a task.
func (tm *Manager) Register(ctx context.Context, req *types.TaskCreateRequest) (taskCreateResponse *types.TaskCreateResponse, err error) {
	// Step1: validate params
	if err := validateParams(req); err != nil {
		return nil, err
	}

	// Step2: add a new Task or update the exist task
	failAccessInterval := tm.cfg.FailAccessInterval
	task, err := tm.addOrUpdateTask(ctx, req, failAccessInterval)
	if err != nil {
		logrus.Infof("failed to add or update task with req %+v: %v", req, err)
		return nil, err
	}
	tm.metrics.tasksRegisterCount.WithLabelValues().Inc()
	logrus.Debugf("success to get task info: %+v", task)
	// TODO: defer rollback the task update

	util.GetLock(task.ID, true)
	defer util.ReleaseLock(task.ID, true)

	// update accessTime for taskID
	if err := tm.accessTimeMap.Add(task.ID, time.Now()); err != nil {
		logrus.Warnf("failed to update accessTime for taskID(%s): %v", task.ID, err)
	}

	// Step3: add a new DfgetTask
	dfgetTask, err := tm.addDfgetTask(ctx, req, task)
	if err != nil {
		logrus.Infof("failed to add dfgetTask %+v: %v", dfgetTask, err)
		return nil, err
	}

	logrus.Debugf("success to add dfgetTask %+v", dfgetTask)
	defer func() {
		if err != nil {
			if err := tm.dfgetTaskMgr.Delete(ctx, req.CID, task.ID); err != nil {
				logrus.Errorf("failed to delete the dfgetTask with taskID %s peerID %s: %v", task.ID, req.PeerID, err)
			}
			logrus.Infof("success to rollback the dfgetTask %+v", dfgetTask)
		}
	}()

	// Step4: init Progress
	if err := tm.progressMgr.InitProgress(ctx, task.ID, req.PeerID, req.CID); err != nil {
		return nil, err
	}
	logrus.Debugf("success to init progress for taskID: %s peerID: %s cID: %s", task.ID, req.PeerID, req.CID)
	// TODO: defer rollback init Progress

	// Step5: trigger CDN
	if err := tm.triggerCdnSyncAction(ctx, task); err != nil {
		return nil, errors.Wrapf(errortypes.ErrSystemError, "failed to trigger cdn: %v", err)
	}

	cdnSource := types.CdnSourceSupernode
	if tm.cfg.CDNPattern == config.CDNPatternSource {
		cdnSource = types.CdnSourceSource
	}
	return &types.TaskCreateResponse{
		ID:         task.ID,
		FileLength: task.HTTPFileLength,
		PieceSize:  task.PieceSize,
		CdnSource:  cdnSource,
	}, nil
}

// Get a task info according to specified taskID.
func (tm *Manager) Get(ctx context.Context, taskID string) (*types.TaskInfo, error) {
	return tm.getTask(taskID)
}

// GetAccessTime gets all task accessTime.
func (tm *Manager) GetAccessTime(ctx context.Context) (*syncmap.SyncMap, error) {
	return tm.accessTimeMap, nil
}

// List returns a list of tasks with filter.
// TODO: implement it.
func (tm *Manager) List(ctx context.Context, filter map[string]string) ([]*types.TaskInfo, error) {
	return nil, nil
}

// CheckTaskStatus checks the task status.
func (tm *Manager) CheckTaskStatus(ctx context.Context, taskID string) (bool, error) {
	util.GetLock(taskID, true)
	defer util.ReleaseLock(taskID, true)

	task, err := tm.getTask(taskID)
	if err != nil {
		return false, err
	}

	// the expected CDNStatus is not nil
	if stringutils.IsEmptyStr(task.CdnStatus) {
		return false, errors.Wrap(errortypes.ErrSystemError, "CDNStatus of TaskInfo")
	}

	return isSuccessCDN(task.CdnStatus), nil
}

// Delete deletes a task.
func (tm *Manager) Delete(ctx context.Context, taskID string) error {
	tm.accessTimeMap.Delete(taskID)
	tm.taskURLUnReachableStore.Delete(taskID)
	tm.taskStore.Delete(taskID)
	return nil
}

// Update the info of task.
func (tm *Manager) Update(ctx context.Context, taskID string, taskInfo *types.TaskInfo) error {
	util.GetLock(taskID, false)
	defer util.ReleaseLock(taskID, false)

	return tm.updateTask(taskID, taskInfo)
}

// GetPieces gets the pieces to be downloaded based on the scheduling result.
func (tm *Manager) GetPieces(ctx context.Context, taskID, clientID string, req *types.PiecePullRequest) (bool, interface{}, error) {
	logrus.Debugf("get pieces request: %+v with taskID(%s) and clientID(%s)", req, taskID, clientID)

	util.GetLock(taskID, true)
	defer util.ReleaseLock(taskID, true)

	// convert piece result and dfgetTask status to dfgetTask status code
	dfgetTaskStatus := convertToDfgetTaskStatus(req.PieceResult, req.DfgetTaskStatus)
	if stringutils.IsEmptyStr(dfgetTaskStatus) {
		return false, nil, errors.Wrapf(errortypes.ErrInvalidValue, "failed to convert piece result (%s) dfgetTaskStatus (%s)", req.PieceResult, req.DfgetTaskStatus)
	}

	dfgetTask, err := tm.dfgetTaskMgr.Get(ctx, clientID, taskID)
	if err != nil {
		return false, nil, errors.Wrapf(err, "failed to get dfgetTask with taskID (%s) clientID (%s)", taskID, clientID)
	}
	logrus.Debugf("success to get dfgetTask: %+v", dfgetTask)

	task, err := tm.getTask(taskID)
	if err != nil {
		return false, nil, errors.Wrapf(err, "failed to get taskID (%s)", taskID)
	}
	logrus.Debugf("success to get task: %+v", task)

	// update accessTime for taskID
	if err := tm.accessTimeMap.Add(task.ID, time.Now()); err != nil {
		logrus.Warnf("failed to update accessTime for taskID(%s): %v", task.ID, err)
	}

	if dfgetTaskStatus == types.DfGetTaskStatusWAITING {
		logrus.Debugf("start to process task(%s) start", taskID)
		return tm.processTaskStart(ctx, clientID, task, dfgetTask)
	}
	if dfgetTaskStatus == types.DfGetTaskStatusRUNNING {
		logrus.Debugf("start to process task(%s) running", taskID)
		return tm.processTaskRunning(ctx, clientID, dfgetTask.PeerID, task, req, dfgetTask)
	}
	logrus.Debugf("start to process task(%s) finish", taskID)
	return true, nil, tm.processTaskFinish(ctx, taskID, clientID, dfgetTaskStatus)
}

// UpdatePieceStatus updates the piece status with specified parameters.
func (tm *Manager) UpdatePieceStatus(ctx context.Context, taskID, pieceRange string, pieceUpdateRequest *types.PieceUpdateRequest) error {
	logrus.Debugf("get update piece status request: %+v with taskID(%s) pieceRange(%s)", pieceUpdateRequest, taskID, pieceRange)

	// calculate the pieceNum according to the pieceRange
	pieceNum := rangeutils.CalculatePieceNum(pieceRange)
	if pieceNum == -1 {
		return errors.Wrapf(errortypes.ErrInvalidValue,
			"failed to parse pieceRange: %s to pieceNum for taskID: %s, clientID: %s",
			pieceRange, taskID, pieceUpdateRequest.ClientID)
	}

	// when a peer success to download a piece from supernode,
	// and the load of supernode for the taskID should be decremented by one.
	if tm.cfg.IsSuperPID(pieceUpdateRequest.DstPID) {
		_, err := tm.progressMgr.UpdateSuperLoad(ctx, taskID, -1, -1)
		if err != nil {
			logrus.Warnf("failed to update superLoad taskID(%s) clientID(%s): %v", taskID, pieceUpdateRequest.ClientID, err)
		}
	}

	// get dfgetTask according to the CID
	srcDfgetTask, err := tm.dfgetTaskMgr.Get(ctx, pieceUpdateRequest.ClientID, taskID)
	if err != nil {
		return err
	}

	// get piece status code according to the pieceUpdateRequest.Result
	pieceStatus, ok := mgr.PieceStatusMap[pieceUpdateRequest.PieceStatus]
	if !ok {
		return errors.Wrapf(errortypes.ErrInvalidValue, "result: %s", pieceUpdateRequest.PieceStatus)
	}

	return tm.progressMgr.UpdateProgress(ctx, taskID, pieceUpdateRequest.ClientID,
		srcDfgetTask.PeerID, pieceUpdateRequest.DstPID, pieceNum, pieceStatus)
}
