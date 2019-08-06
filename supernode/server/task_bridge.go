package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

func (s *Server) registerTask(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	reader := req.Body
	request := &types.TaskCreateRequest{}
	if err := json.NewDecoder(reader).Decode(request); err != nil {
		return errors.Wrap(errortypes.ErrInvalidValue, err.Error())
	}

	if err := request.Validate(strfmt.NewFormats()); err != nil {
		return errors.Wrap(errortypes.ErrInvalidValue, err.Error())
	}

	resp, err := s.TaskMgr.Register(ctx, request)
	if err != nil {
		return err
	}
	return EncodeResponse(rw, http.StatusCreated, resp)
}

func (s *Server) deleteTask(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	id := mux.Vars(req)["id"]

	s.GCMgr.GCTask(ctx, id)

	rw.WriteHeader(http.StatusOK)
	return nil
}

func (s *Server) getTask(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	id := mux.Vars(req)["id"]

	task, err := s.TaskMgr.Get(ctx, id)
	if err != nil {
		return err
	}

	return EncodeResponse(rw, http.StatusOK, task)
}

func (s *Server) listTasks(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	taskList, err := s.TaskMgr.List(ctx, nil)
	if err != nil {
		return err
	}

	return EncodeResponse(rw, http.StatusOK, taskList)
}

func (s *Server) updateTask(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	id := mux.Vars(req)["id"]

	reader := req.Body
	request := &types.TaskInfo{}
	if err := json.NewDecoder(reader).Decode(request); err != nil {
		return errors.Wrap(errortypes.ErrInvalidValue, err.Error())
	}

	if err := request.Validate(strfmt.NewFormats()); err != nil {
		return errors.Wrap(errortypes.ErrInvalidValue, err.Error())
	}

	if err := s.TaskMgr.Update(ctx, id, request); err != nil {
		return err
	}

	rw.WriteHeader(http.StatusOK)
	return nil
}
