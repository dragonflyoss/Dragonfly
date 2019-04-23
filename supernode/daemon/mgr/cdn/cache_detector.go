package cdn

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/store"

	"github.com/sirupsen/logrus"
)

type cacheDetector struct {
	cacheStore      *store.Store
	metaDataManager *fileMetaDataManager
}

func newCacheDetector(cacheStore *store.Store, metaDataManager *fileMetaDataManager) *cacheDetector {
	return &cacheDetector{
		cacheStore:      cacheStore,
		metaDataManager: metaDataManager,
	}
}

// detectCache detects whether there is a corresponding file in the local.
// If any, check whether the entire file has been completely downloaded.
//
// If so, return the md5 of task file and return startPieceNum as -1.
// And if not, return the lastest piece num that has been downloaded.
func (cd *cacheDetector) detectCache(ctx context.Context, task *types.TaskInfo) (int, *fileMetaData, error) {
	var breakNum int
	var metaData *fileMetaData
	var err error

	if metaData, err = cd.metaDataManager.readFileMetaData(ctx, task.ID); err == nil &&
		checkSameFile(task, metaData) {
		breakNum = cd.parseBreakNum(ctx, task, metaData)
	}
	logrus.Infof("detect cache breakNum: %d", breakNum)

	if breakNum == 0 {
		if metaData, err = cd.resetRepo(ctx, task); err != nil {
			return 0, nil, err
		}
	}

	// TODO: update the access time of task meta file for GC module
	return breakNum, metaData, nil
}

func (cd *cacheDetector) parseBreakNum(ctx context.Context, task *types.TaskInfo, metaData *fileMetaData) int {
	expired, err := cutil.IsExpired(task.TaskURL, task.Headers, metaData.LastModified, metaData.ETag)
	if err != nil {
		logrus.Errorf("failed to check whether the task(%s) has expired: %v", task.ID, err)
	}
	if expired {
		return 0
	}

	if metaData.Finish {
		if metaData.Success {
			return -1
		}
		return 0
	}

	supportRange, err := cutil.IsSupportRange(task.TaskURL, task.Headers)
	if err != nil {
		logrus.Errorf("failed to check whether the task(%s) supports partial requests: %v", task.ID, err)
	}
	if !supportRange || task.FileLength < 0 {
		return 0
	}

	return cd.parseBreakNumByCheckFile(ctx, task.ID)
}

func (cd *cacheDetector) parseBreakNumByCheckFile(ctx context.Context, taskID string) int {
	cacheReader := newSuperReader()

	reader, err := cd.cacheStore.Get(ctx, getDownloadRawFunc(taskID))
	if err != nil {
		logrus.Errorf("failed to read key file: %s : %v", taskID, err)
		return 0
	}
	result, err := cacheReader.readFile(ctx, reader, false, false)
	if err != nil {
		logrus.Errorf("read file gets error: %v", err)
	}
	if result != nil {
		return result.pieceCount
	}

	return 0
}

func (cd *cacheDetector) resetRepo(ctx context.Context, task *types.TaskInfo) (*fileMetaData, error) {
	logrus.Infof("reset repo for taskID: %s", task.ID)
	if err := deleteTaskFiles(ctx, cd.cacheStore, task.ID, false); err != nil {
		return nil, err
	}

	return cd.metaDataManager.writeFileMetaDataByTask(ctx, task)
}

func checkSameFile(task *types.TaskInfo, metaData *fileMetaData) bool {
	if cutil.IsNil(task) || cutil.IsNil(metaData) {
		return false
	}

	if metaData.PieceSize != task.PieceSize {
		return false
	}

	if metaData.TaskID != task.ID {
		return false
	}

	if metaData.URL != task.TaskURL {
		return false
	}

	if !cutil.IsEmptyStr(task.Md5) {
		return metaData.Md5 == task.Md5
	}

	if cutil.IsEmptyStr(metaData.Md5) {
		return metaData.Identifier == task.Identifier
	}
	return false
}
