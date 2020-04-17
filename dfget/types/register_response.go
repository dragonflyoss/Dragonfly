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

package types

import (
	"encoding/json"

	apiTypes "github.com/dragonflyoss/Dragonfly/apis/types"
)

// RegisterResponse is the response of register request.
type RegisterResponse struct {
	*BaseResponse
	Data *RegisterResponseData `json:"data,omitempty"`
}

func (res *RegisterResponse) String() string {
	if b, e := json.Marshal(res); e == nil {
		return string(b)
	}
	return ""
}

// RegisterResponseData is the data when registering supernode successfully.
type RegisterResponseData struct {
	TaskID     string             `json:"taskId"`
	FileLength int64              `json:"fileLength"`
	PieceSize  int32              `json:"pieceSize"`
	CDNSource  apiTypes.CdnSource `json:"cdnSource"`

	// in seed pattern, if peer selected as seed, AsSeed sets true.
	AsSeed bool `json:"asSeed"`

	// in seed pattern, if as seed, SeedTaskID is the taskID of seed file.
	SeedTaskID string `json:"seedTaskID"`
}
