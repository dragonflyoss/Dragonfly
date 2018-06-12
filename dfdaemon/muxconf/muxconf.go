// Copyright 1999-2017 Alibaba Group.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package muxconf

import (
	"net/http"

	"github.com/alibaba/Dragonfly/dfdaemon/handler"
)

// InitMux initialize web router of dfdaemon
func InitMux() {
	router := map[string]func(http.ResponseWriter, *http.Request){
		"/":       handler.Process,
		"/args":   handler.GetArgs,
		"/debug/": handler.DebugInfo,
		"/env":    handler.GetEnv,
	}

	for key, value := range router {
		http.HandleFunc(key, value)
	}
}
