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

package config

import (
	"path"
)

var (
	// HTTPSubPath is the name of the parent directory where the upload files are stored.
	HTTPSubPath = "qtdown"

	// DownloadSubPath is the name of the parent directory where the downloaded files are stored.
	DownloadSubPath = "download"

	// DownloadHome is the parent directory where the downloaded files are stored
	// which is a relative path.
	DownloadHome = path.Join("repo", DownloadSubPath)

	// UploadHome is the parent directory where the upload files are stored
	// which is a relative path.
	UploadHome = path.Join("repo", HTTPSubPath)
)
