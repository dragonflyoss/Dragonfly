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

package util

import (
	"fmt"
	"io"
)

// StdPrinter output info to console directly.
type StdPrinter struct {
	Out io.Writer
}

// Println output info to console directly.
func (sp *StdPrinter) Println(msg string) {
	if sp.Out != nil {
		fmt.Fprintln(sp.Out, msg)
	}
}

// Printf formats according to a format specifier.
func (sp *StdPrinter) Printf(format string, a ...interface{}) {
	if sp.Out != nil {
		fmt.Fprintf(sp.Out, format+"\n", a...)
	}
}
