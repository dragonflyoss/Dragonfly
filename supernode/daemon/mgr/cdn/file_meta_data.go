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

package cdn

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/digest"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/store"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type fileMetaData struct {
	TaskID      string `json:"taskID"`
	URL         string `json:"url"`
	PieceSize   int32  `json:"pieceSize"`
	HTTPFileLen int64  `json:"httpFileLen"`
	Identifier  string `json:"bizId"`

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
		return errors.Wrapf(err, "failed to marshal metadata")
	}

	return mm.fileStore.PutBytes(ctx, getMetaDataRawFunc(metaData.TaskID), data)
}

// readFileMetaData returns the fileMetaData info according to the taskID.
func (mm *fileMetaDataManager) readFileMetaData(ctx context.Context, taskID string) (*fileMetaData, error) {
	bytes, err := mm.fileStore.GetBytes(ctx, getMetaDataRawFunc(taskID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get metadata bytes")
	}

	metaData := &fileMetaData{}
	if err := json.Unmarshal(bytes, metaData); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal metadata bytes")
	}
	logrus.Debugf("success to read metadata: %+v for taskID: %s", metaData, taskID)

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
		return errors.Wrapf(err, "failed to get origin metaData")
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
		return errors.Wrapf(err, "failed to get origin metadata")
	}

	originMetaData.Finish = metaData.Finish
	originMetaData.Success = metaData.Success
	if originMetaData.Success {
		originMetaData.FileLength = metaData.FileLength
		if !stringutils.IsEmptyStr(metaData.RealMd5) {
			originMetaData.RealMd5 = metaData.RealMd5
		}
	}

	return mm.writeFileMetaData(ctx, originMetaData)
}

// writePieceMD5s writes the piece md5s to storage for the md5 file of taskID.
//
// And it should append the fileMD5 which means that the md5 of the task file
// and the SHA-1 digest of fileMD5 at the end of the file.
func (mm *fileMetaDataManager) writePieceMD5s(ctx context.Context, taskID, fileMD5 string, pieceMD5s []string) error {
	mm.locker.GetLock(taskID, false)
	defer mm.locker.ReleaseLock(taskID, false)

	if len(pieceMD5s) == 0 {
		logrus.Warnf("failed to write empty pieceMD5s for taskID: %s", taskID)
		return nil
	}

	// append fileMD5
	pieceMD5s = append(pieceMD5s, fileMD5)
	// append the SHA-1 checksum of pieceMD5s
	pieceMD5s = append(pieceMD5s, digest.Sha1(pieceMD5s))

	pieceMD5Str := strings.Join(pieceMD5s, "\n")

	return mm.fileStore.PutBytes(ctx, getMd5DataRawFunc(taskID), []byte(pieceMD5Str))
}

// readPieceMD5s reads the md5 file of the taskID and returns the pieceMD5s.
func (mm *fileMetaDataManager) readPieceMD5s(ctx context.Context, taskID, fileMD5 string) (pieceMD5s []string, err error) {
	mm.locker.GetLock(taskID, true)
	defer mm.locker.ReleaseLock(taskID, true)

	bytes, err := mm.fileStore.GetBytes(ctx, getMd5DataRawFunc(taskID))
	if err != nil {
		return nil, err
	}
	pieceMD5s = strings.Split(strings.TrimSpace(string(bytes)), "\n")

	if len(pieceMD5s) == 0 {
		return nil, nil
	}

	// validate the SHA-1 checksum of pieceMD5s
	pieceMD5sLength := len(pieceMD5s)
	pieceMD5sWithoutSha1Value := pieceMD5s[:pieceMD5sLength-1]
	expectedSha1Value := digest.Sha1(pieceMD5sWithoutSha1Value)
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
