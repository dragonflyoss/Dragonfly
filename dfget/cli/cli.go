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

	cfg "github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/version"
)

// Run is running cli.
func Run() {
	initParameters()
	initProperties()
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
	path := "/etc/dragonfly.conf"
	if err := cfg.Props.Load(path); err != nil {
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

func panicIf(err error, msg string) {
	if err != nil {
		panic(fmt.Errorf("%s:%s", msg, err))
	}
}
