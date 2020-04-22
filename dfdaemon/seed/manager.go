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

import "time"

// Manager is an interface which manages the seeds.
type Manager interface {
	// Register a seed.
	Register(key string, info BaseInfo) (Seed, error)

	// UnRegister seed by key.
	UnRegister(key string) error

	// RefreshExpireTime refreshes expire time of seed.
	RefreshExpireTime(key string, expireTimeDur time.Duration) error

	// NotifyExpired get the expired chan of seed, it will be notified if seed expired.
	NotifyExpired(key string) (<-chan struct{}, error)

	// Prefetch will add seed to the prefetch list, and then prefetch by the concurrent limit.
	Prefetch(key string, perDownloadSize int64) (<-chan struct{}, error)

	// GetPrefetchResult should be called after notify by prefetch chan.
	GetPrefetchResult(key string) (PreFetchResult, error)

	// SetPrefetchLimit limits the concurrency of prefetching seed.
	// Default is defaultDownloadConcurrency.
	SetConcurrentLimit(limit int) (validLimit int)

	// Get gets the seed by key.
	Get(key string) (Seed, error)

	// List lists the seeds.
	List() ([]Seed, error)

	// Stop stops the SeedManager.
	Stop()
}
