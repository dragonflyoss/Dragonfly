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

	log "github.com/Sirupsen/logrus"

	"github.com/alibaba/Dragonfly/dfdaemon/constant"
	g "github.com/alibaba/Dragonfly/dfdaemon/global"
	mux "github.com/alibaba/Dragonfly/dfdaemon/muxconf"
	"github.com/alibaba/Dragonfly/dfdaemon/util"
)

func init() {

	// init part log config
	initLogger()

	log.Info("init...")

	// init command line param
	initParam()

	// http handler mapper
	mux.InitMux()

	// clean local data dir
	go cleanLocalRepo()

	log.Info("init finish")
}

// cleanLocalRepo checks the files at local periodically, and delete the file when
// it comes to a certain age(counted by the last access time).
// TODO: what happens if the disk usage comes to high level?
func cleanLocalRepo() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("recover cleanLocalRepo from err:%v", err)
			go cleanLocalRepo()
		}
	}()
	for {
		time.Sleep(time.Minute * 2)
		log.Info("scan repo and clean expired files")
		filepath.Walk(g.CommandLine.DFRepo, func(path string, info os.FileInfo, err error) error {
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
		if logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_RDWR, 0644); err == nil {
			logFile.Seek(0, 2)
			log.SetOutput(logFile)
			go rotateLog(logFile, logFilePath)
		}
	}

}

func initParam() {
	flag.StringVar(&g.CommandLine.DFRepo, "localrepo", g.HomeDir+".small-dragonfly/dfdaemon/data/", "temp output dir of dfdaemon")

	var defaultPath string
	if path, err := exec.LookPath(os.Args[0]); err == nil {
		if absPath, err := filepath.Abs(path); err == nil {
			g.DfHome = filepath.Dir(absPath)
			defaultPath = g.DfHome + "/dfget"
		}

	}
	flag.StringVar(&g.CommandLine.DfPath, "dfpath", defaultPath, "dfget path")
	flag.StringVar(&g.CommandLine.RateLimit, "ratelimit", "", "net speed limit,format:xxxM/K")
	flag.StringVar(&g.CommandLine.CallSystem, "callsystem", "com_ops_dragonfly", "caller name")
	flag.StringVar(&g.CommandLine.Urlfilter, "urlfilter", "Signature&Expires&OSSAccessKeyId", "filter specified url fields")
	flag.BoolVar(&g.CommandLine.Notbs, "notbs", true, "not try back source to download if throw exception")
	flag.BoolVar(&g.CommandLine.Version, "v", false, "version")
	flag.BoolVar(&g.CommandLine.Verbose, "verbose", false, "verbose")
	flag.BoolVar(&g.CommandLine.Help, "h", false, "help")
	flag.StringVar(&g.CommandLine.HostIp, "hostIp", "127.0.0.1", "dfdaemon host ip, default: 127.0.0.1")
	flag.UintVar(&g.CommandLine.Port, "port", 65001, "dfdaemon will listen the port")
	flag.StringVar(&g.CommandLine.Registry, "registry", "", "registry addr(https://abc.xx.x or http://abc.xx.x) and must exist if dfdaemon is used to mirror mode")
	flag.StringVar(&g.CommandLine.DownRule, "rule", "", "download the url by P2P if url matches the specified pattern,format:reg1,reg2,reg3")
	flag.StringVar(&g.CommandLine.CertFile, "certpem", "", "cert.pem file path")
	flag.StringVar(&g.CommandLine.KeyFile, "keypem", "", "key.pem file path")
	flag.IntVar(&g.CommandLine.MaxProcs, "maxprocs", 4, "the maximum number of CPUs that the dfdaemon can use")

	flag.Parse()

	if g.CommandLine.Version {
		fmt.Print(constant.VERSION)
		os.Exit(0)
	}
	if g.CommandLine.Help {
		flag.Usage()
		os.Exit(0)
	}

	if g.CommandLine.Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if !filepath.IsAbs(g.CommandLine.DFRepo) {
		log.Errorf("local repo:%s is not abs", g.CommandLine.DFRepo)
		os.Exit(constant.CodeExitPathNotAbs)
	}
	if !strings.HasSuffix(g.CommandLine.DFRepo, "/") {
		g.CommandLine.DFRepo += "/"
	}
	if err := os.MkdirAll(g.CommandLine.DFRepo, 0755); err != nil {
		log.Errorf("create local repo:%s err:%v", g.CommandLine.DFRepo, err)
		os.Exit(constant.CodeExitRepoCreateFail)
	}

	if len(g.CommandLine.RateLimit) == 0 {
		g.CommandLine.RateLimit = util.NetLimit()
	} else if isMatch, _ := regexp.MatchString("^[[:digit:]]+[MK]$", g.CommandLine.RateLimit); !isMatch {
		os.Exit(constant.CodeExitRateLimitInvalid)
	}

	if g.CommandLine.Port <= 2000 || g.CommandLine.Port > 65535 {
		os.Exit(constant.CodeExitPortInvalid)
	}

	downRule := strings.Split(g.CommandLine.DownRule, ",")
	for _, rule := range downRule {
		g.UpdateDFPattern(rule)
	}
	if _, err := os.Stat(g.CommandLine.DfPath); err != nil && os.IsNotExist(err) {
		log.Errorf("dfpath:%s not found", g.CommandLine.DfPath)
		os.Exit(constant.CodeExitDfgetNotFound)
	}
	cmd := exec.Command(g.CommandLine.DfPath, "-v")
	version, _ := cmd.CombinedOutput()

	log.Infof("dfget version:%s", string(version))

	if !cmd.ProcessState.Success() {
		fmt.Println("\npython must be 2.7")
		os.Exit(constant.CodeExitDfgetFail)
	}

	if g.CommandLine.CertFile != "" && g.CommandLine.KeyFile != "" {
		g.UseHttps = true
	}

	if g.CommandLine.Registry != "" {
		protoAndDomain := strings.SplitN(g.CommandLine.Registry, "://", 2)
		splitedCount := len(protoAndDomain)
		g.RegProto = "http"
		g.RegDomain = protoAndDomain[splitedCount-1]
		if splitedCount == 2 {
			g.RegProto = protoAndDomain[0]
		}
	}

}
