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
	"fmt"

	"github.com/dragonflyoss/Dragonfly/pkg/constants"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
)

// ResultInfo identify a struct that will be returned to the client.
type ResultInfo struct {
	code int
	msg  string
	data interface{}
}

// NewResultInfoWithError returns a new ResultInfo with error only.
// And it will fill the result code according to the type of error.
func NewResultInfoWithError(err error) ResultInfo {
	if errortypes.IsEmptyValue(err) ||
		errortypes.IsInvalidValue(err) {
		return NewResultInfoWithCodeError(constants.CodeParamError, err)
	}

	if errortypes.IsDataNotFound(err) {
		return NewResultInfoWithCodeError(constants.CodeTargetNotFound, err)
	}

	if errortypes.IsPeerWait(err) {
		return NewResultInfoWithCodeError(constants.CodePeerWait, err)
	}

	if errortypes.IsPeerContinue(err) {
		return NewResultInfoWithCodeError(constants.CodePeerContinue, err)
	}

	if errortypes.IsURLNotReachable(err) {
		return NewResultInfoWithCodeError(constants.CodeURLNotReachable, err)
	}

	// IsConvertFailed
	return NewResultInfoWithCodeError(constants.CodeSystemError, err)
}

// NewResultInfoWithCodeError returns a new ResultInfo with code and error.
// And it will get the err.Error() as the value of ResultInfo.msg.
func NewResultInfoWithCodeError(code int, err error) ResultInfo {
	msg := err.Error()
	return NewResultInfoWithCodeMsg(code, msg)
}

// NewResultInfoWithCode returns a new ResultInfo with code
// and it will get the default msg corresponding to the code as the value of ResultInfo.msg.
func NewResultInfoWithCode(code int) ResultInfo {
	msg := constants.GetMsgByCode(code)
	return NewResultInfoWithCodeMsg(code, msg)
}

// NewResultInfoWithCodeMsg returns a new ResultInfo with code and specified msg.
func NewResultInfoWithCodeMsg(code int, msg string) ResultInfo {
	return NewResultInfo(code, msg, nil)
}

// NewResultInfoWithCodeData returns a new ResultInfo with code and specified data.
func NewResultInfoWithCodeData(code int, data interface{}) ResultInfo {
	return NewResultInfo(code, "", data)
}

// NewResultInfo returns a new ResultInfo.
func NewResultInfo(code int, msg string, data interface{}) ResultInfo {
	return ResultInfo{
		code: code,
		msg:  msg,
		data: data,
	}
}

func (r ResultInfo) Error() string {
	return fmt.Sprintf("{\"Code\":%d,\"Msg\":\"%s\"}", r.code, r.msg)
}

// SuccessCode returns whether the code equals SuccessCode.
func (r ResultInfo) SuccessCode() bool {
	return r.code == constants.Success
}
