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
	"hash"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	"github.com/dragonflyoss/Dragonfly/supernode/store"

	"github.com/sirupsen/logrus"
)

type reporter struct {
	cfg *config.Config

	cacheStore      *store.Store
	progressManager mgr.ProgressMgr
	metaDataManager *fileMetaDataManager
	pieceMD5Manager *pieceMD5Mgr
}

func newReporter(cfg *config.Config, cacheStore *store.Store, progressManager mgr.ProgressMgr,
	metaDataManager *fileMetaDataManager, pieceMD5Manager *pieceMD5Mgr) *reporter {
	return &reporter{
		cfg:             cfg,
		cacheStore:      cacheStore,
		progressManager: progressManager,
		metaDataManager: metaDataManager,
		pieceMD5Manager: pieceMD5Manager,
	}
}

func (re *reporter) reportCache(ctx context.Context, taskID string, metaData *fileMetaData,
	breakNum int) (hash.Hash, *types.TaskInfo, error) {
	// cache not hit
	if breakNum == 0 {
		return nil, nil, nil
	}

	success, updateTaskInfo, err := re.processCacheByQuick(ctx, taskID, metaData, breakNum)
	if err == nil && success {
		// it is possible to succeed only if breakNum equals -1
		return nil, updateTaskInfo, nil
	}
	logrus.Errorf("failed to process cache by quick taskID(%s): %v", taskID, err)

	// If we can't get the information quickly from fileMetaData,
	// and then we have to get that by reading the file.
	return re.processCacheByReadFile(ctx, taskID, metaData, breakNum)
}

func (re *reporter) processCacheByQuick(ctx context.Context, taskID string, metaData *fileMetaData, breakNum int) (bool, *types.TaskInfo, error) {
	if breakNum != -1 {
		logrus.Debugf("failed to processCacheByQuick: breakNum not equals -1 for taskID %s", taskID)
		return false, nil, nil
	}

	// validate the file md5
	if stringutils.IsEmptyStr(metaData.RealMd5) {
		logrus.Debugf("failed to processCacheByQuick: empty RealMd5 for taskID %s", taskID)
		return false, nil, nil
	}

	// validate the piece md5s
	var pieceMd5s []string
	var err error
	if pieceMd5s, err = re.pieceMD5Manager.getPieceMD5sByTaskID(taskID); err != nil {
		logrus.Debugf("failed to processCacheByQuick: failed to get pieceMd5s taskID %s: %v", taskID, err)
		return false, nil, err
	}
	if len(pieceMd5s) == 0 {
		if pieceMd5s, err = re.metaDataManager.readPieceMD5s(ctx, taskID, metaData.RealMd5); err != nil {
			logrus.Debugf("failed to processCacheByQuick: failed to read pieceMd5s taskID %s: %v", taskID, err)
			return false, nil, err
		}
	}
	if len(pieceMd5s) == 0 {
		logrus.Debugf("failed to processCacheByQuick: empty pieceMd5s taskID %s: %v", taskID, err)
		return false, nil, nil
	}

	return true, getUpdateTaskInfo(types.TaskInfoCdnStatusSUCCESS, metaData.Md5, metaData.FileLength),
		re.reportPiecesStatus(ctx, taskID, pieceMd5s)
}

func (re *reporter) processCacheByReadFile(ctx context.Context, taskID string, metaData *fileMetaData, breakNum int) (hash.Hash, *types.TaskInfo, error) {
	var calculateFileMd5 = true
	if breakNum == -1 && !stringutils.IsEmptyStr(metaData.RealMd5) {
		calculateFileMd5 = false
	}

	cacheReader := newSuperReader()
	reader, err := re.cacheStore.Get(ctx, getDownloadRawFunc(taskID))
	if err != nil {
		logrus.Errorf("failed to read key file taskID(%s): %v", taskID, err)
		return nil, nil, err
	}
	result, err := cacheReader.readFile(ctx, reader, true, calculateFileMd5)
	if err != nil {
		logrus.Errorf("failed to read cache file taskID(%s): %v", taskID, err)
		return nil, nil, err
	}
	logrus.Infof("success to get cache result: %+v by read file", result)

	if err := re.reportPiecesStatus(ctx, taskID, result.pieceMd5s); err != nil {
		return nil, nil, err
	}

	if breakNum != -1 {
		return result.fileMd5, nil, nil
	}

	fileMd5Value := metaData.RealMd5
	if stringutils.IsEmptyStr(fileMd5Value) {
		fileMd5Value = fileutils.GetMd5Sum(result.fileMd5, nil)
	}

	fmd := &fileMetaData{
		Finish:     true,
		Success:    true,
		RealMd5:    fileMd5Value,
		FileLength: result.fileLength,
	}
	if err := re.metaDataManager.updateStatusAndResult(ctx, taskID, fmd); err != nil {
		logrus.Errorf("failed to update status and result fileMetaData(%+v) for taskID(%s): %v", fmd, taskID, err)
		return nil, nil, err
	}
	logrus.Infof("success to update status and result fileMetaData(%+v) for taskID(%s)", fmd, taskID)

	return nil, getUpdateTaskInfo(types.TaskInfoCdnStatusSUCCESS, fileMd5Value, result.fileLength),
		re.metaDataManager.writePieceMD5s(ctx, taskID, fileMd5Value, result.pieceMd5s)
}

func (re *reporter) reportPiecesStatus(ctx context.Context, taskID string, pieceMd5s []string) error {
	// report pieces status
	for pieceNum := 0; pieceNum < len(pieceMd5s); pieceNum++ {
		if err := re.reportPieceStatus(ctx, taskID, pieceNum, pieceMd5s[pieceNum], config.PieceSUCCESS); err != nil {
			return err
		}
	}

	return nil
}

func (re *reporter) reportPieceStatus(ctx context.Context, taskID string, pieceNum int, md5 string, pieceStatus int) (err error) {
	defer func() {
		if err == nil {
			logrus.Debugf("success to report piece status with taskID(%s) pieceNum(%d)", taskID, pieceNum)
		}
	}()

	if pieceStatus == config.PieceSUCCESS {
		if err := re.pieceMD5Manager.setPieceMD5(taskID, pieceNum, md5); err != nil {
			return err
		}
	}

	return re.progressManager.UpdateProgress(ctx, taskID, re.cfg.GetSuperCID(taskID), re.cfg.GetSuperPID(), "", pieceNum, pieceStatus)
}
