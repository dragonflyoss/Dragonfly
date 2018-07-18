// Copyright 1999-2017 Alibaba Group.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package initializer

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/alibaba/Dragonfly/cmd/dfdaemon/options"
	"github.com/alibaba/Dragonfly/dfdaemon/constant"
	"github.com/alibaba/Dragonfly/dfdaemon/global"
	g "github.com/alibaba/Dragonfly/dfdaemon/global"
	mux "github.com/alibaba/Dragonfly/dfdaemon/muxconf"
	"github.com/alibaba/Dragonfly/dfdaemon/util"
	"github.com/alibaba/Dragonfly/version"
)

// Init is a pre-setup based on options.
func Init(options *options.Options) {

	// init part log config
	initLogger()

	log.Info("init...")

	// init command line param
	initParam(options)

	// http handler mapper
	mux.InitMux()

	// clean local data dir
	go cleanLocalRepo(options)

	log.Info("init finish")
}

// cleanLocalRepo checks the files at local periodically, and delete the file when
// it comes to a certain age(counted by the last access time).
// TODO: what happens if the disk usage comes to high level?
func cleanLocalRepo(options *options.Options) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("recover cleanLocalRepo from err:%v", err)
			go cleanLocalRepo(options)
		}
	}()
	for {
		time.Sleep(time.Minute * 2)
		log.Info("scan repo and clean expired files")
		filepath.Walk(options.DFRepo, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Warnf("walk file:%s error:%v", path, err)
				return nil
			}
			if !info.Mode().IsRegular() {
				log.Infof("ignore %s: not a regular file", path)
				return nil
			}
			// get the last access time
			statT, ok := info.Sys().(*syscall.Stat_t)
			if !ok {
				log.Warnf("ignore %s: failed to get last access time", path)
				return nil
			}
			// if the last access time is 1 hour ago
			if time.Now().Unix()-Atime(statT) >= 3600 {
				if err := os.Remove(path); err == nil {
					log.Infof("remove file:%s success", path)
				} else {
					log.Warnf("remove file:%s error:%v", path, err)
				}
			}
			return nil
		})
	}
}

// rotateLog truncates the logs file by a certain amount bytes.
func rotateLog(logFile *os.File, logFilePath string) {
	logSizeLimit := int64(20 * 1024 * 1024)
	for {
		time.Sleep(time.Second * 60)
		stat, err := os.Stat(logFilePath)
		if err != nil {
			log.Errorf("failed to stat %s: %s", logFilePath, err)
			continue
		}
		// if it exceeds the 20MB limitation
		if stat.Size() > logSizeLimit {
			log.SetOutput(ioutil.Discard)
			logFile.Sync()
			if transFile, err := os.Open(logFilePath); err == nil {
				// move the pointer to be (end - 10MB)
				transFile.Seek(-10*1024*1024, 2)
				// move the pointer to head
				logFile.Seek(0, 0)
				count, _ := io.Copy(logFile, transFile)
				logFile.Truncate(count)
				log.SetOutput(logFile)
				transFile.Close()
			}
		}
	}

}

func initLogger() {
	if current, err := user.Current(); err == nil {
		g.HomeDir = strings.TrimSpace(current.HomeDir)
	}
	if g.HomeDir != "" {
		if !strings.HasSuffix(g.HomeDir, "/") {
			g.HomeDir += "/"
		}
	} else {
		os.Exit(constant.CodeExitUserHomeNotExist)
	}

	logFilePath := g.HomeDir + ".small-dragonfly/logs/dfdaemon.log"
	log.SetFormatter(&log.TextFormatter{TimestampFormat: "2006-01-02 15:04:00", DisableColors: true})
	if os.MkdirAll(filepath.Dir(logFilePath), 0755) == nil {
		if logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644); err == nil {
			logFile.Seek(0, 2)
			log.SetOutput(logFile)
			go rotateLog(logFile, logFilePath)
		}
	}

}

func initParam(options *options.Options) {
	if options.Version {
		fmt.Println(version.DFDaemonVersion)
		os.Exit(0)
	}
	if options.Help {
		flag.Usage()
		os.Exit(0)
	}

	if options.Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if !filepath.IsAbs(options.DFRepo) {
		log.Errorf("local repo:%s is not abs", options.DFRepo)
		os.Exit(constant.CodeExitPathNotAbs)
	}
	if !strings.HasSuffix(options.DFRepo, "/") {
		options.DFRepo += "/"
	}
	if err := os.MkdirAll(options.DFRepo, 0755); err != nil {
		log.Errorf("create local repo:%s err:%v", options.DFRepo, err)
		os.Exit(constant.CodeExitRepoCreateFail)
	}

	if len(options.RateLimit) == 0 {
		options.RateLimit = util.NetLimit()
	} else if isMatch, _ := regexp.MatchString("^[[:digit:]]+[MK]$", options.RateLimit); !isMatch {
		os.Exit(constant.CodeExitRateLimitInvalid)
	}

	if options.Port <= 2000 || options.Port > 65535 {
		os.Exit(constant.CodeExitPortInvalid)
	}

	downRule := strings.Split(options.DownRule, ",")
	for _, rule := range downRule {
		g.UpdateDFPattern(rule)
	}
	if _, err := os.Stat(options.DfPath); err != nil && os.IsNotExist(err) {
		log.Errorf("dfpath:%s not found", options.DfPath)
		os.Exit(constant.CodeExitDfgetNotFound)
	}
	cmd := exec.Command(options.DfPath, "-v")
	version, _ := cmd.CombinedOutput()

	log.Infof("dfget version:%s", string(version))

	if !cmd.ProcessState.Success() {
		fmt.Println("\npython must be 2.7")
		os.Exit(constant.CodeExitDfgetFail)
	}

	if options.CertFile != "" && options.KeyFile != "" {
		g.UseHTTPS = true
	}

	if options.Registry != "" {
		protoAndDomain := strings.SplitN(options.Registry, "://", 2)
		splitedCount := len(protoAndDomain)
		g.RegProto = "http"
		g.RegDomain = protoAndDomain[splitedCount-1]
		if splitedCount == 2 {
			g.RegProto = protoAndDomain[0]
		}
	}

	// copy options to g.CommandLine so we do not break anything, but finally
	// we should get rid of g.CommandLine totally.
	g.CommandLine = global.CommandParam{
		DfPath:     options.DfPath,
		DFRepo:     options.DFRepo,
		RateLimit:  options.RateLimit,
		CallSystem: options.CallSystem,
		URLFilter:  options.URLFilter,
		Notbs:      options.Notbs,
		HostIP:     options.HostIP,
		Registry:   options.Registry,
	}
}
