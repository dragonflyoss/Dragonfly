package server

import (
	"context"
	"net/http"
)

func (s *Server) ping(context context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte{'O', 'K'})
	return
}
