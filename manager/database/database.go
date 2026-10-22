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

package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	caches "github.com/go-gorm/caches/v4"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	"d7y.io/dragonfly/v2/manager/config"
	"d7y.io/dragonfly/v2/manager/models"
	"d7y.io/dragonfly/v2/manager/types"
	pkgredis "d7y.io/dragonfly/v2/pkg/redis"
	schedulerconfig "d7y.io/dragonfly/v2/scheduler/config"
)

const (
	// DefaultClusterName name for the cluster.
	DefaultClusterName = "cluster-1"
)

const (
	// DragonflyPATEnvName is the environment variable name for generating personal access token.
	DragonflyPATEnvName = "DRAGONFLY_PAT"
)

type Database struct {
	DB  *gorm.DB
	RDB redis.UniversalClient
}

func New(cfg *config.Config) (*Database, error) {
	var (
		db  *gorm.DB
		err error
	)
	switch cfg.Database.Type {
	case config.DatabaseTypeMysql, config.DatabaseTypeMariaDB:
		db, err = newMysql(cfg)
		if err != nil {
			logger.Errorf("mysql: %s", err.Error())
			return nil, err
		}
	case config.DatabaseTypePostgres:
		db, err = newPostgres(cfg)
		if err != nil {
			logger.Errorf("postgres: %s", err.Error())
			return nil, err
		}
	case config.DatabaseTypePolardb:
		db, err = newPolardb(cfg)
		if err != nil {
			logger.Errorf("polardb: %s", err.Error())
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid database type %s", cfg.Database.Type)
	}

	// Add cache for reducing database load.
	cacher, err := newCacher()
	if err != nil {
		logger.Errorf("new cacher: %s", err.Error())
		return nil, err
	}

	if err := db.Use(&caches.Caches{Conf: &caches.Config{
		Easer:  true,
		Cacher: cacher,
	}}); err != nil {
		logger.Errorf("use cache: %s", err.Error())
		return nil, err
	}

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

	return &Database{
		DB:  db,
		RDB: rdb,
	}, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Audit{},
		&models.Job{},
		&models.SeedPeerCluster{},
		&models.SeedPeer{},
		&models.SchedulerCluster{},
		&models.Scheduler{},
		&models.User{},
		&models.Oauth{},
		&models.Config{},
		&models.Application{},
		&models.PersonalAccessToken{},
		&models.Peer{},
	)
}

func seed(db *gorm.DB) error {
	// Create default scheduler cluster.
	var schedulerClusterCount int64
	if err := db.Model(models.SchedulerCluster{}).Count(&schedulerClusterCount).Error; err != nil {
		return err
	}

	if schedulerClusterCount <= 0 {
		schedulerCluster := models.SchedulerCluster{
			BaseModel: models.BaseModel{
				ID: uint(1),
			},
			Name: DefaultClusterName,
			Config: map[string]any{
				"candidate_parent_limit": schedulerconfig.DefaultSchedulerCandidateParentLimit,
				"filter_parent_limit":    schedulerconfig.DefaultSchedulerFilterParentLimit,
				"job_rate_limit":         config.DefaultClusterJobRateLimit,
			},
			ClientConfig: map[string]any{
				"load_limit": schedulerconfig.DefaultPeerConcurrentUploadLimit,
			},
			SeedClientConfig: map[string]any{},
			Scopes:           map[string]any{},
			IsDefault:        true,
		}

		if err := db.Create(&schedulerCluster).Error; err != nil {
			return err
		}

		if err := setPostgresSequence(db, "scheduler_cluster", schedulerCluster.ID); err != nil {
			return err
		}
	}

	// Create default seed peer cluster.
	var seedPeerClusterCount int64
	if err := db.Model(models.SeedPeerCluster{}).Count(&seedPeerClusterCount).Error; err != nil {
		return err
	}

	if seedPeerClusterCount <= 0 {
		seedPeerCluster := models.SeedPeerCluster{
			BaseModel: models.BaseModel{
				ID: uint(1),
			},
			Name: DefaultClusterName,
			Config: map[string]any{
				"load_limit": schedulerconfig.DefaultSeedPeerConcurrentUploadLimit,
			},
		}

		if err := db.Create(&seedPeerCluster).Error; err != nil {
			return err
		}

		if err := setPostgresSequence(db, "seed_peer_cluster", seedPeerCluster.ID); err != nil {
			return err
		}

		seedPeerCluster = models.SeedPeerCluster{}
		if err := db.First(&seedPeerCluster).Error; err != nil {
			return err
		}

		schedulerCluster := models.SchedulerCluster{}
		if err := db.First(&schedulerCluster).Error; err != nil {
			return err
		}

		if err := db.Model(&seedPeerCluster).Association("SchedulerClusters").Append(&schedulerCluster); err != nil {
			return err
		}
	}

	// TODO Compatible with old version.
	// Update scheduler features when features is NULL.
	var schedulers []models.Scheduler
	if err := db.Model(models.Scheduler{}).Find(&schedulers).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}

		return err
	}

	for _, scheduler := range schedulers {
		if scheduler.Features == nil {
			if err := db.Model(&scheduler).Update("features", models.Array(types.DefaultSchedulerFeatures)).Error; err != nil {
				logger.Errorf("update scheduler %d features: %s", scheduler.ID, err.Error())
				continue
			}

			logger.Infof("update scheduler %d default features", scheduler.ID)
		}
	}

	// Create default GC config.
	var config models.Config
	if err := db.Model(models.Config{}).First(&config, models.Config{Name: models.ConfigGC}).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		gcConfig := &models.GCConfig{
			Audit: &models.GCAuditConfig{
				TTL: models.DefaultGCAuditTTL,
			},
			Job: &models.GCJobConfig{
				TTL: models.DefaultGCJobTTL,
			},
		}
		gcConfigVal, err := json.Marshal(gcConfig)
		if err != nil {
			return err
		}

		if err := db.Model(models.Config{}).Create(&models.Config{
			Name:  models.ConfigGC,
			Value: string(gcConfigVal),
		}).Error; err != nil {
			return err
		}
	}

	// If DRAGONFLY_PAT is set, create a default personal access token for user ID 1(root user).
	if pat := os.Getenv(DragonflyPATEnvName); pat != "" {
		if err := db.Model(models.PersonalAccessToken{}).Create(&models.PersonalAccessToken{
			Name:      "default",
			Token:     pat,
			Scopes:    types.DefaultPersonalAccessTokenScopes,
			State:     models.PersonalAccessTokenStateActive,
			ExpiredAt: time.Now().AddDate(10, 0, 0),
			UserID:    1,
		}).Error; err != nil {
			return err
		}
	}

	return nil
}
