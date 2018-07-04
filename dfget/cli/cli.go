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

	"github.com/alibaba/Dragonfly/version"
)

// Run is running cli.
func Run() {
	initParameters(os.Args[1:])

	if Params.Help {
		Usage()
		os.Exit(0)
	}

	if Params.Version {
		fmt.Println(version.DFGetVersion)
		os.Exit(0)
	}
}
