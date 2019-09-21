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

// BaseResponse defines the common fields of responses from supernode.
// Types of supernode's responses could be defines as following:
// 		type XXResponse struct {
// 			*BaseResponse
//			Data *CustomizedDataStruct
// 		}
type BaseResponse struct {
	// Code represents whether the response is successful.
	Code int `json:"code"`
	// Msg describes the detailed error message if the response is failed.
	Msg string `json:"msg,omitempty"`
}

// NewBaseResponse creates a BaseResponse instance.
func NewBaseResponse(code int, msg string) *BaseResponse {
	res := new(BaseResponse)
	res.Code = code
	res.Msg = msg
	return res
}

// IsSuccess is used for determining whether the response is successful.
func (res *BaseResponse) IsSuccess() bool {
	return res.Code == 1
}
