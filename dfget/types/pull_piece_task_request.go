/*
 * Copyright 1999-2018 Alibaba Group.
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

// PullPieceTaskRequest is send to supernodes when pulling pieces.
type PullPieceTaskRequest struct {
	SrcCid string `request:"srcCid"`
	DstCid string `request:"dstCid"`
	Range  string `request:"range"`
	Result string `request:"result"`
	Status string `request:"status"`
	TaskID string `request:"taskId"`
}
