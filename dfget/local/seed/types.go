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

	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"
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

type BaseOpt struct {
	BaseDir string
	Info    BaseInfo

	downPreFunc func(sd Seed)
	Factory     DownloaderFactory
}

type RateOpt struct {
	DownloadRateLimiter *ratelimiter.RateLimiter
}

type NewSeedManagerOpt struct {
	StoreDir           string
	ConcurrentLimit    int
	TotalLimit         int
	DownloadBlockOrder uint32
	OpenMemoryCache    bool

	// if download rate < 0, means no rate limit; else default limit
	DownloadRate int64
	UploadRate   int64

	// water level which is used to expire the seed
	// if HighLevel is reached, start to prepare the expire
	HighLevel uint

	// expire will be stopped util water level is smaller than LowLevel.
	LowLevel uint

	//DownloaderFactory create instance of Downloader.
	Factory DownloaderFactory
}
