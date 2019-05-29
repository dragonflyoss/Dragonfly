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

package initializer

import (
	"encoding/json"
	"fmt"
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

	"github.com/dragonflyoss/Dragonfly/cmd/dfdaemon/app/options"
	"github.com/dragonflyoss/Dragonfly/common/dflog"
	"github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/constant"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/global"
	g "github.com/dragonflyoss/Dragonfly/dfdaemon/global"
	"github.com/dragonflyoss/Dragonfly/version"
)

// Init is a pre-setup based on options.
func Init(options *options.Options) {

	// init part log config
	initLogger()

	log.Info("init...")

	// init command line param
	initParam(options)

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
			statT, ok := util.GetSys(info)
			if !ok {
				log.Warnf("ignore %s: failed to get last access time", path)
				return nil
			}
			// if the last access time is 1 hour ago
			if time.Now().Unix()-util.AtimeSec(statT) >= 3600 {
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
func rotateLog(logFile *os.File) error {
	stat, err := logFile.Stat()
	if err != nil {
		return err
	}
	logSizeLimit := int64(20 * 1024 * 1024)
	// if it exceeds the 20MB limitation
	if stat.Size() > logSizeLimit {
		log.SetOutput(ioutil.Discard)
		// make sure set the output of log back to logFile when error be raised.
		defer log.SetOutput(logFile)
		logFile.Sync()
		truncateSize := logSizeLimit/2 - 1
		mem, err := syscall.Mmap(int(logFile.Fd()), 0, int(stat.Size()),
			syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
		if err != nil {
			return err
		}
		copy(mem[0:], mem[truncateSize:])
		if err := syscall.Munmap(mem); err != nil {
			return err
		}
		if err := logFile.Truncate(stat.Size() - truncateSize); err != nil {
			return err
		}
		if _, err := logFile.Seek(truncateSize, 0); err != nil {
			return err
		}
	}
	return nil
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
	if err := dflog.InitLog(false, logFilePath, fmt.Sprintf("%d", os.Getpid())); err == nil {
		if logFile, ok := (log.StandardLogger().Out).(*os.File); ok {
			go func(logFile *os.File) {
				log.Infoln("rotate log routine start...")
				ticker := time.NewTicker(60 * time.Second)
				for range ticker.C {
					if err := rotateLog(logFile); err != nil {
						log.Errorf("failed to rotate log %s: %v", logFile.Name(), err)
					}
				}
			}(logFile)
		}
	}
}

func initParam(options *options.Options) {
	if options.Version {
		fmt.Println(version.DFDaemonVersion)
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
	cmd := exec.Command(options.DfPath, "version")
	dfgetVersion, _ := cmd.CombinedOutput()
	log.Infof("dfget version:%s", strings.TrimSpace(string(dfgetVersion)))

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

	parsedTrustHosts := make(map[string]string)
	for _, trustHost := range options.TrustHosts {
		if _, ok := parsedTrustHosts[trustHost]; !ok {
			parsedTrustHosts[trustHost] = trustHost
		}
	}

	// copy options to g.CommandLine so we do not break anything, but finally
	// we should get rid of g.CommandLine totally.
	g.CommandLine = global.CommandParam{
		DfPath:        options.DfPath,
		DFRepo:        options.DFRepo,
		RateLimit:     options.RateLimit,
		CallSystem:    options.CallSystem,
		URLFilter:     options.URLFilter,
		Notbs:         options.Notbs,
		HostIP:        options.HostIP,
		Registry:      options.Registry,
		TrustHosts:    parsedTrustHosts,
		SupernodeList: options.SupernodeList,
	}

	initProperties(options)
}

func initProperties(ops *options.Options) {
	props := config.NewProperties()
	if err := props.Load(ops.ConfigPath); err != nil {
		log.Errorf("init properties failed:%v", err)
		if ops.ConfigPath != constant.DefaultConfigPath || !os.IsNotExist(err) {
			fmt.Printf("init properties failed:%v\n", err)
			os.Exit(constant.CodeExitConfigError)
		}
	}

	if ops.Registry != "" {
		u, err := config.NewURL(ops.Registry)
		if err != nil {
			fmt.Printf("invalid registry url from cli parameters: %v\n", err)
			os.Exit(constant.CodeExitConfigError)
		}

		props.RegistryMirror = &config.RegistryMirror{
			Remote: u,
		}
	}

	g.Properties = props
	str, _ := json.Marshal(props)
	log.Infof("init properties:%s", str)
}
