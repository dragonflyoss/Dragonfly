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

package uploader

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/version"
	"github.com/sirupsen/logrus"
)

var (
	defaultExecutor PeerServerExecutor = &peerServerExecutor{}
)

// SetupPeerServerExecutor setup a giving executor instance instead of default implementation.
func SetupPeerServerExecutor(executor PeerServerExecutor) {
	defaultExecutor = executor
}

// GetPeerServerExecutor returns the current executor instance.
func GetPeerServerExecutor() PeerServerExecutor {
	return defaultExecutor
}

// StartPeerServerProcess starts an independent peer server process for uploading downloaded files
// if it doesn't exist.
// This function is invoked when dfget starts to download files in p2p pattern.
func StartPeerServerProcess(cfg *config.Config) (port int, err error) {
	if defaultExecutor != nil {
		return defaultExecutor.StartPeerServerProcess(cfg)
	}
	return 0, fmt.Errorf("executor of peer server hasn't been initialized")
}

// PeerServerExecutor starts an independent peer server process for uploading downloaded files.
type PeerServerExecutor interface {
	StartPeerServerProcess(cfg *config.Config) (port int, err error)
}

// ---------------------------------------------------------------------------
// PeerServerExecutor default implementation

type peerServerExecutor struct {
}

var _ PeerServerExecutor = &peerServerExecutor{}

func (pe *peerServerExecutor) StartPeerServerProcess(cfg *config.Config) (port int, err error) {
	if port = pe.checkPeerServerExist(cfg, 0); port > 0 {
		return port, nil
	}

	cmd := exec.Command(os.Args[0], "server",
		"--ip", cfg.RV.LocalIP,
		"--port", strconv.Itoa(cfg.RV.PeerPort),
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
		port, err = pe.readPort(stdout)
	}
	if err == nil && pe.checkPeerServerExist(cfg, port) <= 0 {
		err = fmt.Errorf("invalid server on port:%d", port)
		port = 0
	}

	return
}

func (pe *peerServerExecutor) readPort(r io.Reader) (int, error) {
	done := make(chan error)
	var port int32

	go func() {
		buf := make([]byte, 256)

		n, err := r.Read(buf)
		if err != nil {
			done <- err
		}

		content := strings.TrimSpace(string(buf[:n]))
		portValue, err := strconv.Atoi(content)
		// avoid data race
		atomic.StoreInt32(&port, int32(portValue))
		done <- err
	}()

	select {
	case err := <-done:
		return int(atomic.LoadInt32(&port)), err
	case <-time.After(time.Second):
		return 0, fmt.Errorf("get peer server's port timeout")
	}
}

// checkPeerServerExist checks the peer server on port whether is available.
// if the parameter port <= 0, it will get port from meta file and checks.
func (pe *peerServerExecutor) checkPeerServerExist(cfg *config.Config, port int) int {
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
