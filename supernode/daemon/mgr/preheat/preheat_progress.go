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
package preheat

import (
	"bytes"
	"os/exec"
)

type PreheatProgress struct {
	output string
	cmd *exec.Cmd
	errmsg *bytes.Buffer
}

func NewPreheatProgress(output string, cmd *exec.Cmd) *PreheatProgress {
	p := &PreheatProgress{
		output: output,
		cmd: cmd,
		errmsg: bytes.NewBuffer(make([]byte, 0, 128)),
	}
	cmd.Stderr = p.errmsg
	cmd.Stdout = p.errmsg
	return p
}