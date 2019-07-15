package server

import (
	"context"
	"net/http"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/common/constants"
)

func (s *Server) ping(context context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte{'O', 'K'})
	return
}
func (s *Server) status(context context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	status := s.HaMgr.GetSupernodeStatus()
	return EncodeResponse(rw, http.StatusOK, &types.ResultInfo{
		Code: int32(status),
		Msg:  constants.GetMsgByCode(constants.Success),
	})
}
