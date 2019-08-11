package task

import (
	"context"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"
	"github.com/dragonflyoss/Dragonfly/pkg/timeutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	dutil "github.com/dragonflyoss/Dragonfly/supernode/daemon/util"
	"github.com/dragonflyoss/Dragonfly/supernode/httpclient"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	key = ">I$pg-~AS~sP'rqu_`Oh&lz#9]\"=;nE%"
)

var _ mgr.TaskMgr = &Manager{}

// Manager is an implementation of the interface of TaskMgr.
type Manager struct {
	cfg *config.Config

	taskStore               *dutil.Store
	taskLocker              *util.LockerPool
	accessTimeMap           *syncmap.SyncMap
	taskURLUnReachableStore *syncmap.SyncMap

	peerMgr      mgr.PeerMgr
	dfgetTaskMgr mgr.DfgetTaskMgr
	progressMgr  mgr.ProgressMgr
	cdnMgr       mgr.CDNMgr
	schedulerMgr mgr.SchedulerMgr
	OriginClient httpclient.OriginHTTPClient
}

// NewManager returns a new Manager Object.
func NewManager(cfg *config.Config, peerMgr mgr.PeerMgr, dfgetTaskMgr mgr.DfgetTaskMgr,
	progressMgr mgr.ProgressMgr, cdnMgr mgr.CDNMgr, schedulerMgr mgr.SchedulerMgr, originClient httpclient.OriginHTTPClient) (*Manager, error) {
	return &Manager{
		cfg:                     cfg,
		taskStore:               dutil.NewStore(),
		taskLocker:              util.NewLockerPool(),
		peerMgr:                 peerMgr,
		dfgetTaskMgr:            dfgetTaskMgr,
		progressMgr:             progressMgr,
		cdnMgr:                  cdnMgr,
		schedulerMgr:            schedulerMgr,
		accessTimeMap:           syncmap.NewSyncMap(),
		taskURLUnReachableStore: syncmap.NewSyncMap(),
		OriginClient:            originClient,
	}, nil
}

// Register will not only register a task.
func (tm *Manager) Register(ctx context.Context, req *types.TaskCreateRequest) (taskCreateResponse *types.TaskCreateResponse, err error) {
	// Step1: validate params
	if err := validateParams(req); err != nil {
		return nil, err
	}

	// Step2: add a new Task or update the exist task
	failAccessInterval := tm.cfg.FailAccessInterval * time.Minute
	task, err := tm.addOrUpdateTask(ctx, req, failAccessInterval)
	if err != nil {
		logrus.Infof("failed to add or update task with req %+v: %v", req, err)
		return nil, err
	}
	logrus.Debugf("success to get task info: %+v", task)
	// TODO: defer rollback the task update

	// update accessTime for taskID
	if err := tm.accessTimeMap.Add(task.ID, timeutils.GetCurrentTimeMillis()); err != nil {
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

	return &types.TaskCreateResponse{
		ID:         task.ID,
		FileLength: task.HTTPFileLength,
		PieceSize:  task.PieceSize,
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

// CheckTaskStatus check the task status.
func (tm *Manager) CheckTaskStatus(ctx context.Context, taskID string) (bool, error) {
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
	tm.taskStore.Delete(taskID)
	return nil
}

// Update the info of task.
func (tm *Manager) Update(ctx context.Context, taskID string, taskInfo *types.TaskInfo) error {
	return tm.updateTask(taskID, taskInfo)
}

// GetPieces get the pieces to be downloaded based on the scheduling result.
func (tm *Manager) GetPieces(ctx context.Context, taskID, clientID string, req *types.PiecePullRequest) (bool, interface{}, error) {
	logrus.Debugf("get pieces request: %+v with taskID(%s) and clientID(%s)", req, taskID, clientID)

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
	if err := tm.accessTimeMap.Add(task.ID, timeutils.GetCurrentTimeMillis()); err != nil {
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

// UpdatePieceStatus update the piece status with specified parameters.
func (tm *Manager) UpdatePieceStatus(ctx context.Context, taskID, pieceRange string, pieceUpdateRequest *types.PieceUpdateRequest) error {
	// calculate the pieceNum according to the pieceRange
	pieceNum := util.CalculatePieceNum(pieceRange)
	if pieceNum == -1 {
		return errors.Wrapf(errortypes.ErrInvalidValue,
			"failed to parse pieceRange: %s to pieceNum for taskID: %s, clientID: %s",
			pieceRange, taskID, pieceUpdateRequest.ClientID)
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
