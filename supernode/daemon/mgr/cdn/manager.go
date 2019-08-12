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
	"crypto/md5"
	"path"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/limitreader"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	"github.com/dragonflyoss/Dragonfly/supernode/store"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/dragonflyoss/Dragonfly/supernode/httpclient"
	"github.com/sirupsen/logrus"
)

var _ mgr.CDNMgr = &Manager{}

// Manager is an implementation of the interface of CDNMgr.
type Manager struct {
	cfg             *config.Config
	cacheStore      *store.Store
	limiter         *ratelimiter.RateLimiter
	cdnLocker       *util.LockerPool
	progressManager mgr.ProgressMgr

	metaDataManager *fileMetaDataManager
	cdnReporter     *reporter
	detector        *cacheDetector
	originClient    httpclient.OriginHTTPClient
	pieceMD5Manager *pieceMD5Mgr
	writer          *superWriter
}

// NewManager returns a new Manager.
func NewManager(cfg *config.Config, cacheStore *store.Store, progressManager mgr.ProgressMgr, originClient httpclient.OriginHTTPClient) (*Manager, error) {
	rateLimiter := ratelimiter.NewRateLimiter(ratelimiter.TransRate(config.TransLimit(cfg.MaxBandwidth-cfg.SystemReservedBandwidth)), 2)
	metaDataManager := newFileMetaDataManager(cacheStore)
	pieceMD5Manager := newpieceMD5Mgr()
	cdnReporter := newReporter(cfg, cacheStore, progressManager, metaDataManager, pieceMD5Manager)
	return &Manager{
		cfg:             cfg,
		cacheStore:      cacheStore,
		limiter:         rateLimiter,
		cdnLocker:       util.NewLockerPool(),
		progressManager: progressManager,
		metaDataManager: metaDataManager,
		pieceMD5Manager: pieceMD5Manager,
		cdnReporter:     cdnReporter,
		detector:        newCacheDetector(cacheStore, metaDataManager, originClient),
		originClient:    originClient,
		writer:          newSuperWriter(cacheStore, cdnReporter),
	}, nil
}

// TriggerCDN will trigger CDN to download the file from sourceUrl.
func (cm *Manager) TriggerCDN(ctx context.Context, task *types.TaskInfo) (*types.TaskInfo, error) {
	httpFileLength := task.HTTPFileLength
	if httpFileLength == 0 {
		httpFileLength = -1
	}

	cm.cdnLocker.GetLock(task.ID, false)
	defer cm.cdnLocker.ReleaseLock(task.ID, false)
	// detect Cache
	startPieceNum, metaData, err := cm.detector.detectCache(ctx, task)
	if err != nil {
		logrus.Errorf("failed to detect cache for task %s: %v", task.ID, err)
	}
	fileMD5, updateTaskInfo, err := cm.cdnReporter.reportCache(ctx, task.ID, metaData, startPieceNum)
	if err != nil {
		logrus.Errorf("failed to report cache for taskId: %s : %v", task.ID, err)
	}

	if startPieceNum == -1 {
		logrus.Infof("cache full hit for taskId:%s on local", task.ID)
		return updateTaskInfo, nil
	}

	if fileMD5 == nil {
		fileMD5 = md5.New()
	}

	// get piece content size which not including the piece header and trailer
	pieceContSize := task.PieceSize - config.PieceWrapSize

	// start to download the source file
	resp, err := cm.download(ctx, task.ID, task.RawURL, task.Headers, startPieceNum, httpFileLength, pieceContSize)
	if err != nil {
		return getUpdateTaskInfoWithStatusOnly(types.TaskInfoCdnStatusFAILED), err
	}
	defer resp.Body.Close()

	cm.updateLastModifiedAndETag(ctx, task.ID, resp.Header.Get("Last-Modified"), resp.Header.Get("Etag"))
	reader := limitreader.NewLimitReaderWithLimiterAndMD5Sum(resp.Body, cm.limiter, fileMD5)
	downloadMetadata, err := cm.writer.startWriter(ctx, cm.cfg, reader, task, startPieceNum, httpFileLength, pieceContSize)
	if err != nil {
		logrus.Errorf("failed to write for task %s: %v", task.ID, err)
		return nil, err
	}

	realMD5 := reader.Md5()
	success, err := cm.handleCDNResult(ctx, task, realMD5, httpFileLength, downloadMetadata.realHTTPFileLength, downloadMetadata.realFileLength)
	if err != nil || success == false {
		return getUpdateTaskInfoWithStatusOnly(types.TaskInfoCdnStatusFAILED), err
	}

	return getUpdateTaskInfo(types.TaskInfoCdnStatusSUCCESS, realMD5, downloadMetadata.realFileLength), nil
}

// GetHTTPPath returns the http download path of taskID.
// The returned path joined the DownloadRaw.Bucket and DownloadRaw.Key.
func (cm *Manager) GetHTTPPath(ctx context.Context, taskID string) (string, error) {
	raw := getDownloadRawFunc(taskID)
	return path.Join("/", raw.Bucket, raw.Key), nil
}

// GetStatus get the status of the file.
func (cm *Manager) GetStatus(ctx context.Context, taskID string) (cdnStatus string, err error) {
	return "", nil
}

// Delete the file from disk with specified taskID.
func (cm *Manager) Delete(ctx context.Context, taskID string) error {
	return nil
}

func (cm *Manager) handleCDNResult(ctx context.Context, task *types.TaskInfo, realMd5 string, httpFileLength, realHTTPFileLength, realFileLength int64) (bool, error) {
	var isSuccess = true
	if !stringutils.IsEmptyStr(task.Md5) && task.Md5 != realMd5 {
		logrus.Errorf("taskId:%s url:%s file md5 not match expected:%s real:%s", task.ID, task.TaskURL, task.Md5, realMd5)
		isSuccess = false
	}
	if isSuccess && httpFileLength >= 0 && httpFileLength != realHTTPFileLength {
		logrus.Errorf("taskId:%s url:%s file length not match expected:%d real:%d", task.ID, task.TaskURL, httpFileLength, realHTTPFileLength)
		isSuccess = false
	}

	if !isSuccess {
		realFileLength = 0
	}
	if err := cm.metaDataManager.updateStatusAndResult(ctx, task.ID, &fileMetaData{
		Finish:     true,
		Success:    isSuccess,
		RealMd5:    realMd5,
		FileLength: realFileLength,
	}); err != nil {
		return false, err
	}

	if !isSuccess {
		return false, nil
	}

	logrus.Infof("success to get taskID: %s fileLength: %d realMd5: %s", task.ID, realFileLength, realMd5)

	pieceMD5s, err := cm.pieceMD5Manager.getPieceMD5sByTaskID(task.ID)
	if err != nil {
		return false, err
	}

	if err := cm.metaDataManager.writePieceMD5s(ctx, task.ID, realMd5, pieceMD5s); err != nil {
		return false, err
	}
	return true, nil
}

func (cm *Manager) updateLastModifiedAndETag(ctx context.Context, taskID, lastModified, eTag string) {
	lastModifiedInt, _ := netutils.ConvertTimeStringToInt(lastModified)
	if err := cm.metaDataManager.updateLastModifiedAndETag(ctx, taskID, lastModifiedInt, eTag); err != nil {
		logrus.Errorf("failed to update LastModified(%s) and ETag(%s) for taskID %s: %v", lastModified, eTag, taskID, err)
	}
	logrus.Infof("success to update LastModified(%s) and ETag(%s) for taskID: %s", lastModified, eTag, taskID)
}
