/*
 *     Copyright 2020 The Dragonfly Authors
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

package cache

import (
	"time"

	"github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"

	"d7y.io/dragonfly/v2/manager/config"
	pkgredis "d7y.io/dragonfly/v2/pkg/redis"
)

// Cache is cache client.
type Cache struct {
	*cache.Cache
	TTL time.Duration
}

// New cache instance.
func New(cfg *config.Config) (*Cache, error) {
	var localCache *cache.TinyLFU
	localCache = cache.NewTinyLFU(cfg.Cache.Local.Size, cfg.Cache.Local.TTL)

	redisOpts := &redis.UniversalOptions{
		Addrs:            cfg.Database.Redis.Addrs,
		MasterName:       cfg.Database.Redis.MasterName,
		DB:               cfg.Database.Redis.DB,
		Username:         cfg.Database.Redis.Username,
		Password:         cfg.Database.Redis.Password,
		SentinelUsername: cfg.Database.Redis.SentinelUsername,
		SentinelPassword: cfg.Database.Redis.SentinelPassword,
		PoolSize:         cfg.Database.Redis.PoolSize,
		PoolTimeout:      cfg.Database.Redis.PoolTimeout,
	}

	if redisTLS := cfg.Database.Redis.TLS; redisTLS != nil {
		tlsCfg, err := pkgredis.NewTLSClientConfig(redisTLS.CACert, redisTLS.Cert, redisTLS.Key, redisTLS.InsecureSkipVerify)
		if err != nil {
			return nil, err
		}

		redisOpts.TLSConfig = tlsCfg
	}

	rdb, err := pkgredis.NewRedis(redisOpts)
	if err != nil {
		return nil, err
	}

	// If the attribute TTL of cache.Item(cache's instance) is 0, redis expiration time is 1 hour.
	// cfg.TTL Set the expiration time of TinyLFU.
	return &Cache{
		Cache: cache.New(&cache.Options{
			Redis:      rdb,
			LocalCache: localCache,
		}),
		TTL: cfg.Cache.Redis.TTL,
	}, nil
}
