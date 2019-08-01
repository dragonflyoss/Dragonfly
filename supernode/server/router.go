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

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/pprof"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// versionMatcher defines to parse version url path.
const versionMatcher = "/v{version:[0-9.]+}"

var m = newMetrics(prometheus.DefaultRegisterer)

func initRoute(s *Server) *mux.Router {
	r := mux.NewRouter()
	handlers := []*HandlerSpec{
		// system
		{Method: http.MethodGet, Path: "/_ping", HandlerFunc: s.ping},
		{Method: http.MethodGet, Path: "/version", HandlerFunc: version.HandlerWithCtx},

		// v0.3
		{Method: http.MethodPost, Path: "/peer/registry", HandlerFunc: s.registry},
		{Method: http.MethodGet, Path: "/peer/task", HandlerFunc: s.pullPieceTask},
		{Method: http.MethodGet, Path: "/peer/piece/suc", HandlerFunc: s.reportPiece},
		{Method: http.MethodGet, Path: "/peer/service/down", HandlerFunc: s.reportServiceDown},

		// v1
		// peer
		{Method: http.MethodPost, Path: "/peers", HandlerFunc: s.registerPeer},
		{Method: http.MethodDelete, Path: "/peers/{id}", HandlerFunc: s.deRegisterPeer},
		{Method: http.MethodGet, Path: "/peers/{id}", HandlerFunc: s.getPeer},
		{Method: http.MethodGet, Path: "/peers", HandlerFunc: s.listPeers},

		{Method: http.MethodGet, Path: "/metricsutils", HandlerFunc: handleMetrics},
	}

	// register API
	for _, h := range handlers {
		if h != nil {
			r.Path(versionMatcher + h.Path).Methods(h.Method).Handler(m.instrumentHandler(h.Path, filter(h.HandlerFunc)))
			r.Path(h.Path).Methods(h.Method).Handler(m.instrumentHandler(h.Path, filter(h.HandlerFunc)))
		}
	}

	if s.Config.Debug || s.Config.EnableProfiler {
		r.PathPrefix("/debug/pprof/cmdline").HandlerFunc(pprof.Cmdline)
		r.PathPrefix("/debug/pprof/profile").HandlerFunc(pprof.Profile)
		r.PathPrefix("/debug/pprof/symbol").HandlerFunc(pprof.Symbol)
		r.PathPrefix("/debug/pprof/trace").HandlerFunc(pprof.Trace)
		r.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
	}
	return r
}

func handleMetrics(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	promhttp.Handler().ServeHTTP(rw, req)
	return nil
}

func filter(handler Handler) http.HandlerFunc {
	pctx := context.Background()

	return func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithCancel(pctx)
		defer cancel()

		// Start to handle request.

		if err := handler(ctx, w, req); err != nil {
			// Handle error if request handling fails.
			HandleErrorResponse(w, err)
		}
		return
	}
}

// EncodeResponse encodes response in json.
func EncodeResponse(rw http.ResponseWriter, statusCode int, data interface{}) error {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)
	return json.NewEncoder(rw).Encode(data)
}

// HandleErrorResponse handles err from daemon side and constructs response for client side.
func HandleErrorResponse(w http.ResponseWriter, err error) {
	var (
		code   int
		errMsg string
	)

	// By default, daemon side returns code 500 if error happens.
	code = http.StatusInternalServerError
	errMsg = NewResultInfoWithError(err).Error()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	resp := types.Error{
		Message: errMsg,
	}
	enc.Encode(resp)
}
