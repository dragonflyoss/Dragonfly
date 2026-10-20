/*
 *     Copyright 2023 The Dragonfly Authors
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

package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"time"

	cachev9 "github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"

	commonv2 "d7y.io/api/v2/pkg/apis/common/v2"
	managerv2 "d7y.io/api/v2/pkg/apis/manager/v2"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	"d7y.io/dragonfly/v2/manager/cache"
	"d7y.io/dragonfly/v2/manager/config"
	"d7y.io/dragonfly/v2/manager/database"
	"d7y.io/dragonfly/v2/manager/metrics"
	"d7y.io/dragonfly/v2/manager/models"
	"d7y.io/dragonfly/v2/manager/searcher"
	"d7y.io/dragonfly/v2/manager/types"
	pkgredis "d7y.io/dragonfly/v2/pkg/redis"
)

// managerServerV2 is v2 version of the manager grpc server.
type managerServerV2 struct {
	// Manager configuration.
	config *config.Config

	// GORM instance.
	db *gorm.DB

	// Redis universal client interface.
	rdb redis.UniversalClient

	// Cache instance.
	cache *cache.Cache

	// Searcher interface.
	searcher searcher.Searcher
}

// newManagerServerV2 returns v2 version of the manager server.
func newManagerServerV2(cfg *config.Config, database *database.Database, cache *cache.Cache, searcher searcher.Searcher) managerv2.ManagerServer {
	return &managerServerV2{
		config:   cfg,
		db:       database.DB,
		rdb:      database.RDB,
		cache:    cache,
		searcher: searcher,
	}
}

// Get SeedPeer and SeedPeer cluster configuration.
func (s *managerServerV2) GetSeedPeer(ctx context.Context, req *managerv2.GetSeedPeerRequest) (*managerv2.SeedPeer, error) {
	log := logger.WithHostnameAndIP(req.Hostname, req.Ip)
	cacheKey := pkgredis.MakeSeedPeerKeyInManager(uint(req.SeedPeerClusterId), req.Hostname, req.Ip)

	// Cache hit.
	var pbSeedPeer managerv2.SeedPeer
	if err := s.cache.Get(ctx, cacheKey, &pbSeedPeer); err != nil {
		log.Warnf("%s cache miss because of %s", cacheKey, err.Error())
	} else {
		log.Debugf("%s cache hit", cacheKey)
		return &pbSeedPeer, nil
	}

	// Cache miss and search seed peer.
	seedPeer := models.SeedPeer{}
	if err := s.db.WithContext(ctx).Preload("SeedPeerCluster").Preload("SeedPeerCluster.SchedulerClusters.Schedulers", &models.Scheduler{
		State: models.SchedulerStateActive,
	}).First(&seedPeer, &models.SeedPeer{
		Hostname:          req.Hostname,
		SeedPeerClusterID: uint(req.SeedPeerClusterId),
	}).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Marshal config of seed peer cluster.
	config, err := seedPeer.SeedPeerCluster.Config.MarshalJSON()
	if err != nil {
		return nil, status.Error(codes.DataLoss, err.Error())
	}

	// Construct schedulers.
	var pbSchedulers []*managerv2.Scheduler
	for _, schedulerCluster := range seedPeer.SeedPeerCluster.SchedulerClusters {
		for _, scheduler := range schedulerCluster.Schedulers {
			// Marshal features of scheduler.
			features, err := scheduler.Features.MarshalJSON()
			if err != nil {
				return nil, status.Error(codes.DataLoss, err.Error())
			}

			pbSchedulers = append(pbSchedulers, &managerv2.Scheduler{
				Id:       uint64(scheduler.ID),
				Hostname: scheduler.Hostname,
				Idc:      &scheduler.IDC,
				Location: &scheduler.Location,
				Ip:       scheduler.IP,
				Port:     scheduler.Port,
				State:    scheduler.State,
				Features: features,
			})
		}
	}

	// Construct seed peer.
	pbSeedPeer = managerv2.SeedPeer{
		Id:                uint64(seedPeer.ID),
		Type:              seedPeer.Type,
		Hostname:          seedPeer.Hostname,
		Idc:               &seedPeer.IDC,
		Location:          &seedPeer.Location,
		Ip:                seedPeer.IP,
		Port:              seedPeer.Port,
		DownloadPort:      seedPeer.DownloadPort,
		State:             seedPeer.State,
		SeedPeerClusterId: uint64(seedPeer.SeedPeerClusterID),
		SeedPeerCluster: &managerv2.SeedPeerCluster{
			Id:     uint64(seedPeer.SeedPeerCluster.ID),
			Name:   seedPeer.SeedPeerCluster.Name,
			Bio:    seedPeer.SeedPeerCluster.BIO,
			Config: config,
		},
		Schedulers: pbSchedulers,
	}

	// Cache data.
	if err := s.cache.Once(&cachev9.Item{
		Ctx:   ctx,
		Key:   cacheKey,
		Value: &pbSeedPeer,
		TTL:   s.cache.TTL,
	}); err != nil {
		log.Error(err)
	}

	return &pbSeedPeer, nil
}

// List active seed peers configuration.
func (s *managerServerV2) ListSeedPeers(ctx context.Context, req *managerv2.ListSeedPeersRequest) (*managerv2.ListSeedPeersResponse, error) {
	log := logger.WithHostnameAndIP(req.Hostname, req.Ip)
	log.Debugf("list seed peers, version %s, commit %s", req.Version, req.Commit)

	// Cache hit.
	var pbListSeedPeersResponse managerv2.ListSeedPeersResponse
	cacheKey := pkgredis.MakeSeedPeersKeyForPeerInManager(req.Hostname, req.Ip)

	if err := s.cache.Get(ctx, cacheKey, &pbListSeedPeersResponse); err != nil {
		log.Warnf("%s cache miss because of %s", cacheKey, err.Error())
	} else {
		log.Debugf("%s cache hit", cacheKey)
		return &pbListSeedPeersResponse, nil
	}

	// Cache miss and search seed peer.
	seedPeer := models.SeedPeer{}
	if err := s.db.WithContext(ctx).Preload("SeedPeerCluster").First(&seedPeer, models.SeedPeer{
		Hostname: req.Hostname,
		IP:       req.Ip,
	}).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var seedPeers []models.SeedPeer
	if err := s.db.WithContext(ctx).Find(&seedPeers, models.SeedPeer{
		State:             models.SeedPeerStateActive,
		SeedPeerClusterID: uint(seedPeer.SeedPeerClusterID),
	}).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(seedPeers) == 0 {
		return nil, status.Error(codes.NotFound, "seed peer not found")
	}

	// Construct seed peers.
	for _, seedPeer := range seedPeers {
		pbListSeedPeersResponse.SeedPeers = append(pbListSeedPeersResponse.SeedPeers, &managerv2.SeedPeer{
			Id:                uint64(seedPeer.ID),
			Hostname:          seedPeer.Hostname,
			Type:              seedPeer.Type,
			Idc:               &seedPeer.IDC,
			Location:          &seedPeer.Location,
			Ip:                seedPeer.IP,
			Port:              seedPeer.Port,
			DownloadPort:      seedPeer.DownloadPort,
			State:             seedPeer.State,
			SeedPeerClusterId: uint64(seedPeer.SeedPeerClusterID),
			SeedPeerCluster: &managerv2.SeedPeerCluster{
				Id:   uint64(seedPeer.SeedPeerCluster.ID),
				Name: seedPeer.SeedPeerCluster.Name,
				Bio:  seedPeer.SeedPeerCluster.BIO,
			},
		})
	}

	// Cache data.
	if err := s.cache.Once(&cachev9.Item{
		Ctx:   ctx,
		Key:   cacheKey,
		Value: &pbListSeedPeersResponse,
		TTL:   s.cache.TTL,
	}); err != nil {
		log.Error(err)
	}

	return &pbListSeedPeersResponse, nil
}

// Update SeedPeer configuration.
func (s *managerServerV2) UpdateSeedPeer(ctx context.Context, req *managerv2.UpdateSeedPeerRequest) (*managerv2.SeedPeer, error) {
	log := logger.WithHostnameAndIP(req.Hostname, req.Ip)
	seedPeer := models.SeedPeer{}
	if err := s.db.WithContext(ctx).First(&seedPeer, models.SeedPeer{
		Hostname:          req.Hostname,
		IP:                req.Ip,
		SeedPeerClusterID: uint(req.SeedPeerClusterId),
	}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.createSeedPeer(ctx, req)
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.db.WithContext(ctx).Model(&seedPeer).Updates(models.SeedPeer{
		Type:              req.GetType(),
		IDC:               req.GetIdc(),
		Location:          req.GetLocation(),
		IP:                req.GetIp(),
		Port:              req.GetPort(),
		DownloadPort:      req.GetDownloadPort(),
		State:             models.SeedPeerStateActive,
		SeedPeerClusterID: uint(req.GetSeedPeerClusterId()),
	}).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.cache.Delete(
		ctx,
		pkgredis.MakeSeedPeerKeyInManager(seedPeer.SeedPeerClusterID, seedPeer.Hostname, seedPeer.IP),
	); err != nil {
		log.Warn(err)
	}

	return &managerv2.SeedPeer{
		Id:                uint64(seedPeer.ID),
		Hostname:          seedPeer.Hostname,
		Type:              seedPeer.Type,
		Idc:               &seedPeer.IDC,
		Location:          &seedPeer.Location,
		Ip:                seedPeer.IP,
		Port:              seedPeer.Port,
		DownloadPort:      seedPeer.DownloadPort,
		State:             seedPeer.State,
		SeedPeerClusterId: uint64(seedPeer.SeedPeerClusterID),
	}, nil
}

// Create SeedPeer and associate cluster.
func (s *managerServerV2) createSeedPeer(ctx context.Context, req *managerv2.UpdateSeedPeerRequest) (*managerv2.SeedPeer, error) {
	seedPeer := models.SeedPeer{
		Hostname:          req.GetHostname(),
		Type:              req.GetType(),
		IDC:               req.GetIdc(),
		Location:          req.GetLocation(),
		IP:                req.GetIp(),
		Port:              req.GetPort(),
		DownloadPort:      req.GetDownloadPort(),
		State:             models.SeedPeerStateActive,
		SeedPeerClusterID: uint(req.GetSeedPeerClusterId()),
	}

	if err := s.db.WithContext(ctx).Create(&seedPeer).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &managerv2.SeedPeer{
		Id:                uint64(seedPeer.ID),
		Hostname:          seedPeer.Hostname,
		Type:              seedPeer.Type,
		Idc:               &seedPeer.IDC,
		Location:          &seedPeer.Location,
		Ip:                seedPeer.IP,
		Port:              seedPeer.Port,
		DownloadPort:      seedPeer.DownloadPort,
		SeedPeerClusterId: uint64(seedPeer.SeedPeerClusterID),
		State:             seedPeer.State,
	}, nil
}

// Delete SeedPeer configuration.
func (s *managerServerV2) DeleteSeedPeer(ctx context.Context, req *managerv2.DeleteSeedPeerRequest) (*emptypb.Empty, error) {
	if err := s.db.WithContext(ctx).Unscoped().Delete(&models.SeedPeer{}, models.SeedPeer{
		Hostname:          req.Hostname,
		IP:                req.Ip,
		SeedPeerClusterID: uint(req.SeedPeerClusterId),
	}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	return new(emptypb.Empty), nil
}

// Get Scheduler and Scheduler cluster configuration.
func (s *managerServerV2) GetScheduler(ctx context.Context, req *managerv2.GetSchedulerRequest) (*managerv2.Scheduler, error) {
	log := logger.WithHostnameAndIP(req.Hostname, req.Ip)
	cacheKey := pkgredis.MakeSchedulerKeyInManager(uint(req.SchedulerClusterId), req.Hostname, req.Ip)

	// Cache hit.
	var pbScheduler managerv2.Scheduler
	if err := s.cache.Get(ctx, cacheKey, &pbScheduler); err != nil {
		log.Warnf("%s cache miss because of %s", cacheKey, err.Error())
	} else {
		log.Debugf("%s cache hit", cacheKey)
		return &pbScheduler, nil
	}

	// Cache miss and search scheduler.
	scheduler := models.Scheduler{}
	if err := s.db.WithContext(ctx).Preload("SchedulerCluster").Preload("SchedulerCluster.SeedPeerClusters.SeedPeers", &models.SeedPeer{
		State: models.SeedPeerStateActive,
	}).First(&scheduler, &models.Scheduler{
		Hostname:           req.Hostname,
		SchedulerClusterID: uint(req.SchedulerClusterId),
	}).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Marshal config of scheduler.
	schedulerClusterConfig, err := scheduler.SchedulerCluster.Config.MarshalJSON()
	if err != nil {
		return nil, status.Error(codes.DataLoss, err.Error())
	}

	// Marshal config of client.
	schedulerClusterClientConfig, err := scheduler.SchedulerCluster.ClientConfig.MarshalJSON()
	if err != nil {
		return nil, status.Error(codes.DataLoss, err.Error())
	}

	// Marshal scopes of client.
	schedulerClusterScopes, err := scheduler.SchedulerCluster.Scopes.MarshalJSON()
	if err != nil {
		return nil, status.Error(codes.DataLoss, err.Error())
	}

	// Construct seed peers.
	var pbSeedPeers []*managerv2.SeedPeer
	for _, seedPeerCluster := range scheduler.SchedulerCluster.SeedPeerClusters {
		seedPeerClusterConfig, err := seedPeerCluster.Config.MarshalJSON()
		if err != nil {
			return nil, status.Error(codes.DataLoss, err.Error())
		}

		for _, seedPeer := range seedPeerCluster.SeedPeers {
			pbSeedPeers = append(pbSeedPeers, &managerv2.SeedPeer{
				Id:                uint64(seedPeer.ID),
				Hostname:          seedPeer.Hostname,
				Type:              seedPeer.Type,
				Idc:               &seedPeer.IDC,
				Location:          &seedPeer.Location,
				Ip:                seedPeer.IP,
				Port:              seedPeer.Port,
				DownloadPort:      seedPeer.DownloadPort,
				State:             seedPeer.State,
				SeedPeerClusterId: uint64(seedPeer.SeedPeerClusterID),
				SeedPeerCluster: &managerv2.SeedPeerCluster{
					Id:     uint64(seedPeerCluster.ID),
					Name:   seedPeerCluster.Name,
					Bio:    seedPeerCluster.BIO,
					Config: seedPeerClusterConfig,
				},
			})
		}
	}

	// Marshal features of scheduler.
	features, err := scheduler.Features.MarshalJSON()
	if err != nil {
		return nil, status.Error(codes.DataLoss, err.Error())
	}

	// Construct scheduler.
	pbScheduler = managerv2.Scheduler{
		Id:                 uint64(scheduler.ID),
		Hostname:           scheduler.Hostname,
		Idc:                &scheduler.IDC,
		Location:           &scheduler.Location,
		Ip:                 scheduler.IP,
		Port:               scheduler.Port,
		State:              scheduler.State,
		Features:           features,
		SchedulerClusterId: uint64(scheduler.SchedulerClusterID),
		SchedulerCluster: &managerv2.SchedulerCluster{
			Id:           uint64(scheduler.SchedulerCluster.ID),
			Name:         scheduler.SchedulerCluster.Name,
			Bio:          scheduler.SchedulerCluster.BIO,
			Config:       schedulerClusterConfig,
			ClientConfig: schedulerClusterClientConfig,
			Scopes:       schedulerClusterScopes,
		},
		SeedPeers: pbSeedPeers,
	}

	// Cache data.
	if err := s.cache.Once(&cachev9.Item{
		Ctx:   ctx,
		Key:   cacheKey,
		Value: &pbScheduler,
		TTL:   s.cache.TTL,
	}); err != nil {
		log.Error(err)
	}

	return &pbScheduler, nil
}

// Update scheduler configuration.
func (s *managerServerV2) UpdateScheduler(ctx context.Context, req *managerv2.UpdateSchedulerRequest) (*managerv2.Scheduler, error) {
	log := logger.WithHostnameAndIP(req.Hostname, req.Ip)
	scheduler := models.Scheduler{}
	if err := s.db.WithContext(ctx).First(&scheduler, models.Scheduler{
		Hostname:           req.Hostname,
		IP:                 req.Ip,
		SchedulerClusterID: uint(req.SchedulerClusterId),
	}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.createScheduler(ctx, req)
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	// Use default scheduler features if not set, compatible with the old version.
	schedulerFeatures := types.DefaultSchedulerFeatures
	if req.GetFeatures() != nil {
		schedulerFeatures = req.GetFeatures()
	}

	var schedulerConfig models.JSONMap
	if len(req.Config) > 0 {
		if err := json.Unmarshal(req.Config, &schedulerConfig); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	if err := s.db.WithContext(ctx).Model(&scheduler).Updates(models.Scheduler{
		IDC:                req.GetIdc(),
		Location:           req.GetLocation(),
		IP:                 req.GetIp(),
		Port:               req.GetPort(),
		SchedulerClusterID: uint(req.GetSchedulerClusterId()),
		Features:           schedulerFeatures,
		LastKeepAliveAt:    time.Now(),
		Config:             schedulerConfig,
	}).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.cache.Delete(
		ctx,
		pkgredis.MakeSchedulerKeyInManager(scheduler.SchedulerClusterID, scheduler.Hostname, scheduler.IP),
	); err != nil {
		log.Warn(err)
	}

	// Marshal features of scheduler.
	features, err := scheduler.Features.MarshalJSON()
	if err != nil {
		return nil, status.Error(codes.DataLoss, err.Error())
	}

	return &managerv2.Scheduler{
		Id:                 uint64(scheduler.ID),
		Hostname:           scheduler.Hostname,
		Idc:                &scheduler.IDC,
		Location:           &scheduler.Location,
		Ip:                 scheduler.IP,
		Port:               scheduler.Port,
		Features:           features,
		State:              scheduler.State,
		SchedulerClusterId: uint64(scheduler.SchedulerClusterID),
	}, nil
}

// Create scheduler and associate cluster.
func (s *managerServerV2) createScheduler(ctx context.Context, req *managerv2.UpdateSchedulerRequest) (*managerv2.Scheduler, error) {
	// Use default scheduler features if not set, compatible with the old version.
	schedulerFeatures := types.DefaultSchedulerFeatures
	if req.GetFeatures() != nil {
		schedulerFeatures = req.GetFeatures()
	}

	var schedulerConfig models.JSONMap
	if len(req.Config) > 0 {
		if err := json.Unmarshal(req.Config, &schedulerConfig); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	scheduler := models.Scheduler{
		Hostname:           req.GetHostname(),
		IDC:                req.GetIdc(),
		Location:           req.GetLocation(),
		IP:                 req.GetIp(),
		Port:               req.GetPort(),
		Features:           schedulerFeatures,
		SchedulerClusterID: uint(req.GetSchedulerClusterId()),
		LastKeepAliveAt:    time.Now(),
		Config:             schedulerConfig,
	}

	if err := s.db.WithContext(ctx).Create(&scheduler).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Marshal features of scheduler.
	features, err := scheduler.Features.MarshalJSON()
	if err != nil {
		return nil, status.Error(codes.DataLoss, err.Error())
	}

	return &managerv2.Scheduler{
		Id:                 uint64(scheduler.ID),
		Hostname:           scheduler.Hostname,
		Idc:                &scheduler.IDC,
		Location:           &scheduler.Location,
		Ip:                 scheduler.IP,
		Port:               scheduler.Port,
		State:              scheduler.State,
		Features:           features,
		SchedulerClusterId: uint64(scheduler.SchedulerClusterID),
	}, nil
}

// List active schedulers configuration.
func (s *managerServerV2) ListSchedulers(ctx context.Context, req *managerv2.ListSchedulersRequest) (*managerv2.ListSchedulersResponse, error) {
	if req.SchedulerClusterId != 0 {
		return s.listSchedulersByClusterID(ctx, req)
	}

	// Fallback to the legacy way to list schedulers by searcher.
	return s.listSchedulersBySearcher(ctx, req)
}

// listSchedulersBySearcher is the legacy way to list schedulers by searcher, which will use the searcher plugin to query the schedulers
// filter by hostname/ip/idc/location.
func (s *managerServerV2) listSchedulersBySearcher(ctx context.Context, req *managerv2.ListSchedulersRequest) (*managerv2.ListSchedulersResponse, error) {
	log := logger.WithHostnameAndIP(req.Hostname, req.Ip)
	log.Debugf("[legacy] list schedulers, version %s, commit %s", req.Version, req.Commit)
	metrics.SearchSchedulerClusterCount.WithLabelValues(req.Version, req.Commit).Inc()

	// Cache hit.
	var pbListSchedulersResponse managerv2.ListSchedulersResponse
	cacheKey := pkgredis.MakeSchedulersKeyForPeerInManager(req.Hostname, req.Ip, req.Version)

	if err := s.cache.Get(ctx, cacheKey, &pbListSchedulersResponse); err != nil {
		log.Warnf("%s cache miss because of %s", cacheKey, err.Error())
	} else {
		log.Debugf("%s cache hit", cacheKey)
		return &pbListSchedulersResponse, nil
	}

	// Cache miss and search scheduler cluster.
	var schedulerClusters []models.SchedulerCluster
	if err := s.db.WithContext(ctx).Preload("SeedPeerClusters.SeedPeers", "state = ?", "active").Preload("Schedulers", "state = ?", "active").Find(&schedulerClusters).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Remove schedulers which not have schedule feature. As OceanBase does not support JSON type,
	// it is not possible to use datatypes.JSONQuery for filtering.
	var tmpSchedulerClusters []models.SchedulerCluster
	for _, schedulerCluster := range schedulerClusters {
		var tmpSchedulers []models.Scheduler
		for _, scheduler := range schedulerCluster.Schedulers {
			if slices.Contains(scheduler.Features, types.SchedulerFeatureSchedule) {
				tmpSchedulers = append(tmpSchedulers, scheduler)
			}
		}

		if len(tmpSchedulers) != 0 {
			schedulerCluster.Schedulers = tmpSchedulers
			tmpSchedulerClusters = append(tmpSchedulerClusters, schedulerCluster)
		}
	}
	log.Debugf("list scheduler clusters %v, idc is %s, location is %s", getSchedulerClusterNames(tmpSchedulerClusters), req.GetIdc(), req.GetLocation())

	// Search optimal scheduler clusters.
	// If searcher can not found candidate scheduler cluster,
	// return all scheduler clusters.
	var (
		candidateSchedulerClusters []models.SchedulerCluster
		err                        error
	)
	candidateSchedulerClusters, err = s.searcher.FindSchedulerClusters(ctx, tmpSchedulerClusters, req.Ip, req.Hostname,
		map[string]string{searcher.ConditionIDC: req.GetIdc(), searcher.ConditionLocation: req.GetLocation()}, logger.CoreLogger)
	if err != nil {
		log.Error(err)
		metrics.SearchSchedulerClusterFailureCount.WithLabelValues(req.Version, req.Commit).Inc()
		candidateSchedulerClusters = schedulerClusters
	}
	log.Debugf("find matching scheduler cluster %v", getSchedulerClusterNames(candidateSchedulerClusters))

	schedulers := []models.Scheduler{}
	for _, candidateSchedulerCluster := range candidateSchedulerClusters {
		for _, scheduler := range candidateSchedulerCluster.Schedulers {
			scheduler.SchedulerCluster = candidateSchedulerCluster
			schedulers = append(schedulers, scheduler)
		}
	}

	// Construct schedulers.
	for _, scheduler := range schedulers {
		seedPeers := []*managerv2.SeedPeer{}
		for _, seedPeerCluster := range scheduler.SchedulerCluster.SeedPeerClusters {
			// Marshal config of seed peer cluster.
			seedPeerClusterConfig, err := seedPeerCluster.Config.MarshalJSON()
			if err != nil {
				return nil, status.Error(codes.DataLoss, err.Error())
			}

			for _, seedPeer := range seedPeerCluster.SeedPeers {
				seedPeers = append(seedPeers, &managerv2.SeedPeer{
					Id:                uint64(seedPeer.ID),
					Hostname:          seedPeer.Hostname,
					Type:              seedPeer.Type,
					Idc:               &seedPeer.IDC,
					Location:          &seedPeer.Location,
					Ip:                seedPeer.IP,
					Port:              seedPeer.Port,
					DownloadPort:      seedPeer.DownloadPort,
					State:             seedPeer.State,
					SeedPeerClusterId: uint64(seedPeer.SeedPeerClusterID),
					SeedPeerCluster: &managerv2.SeedPeerCluster{
						Id:     uint64(seedPeerCluster.ID),
						Name:   seedPeerCluster.Name,
						Bio:    seedPeerCluster.BIO,
						Config: seedPeerClusterConfig,
					},
				})
			}
		}

		// Marshal features of scheduler.
		features, err := scheduler.Features.MarshalJSON()
		if err != nil {
			return nil, status.Error(codes.DataLoss, err.Error())
		}

		// Marshal config of scheduler.
		config, err := scheduler.SchedulerCluster.Config.MarshalJSON()
		if err != nil {
			return nil, status.Error(codes.DataLoss, err.Error())
		}

		// Marshal client config of scheduler.
		clientConfig, err := scheduler.SchedulerCluster.ClientConfig.MarshalJSON()
		if err != nil {
			return nil, status.Error(codes.DataLoss, err.Error())
		}

		// Marshal seed client config of scheduler.
		seedClientConfig, err := scheduler.SchedulerCluster.SeedClientConfig.MarshalJSON()
		if err != nil {
			return nil, status.Error(codes.DataLoss, err.Error())
		}

		// Marshal scopes config of scheduler.
		scopes, err := scheduler.SchedulerCluster.Scopes.MarshalJSON()
		if err != nil {
			return nil, status.Error(codes.DataLoss, err.Error())
		}

		pbListSchedulersResponse.Schedulers = append(pbListSchedulersResponse.Schedulers, &managerv2.Scheduler{
			Id:                 uint64(scheduler.ID),
			Hostname:           scheduler.Hostname,
			Idc:                &scheduler.IDC,
			Location:           &scheduler.Location,
			Ip:                 scheduler.IP,
			Port:               scheduler.Port,
			State:              scheduler.State,
			Features:           features,
			SchedulerClusterId: uint64(scheduler.SchedulerClusterID),
			SchedulerCluster: &managerv2.SchedulerCluster{
				Id:               uint64(scheduler.SchedulerCluster.ID),
				Name:             scheduler.SchedulerCluster.Name,
				Bio:              scheduler.SchedulerCluster.BIO,
				Config:           config,
				ClientConfig:     clientConfig,
				Scopes:           scopes,
				SeedClientConfig: seedClientConfig,
			},
			SeedPeers: seedPeers,
		})
	}

	// If scheduler is not found, even no default scheduler is returned.
	// It means that the scheduler has not been started,
	// and the results are not cached, waiting for the scheduler to be ready.
	if len(pbListSchedulersResponse.Schedulers) == 0 {
		return &pbListSchedulersResponse, nil
	}

	// Cache data.
	if err := s.cache.Once(&cachev9.Item{
		Ctx:   ctx,
		Key:   cacheKey,
		Value: &pbListSchedulersResponse,
		TTL:   s.cache.TTL,
	}); err != nil {
		log.Error(err)
	}

	return &pbListSchedulersResponse, nil
}

// listSchedulersByClusterID is the direct way to list the schedulers by the scheduler cluster id.
func (s *managerServerV2) listSchedulersByClusterID(ctx context.Context, req *managerv2.ListSchedulersRequest) (*managerv2.ListSchedulersResponse, error) {
	log := logger.WithHostnameAndIP(req.Hostname, req.Ip)
	log.Debugf("list schedulers, version %s, commit %s", req.Version, req.Commit)
	metrics.SearchSchedulerClusterCount.WithLabelValues(req.Version, req.Commit).Inc()

	// Cache hit.
	var pbListSchedulersResponse managerv2.ListSchedulersResponse
	cacheKey := pkgredis.MakeSchedulersByClusterIDKeyForPeerInManager(uint(req.SchedulerClusterId))

	if err := s.cache.Get(ctx, cacheKey, &pbListSchedulersResponse); err != nil {
		log.Warnf("%s cache miss because of %s", cacheKey, err.Error())
	} else {
		log.Debugf("%s cache hit", cacheKey)
		return &pbListSchedulersResponse, nil
	}

	var schedulers []models.Scheduler
	if err := s.db.Find(&schedulers, "scheduler_cluster_id = ? AND state = ?", req.SchedulerClusterId, models.SchedulerStateActive).Error; err != nil {
		log.Errorf("failed to get schedulers by cluster id %d: %s", req.SchedulerClusterId, err.Error())
		return nil, err
	}

	var schedulerCluster models.SchedulerCluster
	if err := s.db.First(&schedulerCluster, "id = ?", req.SchedulerClusterId).Error; err != nil {
		log.Errorf("failed to get scheduler cluster by id %d: %s", req.SchedulerClusterId, err.Error())
		return nil, err
	}

	config, err := schedulerCluster.Config.MarshalJSON()
	if err != nil {
		log.Errorf("failed to marshal scheduler cluster config: %s", err.Error())
		return nil, err
	}

	clientConfig, err := schedulerCluster.ClientConfig.MarshalJSON()
	if err != nil {
		log.Errorf("failed to marshal scheduler cluster client config: %s", err.Error())
		return nil, err
	}

	scopes, err := schedulerCluster.Scopes.MarshalJSON()
	if err != nil {
		log.Errorf("failed to marshal scheduler cluster scopes: %s", err.Error())
		return nil, err
	}

	seedClientConfig, err := schedulerCluster.SeedClientConfig.MarshalJSON()
	if err != nil {
		log.Errorf("failed to marshal scheduler cluster seed client config: %s", err.Error())
		return nil, err
	}

	for _, scheduler := range schedulers {
		// Marshal features of scheduler.
		features, err := scheduler.Features.MarshalJSON()
		if err != nil {
			return nil, status.Error(codes.DataLoss, err.Error())
		}

		pbListSchedulersResponse.Schedulers = append(pbListSchedulersResponse.Schedulers, &managerv2.Scheduler{
			Id:                 uint64(scheduler.ID),
			Hostname:           scheduler.Hostname,
			Idc:                &scheduler.IDC,
			Location:           &scheduler.Location,
			Ip:                 scheduler.IP,
			Port:               scheduler.Port,
			State:              scheduler.State,
			Features:           features,
			SchedulerClusterId: uint64(scheduler.SchedulerClusterID),
			SchedulerCluster: &managerv2.SchedulerCluster{
				Id:               uint64(schedulerCluster.ID),
				Name:             schedulerCluster.Name,
				Bio:              schedulerCluster.BIO,
				Config:           config,
				ClientConfig:     clientConfig,
				Scopes:           scopes,
				SeedClientConfig: seedClientConfig,
			},
		})
	}

	// If scheduler is not found, even no default scheduler is returned.
	// It means that the scheduler has not been started,
	// and the results are not cached, waiting for the scheduler to be ready.
	if len(pbListSchedulersResponse.Schedulers) == 0 {
		return &pbListSchedulersResponse, nil
	}

	// Cache data.
	if err := s.cache.Once(&cachev9.Item{
		Ctx:   ctx,
		Key:   cacheKey,
		Value: &pbListSchedulersResponse,
		TTL:   s.cache.TTL,
	}); err != nil {
		log.Error(err)
	}

	return &pbListSchedulersResponse, nil
}

// List applications configuration.
func (s *managerServerV2) ListApplications(ctx context.Context, req *managerv2.ListApplicationsRequest) (*managerv2.ListApplicationsResponse, error) {
	log := logger.WithHostnameAndIP(req.Hostname, req.Ip)

	// Cache hit.
	var pbListApplicationsResponse managerv2.ListApplicationsResponse
	cacheKey := pkgredis.MakeApplicationsKeyInManager()
	if err := s.cache.Get(ctx, cacheKey, &pbListApplicationsResponse); err != nil {
		log.Warnf("%s cache miss because of %s", cacheKey, err.Error())
	} else {
		log.Debugf("%s cache hit", cacheKey)
		return &pbListApplicationsResponse, nil
	}

	// Cache miss and search applications.
	var applications []models.Application
	if err := s.db.WithContext(ctx).Find(&applications, "priority != ?", "").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(applications) == 0 {
		return nil, status.Error(codes.NotFound, "application not found")
	}

	for _, application := range applications {
		b, err := application.Priority.MarshalJSON()
		if err != nil {
			log.Warn(err)
			continue
		}

		var priority types.PriorityConfig
		if err := json.Unmarshal(b, &priority); err != nil {
			log.Warn(err)
			continue
		}

		var pbURLPriorities []*managerv2.URLPriority
		for _, url := range priority.URLs {
			pbURLPriorities = append(pbURLPriorities, &managerv2.URLPriority{
				Regex: url.Regex,
				Value: commonv2.Priority(url.Value),
			})
		}

		pbListApplicationsResponse.Applications = append(pbListApplicationsResponse.Applications, &managerv2.Application{
			Id:   uint64(application.ID),
			Name: application.Name,
			Url:  application.URL,
			Bio:  application.BIO,
			Priority: &managerv2.ApplicationPriority{
				Value: commonv2.Priority(*priority.Value),
				Urls:  pbURLPriorities,
			},
		})
	}

	// Cache data.
	if err := s.cache.Once(&cachev9.Item{
		Ctx:   ctx,
		Key:   cacheKey,
		Value: &pbListApplicationsResponse,
		TTL:   s.cache.TTL,
	}); err != nil {
		log.Error(err)
	}

	return &pbListApplicationsResponse, nil
}

// KeepAlive with manager.
func (s *managerServerV2) KeepAlive(stream managerv2.Manager_KeepAliveServer) error {
	req, err := stream.Recv()
	if err != nil {
		logger.Errorf("keepalive failed for the first time: %s", err.Error())
		return status.Error(codes.Internal, err.Error())
	}
	hostname := req.Hostname
	ip := req.Ip
	sourceType := req.SourceType
	clusterID := uint(req.ClusterId)

	log := logger.WithKeepAlive(hostname, ip, sourceType.Enum().String(), req.ClusterId)
	log.Info("keepalive for the first time")

	// Initialize active scheduler.
	if sourceType == managerv2.SourceType_SCHEDULER_SOURCE {
		scheduler := models.Scheduler{}
		if err := s.db.First(&scheduler, models.Scheduler{
			Hostname:           hostname,
			IP:                 ip,
			SchedulerClusterID: clusterID,
		}).Updates(models.Scheduler{
			State: models.SchedulerStateActive,
		}).Error; err != nil {
			return status.Error(codes.Internal, err.Error())
		}

		// Clean legacy scheduler cache.
		if err := s.cache.Delete(
			stream.Context(),
			pkgredis.MakeSchedulerKeyInManager(clusterID, hostname, ip),
		); err != nil {
			log.Warnf("refresh keepalive status failed: %s", err.Error())
		}

		// Clean scheduler cache.
		if err := s.cache.Delete(
			stream.Context(),
			pkgredis.MakeSchedulersByClusterIDKeyForPeerInManager(uint(clusterID)),
		); err != nil {
			log.Warnf("refresh keepalive status failed: %s", err.Error())
		}
	}

	// Initialize active seed peer.
	if sourceType == managerv2.SourceType_SEED_PEER_SOURCE {
		seedPeer := models.SeedPeer{}
		if err := s.db.First(&seedPeer, models.SeedPeer{
			Hostname:          hostname,
			IP:                ip,
			SeedPeerClusterID: clusterID,
		}).Updates(models.SeedPeer{
			State: models.SeedPeerStateActive,
		}).Error; err != nil {
			return status.Error(codes.Internal, err.Error())
		}

		if err := s.cache.Delete(
			stream.Context(),
			pkgredis.MakeSeedPeerKeyInManager(clusterID, hostname, ip),
		); err != nil {
			log.Warnf("refresh keepalive status failed: %s", err.Error())
		}
	}

	for {
		if _, err := stream.Recv(); err != nil {
			// The stream context is canceled once Recv fails, so detach cancellation
			// for the cleanup below while keeping values for tracing.
			ctx := context.WithoutCancel(stream.Context())

			// Inactive scheduler.
			if sourceType == managerv2.SourceType_SCHEDULER_SOURCE {
				scheduler := models.Scheduler{}
				if err := s.db.First(&scheduler, models.Scheduler{
					Hostname:           hostname,
					IP:                 ip,
					SchedulerClusterID: clusterID,
				}).Updates(models.Scheduler{
					State: models.SchedulerStateInactive,
				}).Error; err != nil {
					return status.Error(codes.Internal, err.Error())
				}

				// Clean legacy scheduler cache.
				if err := s.cache.Delete(
					ctx,
					pkgredis.MakeSchedulerKeyInManager(clusterID, hostname, ip),
				); err != nil {
					log.Warnf("refresh keepalive status failed: %s", err.Error())
				}

				// Clean scheduler cache.
				if err := s.cache.Delete(
					ctx,
					pkgredis.MakeSchedulersByClusterIDKeyForPeerInManager(uint(clusterID)),
				); err != nil {
					log.Warnf("refresh keepalive status failed: %s", err.Error())
				}
			}

			// Inactive seed peer.
			if sourceType == managerv2.SourceType_SEED_PEER_SOURCE {
				seedPeer := models.SeedPeer{}
				if err := s.db.First(&seedPeer, models.SeedPeer{
					Hostname:          hostname,
					IP:                ip,
					SeedPeerClusterID: clusterID,
				}).Updates(models.SeedPeer{
					State: models.SeedPeerStateInactive,
				}).Error; err != nil {
					return status.Error(codes.Internal, err.Error())
				}

				if err := s.cache.Delete(
					ctx,
					pkgredis.MakeSeedPeerKeyInManager(clusterID, hostname, ip),
				); err != nil {
					log.Warnf("refresh keepalive status failed: %s", err.Error())
				}
			}

			if err == io.EOF {
				log.Info("keepalive closed")
				return nil
			}

			log.Errorf("keepalive failed: %s", err.Error())
			return status.Error(codes.Unknown, err.Error())
		}

		// Keepalive successful, update last heartbeat time.
		if sourceType == managerv2.SourceType_SCHEDULER_SOURCE {
			scheduler := models.Scheduler{}
			if err := s.db.First(&scheduler, models.Scheduler{
				Hostname:           hostname,
				IP:                 ip,
				SchedulerClusterID: clusterID,
			}).Updates(models.Scheduler{
				LastKeepAliveAt: time.Now(),
			}).Error; err != nil {
				log.Errorf("update scheduler last heartbeat failed: %s", err.Error())
			}
		}
	}
}
