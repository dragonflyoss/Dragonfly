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

package handler

import (
	"net/http"
	// pprof will inject handlers for users to profile this program
	_ "net/http/pprof"

	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// New returns a new http mux for dfdaemon.
func New() *http.ServeMux {
	s := http.DefaultServeMux
	s.HandleFunc("/args", getArgs)
	s.HandleFunc("/env", getEnv)
	s.HandleFunc("/debug/version", version.Handler)
	s.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)
	return s
}
