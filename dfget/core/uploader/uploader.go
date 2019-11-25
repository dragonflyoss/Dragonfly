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

// Package uploader implements an uploader server. It is the important role
// - peer - in P2P pattern that will wait for other P2PDownloader to download
// its downloaded files.
package uploader

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"

	"github.com/sirupsen/logrus"
)

const (
	ctype = "application/octet-stream"
)

var (
	p2p *peerServer
)

var (
	aliveQueue  = queue.NewQueue(0)
	uploaderAPI = api.NewUploaderAPI(httputils.DefaultTimeout)
)

// -----------------------------------------------------------------------------
// dfget server functions

// WaitForShutdown waits for peer server shutdown.
func WaitForShutdown() {
	if p2p != nil {
		p2p.waitForShutdown()
	}
}

// LaunchPeerServer launches a server to send piece data.
func LaunchPeerServer(cfg *config.Config) (int, error) {
	// avoid data race caused by reading and writing variable 'p2p'
	// in different routines
	var p2pPtr unsafe.Pointer

	logrus.Infof("********************")
	logrus.Infof("start peer server...")

	res := make(chan error)
	go func() {
		res <- launch(cfg, &p2pPtr)
	}()

	if err := waitForStartup(res, &p2pPtr); err != nil {
		logrus.Errorf("start peer server error:%v, exit directly", err)
		return 0, err
	}

	p2p = loadSrvPtr(&p2pPtr)
	updateServicePortInMeta(cfg.RV.MetaPath, p2p.port)
	logrus.Infof("start peer server success, host:%s, port:%d",
		p2p.host, p2p.port)
	go monitorAlive(cfg, 15*time.Second)
	return p2p.port, nil
}

func launch(cfg *config.Config, p2pPtr *unsafe.Pointer) error {
	var (
		retryCount         = 10
		port               = 0
		shouldGeneratePort = true
	)
	if cfg.RV.PeerPort > 0 {
		retryCount = 1
		port = cfg.RV.PeerPort
		shouldGeneratePort = false
	}
	for i := 0; i < retryCount; i++ {
		if shouldGeneratePort {
			port = generatePort(i)
		}
		tmp := newPeerServer(cfg, port)
		storeSrvPtr(p2pPtr, tmp)
		if err := tmp.ListenAndServe(); err != nil {
			if !strings.Contains(err.Error(), "address already in use") {
				// start failed or shutdown
				return err
			} else if uploaderAPI.PingServer(tmp.host, tmp.port) {
				// a peer server is already existing
				return nil
			}
			logrus.Warnf("start error:%v, remain retry times:%d",
				err, retryCount-i)
		}
	}
	return fmt.Errorf("start peer server error and retried at most %d times", retryCount)
}

// waitForStartup It's a goal to start 'dfget server' process and make it working
// within 300ms, such as in the case of downloading very small files, especially
// in parallel.
// The ticker which has a 5ms period can test the server whether is working
// successfully as soon as possible.
// Actually, it costs about 70ms for 'dfget client' to start a `dfget server`
// process if everything goes right without any failure. So the remaining time
// for retrying to launch server internal is about 230ms. And '233' is just
// right the smallest number which is greater than 230, a prime, and not a
// multiple of '5'.
// And there is only one situation which should be retried again: the address
// already in use. The remaining time is enough for it to retry 10 times to find
// another available address in majority of cases.
func waitForStartup(result chan error, p2pPtr *unsafe.Pointer) (err error) {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(233 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			tmp := loadSrvPtr(p2pPtr)
			if tmp != nil && uploaderAPI.PingServer(tmp.host, tmp.port) {
				return nil
			}
		case err = <-result:
			tmp := loadSrvPtr(p2pPtr)
			if err == nil {
				logrus.Infof("reuse exist server on port:%d", tmp.port)
				tmp.setFinished()
			}
			return err
		case <-timeout:
			// The peer server go routine will block and serve if it starts successfully.
			// So we have to wait a moment and check again whether the peer server is
			// started.
			tmp := loadSrvPtr(p2pPtr)
			if tmp == nil {
				return fmt.Errorf("initialize peer server error")
			}
			if !uploaderAPI.PingServer(tmp.host, tmp.port) {
				return fmt.Errorf("can't ping port:%d", tmp.port)
			}
			return nil
		}
	}
}

func serverGC(cfg *config.Config, interval time.Duration) {
	logrus.Info("start server gc, expireTime:", cfg.RV.DataExpireTime)

	var walkFn filepath.WalkFunc = func(path string, info os.FileInfo, err error) error {
		if path == cfg.RV.SystemDataDir || info == nil || err != nil {
			return nil
		}
		if info.IsDir() {
			os.RemoveAll(path)
			return filepath.SkipDir
		}
		if p2p != nil && p2p.deleteExpiredFile(path, info, cfg.RV.DataExpireTime) {
			logrus.Info("server gc, delete file:", path)
		}
		return nil
	}

	for {
		if !isRunning() {
			return
		}
		if err := filepath.Walk(cfg.RV.SystemDataDir, walkFn); err != nil {
			logrus.Warnf("server gc error:%v", err)
		}
		time.Sleep(interval)
	}
}

func captureQuitSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	s := <-c
	logrus.Infof("capture stop signal: %s, will shutdown...", s)

	if p2p != nil {
		p2p.shutdown()
	}
}

func monitorAlive(cfg *config.Config, interval time.Duration) {
	if !isRunning() {
		return
	}

	logrus.Info("monitor peer server whether is alive, aliveTime:",
		cfg.RV.ServerAliveTime)
	go serverGC(cfg, interval)
	go captureQuitSignal()

	if cfg.RV.ServerAliveTime <= 0 {
		return
	}

	for {
		if _, ok := aliveQueue.PollTimeout(cfg.RV.ServerAliveTime); !ok {
			if aliveQueue.Len() > 0 {
				continue
			}
			if p2p != nil {
				logrus.Info("no more task, peer server will stop...")
				p2p.shutdown()
			}
			return
		}
	}
}

func sendAlive(cfg *config.Config) {
	if cfg.RV.ServerAliveTime <= 0 {
		return
	}
	aliveQueue.Put(true)
}

func isRunning() bool {
	return p2p != nil && !p2p.isFinished()
}

// -----------------------------------------------------------------------------
// helper functions

func storeSrvPtr(addr *unsafe.Pointer, ptr *peerServer) {
	atomic.StorePointer(addr, unsafe.Pointer(ptr))
}

func loadSrvPtr(addr *unsafe.Pointer) *peerServer {
	return (*peerServer)(atomic.LoadPointer(addr))
}
