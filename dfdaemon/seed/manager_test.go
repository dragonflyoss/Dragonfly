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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-check/check"
	"github.com/pborman/uuid"
)

func (suite *SeedTestSuite) TestOneSeed(c *check.C) {
	smOpt := NewSeedManagerOpt{
		StoreDir:           filepath.Join(suite.cacheDir, "TestOneSeed"),
		ConcurrentLimit:    2,
		TotalLimit:         4,
		DownloadBlockOrder: 14,
		OpenMemoryCache:    true,
		DownloadRate:       -1,
		UploadRate:         -1,
		HighLevel:          100,
		LowLevel:           90,
	}

	sm, err := newSeedManager(smOpt)
	c.Assert(err, check.IsNil)

	defer sm.Stop()

	fileName := "fileA"
	fileLength := int64(500 * 1024)

	preInfo := BaseInfo{
		// fileA: 500KB
		URL:           fmt.Sprintf("http://%s/%s", suite.host, fileName),
		ExpireTimeDur: time.Second * 10,
	}

	taskID := uuid.New()
	sd, err := sm.Register(taskID, preInfo)
	c.Assert(err, check.IsNil)

	c.Assert(sd.GetFullSize(), check.Equals, fileLength)

	finishCh, err := sm.Prefetch(taskID, 1024)
	c.Assert(err, check.IsNil)

	prefetchTimeout := time.NewTimer(time.Second * 20)
	defer prefetchTimeout.Stop()

	select {
	case <-prefetchTimeout.C:
		c.Fatalf("expected not time out")
	case <-finishCh:
		break
	}

	prefetchResult, err := sm.GetPrefetchResult(taskID)
	c.Assert(err, check.IsNil)
	c.Assert(prefetchResult.Success, check.Equals, true)
	c.Assert(prefetchResult.Canceled, check.Equals, false)
	c.Assert(prefetchResult.Err, check.IsNil)

	// download all
	rc, err := sd.Download(0, -1)
	c.Assert(err, check.IsNil)
	obtainedData, err := ioutil.ReadAll(rc)
	rc.Close()
	c.Assert(err, check.IsNil)
	suite.checkDataWithFileServer(c, fileName, 0, -1, obtainedData)

	// download 0-100*1024
	rc, err = sd.Download(0, 100*1024)
	c.Assert(err, check.IsNil)
	obtainedData, err = ioutil.ReadAll(rc)
	rc.Close()
	c.Assert(err, check.IsNil)
	suite.checkDataWithFileServer(c, fileName, 0, 100*1024, obtainedData)

	// download 100*1024-(500*1024-1)
	rc, err = sd.Download(100*1024, 400*1024)
	c.Assert(err, check.IsNil)
	obtainedData, err = ioutil.ReadAll(rc)
	rc.Close()
	c.Assert(err, check.IsNil)
	suite.checkDataWithFileServer(c, fileName, 100*1024, 400*1024, obtainedData)

	suite.checkFileWithSeed(c, fileName, fileLength, sd)

	expiredCh, err := sm.NotifyPrepareExpired(taskID)
	c.Assert(err, check.IsNil)
	// try to gc
	time.Sleep(time.Second * 11)
	lsm := sm.(*seedManager)
	// gc expired seed
	lsm.gcExpiredSeed()

	receiveExpired := false
	select {
	case <-expiredCh:
		receiveExpired = true
	default:
	}

	c.Assert(receiveExpired, check.Equals, true)

	rc, err = sd.Download(0, -1)
	c.Assert(err, check.IsNil)
	rc.Close()

	sm.UnRegister(taskID)

	_, err = sd.Download(0, -1)
	c.Assert(err, check.NotNil)

	_, err = sm.Get(taskID)
	c.Assert(err, check.NotNil)
}

func (suite *SeedTestSuite) TestManySeed(c *check.C) {
	smOpt := NewSeedManagerOpt{
		StoreDir:           filepath.Join(suite.cacheDir, "TestManySeed"),
		ConcurrentLimit:    2,
		TotalLimit:         6,
		DownloadBlockOrder: 14,
		OpenMemoryCache:    true,
		DownloadRate:       -1,
		UploadRate:         -1,
		HighLevel:          90,
		LowLevel:           85,
	}

	sm, err := newSeedManager(smOpt)
	c.Assert(err, check.IsNil)

	defer sm.Stop()

	filePaths := []string{"fileB", "fileC", "fileD", "fileE", "fileF"}
	fileLens := []int64{1024 * 1024, 1500 * 1024, 2048 * 1024, 9500 * 1024, 10 * 1024 * 1024}
	taskIDArr := make([]string, 5)
	seedArr := make([]Seed, 5)
	expireChs := make([]<-chan struct{}, 5)

	wg := &sync.WaitGroup{}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		taskIDArr[i] = uuid.New()
		sd, err := sm.Register(taskIDArr[i], BaseInfo{URL: fmt.Sprintf("http://%s/%s", suite.host, filePaths[i]),
			TaskID: taskIDArr[i], ExpireTimeDur: 30 * time.Second})
		c.Assert(err, check.IsNil)
		seedArr[i] = sd
		expireChs[i], err = sm.NotifyPrepareExpired(taskIDArr[i])
		c.Assert(err, check.IsNil)

		go func(lsd Seed, path string, fileLength int64, taskID string) {
			suite.checkSeedFileBySeedManager(c, path, fileLength, taskID, 64*1024, lsd, sm, wg)
		}(sd, filePaths[i], fileLens[i], taskIDArr[i])
	}

	wg.Wait()
	// refresh expired time
	for i := 1; i < 4; i++ {
		rc, err := seedArr[i].Download(0, 10)
		c.Assert(err, check.IsNil)
		rc.Close()
	}

	// new one seed, it may wide out the oldest one
	taskIDArr[4] = uuid.New()
	seedArr[4], err = sm.Register(taskIDArr[4], BaseInfo{URL: fmt.Sprintf("http://%s/%s", suite.host, filePaths[4]),
		TaskID: taskIDArr[4], ExpireTimeDur: 30 * time.Second})
	c.Assert(err, check.IsNil)
	expireChs[4], err = sm.NotifyPrepareExpired(taskIDArr[4])
	c.Assert(err, check.IsNil)
	suite.checkSeedFileBySeedManager(c, filePaths[4], fileLens[4], taskIDArr[4], 64*1024, seedArr[4], sm, nil)

	time.Sleep(2 * time.Second)
	receiveExpired := false
	select {
	case <-expireChs[0]:
		receiveExpired = true
		sm.UnRegister(taskIDArr[0])
	default:
	}

	c.Assert(receiveExpired, check.Equals, true)

	// check the oldest one taskIDArr[0], it should be wide out
	_, err = sm.Get(taskIDArr[0])
	c.Assert(err, check.NotNil)

	// refresh taskIDArr[1]
	sm.RefreshExpireTime(taskIDArr[1], 0)
	time.Sleep(35 * time.Second)
	lsm := sm.(*seedManager)
	// gc expired seed
	lsm.gcExpiredSeed()

	for i := 2; i < 5; i++ {
		receiveExpired := false
		select {
		case <-expireChs[i]:
			receiveExpired = true
			sm.UnRegister(taskIDArr[i])
		default:
		}

		c.Assert(receiveExpired, check.Equals, true)
	}
}

func (suite *SeedTestSuite) TestSeedRestoreInManager(c *check.C) {
	smOpt := NewSeedManagerOpt{
		StoreDir:           filepath.Join(suite.cacheDir, "TestSeedRestoreInManager"),
		ConcurrentLimit:    2,
		TotalLimit:         6,
		DownloadBlockOrder: 14,
		OpenMemoryCache:    true,
		DownloadRate:       -1,
		UploadRate:         -1,
		HighLevel:          90,
		LowLevel:           85,
	}

	sm, err := newSeedManager(smOpt)
	c.Assert(err, check.IsNil)

	defer sm.Stop()

	filePaths := []string{"fileB", "fileC", "fileD", "fileE", "fileF"}
	fileLens := []int64{1024 * 1024, 1500 * 1024, 2048 * 1024, 9500 * 1024, 10 * 1024 * 1024}
	taskIDArr := make([]string, 5)
	seedArr := make([]Seed, 5)
	expireChs := make([]<-chan struct{}, 5)

	wg := &sync.WaitGroup{}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		taskIDArr[i] = uuid.New()
		sd, err := sm.Register(taskIDArr[i], BaseInfo{URL: fmt.Sprintf("http://%s/%s", suite.host, filePaths[i]),
			TaskID: taskIDArr[i], ExpireTimeDur: time.Second * 30})
		c.Assert(err, check.IsNil)
		seedArr[i] = sd

		expireChs[i], err = sm.NotifyPrepareExpired(taskIDArr[i])
		c.Assert(err, check.IsNil)

		go func(lsd Seed, path string, fileLength int64, taskID string) {
			suite.checkSeedFileBySeedManager(c, path, fileLength, taskID, 64*1024, lsd, sm, wg)
		}(sd, filePaths[i], fileLens[i], taskIDArr[i])
	}

	wg.Wait()
	// refresh expired time, seed[0] and seed[1] will be wide out before next restore.
	sm.RefreshExpireTime(taskIDArr[0], 30*time.Second)
	sm.RefreshExpireTime(taskIDArr[1], 30*time.Second)

	for i := 2; i < 4; i++ {
		sm.RefreshExpireTime(taskIDArr[i], 180*time.Second)
	}

	// stop sm
	sm.Stop()

	time.Sleep(40 * time.Second)
	// restore seedManager
	sm, err = newSeedManager(smOpt)
	c.Assert(err, check.IsNil)

	_, seedArr, err = sm.List()
	c.Assert(len(seedArr), check.Equals, 2)

	_, err = sm.Get(taskIDArr[0])
	c.Assert(err, check.NotNil)

	_, err = sm.Get(taskIDArr[1])
	c.Assert(err, check.NotNil)

	for i := 2; i < 4; i++ {
		sd, err := sm.Get(taskIDArr[i])
		c.Assert(err, check.IsNil)

		suite.checkFileWithSeed(c, filePaths[i], fileLens[i], sd)
	}
}

func (suite *SeedTestSuite) TestSeedSyncWriteAndRead(c *check.C) {
	smOpt := NewSeedManagerOpt{
		StoreDir:           filepath.Join(suite.cacheDir, "TestSeedSyncWriteAndRead"),
		ConcurrentLimit:    2,
		TotalLimit:         6,
		DownloadBlockOrder: 14,
		OpenMemoryCache:    true,
		DownloadRate:       -1,
		UploadRate:         -1,
		HighLevel:          90,
		LowLevel:           85,
	}
	sm, err := newSeedManager(smOpt)
	c.Assert(err, check.IsNil)

	defer sm.Stop()

	filePaths := []string{"fileF"}
	fileLens := []int64{10 * 1024 * 1024}
	taskIDArr := make([]string, 1)
	taskIDArr[0] = uuid.New()

	sd, err := sm.Register(taskIDArr[0], BaseInfo{URL: fmt.Sprintf("http://%s/%s", suite.host, filePaths[0]),
		TaskID: taskIDArr[0], ExpireTimeDur: time.Second * 30})
	c.Assert(err, check.IsNil)

	finishCh, err := sm.Prefetch(taskIDArr[0], 64*1024)
	c.Assert(err, check.IsNil)

	go func(lsd Seed, key string, path string, fileLength int64) {
		<-finishCh
		prefetchResult, err := sm.GetPrefetchResult(key)
		c.Assert(err, check.IsNil)

		c.Assert(prefetchResult.Err, check.IsNil)
		c.Assert(prefetchResult.Success, check.Equals, true)
		c.Assert(prefetchResult.Canceled, check.Equals, false)
		c.Assert(lsd.GetFullSize(), check.Equals, fileLength)
		suite.checkFileWithSeed(c, path, fileLength, lsd)
	}(sd, taskIDArr[0], filePaths[0], fileLens[0])

	for {
		for i := 0; i < 5*100; i++ {
			rc, err := sd.Download(int64(i*20000), 20000)
			c.Assert(err, check.IsNil)
			obtainedData, err := ioutil.ReadAll(rc)
			rc.Close()
			c.Assert(err, check.IsNil)
			suite.checkDataWithFileServer(c, filePaths[0], int64(i*20000), 20000, obtainedData)
		}

		if sd.GetStatus() == FinishedStatus {
			break
		}
	}
}
