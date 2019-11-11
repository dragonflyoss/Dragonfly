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
