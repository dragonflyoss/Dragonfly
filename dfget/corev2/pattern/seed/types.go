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
	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/pkg/rate"
)

type Config struct {
	config.DFGetCommonConfig

	MetaDir string

	// seed pattern config
	// high level of water level which to control the weed out seed file
	HighLevel int

	// low level of water level which to control the weed out seed file
	LowLevel int

	// DefaultBlockOrder represents the default block order of seed file. it should be in range [10-31].
	DefaultBlockOrder int

	PerDownloadBlocks int

	DownRate   rate.Rate
	UploadRate rate.Rate

	TotalLimit      int
	ConcurrentLimit int

	DisableOpenMemoryCache bool
}
