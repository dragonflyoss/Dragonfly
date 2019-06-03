package cdn

import (
	"context"
	"encoding/json"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/store"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/sirupsen/logrus"
)

type fileMetaData struct {
	TaskID      string `json:"taskID"`
	URL         string `json:"url"`
	PieceSize   int32  `json:"pieceSize"`
	HTTPFileLen int64  `json:"httpFileLen"`
	Identifier  string `json:"identifier"`

	AccessTime   int64  `json:"accessTime"`
	Interval     int64  `json:"interval"`
	FileLength   int64  `json:"fileLength"`
	Md5          string `json:"md5"`
	RealMd5      string `json:"realMd5"`
	LastModified int64  `json:"lastModified"`
	ETag         string `json:"eTag"`
	Finish       bool   `json:"finish"`
	Success      bool   `json:"success"`
}

// fileMetaDataManager manages the meta file and md5 file of each taskID.
type fileMetaDataManager struct {
	fileStore *store.Store
	locker    *util.LockerPool
}

func newFileMetaDataManager(store *store.Store) *fileMetaDataManager {
	return &fileMetaDataManager{
		fileStore: store,
		locker:    util.NewLockerPool(),
	}
}

// writeFileMetaData stores the metadata of task.ID to storage.
func (mm *fileMetaDataManager) writeFileMetaDataByTask(ctx context.Context, task *types.TaskInfo) (*fileMetaData, error) {
	metaData := &fileMetaData{
		TaskID:      task.ID,
		URL:         task.TaskURL,
		PieceSize:   task.PieceSize,
		HTTPFileLen: task.HTTPFileLength,
		Identifier:  task.Identifier,
		AccessTime:  getCurrentTimeMillisFunc(),
		FileLength:  task.FileLength,
		Md5:         task.Md5,
	}

	if err := mm.writeFileMetaData(ctx, metaData); err != nil {
		return nil, err
	}

	return metaData, nil
}

// writeFileMetaData stores the metadata of task.ID to storage.
func (mm *fileMetaDataManager) writeFileMetaData(ctx context.Context, metaData *fileMetaData) error {
	data, err := json.Marshal(metaData)
	if err != nil {
		return err
	}

	return mm.fileStore.PutBytes(ctx, getMetaDataRawFunc(metaData.TaskID), data)
}

// readFileMetaData returns the fileMetaData info according to the taskID.
func (mm *fileMetaDataManager) readFileMetaData(ctx context.Context, taskID string) (*fileMetaData, error) {
	bytes, err := mm.fileStore.GetBytes(ctx, getMetaDataRawFunc(taskID))
	if err != nil {
		return nil, err
	}

	metaData := &fileMetaData{}
	if err := json.Unmarshal(bytes, metaData); err != nil {
		return nil, err
	}
	logrus.Infof("success to read metadata: %+v for taskID: %s", metaData, taskID)

	if metaData.PieceSize == 0 {
		metaData.PieceSize = config.DefaultPieceSize
	}
	return metaData, nil
}

func (mm *fileMetaDataManager) updateAccessTime(ctx context.Context, taskID string, accessTime int64) error {
	mm.locker.GetLock(taskID, false)
	defer mm.locker.ReleaseLock(taskID, false)

	originMetaData, err := mm.readFileMetaData(ctx, taskID)
	if err != nil {
		return err
	}

	interval := accessTime - originMetaData.AccessTime
	originMetaData.Interval = interval
	if interval <= 0 {
		logrus.Warnf("taskId:%s file hit interval:%d", taskID, interval)
		originMetaData.Interval = 0
	}

	originMetaData.AccessTime = accessTime

	return mm.writeFileMetaData(ctx, originMetaData)
}

func (mm *fileMetaDataManager) updateLastModifiedAndETag(ctx context.Context, taskID string, lastModified int64, eTag string) error {
	mm.locker.GetLock(taskID, false)
	defer mm.locker.ReleaseLock(taskID, false)

	originMetaData, err := mm.readFileMetaData(ctx, taskID)
	if err != nil {
		return err
	}

	originMetaData.LastModified = lastModified
	originMetaData.ETag = eTag

	return mm.writeFileMetaData(ctx, originMetaData)
}

func (mm *fileMetaDataManager) updateStatusAndResult(ctx context.Context, taskID string, metaData *fileMetaData) error {
	mm.locker.GetLock(taskID, false)
	defer mm.locker.ReleaseLock(taskID, false)

	originMetaData, err := mm.readFileMetaData(ctx, taskID)
	if err != nil {
		return err
	}

	originMetaData.Finish = metaData.Finish
	originMetaData.Success = metaData.Success
	if originMetaData.Success {
		originMetaData.FileLength = metaData.FileLength
		if !cutil.IsEmptyStr(metaData.RealMd5) {
			originMetaData.RealMd5 = metaData.RealMd5
		}
	}

	return mm.writeFileMetaData(ctx, originMetaData)
}

// writePieceMD5s write the piece md5s to storage for the md5 file of taskID.
//
// And it should append the fileMD5 which means that the md5 of the task file
// and the SHA-1 digest of fileMD5 at the end of the file.
func (mm *fileMetaDataManager) writePieceMD5s(ctx context.Context, taskID, fileMD5 string, pieceMD5s []string) error {
	mm.locker.GetLock(taskID, false)
	defer mm.locker.ReleaseLock(taskID, false)

	if cutil.IsEmptySlice(pieceMD5s) {
		logrus.Warnf("failed to write empty pieceMD5s for taskID: %s", taskID)
		return nil
	}

	// append fileMD5
	pieceMD5s = append(pieceMD5s, fileMD5)
	// append the the SHA-1 checksum of pieceMD5s
	pieceMD5s = append(pieceMD5s, cutil.Sha1(pieceMD5s))

	data, err := json.Marshal(pieceMD5s)
	if err != nil {
		return err
	}

	return mm.fileStore.PutBytes(ctx, getMd5DataRawFunc(taskID), data)
}

// readPieceMD5s read the md5 file of the taskID and returns the pieceMD5s.
func (mm *fileMetaDataManager) readPieceMD5s(ctx context.Context, taskID, fileMD5 string) (pieceMD5s []string, err error) {
	mm.locker.GetLock(taskID, true)
	defer mm.locker.ReleaseLock(taskID, true)

	bytes, err := mm.fileStore.GetBytes(ctx, getMd5DataRawFunc(taskID))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bytes, &pieceMD5s); err != nil {
		return nil, err
	}

	if cutil.IsEmptySlice(pieceMD5s) {
		return nil, nil
	}

	// validate the SHA-1 checksum of pieceMD5s
	pieceMD5sLength := len(pieceMD5s)
	pieceMD5sWithoutSha1Value := pieceMD5s[:pieceMD5sLength-1]
	expectedSha1Value := cutil.Sha1(pieceMD5sWithoutSha1Value)
	realSha1Value := pieceMD5s[pieceMD5sLength-1]
	if expectedSha1Value != realSha1Value {
		logrus.Errorf("failed to validate the SHA-1 checksum of pieceMD5s, expected: %s, real: %s", expectedSha1Value, realSha1Value)
		return nil, nil
	}

	// validate the fileMD5
	realFileMD5 := pieceMD5s[pieceMD5sLength-2]
	if realFileMD5 != fileMD5 {
		logrus.Errorf("failed to validate the fileMD5, expected: %s, real: %s", fileMD5, realFileMD5)
		return nil, nil
	}
	return pieceMD5s[:pieceMD5sLength-2], nil
}
