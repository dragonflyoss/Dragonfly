package seed

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"

	"github.com/go-check/check"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

func (suite *SeedTestSuite) TestNormalSeed(c *check.C) {
	urlA := fmt.Sprintf("http://%s/fileA", suite.host)
	metaDir := filepath.Join(suite.cacheDir, "TestNormalSeed")
	// 8 KB
	blockOrder := uint32(13)

	sOpt := BaseOpt{
		BaseDir: metaDir,
		Info: BaseInfo{
			URL:        urlA,
			TaskID:     uuid.New(),
			FullLength: 500 * 1024,
			BlockOrder: blockOrder,
		},
	}

	sd, err := NewSeed(sOpt, RateOpt{DownloadRateLimiter: ratelimiter.NewRateLimiter(0, 0)}, false)
	c.Assert(err, check.IsNil)

	notifyCh, err := sd.Prefetch(32 * 1024)
	c.Assert(err, check.IsNil)

	// wait for prefetch ok
	<-notifyCh
	rs, err := sd.GetPrefetchResult()
	c.Assert(err, check.IsNil)
	c.Assert(rs.Err, check.IsNil)
	c.Assert(rs.Success, check.Equals, true)

	suite.checkFileWithSeed(c, "fileA", 500*1024, sd)

	suite.checkSeedFile(c, "fileB", 1024*1024, "TestNormalSeed-fileB", 14, 17*1024, nil)
	suite.checkSeedFile(c, "fileC", 1500*1024, "TestNormalSeed-fileC", 12, 10*1024, nil)
	suite.checkSeedFile(c, "fileD", 2048*1024, "TestNormalSeed-fileD", 13, 24*1024, nil)
	suite.checkSeedFile(c, "fileE", 9500*1024, "TestNormalSeed-fileE", 15, 100*1024, nil)
	suite.checkSeedFile(c, "fileF", 10*1024*1024, "TestNormalSeed-fileE", 15, 96*1024, nil)
}

func (suite *SeedTestSuite) TestSeedSyncRead(c *check.C) {
	var (
		start, end, rangeSize int64
	)

	urlF := fmt.Sprintf("http://%s/fileF", suite.host)
	metaDir := filepath.Join(suite.cacheDir, "TestSeedSyncRead")
	// 64 KB
	blockOrder := uint32(16)

	sOpt := BaseOpt{
		BaseDir: metaDir,
		Info: BaseInfo{
			URL:        urlF,
			TaskID:     uuid.New(),
			FullLength: 10 * 1024 * 1024,
			BlockOrder: blockOrder,
		},
	}

	now := time.Now()

	sd, err := NewSeed(sOpt, RateOpt{DownloadRateLimiter: ratelimiter.NewRateLimiter(0, 0)}, false)
	c.Assert(err, check.IsNil)

	notifyCh, err := sd.Prefetch(64 * 1024)
	c.Assert(err, check.IsNil)

	// try to download
	start = 0
	end = 0
	rangeSize = 99 * 1023
	fileLength := int64(10 * 1024 * 1024)

	for {
		end = start + rangeSize - 1
		if end >= fileLength {
			end = fileLength - 1
		}

		if start > end {
			break
		}

		startTime := time.Now()
		rc, err := sd.Download(start, end-start+1)
		logrus.Infof("in TestSeedSyncRead, Download 100KB costs time: %f second", time.Now().Sub(startTime).Seconds())
		c.Assert(err, check.IsNil)
		obtainedData, err := ioutil.ReadAll(rc)
		rc.Close()
		c.Assert(err, check.IsNil)

		startTime = time.Now()
		_, err = suite.readFromFileServer("fileF", start, end-start+1)
		c.Assert(err, check.IsNil)
		logrus.Infof("in TestSeedSyncRead, Download from source 100KB costs time: %f second", time.Now().Sub(startTime).Seconds())

		suite.checkDataWithFileServer(c, "fileF", start, end-start+1, obtainedData)
		start = end + 1
	}

	<-notifyCh
	logrus.Infof("in TestSeedSyncRead, costs time: %f second", time.Now().Sub(now).Seconds())

	rs, err := sd.GetPrefetchResult()
	c.Assert(err, check.IsNil)
	c.Assert(rs.Success, check.Equals, true)
	c.Assert(rs.Err, check.IsNil)

	suite.checkFileWithSeed(c, "fileF", fileLength, sd)
}

func (suite *SeedTestSuite) TestSeedSyncReadPerformance(c *check.C) {
	var (
		rangeSize int64
	)

	fileName := "fileH"
	fileLength := int64(100 * 1024 * 1024)
	urlF := fmt.Sprintf("http://%s/%s", suite.host, fileName)
	metaDir := filepath.Join(suite.cacheDir, "TestSeedSyncReadPerformance")
	// 128 KB
	blockOrder := uint32(17)
	sOpt := BaseOpt{
		BaseDir: metaDir,
		Info: BaseInfo{
			URL:        urlF,
			TaskID:     uuid.New(),
			FullLength: fileLength,
			BlockOrder: blockOrder,
		},
	}

	now := time.Now()

	sd, err := NewSeed(sOpt, RateOpt{DownloadRateLimiter: ratelimiter.NewRateLimiter(0, 0)}, true)
	c.Assert(err, check.IsNil)

	notifyCh, err := sd.Prefetch(128 * 1024)
	c.Assert(err, check.IsNil)

	wg := &sync.WaitGroup{}

	// try to download in 5 goroutine
	for i := 0; i < 5; i++ {
		rangeSize = 99 * 1023
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := int64(0)
			end := int64(0)

			for {
				end = start + rangeSize - 1
				if end >= fileLength {
					end = fileLength - 1
				}

				if start > end {
					break
				}

				startTime := time.Now()
				rc, err := sd.Download(start, end-start+1)
				logrus.Infof("in TestSeedSyncReadPerformance, Download 100KB costs time: %f second", time.Now().Sub(startTime).Seconds())
				c.Assert(err, check.IsNil)
				obtainedData, err := ioutil.ReadAll(rc)
				rc.Close()
				c.Assert(err, check.IsNil)

				startTime = time.Now()
				_, err = suite.readFromFileServer(fileName, start, end-start+1)
				c.Assert(err, check.IsNil)
				logrus.Infof("in TestSeedSyncReadPerformance, Download from source 100KB costs time: %f second", time.Now().Sub(startTime).Seconds())

				suite.checkDataWithFileServer(c, fileName, start, end-start+1, obtainedData)
				start = end + 1
			}
		}()
	}

	<-notifyCh
	logrus.Infof("in TestSeedSyncRead, costs time: %f second", time.Now().Sub(now).Seconds())

	wg.Wait()

	rs, err := sd.GetPrefetchResult()
	c.Assert(err, check.IsNil)
	c.Assert(rs.Success, check.Equals, true)
	c.Assert(rs.Err, check.IsNil)

	//s.checkFileWithSeed(c, fileName, fileLength, sd)
}

func (suite *SeedTestSuite) TestSeedRestore(c *check.C) {
	var (
		rangeSize int64
	)

	fileName := "fileH"
	fileLength := int64(100 * 1024 * 1024)
	urlF := fmt.Sprintf("http://%s/%s", suite.host, fileName)
	metaDir := filepath.Join(suite.cacheDir, "TestSeedRestore")
	// 128 KB
	blockOrder := uint32(17)
	sOpt := BaseOpt{
		BaseDir: metaDir,
		Info: BaseInfo{
			URL:        urlF,
			TaskID:     uuid.New(),
			FullLength: fileLength,
			BlockOrder: blockOrder,
		},
	}

	sd, err := NewSeed(sOpt, RateOpt{DownloadRateLimiter: ratelimiter.NewRateLimiter(0, 0)}, true)
	c.Assert(err, check.IsNil)

	rangeSize = 99 * 1023
	maxReadIndex := fileLength / 2

	for i := 0; i < 1; i++ {
		start := int64(0)
		end := int64(0)

		for {
			end = start + rangeSize - 1
			if end >= maxReadIndex {
				end = maxReadIndex
			}

			if start > end {
				break
			}

			startTime := time.Now()
			rc, err := sd.Download(start, end-start+1)
			logrus.Infof("in TestSeedSyncReadPerformance, Download 100KB costs time: %f second", time.Now().Sub(startTime).Seconds())
			c.Assert(err, check.IsNil)
			obtainedData, err := ioutil.ReadAll(rc)
			rc.Close()
			c.Assert(err, check.IsNil)

			startTime = time.Now()
			_, err = suite.readFromFileServer(fileName, start, end-start+1)
			c.Assert(err, check.IsNil)
			logrus.Infof("in TestSeedSyncReadPerformance, Download from source 100KB costs time: %f second", time.Now().Sub(startTime).Seconds())

			suite.checkDataWithFileServer(c, fileName, start, end-start+1, obtainedData)
			start = end + 1
		}
	}

	sd.Stop()

	// restore seed
	sd, remove, err := RestoreSeed(metaDir, RateOpt{DownloadRateLimiter: ratelimiter.NewRateLimiter(0, 0)}, nil)
	c.Assert(err, check.IsNil)
	c.Assert(remove, check.Equals, true)
	err = sd.Delete()
	c.Assert(err, check.IsNil)

	// new again
	sd, err = NewSeed(sOpt, RateOpt{DownloadRateLimiter: ratelimiter.NewRateLimiter(0, 0)}, true)
	c.Assert(err, check.IsNil)

	//localSd, ok := sd.(*seed)
	//c.Assert(ok, check.Equals, true)
	//cb := localSd.cache

	// read again from local file
	start := int64(0)
	end := int64(0)

	for {
		end = start + rangeSize - 1
		if end >= maxReadIndex {
			end = maxReadIndex
		}

		if start > end {
			break
		}

		startTime := time.Now()
		rc, err := sd.Download(start, end-start+1)
		logrus.Infof("in TestSeedSyncReadPerformance, Download 100KB costs time: %f second", time.Now().Sub(startTime).Seconds())
		c.Assert(err, check.IsNil)
		obtainedData, err := ioutil.ReadAll(rc)
		rc.Close()
		c.Assert(err, check.IsNil)

		startTime = time.Now()
		_, err = suite.readFromFileServer(fileName, start, end-start+1)
		c.Assert(err, check.IsNil)
		logrus.Infof("in TestSeedSyncReadPerformance, Download from source 100KB costs time: %f second", time.Now().Sub(startTime).Seconds())

		suite.checkDataWithFileServer(c, fileName, start, end-start+1, obtainedData)
		start = end + 1
	}

	// read next range again
	start = maxReadIndex + 1
	end = int64(0)

	for {
		end = start + rangeSize - 1
		if end >= fileLength {
			end = fileLength - 1
		}

		if start > end {
			break
		}

		startTime := time.Now()
		rc, err := sd.Download(start, end-start+1)
		logrus.Infof("in TestSeedSyncReadPerformance, Download 100KB costs time: %f second", time.Now().Sub(startTime).Seconds())
		c.Assert(err, check.IsNil)
		obtainedData, err := ioutil.ReadAll(rc)
		rc.Close()
		c.Assert(err, check.IsNil)

		startTime = time.Now()
		_, err = suite.readFromFileServer(fileName, start, end-start+1)
		c.Assert(err, check.IsNil)
		logrus.Infof("in TestSeedSyncReadPerformance, Download from source 100KB costs time: %f second", time.Now().Sub(startTime).Seconds())

		suite.checkDataWithFileServer(c, fileName, start, end-start+1, obtainedData)
		start = end + 1
	}

	localSd, ok := sd.(*seed)
	c.Assert(ok, check.Equals, true)
	localSd.syncCache()
	localSd.setFinished()

	sd.Stop()

	// restore again, and try to read all from local file.
	sd, remove, err = RestoreSeed(metaDir, RateOpt{DownloadRateLimiter: ratelimiter.NewRateLimiter(0, 0)}, nil)
	c.Assert(err, check.IsNil)
	c.Assert(remove, check.Equals, false)

	localSd, ok = sd.(*seed)
	c.Assert(ok, check.Equals, true)
	cb := localSd.cache

	// read again from local file
	start = int64(0)
	end = int64(0)

	for {
		end = start + rangeSize - 1
		if end >= fileLength {
			end = fileLength - 1
		}

		if start > end {
			break
		}

		startTime := time.Now()
		rc, err := cb.ReadStream(start, end-start+1)
		logrus.Infof("in TestSeedSyncReadPerformance, Download 100KB costs time: %f second", time.Now().Sub(startTime).Seconds())
		c.Assert(err, check.IsNil)
		obtainedData, err := ioutil.ReadAll(rc)
		rc.Close()
		c.Assert(err, check.IsNil)

		startTime = time.Now()
		_, err = suite.readFromFileServer(fileName, start, end-start+1)
		c.Assert(err, check.IsNil)
		logrus.Infof("in TestSeedSyncReadPerformance, Download from source 100KB costs time: %f second", time.Now().Sub(startTime).Seconds())

		suite.checkDataWithFileServer(c, fileName, start, end-start+1, obtainedData)
		start = end + 1
	}

	sd.Delete()
}
