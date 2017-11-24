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
package util

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/alibaba/Dragonfly/src/daemon/src/df-daemon/constant"
	. "github.com/alibaba/Dragonfly/src/daemon/src/df-daemon/global"
	. "github.com/alibaba/Dragonfly/src/daemon/src/df-daemon/muxconf"
	"github.com/alibaba/Dragonfly/src/daemon/src/df-daemon/util"
)

func init() {

	//init part log config
	initLogger()

	log.Info("init...")

	// init command line param
	initParam()

	//http handler mapper
	InitMux()

	//clean local data dir
	go cleanLocalRepo()

	log.Info("init finish")
}

func cleanLocalRepo() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("recover cleanLocalRepo from err:%v", err)
			go cleanLocalRepo()
		}
	}()
	for {
		select {
		case <-time.After(time.Minute * 2):
			func() {
				log.Info("scan repo and clean expired files")
				filepath.Walk(G_CommandLine.DFRepo, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						log.Warnf("walk file:%s error:%v", path, err)
					} else {
						if info.Mode().IsRegular() {
							if time.Now().Unix()-reflect.ValueOf(info.Sys()).Elem().FieldByName("Atim").Field(0).Int() >= 3600 {
								if err := os.Remove(path); err == nil {
									log.Infof("remove file:%s success", path)
								} else {
									log.Warnf("remove file:%s error:%v", path, err)
								}
							}
						}
					}
					return nil
				})
			}()
		}
	}
}

func rotateLog(logFile *os.File, logFilePath string) {
	logSizeLimit := int64(20 * 1024 * 1024)
	for {
		select {
		case <-time.After(time.Second * 60):
			if stat, err := os.Stat(logFilePath); err == nil {
				if stat.Size() > logSizeLimit {
					log.SetOutput(ioutil.Discard)
					logFile.Sync()
					if transFile, err := os.Open(logFilePath); err == nil {
						transFile.Seek(-10*1024*1024, 2)
						logFile.Seek(0, 0)
						count, _ := io.Copy(logFile, transFile)
						logFile.Truncate(count)
						log.SetOutput(logFile)
						transFile.Close()
					}
				}
			}

		}
	}

}

func initLogger() {
	if current, err := user.Current(); err == nil {
		G_HomeDir = strings.TrimSpace(current.HomeDir)
	}
	if G_HomeDir != "" {
		if !strings.HasSuffix(G_HomeDir, "/") {
			G_HomeDir += "/"
		}
	} else {
		os.Exit(constant.CODE_EXIT_USER_HOME_NOT_EXIST)
	}

	logFilePath := G_HomeDir + ".small-dragonfly/logs/dfdaemon.log"
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
	flag.StringVar(&G_CommandLine.DFRepo, "localrepo", G_HomeDir+".small-dragonfly/dfdaemon/data/", "temp output dir of daemon")

	var defaultPath string
	if path, err := exec.LookPath(os.Args[0]); err == nil {
		if absPath, err := filepath.Abs(path); err == nil {
			G_DfHome = filepath.Dir(absPath)
			defaultPath = G_DfHome + "/dfget"
		}

	}
	flag.StringVar(&G_CommandLine.DfPath, "dfpath", defaultPath, "dfget path")
	flag.StringVar(&G_CommandLine.RateLimit, "ratelimit", "", "net speed limit,format:xxxM/K")
	flag.StringVar(&G_CommandLine.CallSystem, "callsystem", "com_ops_dragonfly", "caller name")
	flag.StringVar(&G_CommandLine.Urlfilter, "urlfilter", "Signature&Expires&OSSAccessKeyId", "filter specified url fields")
	flag.BoolVar(&G_CommandLine.Notbs, "notbs", true, "not try back source to download if throw exception")
	flag.BoolVar(&G_CommandLine.Version, "v", false, "version")
	flag.BoolVar(&G_CommandLine.Verbose, "verbose", false, "verbose")
	flag.BoolVar(&G_CommandLine.Help, "h", false, "help")
	flag.UintVar(&G_CommandLine.Port, "port", 65001, "daemon will listen the port")
	flag.StringVar(&G_CommandLine.Registry, "registry", "", "registry addr(https://abc.xx.x or http://abc.xx.x) and must exist if df-daemon is used to mirror mode")
	flag.StringVar(&G_CommandLine.DownRule, "rule", "", "download the url by P2P if url matches the specified pattern,format:reg1,reg2,reg3")
	flag.StringVar(&G_CommandLine.CertFile, "certpem", "", "cert.pem file path")
	flag.StringVar(&G_CommandLine.KeyFile, "keypem", "", "key.pem file path")

	flag.Parse()

	if G_CommandLine.Version {
		fmt.Print(constant.VERSION)
		os.Exit(0)
	}
	if G_CommandLine.Help {
		flag.Usage()
		os.Exit(0)
	}

	if G_CommandLine.Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if !filepath.IsAbs(G_CommandLine.DFRepo) {
		log.Errorf("local repo:%s is not abs", G_CommandLine.DFRepo)
		os.Exit(constant.CODE_EXIT_PATH_NOT_ABS)
	}
	if !strings.HasSuffix(G_CommandLine.DFRepo, "/") {
		G_CommandLine.DFRepo += "/"
	}
	if err := os.MkdirAll(G_CommandLine.DFRepo, 0755); err != nil {
		log.Errorf("create local repo:%s err:%v", G_CommandLine.DFRepo, err)
		os.Exit(constant.CODE_EXIT_REPO_CREATE_FAIL)
	}

	if len(G_CommandLine.RateLimit) == 0 {
		G_CommandLine.RateLimit = util.NetLimit()
	} else if isMatch, _ := regexp.MatchString("^[[:digit:]]+[MK]$", G_CommandLine.RateLimit); !isMatch {
		os.Exit(constant.CODE_EXIT_RATE_LIMIT_INVALID)
	}

	if G_CommandLine.Port <= 2000 || G_CommandLine.Port > 65535 {
		os.Exit(constant.CODE_EXIT_PORT_INVALID)
	}

	downRule := strings.Split(G_CommandLine.DownRule, ",")
	for _, rule := range downRule {
		UpdateDFPattern(rule)
	}
	if _, err := os.Stat(G_CommandLine.DfPath); err != nil && os.IsNotExist(err) {
		log.Errorf("dfpath:%s not found", G_CommandLine.DfPath)
		os.Exit(constant.CODE_EXIT_DFGET_NOT_FOUND)
	}
	cmd := exec.Command(G_CommandLine.DfPath, "-v")
	version, _ := cmd.CombinedOutput()

	log.Infof("dfget version:%s", string(version))

	if !cmd.ProcessState.Success() {
		fmt.Println("\npython must be 2.7")
		os.Exit(constant.CODE_EXIT_DFGET_FAIL)
	}

	if G_CommandLine.CertFile != "" && G_CommandLine.KeyFile != "" {
		G_UseHttps = true
	}

	if G_CommandLine.Registry != "" {
		protoAndDomain := strings.SplitN(G_CommandLine.Registry, "://", 2)
		splitedCount := len(protoAndDomain)
		G_RegProto = "http"
		G_RegDomain = protoAndDomain[splitedCount-1]
		if splitedCount == 2 {
			G_RegProto = protoAndDomain[0]
		}
	}

}
