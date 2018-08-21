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

// Package cli controls the flow of all procedures. It parses arguments,
// invokes registrar to register itself on supernode, assigns tasks to
// downloader and other modules.
package cli

import (
	"fmt"
	"os"
	"path"
	"time"

	cfg "github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/dfget/core"
	"github.com/alibaba/Dragonfly/dfget/errors"
	"github.com/alibaba/Dragonfly/dfget/util"
	"github.com/alibaba/Dragonfly/version"
)

// Run is running cli.
func Run() {
	initialize()
	err := core.Start(cfg.Ctx)
	if err != nil {
		util.Printer.Println(resultMsg(cfg.Ctx, time.Now(), err))
		os.Exit(err.Code)
	}
	util.Printer.Println(resultMsg(cfg.Ctx, time.Now(), err))
	os.Exit(0)
}

func initialize() {
	initParameters()
	initLog()
	cfg.AssertContext(cfg.Ctx)
	cfg.Ctx.ClientLogger.Infof("cmd params:%q", os.Args)
	initProperties()
	cfg.Ctx.ClientLogger.Infof("context:%s", cfg.Ctx)
}

func initParameters() {
	if len(os.Args) < 2 {
		fmt.Println("please use '--help' or '-h' to show the help information")
		os.Exit(0)
	}

	setupFlags(os.Args[1:])
	if cfg.Ctx.Help {
		Usage()
		os.Exit(0)
	}

	if cfg.Ctx.Version {
		fmt.Println(version.DFGetVersion)
		os.Exit(0)
	}
}

// initProperties
func initProperties() {
	for _, v := range cfg.Ctx.ConfigFiles {
		if err := cfg.Props.Load(v); err == nil {
			cfg.Ctx.ClientLogger.Debugf("initProperties[%s] success: %v", v, cfg.Props)
			break
		} else {
			cfg.Ctx.ClientLogger.Debugf("initProperties[%s] fail: %v", v, err)
		}
	}

	if cfg.Ctx.Node == nil {
		cfg.Ctx.Node = cfg.Props.Nodes
	}

	if cfg.Ctx.LocalLimit == 0 {
		cfg.Ctx.LocalLimit = cfg.Props.LocalLimit
	}

	if cfg.Ctx.TotalLimit == 0 {
		cfg.Ctx.TotalLimit = cfg.Props.TotalLimit
	}

	if cfg.Ctx.ClientQueueSize == 0 {
		cfg.Ctx.ClientQueueSize = cfg.Props.ClientQueueSize
	}
}

func initLog() {
	var (
		logPath  = path.Join(cfg.Ctx.WorkHome, "logs")
		logLevel = "info"
	)
	if cfg.Ctx.Verbose {
		logLevel = "debug"
	}
	cfg.Ctx.ClientLogger = util.CreateLogger(logPath, "dfclient.log", logLevel, cfg.Ctx.Sign)
	if cfg.Ctx.Console {
		util.AddConsoleLog(cfg.Ctx.ClientLogger)
	}
	if cfg.Ctx.Pattern == "p2p" {
		cfg.Ctx.ServerLogger = util.CreateLogger(logPath, "dfserver.log", logLevel, cfg.Ctx.Sign)
	}
}

func resultMsg(ctx *cfg.Context, end time.Time, e *errors.DFGetError) string {
	if e != nil {
		return fmt.Sprintf("download FAIL(%d) cost:%.3fs length:%d reason:%d error:%v",
			e.Code, end.Sub(ctx.StartTime).Seconds(), ctx.RV.FileLength,
			ctx.BackSourceReason, e)
	}
	return fmt.Sprintf("download SUCCESS(0) cost:%.3fs length:%d reason:%d",
		end.Sub(ctx.StartTime).Seconds(), ctx.RV.FileLength, ctx.BackSourceReason)
}
