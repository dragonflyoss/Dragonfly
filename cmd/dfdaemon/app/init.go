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
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/constant"
	"github.com/dragonflyoss/Dragonfly/pkg/dflog"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	statutil "github.com/dragonflyoss/Dragonfly/pkg/stat"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// initDfdaemon sets up running environment for dfdaemon according to the given config
func initDfdaemon(cfg config.Properties) error {
	// if Options.MaxProcs <= 0, programs run with GOMAXPROCS set to the number of cores available.
	if cfg.MaxProcs > 0 {
		runtime.GOMAXPROCS(cfg.MaxProcs)
	}

	if err := initLogger(cfg); err != nil {
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

	go cleanLocalRepo(cfg.DFRepo)

	dfgetVersion, err := exec.Command(cfg.DFPath, "version").CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "get dfget version")
	}
	logrus.Infof("use dfget %s from %s", bytes.TrimSpace(dfgetVersion), cfg.DFPath)

	return nil
}

// initLogger initialize the global logrus logger
func initLogger(cfg config.Properties) error {
	current, err := user.Current()
	if err != nil {
		return errors.Wrap(err, "get current user")
	}

	// Set the log level here so the following line will be output normally
	// to the console, before setting the log file.
	if cfg.Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logFilePath := filepath.Join(current.HomeDir, ".small-dragonfly/logs/dfdaemon.log")
	logrus.Debugf("use log file %s", logFilePath)

	if err := dflog.InitLog(cfg.Verbose, logFilePath, fmt.Sprintf("%d", os.Getpid())); err != nil {
		return errors.Wrap(err, "init log file")
	}

	logFile, ok := (logrus.StandardLogger().Out).(*os.File)
	if !ok {
		return nil
	}
	go func(logFile *os.File) {
		logrus.Infof("rotate %s every 60 seconds", logFilePath)
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
			if err := rotateLog(logFile); err != nil {
				logrus.Errorf("failed to rotate log %s: %v", logFile.Name(), err)
			}
		}
	}(logFile)

	return nil
}

// rotateLog truncates the logs file by a certain amount bytes.
func rotateLog(logFile *os.File) error {
	fStat, err := logFile.Stat()
	if err != nil {
		return err
	}
	logSizeLimit := int64(20 * 1024 * 1024)

	if fStat.Size() <= logSizeLimit {
		return nil
	}

	// if it exceeds the 20MB limitation
	log.SetOutput(ioutil.Discard)
	// make sure set the output of log back to logFile when error be raised.
	defer log.SetOutput(logFile)
	logFile.Sync()
	truncateSize := logSizeLimit/2 - 1
	mem, err := syscall.Mmap(int(logFile.Fd()), 0, int(fStat.Size()),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return err
	}
	copy(mem[0:], mem[truncateSize:])
	if err := syscall.Munmap(mem); err != nil {
		return err
	}
	if err := logFile.Truncate(fStat.Size() - truncateSize); err != nil {
		return err
	}
	if _, err := logFile.Seek(truncateSize, 0); err != nil {
		return err
	}

	return nil
}

// cleanLocalRepo checks the files at local periodically, and delete the file when
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
