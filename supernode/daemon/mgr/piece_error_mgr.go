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

package mgr

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// PieceErrorMgr as an interface defines all operations to handle piece errors.
type PieceErrorMgr interface {
	// StartHandleError starts a goroutine to handle the piece error.
	StartHandleError(ctx context.Context)

	// HandlePieceError the peer should report the error with related info when
	// it failed to download a piece from supernode.
	// And the supernode should handle the piece Error and do some repair operations.
	HandlePieceError(ctx context.Context, pieceErrorRequest *types.PieceErrorRequest) error
}
