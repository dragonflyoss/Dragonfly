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
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/downloader"
	backDown "github.com/dragonflyoss/Dragonfly/dfget/core/downloader/back_downloader"
	p2pDown "github.com/dragonflyoss/Dragonfly/dfget/core/downloader/p2p_downloader"
	"github.com/dragonflyoss/Dragonfly/dfget/core/regist"
	"github.com/dragonflyoss/Dragonfly/dfget/core/uploader"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
	"github.com/dragonflyoss/Dragonfly/pkg/constants"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/printer"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Start function creates a new task and starts it to download file.
func Start(cfg *config.Config) *errortypes.DfError {
	var (
		supernodeAPI = api.NewSupernodeAPI()
		register     = regist.NewSupernodeRegister(cfg, supernodeAPI)
		err          error
		result       *regist.RegisterResult
	)

	printer.Println(fmt.Sprintf("--%s--  %s",
		cfg.StartTime.Format(config.DefaultTimestampFormat), cfg.URL))

	if err = prepare(cfg); err != nil {
		return errortypes.New(config.CodePrepareError, err.Error())
	}

	if result, err = registerToSuperNode(cfg, register); err != nil {
		return errortypes.New(config.CodeRegisterError, err.Error())
	}

	if err = downloadFile(cfg, supernodeAPI, register, result); err != nil {
		return errortypes.New(config.CodeDownloadError, err.Error())
	}

	return nil
}

// prepare the RV-related information and create the corresponding files.
func prepare(cfg *config.Config) (err error) {
	printer.Printf("dfget version:%s", version.DFGetVersion)
	printer.Printf("workspace:%s", cfg.WorkHome)
	printer.Printf("sign:%s", cfg.Sign)
	logrus.Infof("target file path:%s", cfg.Output)

	rv := &cfg.RV

	rv.RealTarget = cfg.Output
	rv.TargetDir = path.Dir(rv.RealTarget)
	if err = fileutils.CreateDirectory(rv.TargetDir); err != nil {
		return err
	}

	if cfg.RV.TempTarget, err = createTempTargetFile(rv.TargetDir, cfg.Sign); err != nil {
		return err
	}

	if err = fileutils.CreateDirectory(path.Dir(rv.MetaPath)); err != nil {
		return err
	}
	if err = fileutils.CreateDirectory(cfg.WorkHome); err != nil {
		return err
	}
	if err = fileutils.CreateDirectory(rv.SystemDataDir); err != nil {
		return err
	}
	rv.DataDir = cfg.RV.SystemDataDir

	cfg.Node = adjustSupernodeList(cfg.Node)
	if stringutils.IsEmptyStr(rv.LocalIP) {
		rv.LocalIP = checkConnectSupernode(cfg.Node)
	}
	rv.Cid = getCid(rv.LocalIP, cfg.Sign)
	rv.TaskFileName = getTaskFileName(rv.RealTarget, cfg.Sign)
	rv.TaskURL = netutils.FilterURLParam(cfg.URL, cfg.Filter)
	logrus.Info("runtimeVariable: " + cfg.RV.String())

	return nil
}

func launchPeerServer(cfg *config.Config) (err error) {
	port, err := uploader.StartPeerServerProcess(cfg)
	if err == nil && port > 0 {
		cfg.RV.PeerPort = port
	}
	return
}

func registerToSuperNode(cfg *config.Config, register regist.SupernodeRegister) (
	*regist.RegisterResult, error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Warnf("register fail but try to download from source, "+
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
			logrus.Warnf("start peer server error:%v, change to CDN pattern", e)
			cfg.Pattern = config.PatternCDN
		}
	}

	result, e := register.Register(cfg.RV.PeerPort)
	if e != nil {
		if e.Code == constants.CodeNeedAuth {
			return nil, e
		}
		cfg.BackSourceReason = config.BackSourceReasonRegisterFail
		panic(e.Error())
	}
	cfg.RV.FileLength = result.FileLength
	printer.Printf("client:%s connected to node:%s", cfg.RV.LocalIP, result.Node)
	return result, nil
}

func downloadFile(cfg *config.Config, supernodeAPI api.SupernodeAPI,
	register regist.SupernodeRegister, result *regist.RegisterResult) error {
	var getter downloader.Downloader
	if cfg.BackSourceReason > 0 {
		getter = backDown.NewBackDownloader(cfg, result)
	} else {
		printer.Printf("start download by dragonfly...")
		getter = p2pDown.NewP2PDownloader(cfg, supernodeAPI, register, result)
	}

	timeout := netutils.CalculateTimeout(cfg.RV.FileLength, cfg.MinRate, config.DefaultMinRate, 10*time.Second)
	if timeout == 0 && cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}
	success := true
	err := downloader.DoDownloadTimeout(getter, timeout)
	if err != nil {
		success = false
	} else if cfg.RV.FileLength < 0 && fileutils.IsRegularFile(cfg.RV.RealTarget) {
		if info, err := os.Stat(cfg.RV.RealTarget); err == nil {
			cfg.RV.FileLength = info.Size()
		}
	}

	reportFinishedTask(cfg, getter)
	os.Remove(cfg.RV.TempTarget)

	downloadTime := time.Since(cfg.StartTime).Seconds()
	// upload metrics to supernode only if pattern is p2p or cdn and result is not nil
	if cfg.Pattern != config.PatternSource && result != nil {
		reportMetrics(cfg, supernodeAPI, downloadTime, result.TaskID, success)
	}

	if success {
		logrus.Infof("download SUCCESS from supernode %s cost:%.3fs length:%d",
			cfg.Node, time.Since(cfg.StartTime).Seconds(), cfg.RV.FileLength)
	} else {
		logrus.Infof("download FAIL from supernode %s cost:%.3fs length:%d reason:%d",
			cfg.Node, time.Since(cfg.StartTime).Seconds(), cfg.RV.FileLength, cfg.BackSourceReason)
	}
	return err
}

func reportFinishedTask(cfg *config.Config, getter downloader.Downloader) {
	if cfg.RV.PeerPort <= 0 {
		return
	}
	if getter, ok := getter.(*p2pDown.P2PDownloader); ok {
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

func checkConnectSupernode(nodes []string) (localIP string) {
	var (
		e error
	)
	for _, n := range nodes {
		ip, port := netutils.GetIPAndPortFromNode(n, config.DefaultSupernodePort)
		if localIP, e = httputils.CheckConnect(ip, port, 1000); e == nil {
			return localIP
		}
		logrus.Errorf("Connect to node:%s error: %v", n, e)
	}
	return ""
}

func reportMetrics(cfg *config.Config, supernodeAPI api.SupernodeAPI, downloadTime float64, taskID string, success bool) {
	req := &types.TaskMetricsRequest{
		BacksourceReason: strconv.Itoa(cfg.BackSourceReason),
		IP:               cfg.RV.LocalIP,
		CID:              cfg.RV.Cid,
		CallSystem:       cfg.CallSystem,
		Duration:         downloadTime,
		FileLength:       cfg.RV.FileLength,
		Port:             int32(cfg.RV.PeerPort),
		Success:          success,
		TaskID:           taskID,
	}
	for _, node := range cfg.Node {
		resp, err := supernodeAPI.ReportMetrics(node, req)
		if err != nil {
			logrus.Errorf("failed to report metrics to supernode %s: %v", node, err)
		}
		if resp != nil && resp.IsSuccess() {
			return
		}
	}
	return
}
