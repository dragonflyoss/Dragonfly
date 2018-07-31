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
	"time"

	"github.com/alibaba/Dragonfly/dfget/config"
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
	var err error

	util.Printer.Println(fmt.Sprintf("--%s--  %s",
		ctx.StartTime.Format(config.DefaultTimestampFormat), ctx.URL))

	if err = prepare(ctx); err != nil {
		return errors.New(1100, err.Error())
	}

	if err = registerToSuperNode(ctx); err != nil {
		return errors.New(1200, err.Error())
	}

	if err = downloadFile(ctx); err != nil {
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

	ctx.RealTarget = ctx.Output
	ctx.TargetDir = path.Dir(ctx.RealTarget)
	panicIf(util.CreateDirectory(ctx.TargetDir))
	ctx.TempTarget, err = createTempTargetFile(ctx.TargetDir, ctx.Sign)
	panicIf(err)

	panicIf(util.CreateDirectory(path.Dir(ctx.MetaPath)))
	panicIf(util.CreateDirectory(ctx.WorkHome))
	panicIf(util.CreateDirectory(ctx.SystemDataDir))
	ctx.DataDir = ctx.SystemDataDir

	return nil
}

func registerToSuperNode(ctx *config.Context) error {
	return register(ctx)
}

func downloadFile(ctx *config.Context) error {
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

	prefix := "dfget" + sign + ".tmp-"
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

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}
