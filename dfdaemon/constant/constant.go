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

package constant

const (
	// CodeExitConfigError represents that the config provided can not be load successfully.
	CodeExitConfigError = 10 + iota
	// CodeExitUserHomeNotExist represents that the user home directory is not exist.
	CodeExitUserHomeNotExist
	// CodeExitPathNotAbs represents that the repo directory parsed from command-line is not absolute path.
	CodeExitPathNotAbs
	// CodeExitRateLimitInvalid represents that the rate limit is invalid.
	CodeExitRateLimitInvalid
	// CodeExitPortInvalid represents that the port is invalid.
	CodeExitPortInvalid
	// CodeExitDfgetNotFound represents that the dfget cannot be found.
	CodeExitDfgetNotFound
	// CodeExitRepoCreateFail represents that the repo directory is created failed.
	CodeExitRepoCreateFail
	// CodeExitDfgetFail represents that executing dfget failed.
	CodeExitDfgetFail
)

const (
	// CodeReqAuth represents that an authentication failure happens when executing dfget.
	CodeReqAuth = 100 + iota
)

const (
	// DefaultConfigPath is the default path of dfdaemon configuration file.
	DefaultConfigPath = "/etc/dragonfly/dfdaemon.yml"
)

const (
	// Namespace is the prefix of metrics namespace of dragonfly
	Namespace = "dragonfly"
	// Subsystem represents metrics for dfdaemon
	Subsystem = "dfdaemon"
)
