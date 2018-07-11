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

	cfg "github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/dfget/util"
	"github.com/alibaba/Dragonfly/version"
	"github.com/spf13/pflag"
)

// Run is running cli.
func Run() {
	initialize()
}

func initialize() {
	initParameters()
	initLog()
	cfg.Ctx.ClientLogger.Infof("cmd params:%v", pflag.Args())
	initProperties()
	cfg.Ctx.ClientLogger.Infof("context:%s", cfg.Ctx)
}

func initParameters() {
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

func initProperties() {
	if err := cfg.Props.Load(cfg.Ctx.ConfigFile); err != nil {
	}

	if cfg.Ctx.Node == nil {
		cfg.Ctx.Node = cfg.Props.Node
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
	cfg.Ctx.ServerLogger = util.CreateLogger(logPath, "dfserver.log", logLevel, cfg.Ctx.Sign)
}

func panicIf(err error, msg string) {
	if err != nil {
		panic(fmt.Errorf("%s:%v", msg, err))
	}
}
