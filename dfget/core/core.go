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

// Package core implements the core modules of dfget.
package core

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/downloader"
	"github.com/dragonflyoss/Dragonfly/dfget/core/regist"
	"github.com/dragonflyoss/Dragonfly/dfget/core/uploader"
	"github.com/dragonflyoss/Dragonfly/dfget/errors"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Start function creates a new task and starts it to download file.
func Start(cfg *config.Config) *errors.DFGetError {
	var (
		supernodeAPI = api.NewSupernodeAPI()
		register     = regist.NewSupernodeRegister(cfg, supernodeAPI)
		err          error
		result       *regist.RegisterResult
	)

	util.Printer.Println(fmt.Sprintf("--%s--  %s",
		cfg.StartTime.Format(config.DefaultTimestampFormat), cfg.URL))

	if err = prepare(cfg); err != nil {
		return errors.New(1100, err.Error())
	}

	if result, err = registerToSuperNode(cfg, register); err != nil {
		return errors.New(1200, err.Error())
	}

	if err = downloadFile(cfg, supernodeAPI, register, result); err != nil {
		return errors.New(1300, err.Error())
	}

	return nil
}

func prepare(cfg *config.Config) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	util.Printer.Printf("dfget version:%s", version.DFGetVersion)
	util.Printer.Printf("workspace:%s sign:%s", cfg.WorkHome, cfg.Sign)
	cfg.ClientLogger.Infof("target file path:%s", cfg.Output)

	rv := &cfg.RV

	rv.RealTarget = cfg.Output
	rv.TargetDir = path.Dir(rv.RealTarget)
	panicIf(util.CreateDirectory(rv.TargetDir))
	cfg.RV.TempTarget, err = createTempTargetFile(rv.TargetDir, cfg.Sign)
	panicIf(err)

	panicIf(util.CreateDirectory(path.Dir(rv.MetaPath)))
	panicIf(util.CreateDirectory(cfg.WorkHome))
	panicIf(util.CreateDirectory(rv.SystemDataDir))
	rv.DataDir = cfg.RV.SystemDataDir

	cfg.Node = adjustSupernodeList(cfg.Node)
	rv.LocalIP = checkConnectSupernode(cfg.Node, cfg.ClientLogger)
	rv.Cid = getCid(rv.LocalIP, cfg.Sign)
	rv.TaskFileName = getTaskFileName(rv.RealTarget, cfg.Sign)
	rv.TaskURL = getTaskURL(cfg.URL, cfg.Filter)
	cfg.ClientLogger.Info("runtimeVariable: " + cfg.RV.String())

	return nil
}

func launchPeerServer(cfg *config.Config) (err error) {
	var port = 0
	port, err = uploader.StartPeerServerProcess(cfg)
	if err == nil && port > 0 {
		cfg.RV.PeerPort = port
	}
	return
}

func registerToSuperNode(cfg *config.Config, register regist.SupernodeRegister) (
	*regist.RegisterResult, error) {
	defer func() {
		if r := recover(); r != nil {
			cfg.ClientLogger.Warnf("register fail but try to download from source, "+
				"reason:%d(%v)", cfg.BackSourceReason, r)
		}
	}()
	if cfg.Pattern == config.PatternSource {
		cfg.BackSourceReason = config.BackSourceReasonUserSpecified
		panic("user specified")
	}

	if len(cfg.Node) == 0 {
		cfg.BackSourceReason = config.BackSourceReasonNodeEmpty
		panic("supernode empty")
	}

	if cfg.Pattern == config.PatternP2P {
		if e := launchPeerServer(cfg); e != nil {
			cfg.ClientLogger.Warnf("start peer server error:%v, change to CDN pattern", e)
		}
	}

	result, e := register.Register(cfg.RV.PeerPort)
	if e != nil {
		if e.Code == config.TaskCodeNeedAuth {
			return nil, e
		}
		cfg.BackSourceReason = config.BackSourceReasonRegisterFail
		panic(e.Error())
	}
	cfg.RV.FileLength = result.FileLength
	util.Printer.Printf("client:%s connected to node:%s", cfg.RV.LocalIP, result.Node)
	return result, nil
}

func downloadFile(cfg *config.Config, supernodeAPI api.SupernodeAPI,
	register regist.SupernodeRegister, result *regist.RegisterResult) error {
	var getter downloader.Downloader
	if cfg.BackSourceReason > 0 {
		getter = downloader.NewBackDownloader(cfg, result)
	} else {
		util.Printer.Printf("start download by dragonfly")
		getter = downloader.NewP2PDownloader(cfg, supernodeAPI, register, result)
	}

	timeout := calculateTimeout(cfg.RV.FileLength, cfg.Timeout)
	err := downloader.DoDownloadTimeout(getter, timeout)
	success := "SUCCESS"
	if err != nil {
		cfg.ClientLogger.Error(err)
		success = "FAIL"
	} else if cfg.RV.FileLength < 0 && util.IsRegularFile(cfg.RV.RealTarget) {
		if info, err := os.Stat(cfg.RV.RealTarget); err == nil {
			cfg.RV.FileLength = info.Size()
		}
	}

	reportFinishedTask(cfg, getter)

	os.Remove(cfg.RV.TempTarget)
	cfg.ClientLogger.Infof("download %s cost:%.3fs length:%d reason:%d",
		success, time.Since(cfg.StartTime).Seconds(), cfg.RV.FileLength, cfg.BackSourceReason)
	return err
}

func reportFinishedTask(cfg *config.Config, getter downloader.Downloader) {
	if cfg.RV.PeerPort <= 0 {
		return
	}
	if getter, ok := getter.(*downloader.P2PDownloader); ok {
		uploader.FinishTask(cfg.RV.LocalIP, cfg.RV.PeerPort,
			cfg.RV.TaskFileName, cfg.RV.Cid,
			getter.GetTaskID(), getter.GetNode())
	}
}

func createTempTargetFile(targetDir string, sign string) (name string, e error) {
	var (
		f *os.File
	)

	defer func() {
		if e == nil {
			f.Close()
		}
	}()

	prefix := "dfget-" + sign + ".tmp-"
	f, e = ioutil.TempFile(targetDir, prefix)
	if e == nil {
		return f.Name(), e
	}

	f, e = os.OpenFile(path.Join(targetDir, fmt.Sprintf("%s%d", prefix, rand.Uint64())),
		os.O_CREATE|os.O_EXCL, 0755)
	if e == nil {
		return f.Name(), e
	}
	return "", e
}

func getTaskFileName(realTarget string, sign string) string {
	return filepath.Base(realTarget) + "-" + sign
}

func getCid(localIP string, sign string) string {
	return localIP + "-" + sign
}

func getTaskURL(rawURL string, filters []string) string {
	idx := strings.IndexByte(rawURL, '?')
	if len(filters) <= 0 || idx < 0 || idx >= len(rawURL)-1 {
		return rawURL
	}

	var params []string
	for _, p := range strings.Split(rawURL[idx+1:], "&") {
		kv := strings.Split(p, "=")
		if !util.ContainsString(filters, kv[0]) {
			params = append(params, p)
		}
	}
	if len(params) > 0 {
		return rawURL[:idx+1] + strings.Join(params, "&")
	}
	return rawURL[:idx]
}

func adjustSupernodeList(nodes []string) []string {
	switch nodesLen := len(nodes); nodesLen {
	case 0:
		return nodes
	case 1:
		return append(nodes, nodes[0])
	default:
		util.Shuffle(nodesLen, func(i, j int) {
			nodes[i], nodes[j] = nodes[j], nodes[i]
		})
		return append(nodes, nodes...)
	}
}

func checkConnectSupernode(nodes []string, clientLogger *logrus.Logger) (localIP string) {
	var (
		e    error
		port = 8002
	)
	for _, n := range nodes {
		nodeFields := strings.Split(n, ":")
		if len(nodeFields) == 2 {
			port, _ = strconv.Atoi(nodeFields[1])
		}
		if localIP, e = util.CheckConnect(nodeFields[0], port, 1000); e == nil {
			return localIP
		}
		if clientLogger != nil {
			clientLogger.Errorf("Connect to node:%s error: %v", n, e)
		}
	}
	return ""
}

func calculateTimeout(fileLength int64, defaultTimeoutSecond int) time.Duration {
	timeout := 5 * 60

	if defaultTimeoutSecond > 0 {
		timeout = defaultTimeoutSecond
	} else if fileLength > 0 {
		timeout = int(fileLength/(64*1024) + 10)
	}
	return time.Duration(timeout) * time.Second
}

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}
