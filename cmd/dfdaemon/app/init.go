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

package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/constant"
	dfgetcfg "github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/pkg/algorithm"
	"github.com/dragonflyoss/Dragonfly/pkg/dflog"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	statutil "github.com/dragonflyoss/Dragonfly/pkg/stat"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// adjustSupernodeList adjusts the super nodes [a,b] to [a,b,b,a]
func adjustSupernodeList(nodes []string) []string {
	switch nodesLen := len(nodes); nodesLen {
	case 0:
		return nodes
	case 1:
		return append(nodes, nodes[0])
	default:
		algorithm.Shuffle(nodesLen, func(i, j int) {
			nodes[i], nodes[j] = nodes[j], nodes[i]
		})
		return append(nodes, nodes...)
	}
}

// getLocalIP return the localIP which connects to super node
func getLocalIP(nodes []string) (localIP string) {
	var (
		e error
	)
	for _, n := range nodes {
		ip, port := netutils.GetIPAndPortFromNode(n, dfgetcfg.DefaultSupernodePort)
		if localIP, e = httputils.CheckConnect(ip, port, 1000); e == nil {
			return localIP
		}
		logrus.Warnf("Connect to node:%s error: %v", n, e)
	}
	return ""
}

// initDfdaemon sets up running environment for dfdaemon according to the given config.
func initDfdaemon(cfg *config.Properties) error {
	// if Options.MaxProcs <= 0, programs run with GOMAXPROCS set to the number of cores available.
	if cfg.MaxProcs > 0 {
		runtime.GOMAXPROCS(cfg.MaxProcs)
	}

	if err := initLogger(*cfg); err != nil {
		return errors.Wrap(err, "init logger")
	}

	if cfg.Verbose {
		logrus.Infoln("use verbose logging")
	}

	if err := os.MkdirAll(cfg.DFRepo, 0755); err != nil {
		return errortypes.Newf(
			constant.CodeExitRepoCreateFail,
			"ensure local repo %s exists", cfg.DFRepo,
		)
	}
	cfg.SuperNodes = adjustSupernodeList(cfg.SuperNodes)
	if stringutils.IsEmptyStr(cfg.LocalIP) {
		cfg.LocalIP = getLocalIP(cfg.SuperNodes)
	}

	go cleanLocalRepo(cfg.DFRepo)

	if !cfg.StreamMode {
		dfgetVersion, err := exec.Command(cfg.DFPath, "version").CombinedOutput()
		if err != nil {
			return errors.Wrap(err, "get dfget version")
		}
		logrus.Infof("use %s from %s", bytes.TrimSpace(dfgetVersion), cfg.DFPath)
	}

	return nil
}

// initLogger initializes the global logrus logger.
func initLogger(cfg config.Properties) error {
	if cfg.WorkHome == "" {
		current, err := user.Current()
		if err != nil {
			return errors.Wrap(err, "get current user")
		}
		cfg.WorkHome = filepath.Join(current.HomeDir, ".small-dragonfly")
	}
	if cfg.LogConfig.Path == "" {
		cfg.LogConfig.Path = filepath.Join(cfg.WorkHome, "logs", "dfdaemon.log")
	}
	opts := []dflog.Option{
		dflog.WithLogFile(cfg.LogConfig.Path, cfg.LogConfig.MaxSize, cfg.LogConfig.MaxBackups),
		dflog.WithSign(fmt.Sprintf("%d", os.Getpid())),
		dflog.WithDebug(cfg.Verbose),
	}

	logrus.Debugf("use log file %s", cfg.LogConfig.Path)

	return errors.Wrap(dflog.Init(logrus.StandardLogger(), opts...), "init log")
}

// cleanLocalRepo checks the files at local periodically, and deletes the file when
// it comes to a certain age(counted by the last access time).
// TODO: what happens if the disk usage comes to high level?
func cleanLocalRepo(dfpath string) {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("recover cleanLocalRepo from err:%v", err)
			go cleanLocalRepo(dfpath)
		}
	}()
	for {
		time.Sleep(time.Minute * 2)
		logrus.Info("scan repo and clean expired files")
		filepath.Walk(dfpath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logrus.Warnf("walk file:%s error:%v", path, err)
				return nil
			}
			if !info.Mode().IsRegular() {
				logrus.Debugf("ignore %s: not a regular file", path)
				return nil
			}
			// get the last access time
			statT, ok := fileutils.GetSys(info)
			if !ok {
				logrus.Warnf("ignore %s: failed to get last access time", path)
				return nil
			}
			// if the last access time is 1 hour ago
			if time.Since(statutil.Atime(statT)) > time.Hour {
				if err := os.Remove(path); err == nil {
					logrus.Infof("remove file:%s success", path)
				} else {
					logrus.Warnf("remove file:%s error:%v", path, err)
				}
			}
			return nil
		})
	}
}
