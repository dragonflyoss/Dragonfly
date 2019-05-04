package cdn

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	"github.com/dragonflyoss/Dragonfly/supernode/store"

	"github.com/sirupsen/logrus"
)

var _ mgr.CDNMgr = &Manager{}

// Manager is an implementation of the interface of CDNMgr.
type Manager struct {
	cfg             *config.Config
	cacheStore      *store.Store
	progressManager mgr.ProgressMgr

	metaDataManager *fileMetaDataManager
	cdnReporter     *reporter
	detector        *cacheDetector
	pieceMD5Manager *pieceMD5Mgr
}

// NewManager returns a new Manager.
func NewManager(cfg *config.Config, cacheStore *store.Store, progressManager mgr.ProgressMgr) (*Manager, error) {
	metaDataManager := newFileMetaDataManager(cacheStore)
	pieceMD5Manager := newpieceMD5Mgr()
	cdnReporter := newReporter(cfg, cacheStore, progressManager, metaDataManager, pieceMD5Manager)
	return &Manager{
		cfg:             cfg,
		cacheStore:      cacheStore,
		progressManager: progressManager,
		metaDataManager: metaDataManager,
		pieceMD5Manager: pieceMD5Manager,
		cdnReporter:     cdnReporter,
		detector:        newCacheDetector(cacheStore, metaDataManager),
	}, nil
}

// TriggerCDN will trigger CDN to download the file from sourceUrl.
func (cm *Manager) TriggerCDN(ctx context.Context, taskInfo *types.TaskInfo) error {
	httpFileLength := taskInfo.HTTPFileLength
	if httpFileLength == 0 {
		httpFileLength = -1
	}

	// detect Cache
	startPieceNum, metaData, err := cm.detector.detectCache(ctx, taskInfo)
	if err != nil {
		return err
	}
	if _, err := cm.cdnReporter.reportCache(ctx, taskInfo.ID, metaData, startPieceNum); err != nil {
		logrus.Errorf("failed to report cache for taskId: %s : %v", taskInfo.ID, err)
		return err
	}

	if startPieceNum == -1 {
		logrus.Infof("cache full hit for taskId:%s on local", taskInfo.ID)
		return nil
	}

	// get piece content size which not including the piece header and trailer
	pieceContSize := taskInfo.PieceSize - config.PieceWrapSize

	// start to download the source file
	resp, err := cm.download(ctx, taskInfo.ID, taskInfo.TaskURL, taskInfo.Headers, startPieceNum, httpFileLength, pieceContSize)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// TODO: update the LastModified And ETag for taskID.
	reader := cutil.NewLimitReader(resp.Body, cm.cfg.LinkLimit, true)

	return cm.startWriter(ctx, cm.cfg, reader, taskInfo, startPieceNum, httpFileLength, pieceContSize)
}

// GetStatus get the status of the file.
func (cm *Manager) GetStatus(ctx context.Context, taskID string) (cdnStatus string, err error) {
	return "", nil
}

// Delete the file from disk with specified taskID.
func (cm *Manager) Delete(ctx context.Context, taskID string) error {
	return nil
}
