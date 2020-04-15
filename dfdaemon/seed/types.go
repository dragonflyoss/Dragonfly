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

import (
	"time"
)

// BaseInfo describes the base info of seed.
type BaseInfo struct {
	// the url of seed file.
	URL string

	// the taskID of seed file.
	TaskID string

	// the header of seed file.
	Header map[string][]string

	// URL may contains some changeful query parameters such as authentication parameters. Dragonfly will
	// filter these parameter via 'filter'. The usage of it is that different URL may generate the same
	// download url.
	Filters []string

	// the full length of seed file.
	FullLength int64

	// Seed will download data from rss which is divided by blocks.
	// And block size is defined by BlockOrder. It should be limited [10, 31].
	BlockOrder uint32

	// expire time duration of seed file.
	ExpireTimeDur time.Duration
}

// PreFetchResult shows the result of prefetch.
type PreFetchResult struct {
	Success bool
	Err     error
	// if canceled, caller need not to do other.
	Canceled bool
}
