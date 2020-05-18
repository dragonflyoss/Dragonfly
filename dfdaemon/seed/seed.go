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
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/pkg/bitmap"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"

	"github.com/sirupsen/logrus"
)

const (
	FinishedStatus = "finished"
	FetchingStatus = "fetching"
	InitialStatus  = "initial"
	DeadStatus     = "dead"
)

var (
	errContextDone = fmt.Errorf("context done")
)

// Seed describes the seed file which represents the resource file defined by taskUrl.
type Seed interface {
	// Prefetch will start to download seed file to local cache.
	Prefetch(perDownloadSize int64) (<-chan struct{}, error)

	// GetPrefetchResult should be called after notify by prefetch chan.
	GetPrefetchResult() (PreFetchResult, error)

	// Delete will delete the local cache and release the resource.
	Delete() error

	// Download providers the range download, if local cache of seed do not include the range,
	// it will download the range data from rss and reply to request.
	Download(off int64, size int64) (io.ReadCloser, error)

	// stop the internal loop and release execution resource.
	Stop()

	// GetFullSize gets the full size of seed file.
	GetFullSize() int64

	// GetStatus gets the status of seed file.
	GetStatus() string

	// GetURL gets the url of seed file.
	GetURL() string

	// GetHeaders gets the headers of seed file.
	GetHeaders() map[string][]string

	// GetHeaders gets the taskID of seed file.
	GetTaskID() string
}

// seed represents a seed which could be downloaded by other peers
type seed struct {
	sync.RWMutex

	Header      map[string][]string `json:"header"`
	URL         string              `json:"url"`
	ContentPath string              `json:"contentPath"`
	FullSize    int64               `json:"fullSize"`
	TaskID      string              `json:"taskId"`
	Status      string              `json:"status"`

	// BlockOrder should be [10,31], it means the order of block size.
	BlockOrder uint32 `json:"blockOrder"`

	// if OpenMemoryCache sets, cacheBuffer will store seed block in memory and asynchronously refresh to local file.
	OpenMemoryCache bool `json:"openMemoryCache"`

	cache      cacheBuffer
	rate       *ratelimiter.RateLimiter
	uploadRate *ratelimiter.RateLimiter

	metaPath    string
	metaBakPath string

	baseDir string

	down downloader

	// block info
	blockMeta *bitmap.BitMap

	// lockBlock
	lockBlock *bitmap.BitMap

	// the max size of cache is (blockSize * MaxInt32)
	blockSize int32
	blocks    uint32

	// if block is downloading, it set wait chan in blockWaitChMap, and set bits in lockBlock.
	blockWaitChMap map[uint32]chan struct{}

	// prefetch result
	prefetchRs PreFetchResult
	prefetchCh chan struct{}

	// internal context
	doneCtx context.Context
	cancel  context.CancelFunc

	// when call Download(), run the downPreFunc first.
	downPreFunc func(sd Seed)
}

func NewSeed(base BaseOpt, rate RateOpt, openMemoryCache bool) (Seed, error) {
	if base.Info.FullLength == 0 {
		return nil, fmt.Errorf("full size should be set")
	}

	if base.Info.BlockOrder < 10 || base.Info.BlockOrder > 31 {
		return nil, fmt.Errorf("block order should be [10,31]")
	}

	err := os.MkdirAll(base.BaseDir, 0744)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	sd := &seed{
		Status:     InitialStatus,
		URL:        base.Info.URL,
		Header:     base.Info.Header,
		FullSize:   base.Info.FullLength,
		TaskID:     base.Info.TaskID,
		BlockOrder: base.Info.BlockOrder,
		baseDir:    base.BaseDir,
		down:       newLocalDownloader(base.Info.URL, base.Info.Header, rate.DownloadRateLimiter, openMemoryCache),
		//UploadRate: sm.UploadRate,
		prefetchCh:      make(chan struct{}),
		blockWaitChMap:  make(map[uint32]chan struct{}),
		OpenMemoryCache: openMemoryCache,
		doneCtx:         ctx,
		cancel:          cancel,
		downPreFunc:     base.downPreFunc,
	}

	err = sd.initParam(base.BaseDir)
	if err != nil {
		return nil, err
	}

	cache, err := newFileCacheBuffer(sd.ContentPath, base.Info.FullLength, true, openMemoryCache, base.Info.BlockOrder)
	if err != nil {
		return nil, err
	}

	sd.cache = cache
	err = sd.storeMetaData()
	if err != nil {
		sd.Delete()
		return nil, err
	}

	go sd.syncCacheLoop(sd.doneCtx)

	return sd, nil
}

func RestoreSeed(seedDir string, rate RateOpt, downPreFunc func(sd Seed)) (s Seed, remove bool, err error) {
	sd := &seed{
		metaPath:    filepath.Join(seedDir, "meta.json"),
		downPreFunc: downPreFunc,
		baseDir:     seedDir,
	}
	// restore metadata
	metaData, err := ioutil.ReadFile(sd.metaPath)
	if err != nil {
		return nil, false, err
	}

	if err = json.Unmarshal(metaData, sd); err != nil {
		return nil, false, err
	}

	err = sd.initParam(seedDir)
	if err != nil {
		return nil, false, err
	}

	if sd.Status != FinishedStatus {
		return sd, true, nil
	}

	// init downloader and cachebuffer
	sd.down = newLocalDownloader(sd.URL, sd.Header, rate.DownloadRateLimiter, sd.OpenMemoryCache)
	cache, err := newFileCacheBuffer(sd.ContentPath, sd.FullSize, false, sd.OpenMemoryCache, sd.BlockOrder)
	if err != nil {
		return nil, false, err
	}

	sd.cache = cache

	if sd.Status == FinishedStatus || sd.Status == DeadStatus {
		return sd, false, nil
	}

	sd.prefetchCh = make(chan struct{})
	sd.blockWaitChMap = make(map[uint32]chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	sd.doneCtx = ctx
	sd.cancel = cancel

	go sd.syncCacheLoop(sd.doneCtx)

	return sd, false, nil
}

func (sd *seed) Stop() {
	if sd.cancel != nil {
		sd.cancel()
	}

	status := sd.GetStatus()
	if status != FinishedStatus && status != DeadStatus {
		sd.syncCache()
	}

	if sd.cache != nil {
		sd.cache.Close()
	}
}

// Prefetch will prefetch data to buffer, and its download rate will be limited.
func (sd *seed) Prefetch(perDownloadSize int64) (<-chan struct{}, error) {
	sd.Lock()
	defer sd.Unlock()

	if sd.Status != InitialStatus {
		return sd.prefetchCh, nil
	}

	sd.Status = FetchingStatus

	err := sd.storeMetaData()
	if err != nil {
		return nil, err
	}

	go func() {
		err := sd.prefetch(sd.doneCtx, perDownloadSize)

		sd.Lock()
		defer sd.Unlock()

		if err != nil {
			sd.prefetchRs = PreFetchResult{
				Success: false,
				Err:     err,
			}
			sd.Status = InitialStatus
		} else {
			sd.prefetchRs = PreFetchResult{
				Success: true,
				Err:     nil,
			}
			sd.Status = FinishedStatus
		}

		close(sd.prefetchCh)

		err = sd.storeMetaData()
		if err != nil {
			logrus.Errorf("failed to store meta data: %v", err)
		}
	}()

	return sd.prefetchCh, nil
}

func (sd *seed) GetPrefetchResult() (PreFetchResult, error) {
	sd.RLock()
	defer sd.RUnlock()

	if sd.Status == FetchingStatus {
		return PreFetchResult{}, fmt.Errorf("prefetch not finished")
	}

	return sd.prefetchRs, nil
}

func (sd *seed) Delete() error {
	sd.Lock()
	defer sd.Unlock()

	sd.Status = DeadStatus
	if sd.cancel != nil {
		sd.cancel()
	}

	sd.clearResource()

	return nil
}

func (sd *seed) Download(off int64, size int64) (io.ReadCloser, error) {
	off, size, err := sd.checkReadStreamParam(off, size)
	if err != nil {
		return nil, err
	}

	// if seed is not finished status, try to download blocks
	if sd.GetStatus() != FinishedStatus {
		err = sd.tryDownloadAndWaitReady(off, off+size-1, false)
		if err != nil {
			return nil, err
		}
	}

	if sd.downPreFunc != nil {
		sd.downPreFunc(sd)
	}

	return sd.cache.ReadStream(off, size)
}

func (sd *seed) GetFullSize() int64 {
	sd.RLock()
	defer sd.RUnlock()

	return sd.FullSize
}

func (sd *seed) GetStatus() string {
	sd.RLock()
	defer sd.RUnlock()

	return sd.Status
}

func (sd *seed) GetTaskID() string {
	sd.RLock()
	defer sd.RUnlock()

	return sd.TaskID
}

func (sd *seed) GetURL() string {
	sd.RLock()
	defer sd.RUnlock()

	return sd.URL
}

func (sd *seed) GetHeaders() map[string][]string {
	sd.RLock()
	defer sd.RUnlock()

	return sd.Header
}

func (sd *seed) initParam(metaDir string) error {
	var (
		err error
	)

	// init path
	metaPath := filepath.Join(metaDir, "meta.json")
	metaBakPath := filepath.Join(metaDir, "meta.json.bak")
	contentPath := filepath.Join(metaDir, "content")

	sd.metaPath = metaPath
	sd.metaBakPath = metaBakPath
	sd.ContentPath = contentPath

	// init block bitmap
	blockSize := 1 << sd.BlockOrder
	blocks := sd.FullSize / int64(blockSize)
	if (sd.FullSize % int64(blockSize)) > 0 {
		blocks++
	}

	sd.blocks = uint32(blocks)
	sd.blockMeta, err = bitmap.NewBitMapWithNumBits(uint32(blocks), false)
	if err != nil {
		return err
	}

	sd.lockBlock, err = bitmap.NewBitMapWithNumBits(uint32(blocks), false)
	if err != nil {
		return err
	}

	return nil
}

func (sd *seed) checkReadStreamParam(off int64, size int64) (int64, int64, error) {
	sd.RLock()
	defer sd.RUnlock()

	if sd.Status == DeadStatus {
		return 0, 0, fmt.Errorf("dead seed")
	}

	if off < 0 {
		off = 0
	}

	// if size <= 0, set range to [off, fullSize - 1]
	if size <= 0 {
		size = sd.FullSize - off
	}

	if off+size > sd.FullSize {
		return 0, 0, errortypes.NewHTTPError(http.StatusRequestedRangeNotSatisfiable, "out of range")
	}

	return off, size, nil
}

// alignWithBlock will align bytes range to block size, and return the block range.
func (sd *seed) alignWithBlock(start int64, end int64) (uint32, uint32) {
	return uint32(start >> sd.BlockOrder), uint32(end >> sd.BlockOrder)
}

// tryDownloadAndWaitReady downloads blocks which include range [start, end] and wait for ready.
func (sd *seed) tryDownloadAndWaitReady(start, end int64, rateLimit bool) error {
	var (
		allCosts  time.Duration
		metaCosts time.Duration
		waitCount int
	)

	allStartTime := time.Now()
	startBlock, endBlock := sd.alignWithBlock(start, end)
	logrus.Debugf("start to download, start-end: [%d-%d], block[%d-%d]\n", start, end, startBlock, endBlock)

	defer func() {
		allCosts = time.Now().Sub(allStartTime)
		logrus.Debugf("download finished, start-end: [%d-%d], block[%d-%d], wait count: %d, all cost time: %f seconds, "+
			"metaCosts costs time: %f seconds.\n", start, end, startBlock, endBlock, waitCount, allCosts.Seconds(), metaCosts.Seconds())
	}()

	for {
		waitChs := sd.tryDownload(startBlock, endBlock, rateLimit)
		if len(waitChs) == 0 {
			return nil
		}

		metaCosts = time.Now().Sub(allStartTime)
		waitCount++

		// wait for the chan
		for _, ch := range waitChs {
			// todo: set the timeout, if timeout, try to direct download again.
			select {
			case <-ch:
				break
			}
		}
	}
}

func (sd *seed) tryDownload(startBlock, endBlock uint32, rateLimit bool) (waitChs []chan struct{}) {
	rs, _ := sd.blockMeta.Get(startBlock, endBlock, false)
	// if all bits is set, it means the range has been downloaded.
	if len(rs) == 0 {
		return nil
	}

	nextDownloadBlocks := sd.lockBlocksForPrepareDownload(startBlock, endBlock)

	// downloadBlocks will download blocks asynchronously.
	sd.downloadBlocks(nextDownloadBlocks, rateLimit)

	return sd.getWaitChans(startBlock, endBlock)
}

func (sd *seed) downloadToFile(start, end int64, rateLimit bool) error {
	timeout := netutils.CalculateTimeout(end-start+1, 0, config.DefaultMinRate, 10*time.Second)
	_, err := sd.down.DownloadToWriterAt(context.Background(), httputils.RangeStruct{StartIndex: start, EndIndex: end},
		timeout, start, sd.cache, rateLimit)

	return err
}

func (sd *seed) clearResource() {
	os.Remove(sd.metaBakPath)
	os.Remove(sd.metaPath)
	if sd.cache != nil {
		sd.cache.Remove()
	}

	os.Remove(sd.baseDir)
}

// downloadBlocks downloads the blocks, it should be sync called.
func (sd *seed) downloadBlocks(blocks []uint32, rateLimit bool) {
	for i := 0; i < len(blocks); i++ {
		go sd.downloadBlock(blocks[i], blocks[i], rateLimit)
	}
}

func (sd *seed) downloadBlock(blockStartIndex, blockEndIndex uint32, rateLimit bool) {
	startBytes := int64(blockStartIndex) << sd.BlockOrder
	endBytes := int64(blockEndIndex+1)<<sd.BlockOrder - 1
	if endBytes >= sd.FullSize {
		endBytes = sd.FullSize - 1
	}

	defer func() {
		sd.unlockBlocks(blockStartIndex, blockEndIndex)
	}()

	fmt.Printf("start to download file range [%d, %d]\n", startBytes, endBytes)
	err := sd.downloadToFile(startBytes, endBytes, rateLimit)
	if err != nil {
		logrus.Errorf("failed to download to file: %v", err)
		return
	}

	sd.blockMeta.Set(blockStartIndex, blockEndIndex, true)
}

// lockBlocksForPrepareDownload  lock the range, will return next downloading blocks.
func (sd *seed) lockBlocksForPrepareDownload(startBlock, endBlock uint32) (blocks []uint32) {
	sd.Lock()
	defer sd.Unlock()

	needDownloadBlocks, _ := sd.blockMeta.Get(startBlock, endBlock, false)
	unDownloadBlocks := []*bitmap.BitsRange{}
	for _, r := range needDownloadBlocks {
		br, _ := sd.lockBlock.Get(r.StartIndex, r.EndIndex, false)
		unDownloadBlocks = append(unDownloadBlocks, br...)
	}

	ret := []uint32{}
	// lock the unDownloadBlocks
	for _, r := range unDownloadBlocks {
		for i := r.StartIndex; i <= r.EndIndex; i++ {
			sd.blockWaitChMap[i] = make(chan struct{})
			ret = append(ret, i)
		}

		// set bits  in lockBlock to tell other goroutinue the range has been locked
		sd.lockBlock.Set(r.StartIndex, r.EndIndex, true)
	}

	return ret
}

func (sd *seed) unlockBlocks(startBlock, endBlock uint32) {
	sd.Lock()
	defer sd.Unlock()

	for i := startBlock; i <= endBlock; i++ {
		ch, ok := sd.blockWaitChMap[i]
		if !ok {
			continue
		}

		close(ch)
		delete(sd.blockWaitChMap, i)
	}

	sd.lockBlock.Set(startBlock, endBlock, false)
}

func (sd *seed) getWaitChans(startBlock, endBlock uint32) []chan struct{} {
	sd.RLock()
	defer sd.RUnlock()

	res := []chan struct{}{}
	lockBlocks, _ := sd.lockBlock.Get(startBlock, endBlock, true)
	for _, r := range lockBlocks {
		for i := r.StartIndex; i <= r.EndIndex; i++ {
			ch, ok := sd.blockWaitChMap[i]
			if !ok {
				continue
			}

			res = append(res, ch)
		}
	}

	return res
}

func (sd *seed) syncCacheLoop(ctx context.Context) {
	if !sd.OpenMemoryCache {
		return
	}

	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			// if seed is finished, break the loop.
			if sd.GetStatus() == FinishedStatus {
				return
			}

			err := sd.syncCache()
			if err != nil {
				logrus.Errorf("sync cache failed: %v", err)
			}
		}
	}
}

// syncCache will sync the memory cache to local file
func (sd *seed) syncCache() error {
	return sd.cache.Sync()
}

func (sd *seed) setFinished() error {
	sd.Lock()
	defer sd.Unlock()

	sd.Status = FinishedStatus
	return sd.storeMetaData()
}

func (sd *seed) storeMetaData() error {
	data, err := json.Marshal(sd)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(sd.metaBakPath, data, 0644)
	if err != nil {
		return err
	}

	return os.Rename(sd.metaBakPath, sd.metaPath)
}

func (sd *seed) prefetch(ctx context.Context, perDownloadSize int64) error {
	blocks := int32(perDownloadSize >> sd.BlockOrder)
	if blocks == 0 {
		blocks = 1
	}

	var err error
	var try int
	var maxTry = 10

	for {
		if try > maxTry {
			return fmt.Errorf("try %d times to download file %s failed", try, sd.URL)
		}

		try++

		err = sd.prefetchAllBlocks(ctx, blocks)
		if err == errContextDone {
			return errContextDone
		}

		if err != nil {
			logrus.Errorf("failed to prefetch file %s: %v", sd.URL, err)
			continue
		}

		// prefetch success, try to check all blocks
		rs, _ := sd.blockMeta.Get(0, uint32(sd.blocks-1), false)
		if len(rs) == 0 {
			break
		}
		//else try again
	}

	return nil
}

func (sd *seed) prefetchAllBlocks(ctx context.Context, perDownloadBlocks int32) error {
	var (
		blockIndex uint32
		i          int32
	)

	// pendingCh is a queue which uses for producer/consumer model.
	// when a block try to download, it consumes from pendingCh;
	// after a block download finish, it produces to pendingCh;
	// by set the buffer size of pendingCh, we could control
	// the max concurrent downloading blocks.
	pendingCh := make(chan struct{}, perDownloadBlocks)
	for i = 0; i < perDownloadBlocks; i++ {
		pendingCh <- struct{}{}
	}

	for {
		if blockIndex >= sd.blocks {
			break
		}

		select {
		case <-ctx.Done():
			return errContextDone
		case <-pendingCh:
			break
		}

		waitChs := sd.tryDownload(blockIndex, blockIndex, true)
		if len(waitChs) > 0 {
			go func(chs []chan struct{}) {
				for _, ch := range chs {
					select {
					case <-ctx.Done():
						return
					case <-ch:
						break
					}
				}

				pendingCh <- struct{}{}
			}(waitChs)
		} else {
			pendingCh <- struct{}{}
		}

		blockIndex++
	}

	// wait for all downloading goroutine
	for i = 0; i < perDownloadBlocks; i++ {
		select {
		case <-ctx.Done():
			return errContextDone
		case <-pendingCh:
			break
		}
	}

	return nil
}
