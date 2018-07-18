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

package handler

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/alibaba/Dragonfly/version"
	"github.com/sirupsen/logrus"
)

// DebugInfo responds the inner http server running information.
func DebugInfo(w http.ResponseWriter, req *http.Request) {
	logrus.Debugf("access:%s", req.URL.String())

	if strings.HasPrefix(req.URL.Path, "/debug/pprof") {
		if req.URL.Path == "/debug/pprof/symbol" {
			pprof.Symbol(w, req)
		} else {
			pprof.Index(w, req)
		}
	} else if strings.HasPrefix(req.URL.Path, "/debug/version") {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain;charset=utf-8")
		w.Write([]byte(version.DFDaemonVersion))
	}

}
