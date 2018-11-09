package server

import (
	"context"
	"net/http"
)

// HandlerSpec is used to describe a HTTP API.
type HandlerSpec struct {
	Method      string
	Path        string
	HandlerFunc Handler
}

// Handler is the http request handler.
type Handler func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error

// NewHandlerSpec constructs a brand new HandlerSpec.
func NewHandlerSpec(method, path string, handler Handler) *HandlerSpec {
	return &HandlerSpec{
		Method:      method,
		Path:        path,
		HandlerFunc: handler,
	}
}
