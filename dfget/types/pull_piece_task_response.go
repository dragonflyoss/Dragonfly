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

	"github.com/dragonflyoss/Dragonfly/pkg/constants"
)

// PullPieceTaskResponse is the response of PullPieceTaskRequest.
type PullPieceTaskResponse struct {
	*BaseResponse
	Data json.RawMessage `json:"data,omitempty"`
	data interface{}
}

func (res *PullPieceTaskResponse) String() string {
	if b, e := json.Marshal(res); e == nil {
		return string(b)
	}
	return ""
}

// FinishData gets structured data from json.RawMessage when the task is finished.
func (res *PullPieceTaskResponse) FinishData() *PullPieceTaskResponseFinishData {
	if res.Code != constants.CodePeerFinish || res.Data == nil {
		return nil
	}
	if res.data == nil {
		data := new(PullPieceTaskResponseFinishData)
		if e := json.Unmarshal(res.Data, data); e != nil {
			return nil
		}
		res.data = data
	}
	return res.data.(*PullPieceTaskResponseFinishData)
}

// ContinueData gets structured data from json.RawMessage when the task is continuing.
func (res *PullPieceTaskResponse) ContinueData() []*PullPieceTaskResponseContinueData {
	if res.Code != constants.CodePeerContinue || res.Data == nil {
		return nil
	}
	if res.data == nil {
		var data []*PullPieceTaskResponseContinueData
		if e := json.Unmarshal(res.Data, &data); e != nil {
			return nil
		}
		res.data = data
	}
	return res.data.([]*PullPieceTaskResponseContinueData)
}

// PullPieceTaskResponseFinishData is the data when successfully pulling piece task
// and the task is finished.
type PullPieceTaskResponseFinishData struct {
	Md5        string `json:"md5"`
	FileLength int64  `json:"fileLength"`
}

func (data *PullPieceTaskResponseFinishData) String() string {
	b, _ := json.Marshal(data)
	return string(b)
}

// PullPieceTaskResponseContinueData is the data when successfully pulling piece task
// and the task is continuing.
type PullPieceTaskResponseContinueData struct {
	Range     string `json:"range"`
	PieceNum  int    `json:"pieceNum"`
	PieceSize int32  `json:"pieceSize"`
	PieceMd5  string `json:"pieceMd5"`
	Cid       string `json:"cid"`
	PeerIP    string `json:"peerIp"`
	PeerPort  int    `json:"peerPort"`
	Path      string `json:"path"`
	DownLink  int    `json:"downLink"`
}

func (data *PullPieceTaskResponseContinueData) String() string {
	b, _ := json.Marshal(data)
	return string(b)
}
