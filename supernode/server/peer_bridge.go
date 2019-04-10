package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	errTypes "github.com/dragonflyoss/Dragonfly/common/errors"
	dutil "github.com/dragonflyoss/Dragonfly/supernode/daemon/util"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

func (s *Server) registerPeer(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	reader := req.Body
	request := &types.PeerCreateRequest{}
	if err := json.NewDecoder(reader).Decode(request); err != nil {
		return errors.Wrap(errTypes.ErrInvalidValue, err.Error())
	}

	if err := request.Validate(strfmt.NewFormats()); err != nil {
		return errors.Wrap(errTypes.ErrInvalidValue, err.Error())
	}

	resp, err := s.PeerMgr.Register(ctx, request)
	if err != nil {
		return err
	}
	return EncodeResponse(rw, http.StatusCreated, resp)
}

// TODO: update the progress info.
func (s *Server) deRegisterPeer(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	id := mux.Vars(req)["id"]

	if err = s.PeerMgr.DeRegister(ctx, id); err != nil {
		return err
	}

	rw.WriteHeader(http.StatusOK)
	return nil
}

func (s *Server) getPeer(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	id := mux.Vars(req)["id"]

	peer, err := s.PeerMgr.Get(ctx, id)
	if err != nil {
		return err
	}

	return EncodeResponse(rw, http.StatusOK, peer)
}

// TODO: parse filter
func (s *Server) listPeers(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	filter, err := dutil.ParseFilter(req, nil)
	if err != nil {
		return err
	}
	peerList, err := s.PeerMgr.List(ctx, filter)
	if err != nil {
		return err
	}

	return EncodeResponse(rw, http.StatusOK, peerList)
}
