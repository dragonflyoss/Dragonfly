/*
 * Copyright 1999-2018 Alibaba Group.
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

	"github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/dfget/core/api"
	"github.com/alibaba/Dragonfly/dfget/core/downloader"
	"github.com/alibaba/Dragonfly/dfget/errors"
	"github.com/alibaba/Dragonfly/dfget/util"
	"github.com/alibaba/Dragonfly/version"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Start function creates a new task and starts it to download file.
func Start(ctx *config.Context) *errors.DFGetError {
	var (
		supernodeAPI = api.NewSupernodeAPI()
		register     = NewSupernodeRegister(ctx, supernodeAPI)
		err          error
		result       *RegisterResult
	)

	util.Printer.Println(fmt.Sprintf("--%s--  %s",
		ctx.StartTime.Format(config.DefaultTimestampFormat), ctx.URL))

	if err = prepare(ctx); err != nil {
		return errors.New(1100, err.Error())
	}

	if result, err = registerToSuperNode(ctx, register); err != nil {
		return errors.New(1200, err.Error())
	}

	if err = downloadFile(ctx, supernodeAPI, register, result); err != nil {
		return errors.New(1300, err.Error())
	}

	return nil
}

func prepare(ctx *config.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	util.Printer.Printf("dfget version:%s", version.DFGetVersion)
	util.Printer.Printf("workspace:%s sign:%s", ctx.WorkHome, ctx.Sign)
	ctx.ClientLogger.Infof("target file path:%s", ctx.Output)

	rv := &ctx.RV

	rv.RealTarget = ctx.Output
	rv.TargetDir = path.Dir(rv.RealTarget)
	panicIf(util.CreateDirectory(rv.TargetDir))
	ctx.RV.TempTarget, err = createTempTargetFile(rv.TargetDir, ctx.Sign)
	panicIf(err)

	panicIf(util.CreateDirectory(path.Dir(rv.MetaPath)))
	panicIf(util.CreateDirectory(ctx.WorkHome))
	panicIf(util.CreateDirectory(rv.SystemDataDir))
	rv.DataDir = ctx.RV.SystemDataDir

	ctx.Node = adjustSupernodeList(ctx.Node)
	rv.LocalIP = checkConnectSupernode(ctx.Node)
	rv.Cid = getCid(rv.LocalIP, ctx.Sign)
	rv.TaskFileName = getTaskFileName(rv.RealTarget, ctx.Sign)
	rv.TaskURL = getTaskURL(ctx.URL, ctx.Filter)
	ctx.ClientLogger.Info("runtimeVariable: " + ctx.RV.String())

	return nil
}

func launchPeerServer(ctx *config.Context) error {
	return fmt.Errorf("not implemented")
}

func registerToSuperNode(ctx *config.Context, register SupernodeRegister) (*RegisterResult, error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.ClientLogger.Warnf("register fail but try to download from source, "+
				"reason:%d(%v)", ctx.BackSourceReason, r)
		}
	}()
	if ctx.Pattern == config.PatternSource {
		ctx.BackSourceReason = config.BackSourceReasonUserSpecified
		panic("user specified")
	}

	if len(ctx.Node) == 0 {
		ctx.BackSourceReason = config.BackSourceReasonNodeEmpty
		panic("supernode empty")
	}

	if ctx.Pattern == config.PatternP2P {
		if e := launchPeerServer(ctx); e != nil {
			ctx.ClientLogger.Warnf("start peer server error:%v, change to CDN pattern", e)
		}
	}

	result, e := register.Register(ctx.RV.PeerPort)
	if e != nil {
		if e.Code == config.TaskCodeNeedAuth {
			return nil, e
		}
		ctx.BackSourceReason = config.BackSourceReasonRegisterFail
		panic(e.Error())
	}
	ctx.RV.FileLength = result.FileLength
	util.Printer.Printf("client:%s connected to node:%s", ctx.RV.LocalIP, result.Node)
	util.Printer.Printf("start download by dragonfly")
	return result, nil
}

func downloadFile(ctx *config.Context, supernodeAPI api.SupernodeAPI,
	register SupernodeRegister, result *RegisterResult) error {
	var getter downloader.Downloader
	if ctx.BackSourceReason > 0 {
		getter = &downloader.BackDownloader{}
	} else {
		getter = &downloader.P2PDownloader{}
	}
	getter.Run()
	return nil
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

func getTaskPath(taskFileName string) string {
	if !util.IsEmptyStr(taskFileName) {
		return config.PeerHTTPPathPrefix + taskFileName
	}
	return ""
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
		if config.Ctx.ClientLogger != nil {
			config.Ctx.ClientLogger.Errorf("connect to node:%s error: %v", n, e)
		}
	}
	return ""
}

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}
