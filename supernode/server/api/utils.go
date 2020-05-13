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

package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-openapi/strfmt"
	"github.com/sirupsen/logrus"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/util"
)

// ValidateFunc validates the request parameters.
type ValidateFunc func(registry strfmt.Registry) error

// ParseJSONRequest parses the request JSON parameter to a target object.
func ParseJSONRequest(req io.Reader, target interface{}, validator ValidateFunc) error {
	if util.IsNil(target) {
		return errortypes.NewHTTPError(http.StatusInternalServerError, "nil target")
	}
	if err := json.NewDecoder(req).Decode(target); err != nil {
		if err == io.EOF {
			return errortypes.NewHTTPError(http.StatusBadRequest, "empty request")
		}
		return errortypes.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if validator != nil {
		if err := validator(strfmt.Default); err != nil {
			return errortypes.NewHTTPError(http.StatusBadRequest, err.Error())
		}
	}
	return nil
}

// EncodeResponse encodes response in json.
// The response body is empty if the data is nil or empty value.
func EncodeResponse(w http.ResponseWriter, code int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if util.IsNil(data) || data == "" {
		return nil
	}
	return json.NewEncoder(w).Encode(data)
}

// HandleErrorResponse handles err from server side and constructs response
// for client side.
func HandleErrorResponse(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case *errortypes.HTTPError:
		_ = EncodeResponse(w, e.Code, errResp(e.Code, e.Msg))
	default:
		// By default, server side returns code 500 if error happens.
		_ = EncodeResponse(w, http.StatusInternalServerError,
			errResp(http.StatusInternalServerError, e.Error()))
	}
}

// WrapHandler converts the 'api.HandlerFunc' into type 'http.HandlerFunc' and
// format the error response if any error happens.
func WrapHandler(handler HandlerFunc) http.HandlerFunc {
	pCtx := context.Background()

	return func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithCancel(pCtx)
		defer cancel()

		// Start to handle request.
		err := handler(ctx, w, req)
		if err != nil {
			// Handle error if request handling fails.
			HandleErrorResponse(w, err)
		}
		logrus.Debugf("%s %v err:%v", req.Method, req.URL, err)
	}
}

func errResp(code int, msg string) *types.ErrorResponse {
	return &types.ErrorResponse{
		Code:    int64(code),
		Message: msg,
	}
}
