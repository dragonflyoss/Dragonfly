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
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/dragonflyoss/Dragonfly/supernode/server/api"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// versionMatcher defines to parse version url path.
const versionMatcher = "/v{version:[0-9.]+}"

var m = newMetrics(prometheus.DefaultRegisterer)

func createRouter(s *Server) *mux.Router {
	registerCoreHandlers(s)

	r := mux.NewRouter()
	if s.Config.Debug || s.Config.EnableProfiler {
		initDebugRoutes(r)
	}
	initAPIRoutes(r)
	return r
}

func registerCoreHandlers(s *Server) {
	registerV1(s)
	registerLegacy(s)
	registerSystem(s)
}

func registerV1(s *Server) {
	v1Handlers := []*api.HandlerSpec{
		// peer
		{Method: http.MethodPost, Path: "/peers", HandlerFunc: s.registerPeer},
		{Method: http.MethodDelete, Path: "/peers/{id}", HandlerFunc: s.deRegisterPeer},
		{Method: http.MethodGet, Path: "/peers/{id}", HandlerFunc: s.getPeer},
		{Method: http.MethodGet, Path: "/peers", HandlerFunc: s.listPeers},
		{Method: http.MethodGet, Path: "/tasks/{id}", HandlerFunc: s.getTaskInfo},
		{Method: http.MethodPost, Path: "/peer/network", HandlerFunc: s.fetchP2PNetworkInfo},
		{Method: http.MethodPost, Path: "/peer/heartbeat", HandlerFunc: s.reportPeerHealth},

		// task
		{Method: http.MethodDelete, Path: "/tasks/{id}", HandlerFunc: s.deleteTask},

		// piece
		{Method: http.MethodGet, Path: "/tasks/{id}/pieces/{pieceRange}/error", HandlerFunc: s.handlePieceError},
	}

	api.V1.Register(v1Handlers...)
	// add preheat APIs to v1 category
	api.V1.Register(preheatHandlers(s)...)
}

func registerSystem(s *Server) {
	systemHandlers := []*api.HandlerSpec{
		// system
		{Method: http.MethodGet, Path: "/_ping", HandlerFunc: s.ping},
		{Method: http.MethodGet, Path: "/version", HandlerFunc: version.HandlerWithCtx},

		// metrics
		{Method: http.MethodGet, Path: "/metrics", HandlerFunc: handleMetrics},
		{Method: http.MethodPost, Path: "/task/metrics", HandlerFunc: m.handleMetricsReport},
	}
	api.Legacy.Register(systemHandlers...)
}

func registerLegacy(s *Server) {
	legacyHandlers := []*api.HandlerSpec{
		// v0.3
		{Method: http.MethodPost, Path: "/peer/registry", HandlerFunc: s.registry},
		{Method: http.MethodGet, Path: "/peer/task", HandlerFunc: s.pullPieceTask},
		{Method: http.MethodGet, Path: "/peer/piece/suc", HandlerFunc: s.reportPiece},
		{Method: http.MethodGet, Path: "/peer/service/down", HandlerFunc: s.reportServiceDown},
		{Method: http.MethodGet, Path: "/peer/piece/error", HandlerFunc: s.reportPieceError},
		{Method: http.MethodPost, Path: "/peer/network", HandlerFunc: s.fetchP2PNetworkInfo},
		{Method: http.MethodPost, Path: "/peer/heartbeat", HandlerFunc: s.reportPeerHealth},
	}
	api.Legacy.Register(legacyHandlers...)
	api.Legacy.Register(preheatHandlers(s)...)
}

func initAPIRoutes(r *mux.Router) {
	add := func(prefix string, h *api.HandlerSpec) {
		path := h.Path
		if path == "" || path[0] != '/' {
			path = "/" + path
		}
		if !strings.HasPrefix(path, prefix) {
			path = prefix + h.Path
		}
		r.Path(path).Methods(h.Method).
			Handler(m.instrumentHandler(path, api.WrapHandler(h.HandlerFunc)))
		// for sdk client
		r.Path(versionMatcher + path).Methods(h.Method).
			Handler(m.instrumentHandler(path, api.WrapHandler(h.HandlerFunc)))
	}

	api.V1.Range(add)
	api.Extension.Range(add)
	api.Legacy.Range(add)
}

func initDebugRoutes(r *mux.Router) {
	r.PathPrefix("/debug/pprof/cmdline").HandlerFunc(pprof.Cmdline)
	r.PathPrefix("/debug/pprof/profile").HandlerFunc(pprof.Profile)
	r.PathPrefix("/debug/pprof/symbol").HandlerFunc(pprof.Symbol)
	r.PathPrefix("/debug/pprof/trace").HandlerFunc(pprof.Trace)
	r.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
}

func handleMetrics(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	promhttp.Handler().ServeHTTP(rw, req)
	return nil
}

// EncodeResponse encodes response in json and sends it.
func EncodeResponse(rw http.ResponseWriter, statusCode int, data interface{}) error {
	return api.SendResponse(rw, statusCode, data)
}
