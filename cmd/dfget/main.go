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

package main

import (
	"github.com/alibaba/Dragonfly/dfget/cli"
)

func main() {
	// TODO: refactor 'dfget' with GoLang
	// the current dfget is written by python in 'src/getter'
	// This main function is just the entry of dfget, the other operations
	// must be controlled in package CLI.
	cli.Run()
}
