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
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"
	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"
	"github.com/dragonflyoss/Dragonfly/supernode/httpclient"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	localSeedManager Manager
	once             sync.Once
)

const (
	DefaultDownloadConcurrency = 4
	MaxDownloadConcurrency     = 8
	MinTotalLimit              = 4

	defaultGcInterval = 2 * time.Minute

	// 128KB
	defaultBlockOrder = 17
)

// Manager is an interface which manages the seeds.
type Manager interface {
	// Register a seed.
	Register(key string, info BaseInfo) (Seed, error)

	// UnRegister seed by key.
	UnRegister(key string) error

	// RefreshExpireTime refreshes expire time of seed.
	RefreshExpireTime(key string, expireTimeDur time.Duration) error

	// NotifyPrepareExpired notify the user the seed should be deleted later, and user decides when
	// to unregister it.
	NotifyPrepareExpired(key string) (<-chan struct{}, error)

	// Prefetch will add seed to the prefetch list, and then prefetch by the concurrent limit.
	Prefetch(key string, perDownloadSize int64) (<-chan struct{}, error)

	// GetPrefetchResult should be called after notify by prefetch chan.
	GetPrefetchResult(key string) (PreFetchResult, error)

	// Get gets the seed by key.
	Get(key string) (Seed, error)

	// List lists the seeds.
	List() ([]string, []Seed, error)

	// Stop stops the SeedManager.
	Stop()
}

// seedWrapObj wraps the seed and expired info.
type seedWrapObj struct {
	sync.RWMutex
	sd Seed
	// if seed prefetch
	prefetchCh chan struct{}

	// if seed is prepared to be expired, the prepareExpiredCh will be closed.
	prepareExpiredCh chan struct{}

	//prefetchRs  PreFetchResult
	resultAcquirer PreFetchResultAcquirer

	Key             string        `json:"key"`
	ExpireTimeDur   time.Duration `json:"expireTimeDur"`
	ExpireTime      time.Time     `json:"expireTime"`
	PerDownloadSize int64         `json:"perDownloadSize"`

	// if prefetch has been called, set prefetch to true to prevent other goroutine prefetch again.
	prefetch    bool
	metaPath    string
	metaBakPath string
}

func (sw *seedWrapObj) isExpired() bool {
	sw.RLock()
	defer sw.RUnlock()

	// if not finished, do not gc.
	if sw.sd.GetStatus() != FinishedStatus {
		return false
	}

	// if expire time dur is 0, never expired.
	if sw.ExpireTimeDur == 0 || sw.ExpireTime.IsZero() {
		return false
	}

	return sw.ExpireTime.Before(time.Now())
}

func (sw *seedWrapObj) refreshExpireTime(expireTimeDur time.Duration) error {
	sw.Lock()
	defer sw.Unlock()

	if sw.prepareExpiredCh == nil {
		sw.prepareExpiredCh = make(chan struct{})
	}

	if expireTimeDur != 0 {
		sw.ExpireTimeDur = expireTimeDur
	}

	sw.ExpireTime = time.Now().Add(sw.ExpireTimeDur)
	return sw.storeMetaDataWithoutLock()
}

func (sw *seedWrapObj) notifyPrepareExpired() {
	sw.Lock()
	defer sw.Unlock()

	if sw.prepareExpiredCh != nil {
		close(sw.prepareExpiredCh)
	}

	sw.prepareExpiredCh = nil
}

func (sw *seedWrapObj) getNotifyPrepareExpiredCh() chan struct{} {
	sw.RLock()
	defer sw.RUnlock()

	return sw.prepareExpiredCh
}

func (sw *seedWrapObj) release() {
	sw.Lock()
	defer sw.Unlock()

	if sw.prepareExpiredCh != nil {
		close(sw.prepareExpiredCh)
	}

	if sw.sd != nil {
		sw.sd.Stop()
		sw.sd.Delete()
	}

	os.Remove(sw.metaBakPath)
	os.Remove(sw.metaPath)
}

func (sw *seedWrapObj) storeMetaData() error {
	sw.RLock()
	defer sw.RUnlock()

	return sw.storeMetaDataWithoutLock()
}

func (sw *seedWrapObj) storeMetaDataWithoutLock() error {
	data, err := json.Marshal(sw)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(sw.metaBakPath, data, 0644)
	if err != nil {
		return err
	}

	return os.Rename(sw.metaBakPath, sw.metaPath)
}

func (sw *seedWrapObj) getPrefetchResult() (PreFetchResult, error) {
	sw.Lock()
	defer sw.Unlock()

	if sw.resultAcquirer == nil {
		return PreFetchResult{}, fmt.Errorf("wait for prefetch")
	}

	return sw.resultAcquirer.Result()
}

type seedManager struct {
	sync.Mutex

	// the context which is monitor by all loop
	ctx context.Context
	// call cancelFn to stop all loop
	cancelFn func()

	storeDir        string
	concurrentLimit int
	totalLimit      int

	// if current limit overheads or equal than highLimit, it will started to weed out process.
	highLimit int

	// expire process will be stopped util current limit lows lowLimit.
	lowLimit int

	startWeedOutCh chan struct{}

	seedContainer map[string]*seedWrapObj
	// lru queue which wide out the seed file, it is thread safe
	lru *queue.LRUQueue
	// the queue wait for prefetch, it is thread safe
	waitQueue queue.Queue

	// downloadCh notify the seed to prefetch
	downloadCh chan struct{}

	originClient httpclient.OriginHTTPClient

	uploadRate *ratelimiter.RateLimiter
	downRate   *ratelimiter.RateLimiter

	defaultBlockOrder uint32
	openMemoryCache   bool
}

func NewSeedManager(opt NewSeedManagerOpt) Manager {
	once.Do(func() {
		var err error
		// todo: config the total limit
		localSeedManager, err = newSeedManager(opt)
		if err != nil {
			panic(err)
		}
	})

	return localSeedManager
}

func GetSeedManager() Manager {
	return localSeedManager
}

func newSeedManager(opt NewSeedManagerOpt) (Manager, error) {
	if opt.ConcurrentLimit > MaxDownloadConcurrency {
		opt.ConcurrentLimit = MaxDownloadConcurrency
	}

	if opt.ConcurrentLimit <= 0 {
		opt.ConcurrentLimit = DefaultDownloadConcurrency
	}

	if opt.TotalLimit == 0 {
		opt.TotalLimit = MinTotalLimit
	}

	if opt.TotalLimit < MinTotalLimit {
		return nil, fmt.Errorf("TotalLimit should be in bigger than %d", MinTotalLimit)
	}

	if opt.DownloadBlockOrder == 0 {
		opt.DownloadBlockOrder = defaultBlockOrder
	}

	if opt.DownloadBlockOrder < 10 || opt.DownloadBlockOrder > 31 {
		return nil, fmt.Errorf("downloadBlockOrder should be in range[10, 31]")
	}

	if opt.DownloadRate < 0 {
		opt.DownloadRate = 0
	}

	// if UploadRate sets 0, means default limit
	if opt.UploadRate < 0 {
		opt.UploadRate = 0
	}

	if opt.HighLevel > 100 || opt.HighLevel < 50 {
		return nil, fmt.Errorf("high level should be in range [50, 100]")
	}

	if opt.LowLevel >= opt.HighLevel || opt.LowLevel < 10 {
		return nil, fmt.Errorf("low level should be small than high level, and should be bigger than 10")
	}

	highLimit := int(opt.TotalLimit * int(opt.HighLevel) / 100)
	lowLimit := int(opt.TotalLimit * int(opt.LowLevel) / 100)

	if lowLimit < 2 {
		return nil, fmt.Errorf("LowLevel * TotalLimit should be bigger or euqal than 2, but now is %d", lowLimit)
	}

	downloadCh := make(chan struct{}, opt.ConcurrentLimit)
	for i := 0; i < opt.ConcurrentLimit; i++ {
		downloadCh <- struct{}{}
	}

	// mkdir store dir
	err := os.MkdirAll(opt.StoreDir, 0774)
	if err != nil {
		return nil, err
	}

	// mkdir store seed dir
	err = os.MkdirAll(filepath.Join(opt.StoreDir, "seed"), 0774)
	if err != nil {
		return nil, err
	}

	// mkdir store seed meta dir
	err = os.MkdirAll(filepath.Join(opt.StoreDir, "meta"), 0774)
	if err != nil {
		return nil, err
	}

	ctx, cancelFn := context.WithCancel(context.Background())

	sm := &seedManager{
		ctx:               ctx,
		cancelFn:          cancelFn,
		storeDir:          opt.StoreDir,
		concurrentLimit:   opt.ConcurrentLimit,
		totalLimit:        opt.TotalLimit,
		seedContainer:     make(map[string]*seedWrapObj),
		lru:               queue.NewLRUQueue(opt.TotalLimit),
		waitQueue:         queue.NewQueue(0),
		downloadCh:        downloadCh,
		originClient:      httpclient.NewOriginClient(),
		uploadRate:        ratelimiter.NewRateLimiter(opt.UploadRate, 2),
		downRate:          ratelimiter.NewRateLimiter(opt.DownloadRate, 2),
		defaultBlockOrder: opt.DownloadBlockOrder,
		openMemoryCache:   opt.OpenMemoryCache,
		highLimit:         highLimit,
		lowLimit:          lowLimit,
		startWeedOutCh:    make(chan struct{}, 10),
	}

	sm.restore(ctx)

	go sm.prefetchLoop(ctx)
	go sm.gcLoop(ctx)
	go sm.weedOutLoop(ctx)

	return sm, nil
}

func (sm *seedManager) Register(key string, info BaseInfo) (Seed, error) {
	sm.Lock()
	defer sm.Unlock()

	obj, ok := sm.seedContainer[key]
	if ok {
		return obj.sd, errortypes.ErrTaskIDDuplicate
	}

	if info.BlockOrder == 0 {
		info.BlockOrder = sm.defaultBlockOrder
	}

	if info.FullLength == 0 {
		// get seed file length
		hd := map[string]string{}
		for k, v := range info.Header {
			hd[k] = v[0]
		}

		fullSize, err := sm.getHTTPFileLength(key, info.URL, hd)
		if err != nil {
			return nil, err
		}

		info.FullLength = fullSize
	}

	opt := BaseOpt{
		BaseDir: filepath.Join(sm.storeDir, "seed", key),
		Info:    info,
		downPreFunc: func(sd Seed) {
			sm.downPreFunc(key, sd)
		},
	}

	sd, err := NewSeed(opt, RateOpt{DownloadRateLimiter: sm.downRate}, sm.openMemoryCache)
	if err != nil {
		return nil, err
	}

	sm.seedContainer[key] = &seedWrapObj{
		sd:               sd,
		prefetchCh:       make(chan struct{}),
		prepareExpiredCh: make(chan struct{}),
		Key:              key,
		metaPath:         sm.seedWrapMetaPath(key),
		metaBakPath:      sm.seedWrapMetaBakPath(key),
		ExpireTimeDur:    info.ExpireTimeDur,
	}

	return sd, nil
}

func (sm *seedManager) UnRegister(key string) error {
	sw, err := sm.getSeedWrapObj(key)
	if err != nil {
		return err
	}

	sm.gcSeed(key, sw.sd)
	return nil
}

func (sm *seedManager) RefreshExpireTime(key string, expireTimeDur time.Duration) error {
	sw, err := sm.getSeedWrapObj(key)
	if err != nil {
		return err
	}

	err = sw.refreshExpireTime(expireTimeDur)
	if err != nil {
		return err
	}

	sm.updateLRU(key, sw.sd)
	return nil
}

func (sm *seedManager) NotifyPrepareExpired(key string) (<-chan struct{}, error) {
	sw, err := sm.getSeedWrapObj(key)
	if err != nil {
		return nil, err
	}

	ch := sw.getNotifyPrepareExpiredCh()
	// if chan is nil, new and close one, and return.
	if ch == nil {
		ch = make(chan struct{})
		close(ch)
	}

	return ch, nil
}

func (sm *seedManager) List() ([]string, []Seed, error) {
	sm.Lock()
	defer sm.Unlock()

	ret := make([]Seed, len(sm.seedContainer))
	keys := make([]string, len(sm.seedContainer))
	i := 0
	for _, obj := range sm.seedContainer {
		ret[i] = obj.sd
		keys[i] = obj.Key
		i++
	}

	return keys, ret, nil
}

func (sm *seedManager) Prefetch(key string, perDownloadSize int64) (<-chan struct{}, error) {
	sw, err := sm.getSeedWrapObj(key)
	if err != nil {
		return nil, err
	}

	sw.Lock()
	defer sw.Unlock()
	if !sw.prefetch {
		sw.prefetch = true
		sw.PerDownloadSize = perDownloadSize
		sw.prefetchCh = make(chan struct{})
		err = sw.storeMetaDataWithoutLock()
		if err != nil {
			return nil, err
		}

		// add seed to waitQueue, it will be polled out by handles goroutine to start to prefetch
		sm.waitQueue.Put(sw)
	}

	return sw.prefetchCh, nil
}

func (sm *seedManager) GetPrefetchResult(key string) (PreFetchResult, error) {
	sw, err := sm.getSeedWrapObj(key)
	if err != nil {
		return PreFetchResult{}, err
	}

	return sw.getPrefetchResult()
}

func (sm *seedManager) Get(key string) (Seed, error) {
	obj, err := sm.getSeedWrapObj(key)
	if err != nil {
		return nil, err
	}

	return obj.sd, nil
}

func (sm *seedManager) Stop() {
	if sm.cancelFn != nil {
		sm.cancelFn()
	}
}

func (sm *seedManager) restore(ctx context.Context) {
	sm.restoreSeedWraps()
	sm.restoreSeeds()

	var sws, validSws []*seedWrapObj

	for _, sw := range sm.seedContainer {
		sws = append(sws, sw)
	}

	// check if seed is expired
	for _, sw := range sws {
		if sw.isExpired() {
			// if expired, release the seed file.
			delete(sm.seedContainer, sw.Key)
			sw.release()
			continue
		}

		validSws = append(validSws, sw)
	}

	// if not set expired time, consider it as dead line is after all seed.
	sort.Slice(validSws, func(i, j int) bool {
		if validSws[i].ExpireTime.IsZero() {
			return true
		}

		if validSws[j].ExpireTime.IsZero() {
			return false
		}

		return validSws[i].ExpireTime.Before(validSws[j].ExpireTime)
	})

	// update seed to url queue
	for _, sw := range validSws {
		sm.updateLRU(sw.Key, sw.sd)
	}
}

// restoreSeedWraps reads the metadata of seed wrap object which wraps the seed.
func (sm *seedManager) restoreSeedWraps() {
	seedWrapDir := filepath.Join(sm.storeDir, "meta")
	fis, err := ioutil.ReadDir(seedWrapDir)
	if err != nil {
		logrus.Errorf("failed to read seed meta dir %s: %v", seedWrapDir, err)
		return
	}

	for _, fi := range fis {
		if !fi.Mode().IsRegular() {
			continue
		}

		if !strings.HasSuffix(fi.Name(), ".meta") {
			continue
		}

		key := strings.TrimSuffix(fi.Name(), ".meta")
		sw, err := sm.restoreSeedWrapObj(key, filepath.Join(seedWrapDir, fi.Name()))
		if err != nil {
			logrus.Errorf("failed to restore seed wrap obj, key: %s, error: %v", key, err)
			continue
		}

		sm.seedContainer[key] = sw
	}
}

// restoreSeeds reads the seed object.
func (sm *seedManager) restoreSeeds() {
	seedDir := filepath.Join(sm.storeDir, "seed")
	fis, err := ioutil.ReadDir(seedDir)
	if err != nil {
		logrus.Errorf("failed to read seed dir %s: %v", seedDir, err)
		return
	}

	tmpSeedMap := map[string]Seed{}

	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}

		key := fi.Name()
		if _, ok := sm.seedContainer[key]; !ok {
			logrus.Errorf("seed dir %s not found in meta data", key)
			continue
		}

		sd, remove, err := RestoreSeed(filepath.Join(seedDir, key), RateOpt{DownloadRateLimiter: sm.downRate}, func(sd Seed) {
			sm.downPreFunc(key, sd)
		})
		if err != nil {
			logrus.Errorf("failed to restore seed %s: %v", key, err)
			continue
		}

		if remove {
			sd.Delete()
			continue
		}

		tmpSeedMap[key] = sd
	}

	sws := []*seedWrapObj{}
	for _, sw := range sm.seedContainer {
		sws = append(sws, sw)
	}

	for _, sw := range sws {
		sd, ok := tmpSeedMap[sw.Key]
		if !ok {
			logrus.Errorf("seed key %s, found the metadata, but not found seed file", sw.Key)
			delete(sm.seedContainer, sw.Key)
			sw.release()
			// todo: remove the seedWrapObj from local file system if seed file not found.
			continue
		}

		sw.sd = sd
	}
}

func (sm *seedManager) restoreSeedWrapObj(key, path string) (*seedWrapObj, error) {
	sw := &seedWrapObj{
		Key: key,
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, sw)
	if err != nil {
		return nil, err
	}

	sw.Key = key
	sw.metaPath = sm.seedWrapMetaPath(key)
	sw.metaBakPath = sm.seedWrapMetaBakPath(key)
	sw.prepareExpiredCh = make(chan struct{})
	sw.prefetchCh = make(chan struct{})

	return sw, nil
}

func (sm *seedManager) listSeedWrapObj() []*seedWrapObj {
	sm.Lock()
	defer sm.Unlock()

	ret := make([]*seedWrapObj, len(sm.seedContainer))
	i := 0
	for _, obj := range sm.seedContainer {
		ret[i] = obj
		i++
	}

	return ret
}

func (sm *seedManager) getSeedWrapObj(key string) (*seedWrapObj, error) {
	sm.Lock()
	defer sm.Unlock()

	obj, ok := sm.seedContainer[key]
	if !ok {
		return obj, errortypes.ErrDataNotFound
	}

	return obj, nil
}

// prepareGcSeed will notify the user, then let user decide to unregister the seed.
// By the way, user could control the time when to release the seed.
func (sm *seedManager) prepareGcSeed(key string, sd Seed) {
	logrus.Infof("prepare gc seed SeedKey  %s, Url %s", key, sd.GetURL())
	sw, err := sm.getSeedWrapObj(key)
	if err != nil {
		return
	}

	// delete the seed from lru queue.
	sm.Lock()
	sm.lru.Delete(key)
	sm.Unlock()

	sw.notifyPrepareExpired()
}

// gcSeed will release the seed resource.
func (sm *seedManager) gcSeed(key string, sd Seed) {
	logrus.Infof("gc seed SeedKey  %s, Url %s", key, sd.GetURL())
	sw, err := sm.getSeedWrapObj(key)
	if err != nil {
		return
	}

	// delete from map
	sm.Lock()
	delete(sm.seedContainer, key)
	sm.lru.Delete(key)
	sm.Unlock()

	sw.release()
}

// prefetchLoop poll the seeds from waitQueue and start to prefetch them.
func (sm *seedManager) prefetchLoop(ctx context.Context) {
	for {
		ob, exist := sm.waitQueue.PollTimeout(2 * time.Second)
		if !exist {
			continue
		}

		sw, ok := ob.(*seedWrapObj)
		if !ok {
			continue
		}

		select {
		case <-ctx.Done():
			return
		// 	downloadCh will control the limit of concurrent prefetch.
		case <-sm.downloadCh:
			break
		}

		go sm.downloadSeed(ctx, sw)
	}
}

func (sm *seedManager) downloadSeed(ctx context.Context, sw *seedWrapObj) {
	sw.RLock()
	perDownloadSize := sw.PerDownloadSize
	sw.RUnlock()

	waitPrefetchCh, resultAcquirer, err := sw.sd.Prefetch(perDownloadSize)
	if err != nil {
		logrus.Errorf("failed to prefetch seed file %s: %v", sw.Key, err)
		return
	}

	sw.Lock()
	sw.resultAcquirer = resultAcquirer
	sw.Unlock()

	select {
	case <-ctx.Done():
		// todo:
	case <-waitPrefetchCh:
		break
	}

	sw.Lock()
	close(sw.prefetchCh)
	sw.Unlock()

	// notify the prefetchLoop to prefetch next seed.
	sm.downloadCh <- struct{}{}

	result, err := sw.getPrefetchResult()
	if err != nil {
		return
	}

	// if prefetch success, add seed to lru queue.
	if result.Success {
		sm.updateLRU(sw.Key, sw.sd)
	}

	return
}

func (sm *seedManager) updateLRU(key string, sd Seed) {
	obsoleteKey, obsoleteData := sm.lru.Put(key, sd)
	if obsoleteKey != "" {
		go sm.prepareGcSeed(obsoleteKey, obsoleteData.(Seed))
	}

	if sm.needWeedOut() {
		go sm.startWeedOut()
	}
}

// gcLoop runs the loop to gc the seed file.
func (sm *seedManager) gcLoop(ctx context.Context) {
	ticker := time.NewTicker(defaultGcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logrus.Infof("context done, return gcLoop")
			return

		case <-ticker.C:
			logrus.Infof("start to  gc loop")
			sm.gcExpiredSeed()
		}
	}
}

// gc the  expired seed
func (sm *seedManager) gcExpiredSeed() {
	list := sm.listSeedWrapObj()
	for _, sw := range list {
		if sw.isExpired() {
			sm.prepareGcSeed(sw.Key, sw.sd)
		}
	}
}

func (sm *seedManager) seedWrapMetaPath(key string) string {
	return filepath.Join(sm.storeDir, "meta", key+".meta")
}

func (sm *seedManager) seedWrapMetaBakPath(key string) string {
	return filepath.Join(sm.storeDir, "meta", key+".meta.bak")
}

func (sm *seedManager) getHTTPFileLength(key, url string, headers map[string]string) (int64, error) {
	fileLength, code, err := sm.originClient.GetContentLength(url, headers)
	if err != nil {
		return -1, errors.Wrapf(errortypes.ErrUnknownError, "failed to get http file Length: %v", err)
	}

	if code == http.StatusUnauthorized || code == http.StatusProxyAuthRequired {
		return -1, errors.Wrapf(errortypes.ErrAuthenticationRequired, "taskID: %s,code: %d", key, code)
	}

	if code != http.StatusOK && code != http.StatusPartialContent {
		logrus.Warnf("failed to get http file length with unexpected code: %d", code)
		if code == http.StatusNotFound {
			return -1, errors.Wrapf(errortypes.ErrURLNotReachable, "taskID: %s, url: %s", key, url)
		}
		return -1, nil
	}

	return fileLength, nil
}

func (sm *seedManager) downPreFunc(key string, sd Seed) {
	sm.RefreshExpireTime(key, 0)
}

// needWeedOut checks whether to start to weed out seed.
func (sm *seedManager) needWeedOut() bool {
	currentLen := sm.lru.Len()
	return currentLen >= sm.highLimit
}

func (sm *seedManager) startWeedOut() {
	sm.startWeedOutCh <- struct{}{}
}

// runWeedOut will weed out the seed util current count of seeds lows than low limit
func (sm *seedManager) runWeedOut(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		currentLen := sm.lru.Len()
		if currentLen < sm.lowLimit {
			return
		}

		key, sd := sm.lru.Poll()
		if key == "" {
			return
		}

		sm.prepareGcSeed(key, sd.(Seed))
	}
}

func (sm *seedManager) weedOutLoop(ctx context.Context) {
	runWeedOutProcess := false
	weedOutProcessEndCh := make(chan struct{})

	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.startWeedOutCh:
			if runWeedOutProcess {
				continue
			}
			runWeedOutProcess = true
			go func() {
				sm.runWeedOut(ctx)
				// if process is end, notify by weedOutProcessEndCh.
				weedOutProcessEndCh <- struct{}{}
			}()
			break
		case <-weedOutProcessEndCh:
			runWeedOutProcess = false
			break
		}
	}
}
