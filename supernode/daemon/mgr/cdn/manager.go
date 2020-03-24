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
	"fmt"
	"path"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/limitreader"
	"github.com/dragonflyoss/Dragonfly/pkg/metricsutils"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/rangeutils"
	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	"github.com/dragonflyoss/Dragonfly/supernode/httpclient"
	"github.com/dragonflyoss/Dragonfly/supernode/store"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	PieceMd5SourceDefault = "default"
	PieceMd5SourceMemory  = "memory"
	PieceMd5SourceMeta    = "meta"
	PieceMd5SourceFile    = "file"
)

var _ mgr.CDNMgr = &Manager{}

type metrics struct {
	cdnCacheHitCount     *prometheus.CounterVec
	cdnDownloadCount     *prometheus.CounterVec
	cdnDownloadFailCount *prometheus.CounterVec
}

func newMetrics(register prometheus.Registerer) *metrics {
	return &metrics{
		cdnCacheHitCount: metricsutils.NewCounter(config.SubsystemSupernode, "cdn_cache_hit_total",
			"Total times of hitting cdn cache", []string{}, register),

		cdnDownloadCount: metricsutils.NewCounter(config.SubsystemSupernode, "cdn_download_total",
			"Total times of cdn download", []string{}, register),

		cdnDownloadFailCount: metricsutils.NewCounter(config.SubsystemSupernode, "cdn_download_failed_total",
			"Total failure times of cdn download", []string{}, register),
	}
}

func init() {
	mgr.Register(config.CDNPatternLocal, NewManager)
}

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
	metrics         *metrics
}

// NewManager returns a new Manager.
func NewManager(cfg *config.Config, cacheStore *store.Store, progressManager mgr.ProgressMgr,
	originClient httpclient.OriginHTTPClient, register prometheus.Registerer) (mgr.CDNMgr, error) {
	return newManager(cfg, cacheStore, progressManager, originClient, register)
}

func newManager(cfg *config.Config, cacheStore *store.Store, progressManager mgr.ProgressMgr,
	originClient httpclient.OriginHTTPClient, register prometheus.Registerer) (*Manager, error) {
	rateLimiter := ratelimiter.NewRateLimiter(ratelimiter.TransRate(int64(cfg.MaxBandwidth-cfg.SystemReservedBandwidth)), 2)
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
		metrics:         newMetrics(register),
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
		cm.metrics.cdnCacheHitCount.WithLabelValues().Inc()
		return updateTaskInfo, nil
	}

	if fileMD5 == nil {
		fileMD5 = md5.New()
	}

	// get piece content size which not including the piece header and trailer
	pieceContSize := task.PieceSize - config.PieceWrapSize

	// start to download the source file
	resp, err := cm.download(ctx, task.ID, task.RawURL, task.Headers, startPieceNum, httpFileLength, pieceContSize)
	cm.metrics.cdnDownloadCount.WithLabelValues().Inc()
	if err != nil {
		cm.metrics.cdnDownloadFailCount.WithLabelValues().Inc()
		return getUpdateTaskInfoWithStatusOnly(types.TaskInfoCdnStatusFAILED), err
	}
	defer resp.Body.Close()

	cm.updateLastModifiedAndETag(ctx, task.ID, resp.Header.Get("Last-Modified"), resp.Header.Get("Etag"))
	reader := limitreader.NewLimitReaderWithLimiterAndMD5Sum(resp.Body, cm.limiter, fileMD5)
	downloadMetadata, err := cm.writer.startWriter(ctx, cm.cfg, reader, task, startPieceNum, httpFileLength, pieceContSize)
	if err != nil {
		logrus.Errorf("failed to write for task %s: %v", task.ID, err)
		return getUpdateTaskInfoWithStatusOnly(types.TaskInfoCdnStatusFAILED), err
	}

	realMD5 := reader.Md5()
	success, err := cm.handleCDNResult(ctx, task, realMD5, httpFileLength, downloadMetadata.realHTTPFileLength, downloadMetadata.realFileLength)
	if err != nil || !success {
		return getUpdateTaskInfoWithStatusOnly(types.TaskInfoCdnStatusFAILED), err
	}

	return getUpdateTaskInfo(types.TaskInfoCdnStatusSUCCESS, realMD5, downloadMetadata.realFileLength), nil
}

// GetHTTPPath returns the http download path of taskID.
// The returned path joined the DownloadRaw.Bucket and DownloadRaw.Key.
func (cm *Manager) GetHTTPPath(ctx context.Context, taskInfo *types.TaskInfo) (string, error) {
	raw := getDownloadRawFunc(taskInfo.ID)
	return path.Join("/", raw.Bucket, raw.Key), nil
}

// GetStatus gets the status of the file.
func (cm *Manager) GetStatus(ctx context.Context, taskID string) (cdnStatus string, err error) {
	return types.TaskInfoCdnStatusSUCCESS, nil
}

// GetPieceMD5 gets the piece Md5 accorrding to the specified taskID and pieceNum.
func (cm *Manager) GetPieceMD5(ctx context.Context, taskID string, pieceNum int, pieceRange, source string) (pieceMd5 string, err error) {
	if stringutils.IsEmptyStr(source) ||
		source == PieceMd5SourceDefault ||
		source == PieceMd5SourceMemory {
		return cm.pieceMD5Manager.getPieceMD5(taskID, pieceNum)
	}

	if source == PieceMd5SourceMeta {
		// get file meta data
		fileMeta, err := cm.metaDataManager.readFileMetaData(ctx, taskID)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get file meta data taskID(%s)", taskID)
		}

		// get piece md5s from meta data file
		pieceMD5s, err := cm.metaDataManager.readPieceMD5s(ctx, taskID, fileMeta.RealMd5)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get piece MD5s from meta data taskID(%s)", taskID)
		}

		if len(pieceMD5s) < pieceNum {
			return "", fmt.Errorf("not enough piece MD5 for pieceNum(%d)", pieceNum)
		}

		return pieceMD5s[pieceNum], nil
	}

	if source == PieceMd5SourceFile {
		// get piece length
		start, end, err := rangeutils.ParsePieceIndex(pieceRange)
		if err != nil {
			return "", errors.Wrapf(err, "failed to parse piece range(%s)", pieceRange)
		}
		pieceLength := end - start + 1

		// get piece content reader
		pieceRaw := getDownloadRawFunc(taskID)
		pieceRaw.Offset = start
		pieceRaw.Length = pieceLength
		reader, err := cm.cacheStore.Get(ctx, pieceRaw)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get file reader taskID(%s)", taskID)
		}

		// get piece Md5 by read source file
		return getMD5ByReadFile(reader, int32(pieceLength))
	}

	return "", nil
}

// CheckFile checks the file whether exists.
func (cm *Manager) CheckFile(ctx context.Context, taskID string) bool {
	if _, err := cm.cacheStore.Stat(ctx, getDownloadRaw(taskID)); err != nil {
		return false
	}
	return true
}

// Delete the cdn meta with specified taskID.
// It will also delete the files on the disk when the force equals true.
func (cm *Manager) Delete(ctx context.Context, taskID string, force bool) error {
	if !force {
		return cm.pieceMD5Manager.removePieceMD5sByTaskID(taskID)
	}

	return deleteTaskFiles(ctx, cm.cacheStore, taskID)
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
