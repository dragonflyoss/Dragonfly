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
	"io"
	"net/http"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// ---------------------------------------------------------------------------
// handlers of preheat http apis

func (s *Server) createPreheatTask(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	request := &types.PreheatCreateRequest{}
	if err := parseRequest(req.Body, request, request.Validate); err != nil {
		return err
	}
	preheatID, err := s.PreheatMgr.Create(ctx, request)
	if err != nil {
		return err
	}
	resp := types.PreheatCreateResponse{ID: preheatID}
	return EncodeResponse(rw, http.StatusCreated, resp)
}

func (s *Server) getAllPreheatTasks(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	tasks, err := s.PreheatMgr.GetAll(ctx)
	if err != nil {
		return err
	}
	return EncodeResponse(rw, http.StatusOK, tasks)
}

func (s *Server) getPreheatTask(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	task, err := s.PreheatMgr.Get(ctx, id)
	if err != nil {
		return err
	}
	resp := types.PreheatInfo{
		ID:         task.ID,
		FinishTime: strfmt.NewDateTime(),
		StartTime:  strfmt.NewDateTime(),
		Status:     task.Status,
	}
	return EncodeResponse(rw, http.StatusOK, resp)
}

func (s *Server) deletePreheatTask(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	if err := s.PreheatMgr.Delete(ctx, id); err != nil {
		return err
	}
	return EncodeResponse(rw, http.StatusOK, true)
}

// ---------------------------------------------------------------------------
// helper functions

type validateFunc func(registry strfmt.Registry) error

func parseRequest(body io.Reader, request interface{}, validator validateFunc) error {
	if err := json.NewDecoder(body).Decode(request); err != nil {
		if err == io.EOF {
			return errortypes.New(http.StatusBadRequest, "empty body")
		}
		return errortypes.New(http.StatusBadRequest, err.Error())
	}
	if validator != nil {
		if err := validator(strfmt.NewFormats()); err != nil {
			return errortypes.New(http.StatusBadRequest, err.Error())
		}
	}
	return nil
}

// initPreheatHandlers register preheat apis
func initPreheatHandlers(s *Server, r *mux.Router) {
	handlers := []*HandlerSpec{
		{Method: http.MethodPost, Path: "/preheats", HandlerFunc: s.createPreheatTask},
		{Method: http.MethodGet, Path: "/preheats", HandlerFunc: s.getAllPreheatTasks},
		{Method: http.MethodGet, Path: "/preheats/{id}", HandlerFunc: s.getPreheatTask},
		{Method: http.MethodDelete, Path: "/preheats/{id}", HandlerFunc: s.deletePreheatTask},
	}
	// register API
	for _, h := range handlers {
		if h != nil {
			r.Path(versionMatcher + h.Path).Methods(h.Method).
				Handler(m.instrumentHandler(h.Path, postPreheatHandler(h.HandlerFunc)))
			r.Path("/api/v1" + h.Path).Methods(h.Method).
				Handler(m.instrumentHandler(h.Path, postPreheatHandler(h.HandlerFunc)))
			r.Path(h.Path).Methods(h.Method).
				Handler(m.instrumentHandler(h.Path, postPreheatHandler(h.HandlerFunc)))
		}
	}
}

func postPreheatHandler(h Handler) http.HandlerFunc {
	pctx := context.Background()

	return func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithCancel(pctx)
		defer cancel()

		// Start to handle request.
		err := h(ctx, w, req)
		if err != nil {
			// Handle error if request handling fails.
			handlePreheatErrorResponse(w, err)
		}
		logrus.Debugf("%s %v err:%v", req.Method, req.URL, err)
	}
}

func handlePreheatErrorResponse(w http.ResponseWriter, err error) {
	var (
		code   int
		errMsg string
	)

	// By default, daemon side returns code 500 if error happens.
	code = http.StatusInternalServerError
	if e, ok := err.(*errortypes.DfError); ok {
		code = e.Code
		errMsg = e.Msg
	}

	_ = EncodeResponse(w, code, types.ErrorResponse{
		Code:    int64(code),
		Message: errMsg,
	})
}
