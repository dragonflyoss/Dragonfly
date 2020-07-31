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

// Module cache implements the cache for stream mode.
// dfget will initialize the cache window during register phase;
// client stream writer in downloader will pass the successfully downloaded pieces to cache;
// uploader will visit the cache to fetch the pieces to share.
package uploader

import (
	"github.com/dragonflyoss/Dragonfly/dfget/config"
)

type cacheManager interface {
	// Register registers the cache for the task. The error would be returned if the cache has already existed.
	register(cfg *config.Config, taskID string, windowSize int) error

	// Store stores the piece to the cache of the given taskID. The cache will be updated.
	store(taskID string, content []byte) error

	// Load loads the piece content from the cache to uploader to share.
	load(taskID string, up *uploadParam) ([]byte, int64, error)
}

type lruCacheManager struct {
}

var _ cacheManager = &lruCacheManager{}

func newCacheManager() cacheManager {
	return &lruCacheManager{}
}

func (lcm *lruCacheManager) register(cfg *config.Config, taskID string, windowSize int) error {
	return nil
}

func (lcm *lruCacheManager) store(taskID string, content []byte) error {
	return nil
}

func (lcm *lruCacheManager) load(taskID string, up *uploadParam) ([]byte, int64, error) {
	return nil, 0, nil
}
