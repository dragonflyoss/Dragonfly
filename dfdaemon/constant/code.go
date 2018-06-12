// Copyright 1999-2017 Alibaba Group.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package constant

var (
	// CodeExitUserHomeNotExist represents that the user home directory is not exist.
	CodeExitUserHomeNotExist = 15
	// CodeExitPathNotAbs represents that the repo directory parsed from command-line is not absolute path.
	CodeExitPathNotAbs = 16
	// CodeExitRateLimitInvalid represents that the rate limit is invalid.
	CodeExitRateLimitInvalid = 17
	// CodeExitPortInvalid represents that the port is invalid.
	CodeExitPortInvalid = 18
	// CodeExitDfgetNotFound represents that the dfget cannot be found.
	CodeExitDfgetNotFound = 19
	// CodeExitRepoCreateFail represents that the repo directory is created failed.
	CodeExitRepoCreateFail = 20
	// CodeExitDfgetFail represents that executing dfget failed.
	CodeExitDfgetFail = 21

	// CodeReqAuth represents that an authentication failure happens when executing dfget.
	CodeReqAuth = 22
)
