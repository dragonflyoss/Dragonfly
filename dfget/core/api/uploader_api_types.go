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

import "bytes"

// ParseRateRequest wraps the request which is sent to uploader
// in order to calculate the rate limit dynamically.
type ParseRateRequest struct {
	TaskFileName string
	RateLimit    int
}

// CheckServerRequest wraps the request which is sent to uploader
// for check the peer server on port whether is available.
type CheckServerRequest struct {
	TaskFileName string
	DataDir      string
	TotalLimit   int
}

// FinishTaskRequest wraps the request which is sent to uploader
// in order to report a finished task.
type FinishTaskRequest struct {
	TaskFileName string `request:"taskFileName"`
	TaskID       string `request:"taskID"`
	ClientID     string `request:"cid"`
	Node         string `request:"superNode"`
}

// RegisterStreamTaskRequest wraps the request which is sent to uploader
// in order to initialize the stream cache window and create the instance in cache manager.
type RegisterStreamTaskRequest struct {
	TaskID     string `request:"taskID"`
	WindowSize string `request:"windowSize"`
	PieceSize  string `request:"pieceSize"`
	Node       string `request:"node"`
	CID        string `request:"CID"`
}

// UploadStreamPieceRequest wraps the request which is sent to uploader
// in order to upload the successfully downloaded piece to the uploader for cache.
type UploadStreamPieceRequest struct {
	TaskID    string        `request:"taskID"`
	PieceNum  int           `request:"pieceNum"`
	PieceSize int32         `request:"pieceSize"`
	Content   *bytes.Buffer `request:"content"`
}
