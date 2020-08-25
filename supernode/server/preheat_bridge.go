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
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/supernode/server/api"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/mux"
)

// ---------------------------------------------------------------------------
// handlers of preheat http apis

func (s *Server) createPreheatTask(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	request := &types.PreheatCreateRequest{}
	if err := api.ParseJSONRequest(req.Body, request, request.Validate); err != nil {
		return err
	}
	preheatID, err := s.PreheatMgr.Create(ctx, request)
	if err != nil {
		return httpErr(err)
	}
	resp := types.PreheatCreateResponse{ID: preheatID}
	return EncodeResponse(rw, http.StatusCreated, resp)
}

func (s *Server) getAllPreheatTasks(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	tasks, err := s.PreheatMgr.GetAll(ctx)
	if err != nil {
		return httpErr(err)
	}
	return EncodeResponse(rw, http.StatusOK, tasks)
}

func (s *Server) getPreheatTask(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	task, err := s.PreheatMgr.Get(ctx, id)
	if err != nil {
		return httpErr(err)
	}
	resp := types.PreheatInfo{
		ID:         task.ID,
		FinishTime: strfmt.DateTime(time.Unix(task.FinishTime/1000, task.FinishTime%1000*int64(time.Millisecond)).UTC()),
		StartTime:  strfmt.DateTime(time.Unix(task.StartTime/1000, task.StartTime%1000*int64(time.Millisecond)).UTC()),
		Status:     task.Status,
		ErrorMsg:   task.ErrorMsg,
	}
	return EncodeResponse(rw, http.StatusOK, resp)
}

func (s *Server) deletePreheatTask(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	if err := s.PreheatMgr.Delete(ctx, id); err != nil {
		return httpErr(err)
	}
	return EncodeResponse(rw, http.StatusOK, true)
}

// ---------------------------------------------------------------------------
// helper functions

func httpErr(err error) error {
	if e, ok := err.(*errortypes.DfError); ok {
		return errortypes.NewHTTPError(e.Code, e.Msg)
	}
	return err
}

// preheatHandlers returns all the preheats handlers.
func preheatHandlers(s *Server) []*api.HandlerSpec {
	return []*api.HandlerSpec{
		{Method: http.MethodPost, Path: "/preheats", HandlerFunc: s.createPreheatTask},
		{Method: http.MethodGet, Path: "/preheats", HandlerFunc: s.getAllPreheatTasks},
		{Method: http.MethodGet, Path: "/preheats/{id}", HandlerFunc: s.getPreheatTask},
		{Method: http.MethodDelete, Path: "/preheats/{id}", HandlerFunc: s.deletePreheatTask},
	}
}
