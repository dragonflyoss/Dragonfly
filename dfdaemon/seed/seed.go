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

package seed

import "io"

// Seed describes the seed file which represents the resource file defined by taskUrl.
type Seed interface {
	// Prefetch will start to download seed file to local cache.
	Prefetch(perDownloadSize int64) (<-chan struct{}, error)

	// GetPrefetchResult should be called after notify by prefetch chan.
	GetPrefetchResult() (PreFetchResult, error)

	// Delete will delete the local cache and release the resource.
	Delete() error

	// Download providers the range download, if local cache of seed do not include the range,
	// it will download the range data from rss and reply to request.
	Download(off int64, size int64) (io.ReadCloser, error)

	// stop the internal loop and release execution resource.
	Stop()

	// GetFullSize gets the full size of seed file.
	GetFullSize() int64

	// GetStatus gets the status of seed file.
	GetStatus() string

	// GetURL gets the url of seed file.
	GetURL() string

	// GetHeaders gets the headers of seed file.
	GetHeaders() map[string][]string

	// GetHeaders gets the taskID of seed file.
	GetTaskID() string
}
