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
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/sirupsen/logrus"
)

const (
	ctype = "application/octet-stream"
)

var (
	p2p *peerServer
)

var (
	aliveQueue  = util.NewQueue(0)
	uploaderAPI = api.NewUploaderAPI(util.DefaultTimeout)
)

// TODO: Move this part out of the uploader

// StartPeerServerProcess starts an independent peer server process for uploading downloaded files
// if it doesn't exist.
// This function is invoked when dfget starts to download files in p2p pattern.
func StartPeerServerProcess(cfg *config.Config) (port int, err error) {
	if port = checkPeerServerExist(cfg, 0); port > 0 {
		return port, nil
	}

	cmd := exec.Command(os.Args[0], "server",
		"--ip", cfg.RV.LocalIP,
		"--meta", cfg.RV.MetaPath,
		"--data", cfg.RV.SystemDataDir,
		"--expiretime", cfg.RV.DataExpireTime.String(),
		"--alivetime", cfg.RV.ServerAliveTime.String())
	if cfg.Verbose {
		cmd.Args = append(cmd.Args, "--verbose")
	}

	var stdout io.ReadCloser
	if stdout, err = cmd.StdoutPipe(); err != nil {
		return 0, err
	}
	if err = cmd.Start(); err == nil {
		port, err = readPort(stdout)
	}
	if err == nil && checkPeerServerExist(cfg, port) <= 0 {
		err = fmt.Errorf("invalid server on port:%d", port)
		port = 0
	}

	return
}

func readPort(r io.Reader) (int, error) {
	done := make(chan error)
	var port int

	go func() {
		var n = 0
		var err error
		buf := make([]byte, 256)

		n, err = r.Read(buf)
		if err != nil {
			done <- err
		}

		content := strings.TrimSpace(string(buf[:n]))
		port, err = strconv.Atoi(content)
		done <- err
		close(done)
	}()

	select {
	case err := <-done:
		return port, err
	case <-time.After(time.Second):
		return 0, fmt.Errorf("get peer server's port timeout")
	}
}

// checkPeerServerExist checks the peer server on port whether is available.
// if the parameter port <= 0, it will get port from meta file and checks.
func checkPeerServerExist(cfg *config.Config, port int) int {
	taskFileName := cfg.RV.TaskFileName
	if port <= 0 {
		port = getPortFromMeta(cfg.RV.MetaPath)
	}

	// check the peer server whether is available
	result, err := checkServer(cfg.RV.LocalIP, port, cfg.RV.DataDir, taskFileName, cfg.TotalLimit)
	logrus.Infof("local http result:%s err:%v, port:%d path:%s",
		result, err, port, config.LocalHTTPPathCheck)

	if err == nil {
		if result == taskFileName {
			logrus.Infof("use peer server on port:%d", port)
			return port
		}
		logrus.Warnf("not found process on port:%d, version:%s", port, version.DFGetVersion)
	}
	return 0
}

// ----------------------------------------------------------------------------
// dfget server functions

// WaitForShutdown wait for peer server shutdown
func WaitForShutdown() {
	if p2p != nil {
		<-p2p.finished
	}
}

// LaunchPeerServer launch a server to send piece data
func LaunchPeerServer(cfg *config.Config) (int, error) {
	logrus.Infof("********************")
	logrus.Infof("start peer server...")

	res := make(chan error)
	go func() {
		res <- launch(cfg)
	}()

	if err := waitForStartup(res); err != nil {
		logrus.Errorf("start peer server error:%v, exit directly", err)
		return 0, err
	}
	updateServicePortInMeta(cfg.RV.MetaPath, p2p.port)
	logrus.Infof("start peer server success, host:%s, port:%d",
		p2p.host, p2p.port)
	go monitorAlive(cfg, 15*time.Second)
	return p2p.port, nil
}

func launch(cfg *config.Config) error {
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
		p2p = newPeerServer(cfg, port)
		if err := p2p.ListenAndServe(); err != nil {
			if strings.Index(err.Error(), "address already in use") < 0 {
				// start failed or shutdown
				return err
			} else if uploaderAPI.PingServer(p2p.host, p2p.port) {
				// a peer server is already existing
				return nil
			}
			logrus.Warnf("start error:%v, remain retry times:%d",
				err, retryCount-i)
		}
	}
	return fmt.Errorf("star peer server error and retried at most %d times", retryCount)
}

func waitForStartup(result chan error) error {
	select {
	case err := <-result:
		if err == nil {
			logrus.Infof("reuse exist server on port:%d", p2p.port)
			close(p2p.finished)
		}
		return err
	case <-time.After(100 * time.Millisecond):
		// The peer server go routine will block and serve if it starts successfully.
		// So we have to wait a moment and check again whether the peer server is
		// started.
		if p2p == nil {
			return fmt.Errorf("initialize peer server error")
		}
		if !uploaderAPI.PingServer(p2p.host, p2p.port) {
			return fmt.Errorf("cann't ping port:%d", p2p.port)
		}
		return nil
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
		if err := filepath.Walk(cfg.RV.SystemDataDir, walkFn); err != nil {
			logrus.Warnf("server gc error:%v", err)
		}
		time.Sleep(interval)
	}
}

func captureQuitSignal() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSTOP)
	s := <-c
	logrus.Infof("capture stop signal: %s, will shutdown...", s)

	if p2p == nil {
		return
	}
	p2p.shutdown()
}

func monitorAlive(cfg *config.Config, interval time.Duration) {
	if !isRunning() {
		return
	}

	logrus.Info("monitor peer server whether is alive, aliveTime:",
		cfg.RV.ServerAliveTime)
	go serverGC(cfg, interval)
	go captureQuitSignal()

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

func isRunning() bool {
	if p2p == nil {
		return false
	}
	select {
	case <-p2p.finished:
		return false
	default:
		return true
	}
}
