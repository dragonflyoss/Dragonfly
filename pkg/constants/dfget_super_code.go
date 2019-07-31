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

package constants

// This file defines the code required for both dfget and supernode.

var cmmap = make(map[int]string)

func init() {
	cmmap[Success] = "success"

	cmmap[CodeSystemError] = "system error"
	cmmap[CodeParamError] = "param is illegal"
	cmmap[CodeTargetNotFound] = "target not found"

	cmmap[CodePeerFinish] = "peer task end"
	cmmap[CodePeerContinue] = "peer task go on"
	cmmap[CodePeerWait] = "peer task wait"
	cmmap[CodePeerLimited] = "peer down limit"
	cmmap[CodeSuperFail] = "super node sync source fail"
	cmmap[CodeUnknownError] = "unknown error"
	cmmap[CodeTaskConflict] = "task conflict"
	cmmap[CodeURLNotReachable] = "url is not reachable"
	cmmap[CodeNeedAuth] = "need auth"
	cmmap[CodeWaitAuth] = "wait auth"
}

// GetMsgByCode gets the description of the code.
func GetMsgByCode(code int) string {
	if v, ok := cmmap[code]; ok {
		return v
	}
	return ""
}

const (
	// HTTPError represents that there is an error between client and server.
	HTTPError = -100
)

/* the response code returned by supernode */
const (
	// Success represents the request is success.
	Success = 200

	CodeSystemError    = 500
	CodeParamError     = 501
	CodeTargetNotFound = 502

	CodePeerFinish      = 600
	CodePeerContinue    = 601
	CodePeerWait        = 602
	CodePeerLimited     = 603
	CodeSuperFail       = 604
	CodeUnknownError    = 605
	CodeTaskConflict    = 606
	CodeURLNotReachable = 607
	CodeNeedAuth        = 608
	CodeWaitAuth        = 609
	CodeSourceError     = 610
	CodeGetPieceReport  = 611
	CodeGetPeerDown     = 612
)

/* the code of task result that dfget will report to supernode */
const (
	ResultFail    = 500
	ResultSuc     = 501
	ResultInvalid = 502
	// ResultSemiSuc represents the result is partial successful.
	ResultSemiSuc = 503
)

/* the code of task status that dfget will report to supernode */
const (
	TaskStatusStart   = 700
	TaskStatusRunning = 701
	TaskStatusFinish  = 702
)

/* the client error when downloading from supernode that dfget will report to supernode */
const (
	ClientErrorFileNotExist    = "FILE_NOT_EXIST"
	ClientErrorFileMd5NotMatch = "FILE_MD5_NOT_MATCH"
)
