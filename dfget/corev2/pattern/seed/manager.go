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

package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	dfdaemonDownloader "github.com/dragonflyoss/Dragonfly/dfdaemon/downloader"
	dfgetcfg "github.com/dragonflyoss/Dragonfly/dfget/config"
	coreAPI "github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/basic"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/api"
	uploader2 "github.com/dragonflyoss/Dragonfly/dfget/corev2/uploader"
	"github.com/dragonflyoss/Dragonfly/dfget/local/seed"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/pkg/algorithm"
	"github.com/dragonflyoss/Dragonfly/pkg/constants"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/protocol"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"

	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

type rangeRequest struct {
	url    string
	off    int64
	size   int64
	header map[string]string
}

func (rr rangeRequest) URL() string {
	return rr.url
}

func (rr rangeRequest) Offset() int64 {
	return rr.off
}

func (rr rangeRequest) Size() int64 {
	return rr.size
}

func (rr rangeRequest) Header() map[string]string {
	return rr.header
}

func (rr rangeRequest) Extra() interface{} {
	return nil
}

const (
	defaultUploadRate = 100 * 1024 * 1024

	defaultDownloadRate = 100 * 1024 * 1024

	// 512KB
	defaultBlockOrder = 19

	maxTry = 20
)

var (
	localManager *Manager
	once         sync.Once
)

func init() {
	dfdaemonDownloader.Register("seed", func(patternConfig config.PatternConfig, commonCfg config.DFGetCommonConfig, c config.Properties) dfdaemonDownloader.Stream {
		return NewManager(patternConfig, commonCfg, c)
	})
}

// Manager provides an implementation of downloader.Stream.
//
type Manager struct {
	//client        protocol.Client
	downloaderAPI api.DownloadAPI
	supernodeAPI  coreAPI.SupernodeAPI

	seedManager seed.Manager
	sm          *supernodeManager
	evQueue     queue.Queue

	cfg *Config

	ctx    context.Context
	cancel func()
}

func NewManager(patternConfig config.PatternConfig, commonCfg config.DFGetCommonConfig, c config.Properties) *Manager {
	once.Do(func() {
		m := newManager(patternConfig, commonCfg, c)
		localManager = m
	})

	return localManager
}

func newManager(pCfg config.PatternConfig, commonCfg config.DFGetCommonConfig, config config.Properties) *Manager {
	cfg := &Config{}
	data, err := json.Marshal(pCfg.Opts)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, cfg)
	if err != nil {
		panic(err)
	}

	cfg.DFGetCommonConfig = commonCfg

	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		cfg:           cfg,
		supernodeAPI:  coreAPI.NewSupernodeAPI(),
		downloaderAPI: api.NewDownloadAPI(),
		ctx:           ctx,
		cancel:        cancel,
	}

	if cfg.HighLevel <= 0 {
		cfg.HighLevel = 90
	}

	if cfg.LowLevel <= 0 {
		cfg.LowLevel = 80
	}

	if cfg.DefaultBlockOrder <= 0 {
		cfg.DefaultBlockOrder = defaultBlockOrder
	}

	if cfg.PerDownloadBlocks <= 0 {
		cfg.PerDownloadBlocks = 4
	}

	if cfg.PerDownloadBlocks >= 1000 {
		cfg.PerDownloadBlocks = 1000
	}

	if cfg.TotalLimit <= 0 {
		cfg.TotalLimit = 50
	}

	if cfg.ConcurrentLimit <= 0 {
		cfg.ConcurrentLimit = 4
	}

	config.SuperNodes = algorithm.DedupStringArr(config.SuperNodes)
	m.sm = NewSupernodeManager(ctx, cfg, config.SuperNodes, m.supernodeAPI)
	m.seedManager = seed.NewSeedManager(seed.NewSeedManagerOpt{
		StoreDir:           filepath.Join(cfg.WorkHome, "localSeed"),
		ConcurrentLimit:    cfg.ConcurrentLimit,
		TotalLimit:         cfg.TotalLimit,
		DownloadBlockOrder: uint32(cfg.DefaultBlockOrder),
		OpenMemoryCache:    !cfg.DisableOpenMemoryCache,
		DownloadRate:       int64(cfg.DownRate),
		UploadRate:         int64(cfg.UploadRate),
		HighLevel:          uint(cfg.HighLevel),
		LowLevel:           uint(cfg.LowLevel),
	})

	uploader2.RegisterUploader("seed", newUploader(m.seedManager))
	m.reportLocalSeedsToSuperNode()

	//todo: init Manager by input config.
	return m
}

//todo: add local seed manager and p2pNetwork updater.

// DownloadStreamContext implementation of downloader.Stream.
func (m *Manager) DownloadStreamContext(ctx context.Context, url string, header map[string][]string, name string) (io.ReadCloser, error) {
	reqRange, err := m.getRangeFromHeader(header)
	if err != nil {
		return nil, err
	}

	m.sm.AddRequest(url)

	logrus.Debugf("start to download stream in seed pattern, url: %s, header: %v, range: [%d, %d]", url,
		header, reqRange.StartIndex, reqRange.EndIndex)

	hd := make(map[string]string)
	for k, v := range header {
		hd[k] = v[0]
	}

	rr := &rangeRequest{
		url:    url,
		off:    reqRange.StartIndex,
		size:   reqRange.EndIndex - reqRange.StartIndex + 1,
		header: hd,
	}

	for i := 0; i < maxTry; i++ {
		// try to get the peer by internal schedule
		dwInfos := m.sm.Schedule(ctx, rr)
		if len(dwInfos) == 0 {
			// try to apply to be the seed node
			m.tryToApplyForSeedNode(m.ctx, url, header)
			continue
		}

		return m.runStream(ctx, rr, dwInfos)
	}

	return nil, errortypes.NewHTTPError(http.StatusInternalServerError, "failed to select a peer to download")
}

func (m *Manager) runStream(ctx context.Context, rr basic.RangeRequest, peers []*basic.SchedulePeerInfo) (io.ReadCloser, error) {
	var (
		rc  io.ReadCloser
		err error
	)

	for _, peer := range peers {
		rc, err = m.tryToDownload(ctx, peer, rr)
		if err != nil {
			continue
		}
	}

	if rc == nil {
		return nil, fmt.Errorf("failed to select a peer to download")
	}

	var (
		pr, pw = io.Pipe()
	)

	// todo: divide request data into pieces in consideration of peer disconnect.
	go func(sourceRc io.ReadCloser) {
		cw := NewClientWriter()
		notify, err := cw.Run(ctx, pw)
		if err != nil {
			pw.CloseWithError(err)
			sourceRc.Close()
			return
		}

		defer func() {
			<-notify.Done()
			sourceRc.Close()
		}()

		data, _ := NewSeedData(sourceRc, rr.Size(), true)
		cw.PutData(data)
		cw.PutData(protocol.NewEoFDistributionData())
	}(rc)

	return pr, nil
}

func (m *Manager) tryToDownload(ctx context.Context, peer *basic.SchedulePeerInfo, rr basic.RangeRequest) (io.ReadCloser, error) {
	down := NewDownloader(peer, 0, m.downloaderAPI)
	rc, err := down.Download(ctx, rr.Offset(), rr.Size())
	if err != nil {
		return nil, err
	}

	return rc, nil
}

func (m *Manager) getRangeFromHeader(header map[string][]string) (*httputils.RangeStruct, error) {
	hr := http.Header(header)
	if headerStr := hr.Get(dfgetcfg.StrRange); headerStr != "" {
		ds, err := httputils.GetRangeSE(headerStr, math.MaxInt64)
		if err != nil {
			return nil, err
		}

		// todo: support the multi range
		if len(ds) != 1 {
			return nil, fmt.Errorf("not support multi range")
		}

		// if EndIndex is max int64, set EndIndex to (StartIndex - 1),
		// so that means the end index is tail of file length.
		if ds[0].EndIndex == math.MaxInt64-1 {
			ds[0].EndIndex = ds[0].StartIndex - 1
		}

		return ds[0], nil
	}

	return &httputils.RangeStruct{
		StartIndex: 0,
		EndIndex:   -1,
	}, nil
}

func (m *Manager) tryToApplyForSeedNode(ctx context.Context, url string, header map[string][]string) {
	//todo: apply for seed node.

	path := uuid.New()
	cHeader := CopyHeader(header)
	hr := http.Header(cHeader)
	hr.Del(dfgetcfg.StrRange)

	asSeed, taskID := m.applyForSeedNode(url, cHeader, path)
	if !asSeed {
		waitCh := make(chan struct{})
		m.sm.ActiveFetchP2PNetwork(activeFetchSt{url: url, waitCh: waitCh})
		<-waitCh
		return
	}

	m.registerLocalSeed(url, cHeader, path, taskID, defaultBlockOrder)
	go m.tryToPrefetchSeedFile(ctx, path, taskID, defaultBlockOrder)
}

func (m *Manager) applyForSeedNode(url string, header map[string][]string, path string) (asSeed bool, seedTaskID string) {
	req := &types.RegisterRequest{
		RawURL:     url,
		TaskURL:    url,
		Cid:        m.cfg.Cid,
		Headers:    FlattenHeader(header),
		Dfdaemon:   m.cfg.Dfdaemon,
		IP:         m.cfg.IP,
		Port:       m.cfg.Port,
		Version:    m.cfg.Version,
		Identifier: m.cfg.Identifier,
		RootCAs:    m.cfg.RootCAs,
		HostName:   m.cfg.HostName,
		AsSeed:     true,
		Path:       path,
	}

	node := m.sm.GetSupernode(url)
	if node == "" {
		logrus.Errorf("failed to found supernode %s in register map", node)
		return false, ""
	}

	resp, err := m.supernodeAPI.ApplyForSeedNode(node, req)
	if err != nil {
		logrus.Errorf("failed to apply for seed node: %v", err)
		return false, ""
	}

	logrus.Debugf("ApplyForSeedNode resp body: %v", resp)

	if resp.Code != constants.Success {
		return false, ""
	}

	return resp.Data.AsSeed, resp.Data.SeedTaskID
}

// syncLocalSeed will sync local seed to all scheduler.
func (m *Manager) syncLocalSeed(path string, taskID string, sd seed.Seed) {
	m.sm.AddLocalSeed(path, taskID, sd)
}

func (m *Manager) registerLocalSeed(url string, header map[string][]string, path string, taskID string, blockOrder uint32) {
	info := seed.BaseInfo{
		URL:           url,
		Header:        header,
		BlockOrder:    blockOrder,
		ExpireTimeDur: time.Hour,
		TaskID:        taskID,
	}
	sd, err := m.seedManager.Register(path, info)
	if err == errortypes.ErrTaskIDDuplicate {
		return
	}

	if err != nil {
		logrus.Errorf("failed to register seed, info: %v, err:%v", info, err)
		return
	}

	m.syncLocalSeed(path, taskID, sd)
}

// tryToPrefetchSeedFile will try to prefetch the seed file
func (m *Manager) tryToPrefetchSeedFile(ctx context.Context, path string, taskID string, blockOrder uint32) {
	finishCh, err := m.seedManager.Prefetch(path, m.computePerDownloadSize(blockOrder))
	if err != nil {
		logrus.Errorf("failed to prefetch: %v", err)
		return
	}

	<-finishCh

	result, err := m.seedManager.GetPrefetchResult(path)
	if err != nil {
		logrus.Errorf("failed to get prefetch result: %v", err)
		return
	}

	if !result.Success {
		logrus.Warnf("path: %s, taskID: %s, prefetch result : %v", path, taskID, result)
		return
	}

	go m.monitorExpiredSeed(ctx, path)
}

// monitor the expired event of seed
func (m *Manager) monitorExpiredSeed(ctx context.Context, path string) {
	sd, err := m.seedManager.Get(path)
	if err != nil {
		logrus.Errorf("failed to get seed file %s: %v", path, err)
		return
	}

	// if a seed is prepared to be expired, the expired chan will be notified.
	expiredCh, err := m.seedManager.NotifyPrepareExpired(path)
	if err != nil {
		logrus.Errorf("failed to get expired chan of seed, url:%s, key: %s: %v", sd.GetURL(), sd.GetTaskID(), err)
		return
	}

	select {
	case <-ctx.Done():
		return
	case <-expiredCh:
		logrus.Infof("seed url: %s, key: %s, has been expired, try to clear resource of it", sd.GetURL(), sd.GetTaskID())
		break
	}

	timer := time.NewTimer(60 * time.Second)
	defer timer.Stop()

	for {
		needBreak := false
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			logrus.Infof("seed %s, url %s will be deleted after %d seconds", path, sd.GetURL(), 60)
			needBreak = true
			break
		default:
		}

		if needBreak {
			break
		}

		// report the seed prepare to delete to super node
		if m.reportSeedPrepareDelete(sd.GetURL(), sd.GetTaskID()) {
			break
		}

		time.Sleep(20 * time.Second)
	}

	// try to clear resource and report to super node
	m.removeLocalSeedFromScheduler(sd.GetURL())

	// unregister the seed file
	m.seedManager.UnRegister(path)
}

func (m *Manager) removeLocalSeedFromScheduler(url string) {
	m.sm.RemoveLocalSeed(url)
}

func (m *Manager) computePerDownloadSize(blockOrder uint32) int64 {
	return (1 << blockOrder) * int64(m.cfg.PerDownloadBlocks)
}

func (m *Manager) reportSeedPrepareDelete(url string, taskID string) bool {
	node := m.sm.GetSupernode(url)
	if node == "" {
		return true
	}

	deleted := m.reportSeedPrepareDeleteToSuperNodes(taskID, node)
	if !deleted {
		return false
	}

	return true
}

func (m *Manager) reportSeedPrepareDeleteToSuperNodes(taskID string, node string) bool {
	resp, err := m.supernodeAPI.ReportResourceDeleted(node, taskID, m.cfg.Cid)
	if err != nil {
		return true
	}

	return resp.Code == constants.CodeGetPeerDown
}

func (m *Manager) reportLocalSeedToSuperNode(path string, sd seed.Seed, targetSuperNode string) {
	req := &types.RegisterRequest{
		RawURL:     sd.GetURL(),
		TaskURL:    sd.GetURL(),
		TaskID:     sd.GetTaskID(),
		Cid:        m.cfg.Cid,
		Headers:    FlattenHeader(sd.GetHeaders()),
		Dfdaemon:   m.cfg.Dfdaemon,
		IP:         m.cfg.IP,
		Port:       m.cfg.Port,
		Version:    m.cfg.Version,
		Identifier: m.cfg.Identifier,
		RootCAs:    m.cfg.RootCAs,
		HostName:   m.cfg.HostName,
		AsSeed:     true,
		Path:       path,
	}

	resp, err := m.supernodeAPI.ReportResource(targetSuperNode, req)
	if err != nil || resp.Code != constants.Success {
		logrus.Errorf("failed to report resouce to supernode, resp: %v, err: %v", resp, err)
	}
}

// restoreLocalSeed will report local seed to supernode
func (m *Manager) restoreLocalSeed(ctx context.Context, syncLocal bool, monitor bool) {
	keys, sds, err := m.seedManager.List()
	if err != nil {
		logrus.Errorf("failed to list local seeds : %v", err)
		return
	}

	for i := 0; i < len(keys); i++ {
		node := m.sm.GetSupernode(sds[i].GetURL())
		if node == "" {
			// todo: all supernode is down?
			continue
		}

		m.reportLocalSeedToSuperNode(keys[i], sds[i], node)
		if syncLocal {
			m.syncLocalSeed(keys[i], sds[i].GetTaskID(), sds[i])
		}
		if monitor {
			go m.monitorExpiredSeed(ctx, keys[i])
		}
	}
}

func (m *Manager) reportLocalSeedsToSuperNode() {
	keys, sds, err := m.seedManager.List()
	if err != nil {
		logrus.Errorf("failed to list local seeds : %v", err)
		return
	}

	for i := 0; i < len(keys); i++ {
		targetNode := m.sm.GetSupernode(sds[i].GetURL())
		if targetNode == "" {
			continue
		}

		m.reportLocalSeedToSuperNode(keys[i], sds[i], targetNode)
	}
}
