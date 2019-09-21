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
	"net/http"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

func (s *Server) handlePieceError(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	taskID := mux.Vars(req)["id"]
	pieceRange := mux.Vars(req)["pieceRange"]

	reader := req.Body
	request := &types.PieceErrorRequest{}
	if err := json.NewDecoder(reader).Decode(request); err != nil {
		return errors.Wrap(errortypes.ErrInvalidValue, err.Error())
	}

	if err := request.Validate(strfmt.NewFormats()); err != nil {
		return errors.Wrap(errortypes.ErrInvalidValue, err.Error())
	}

	if stringutils.IsEmptyStr(request.DstPid) {
		return errors.Wrap(errortypes.ErrEmptyValue, "dstPid")
	}

	// Fulfill the taskID and pieceRange if they are empty.
	if stringutils.IsEmptyStr(request.TaskID) {
		request.TaskID = taskID
	}
	if stringutils.IsEmptyStr(request.Range) {
		request.Range = pieceRange
	}

	if err := s.PieceErrorMgr.HandlePieceError(ctx, request); err != nil {
		return err
	}

	rw.WriteHeader(http.StatusOK)
	return nil
}
