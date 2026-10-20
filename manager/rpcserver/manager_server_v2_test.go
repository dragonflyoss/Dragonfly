/*
 *     Copyright 2026 The Dragonfly Authors
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
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	cachev9 "github.com/go-redis/cache/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	commonv2 "d7y.io/api/v2/pkg/apis/common/v2"
	managerv2 "d7y.io/api/v2/pkg/apis/manager/v2"
	managerv2mocks "d7y.io/api/v2/pkg/apis/manager/v2/mocks"

	"d7y.io/dragonfly/v2/manager/models"
	"d7y.io/dragonfly/v2/manager/searcher"
	"d7y.io/dragonfly/v2/manager/searcher/mocks"
	"d7y.io/dragonfly/v2/manager/types"
	pkgredis "d7y.io/dragonfly/v2/pkg/redis"
)

func TestManagerServerV2_GetSeedPeer(t *testing.T) {
	tests := []struct {
		name   string
		req    *managerv2.GetSeedPeerRequest
		cached *managerv2.SeedPeer
		expect func(t *testing.T, s *managerServerV2, resp *managerv2.SeedPeer, err error)
	}{
		{
			name: "cache miss loads seed peer with cluster config and active schedulers",
			req:  &managerv2.GetSeedPeerRequest{Hostname: mockActiveSeedPeerHostname, Ip: mockActiveSeedPeerIP, SeedPeerClusterId: uint64(mockSeedPeerClusterID)},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.SeedPeer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var seedPeer models.SeedPeer
				assert.NoError(s.db.First(&seedPeer, models.SeedPeer{Hostname: mockActiveSeedPeerHostname}).Error)
				assert.Equal(uint64(seedPeer.ID), resp.GetId())
				assert.Equal(mockActiveSeedPeerHostname, resp.GetHostname())
				assert.Equal(mockActiveSeedPeerIP, resp.GetIp())
				assert.Equal(mockSeedPeerType, resp.GetType())
				assert.Equal(mockSeedPeerPort, resp.GetPort())
				assert.Equal(mockSeedPeerDownloadPort, resp.GetDownloadPort())
				assert.Equal(models.SeedPeerStateActive, resp.GetState())
				assert.Equal(uint64(mockSeedPeerClusterID), resp.GetSeedPeerClusterId())
				assert.Equal(uint64(mockSeedPeerClusterID), resp.GetSeedPeerCluster().GetId())
				assert.Equal(mockSeedPeerClusterName, resp.GetSeedPeerCluster().GetName())
				assert.JSONEq(mockSeedPeerClusterConfigJSON, string(resp.GetSeedPeerCluster().GetConfig()))
				assert.Len(resp.GetSchedulers(), 1)
				for _, scheduler := range resp.GetSchedulers() {
					assert.Equal(mockActiveSchedulerHostname, scheduler.GetHostname())
					assert.Equal(models.SchedulerStateActive, scheduler.GetState())
					assert.JSONEq(mockDefaultFeaturesJSON, string(scheduler.GetFeatures()))
				}

				assert.True(s.cache.Exists(context.Background(), pkgredis.MakeSeedPeerKeyInManager(mockSeedPeerClusterID, mockActiveSeedPeerHostname, mockActiveSeedPeerIP)))
			},
		},
		{
			name:   "cache hit returns cached seed peer without reading db",
			req:    &managerv2.GetSeedPeerRequest{Hostname: mockActiveSeedPeerHostname, Ip: mockActiveSeedPeerIP, SeedPeerClusterId: uint64(mockSeedPeerClusterID)},
			cached: &managerv2.SeedPeer{Hostname: mockCachedHostname},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.SeedPeer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(mockCachedHostname, resp.GetHostname())
			},
		},
		{
			name: "unknown seed peer returns internal",
			req:  &managerv2.GetSeedPeerRequest{Hostname: mockUnknownHostname, Ip: mockUnknownIP, SeedPeerClusterId: uint64(mockSeedPeerClusterID)},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.SeedPeer, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.Equal(codes.Internal, status.Code(err))
			},
		},
		{
			name: "seed peer in another cluster returns internal",
			req:  &managerv2.GetSeedPeerRequest{Hostname: mockActiveSeedPeerHostname, Ip: mockActiveSeedPeerIP, SeedPeerClusterId: uint64(mockSecondSeedPeerClusterID)},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.SeedPeer, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.Equal(codes.Internal, status.Code(err))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &managerServerV2{db: mockDB(t), cache: mockCache()}
			if tc.cached != nil {
				if err := s.cache.Set(&cachev9.Item{Key: pkgredis.MakeSeedPeerKeyInManager(uint(tc.req.SeedPeerClusterId), tc.req.Hostname, tc.req.Ip), Value: tc.cached}); err != nil {
					t.Fatal(err)
				}
			}

			resp, err := s.GetSeedPeer(context.Background(), tc.req)
			tc.expect(t, s, resp, err)
		})
	}
}

func TestManagerServerV2_ListSeedPeers(t *testing.T) {
	tests := []struct {
		name   string
		req    *managerv2.ListSeedPeersRequest
		cached *managerv2.ListSeedPeersResponse
		expect func(t *testing.T, s *managerServerV2, resp *managerv2.ListSeedPeersResponse, err error)
	}{
		{
			name: "returns only active seed peers of the caller cluster and caches them",
			req:  &managerv2.ListSeedPeersRequest{Hostname: mockActiveSeedPeerHostname, Ip: mockActiveSeedPeerIP, Version: mockVersion, Commit: mockCommit},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.ListSeedPeersResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.GetSeedPeers(), 1)
				for _, seedPeer := range resp.GetSeedPeers() {
					assert.Equal(mockActiveSeedPeerHostname, seedPeer.GetHostname())
					assert.Equal(models.SeedPeerStateActive, seedPeer.GetState())
					assert.Equal(uint64(mockSeedPeerClusterID), seedPeer.GetSeedPeerClusterId())
				}

				assert.True(s.cache.Exists(context.Background(), pkgredis.MakeSeedPeersKeyForPeerInManager(mockActiveSeedPeerHostname, mockActiveSeedPeerIP)))
			},
		},
		{
			name: "inactive caller still lists the active seed peers of its cluster",
			req:  &managerv2.ListSeedPeersRequest{Hostname: mockInactiveSeedPeerHostname, Ip: mockInactiveSeedPeerIP},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.ListSeedPeersResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.GetSeedPeers(), 1)
				for _, seedPeer := range resp.GetSeedPeers() {
					assert.Equal(mockActiveSeedPeerHostname, seedPeer.GetHostname())
					assert.Equal(models.SeedPeerStateActive, seedPeer.GetState())
					assert.Equal(uint64(mockSeedPeerClusterID), seedPeer.GetSeedPeerClusterId())
				}
			},
		},
		{
			name: "cluster without active seed peers returns not found and is not cached",
			req:  &managerv2.ListSeedPeersRequest{Hostname: mockLonelySeedPeerHostname, Ip: mockLonelySeedPeerIP},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.ListSeedPeersResponse, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.Equal(codes.NotFound, status.Code(err))
				assert.False(s.cache.Exists(context.Background(), pkgredis.MakeSeedPeersKeyForPeerInManager(mockLonelySeedPeerHostname, mockLonelySeedPeerIP)))
			},
		},
		{
			name: "unknown seed peer returns internal",
			req:  &managerv2.ListSeedPeersRequest{Hostname: mockUnknownHostname, Ip: mockUnknownIP},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.ListSeedPeersResponse, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.Equal(codes.Internal, status.Code(err))
			},
		},
		{
			name:   "cache hit returns cached response without reading db",
			req:    &managerv2.ListSeedPeersRequest{Hostname: mockActiveSeedPeerHostname, Ip: mockActiveSeedPeerIP},
			cached: &managerv2.ListSeedPeersResponse{SeedPeers: []*managerv2.SeedPeer{{Hostname: mockCachedHostname}}},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.ListSeedPeersResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.GetSeedPeers(), 1)
				assert.Equal(mockCachedHostname, resp.GetSeedPeers()[0].GetHostname())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &managerServerV2{db: mockDB(t), cache: mockCache()}
			if tc.cached != nil {
				if err := s.cache.Set(&cachev9.Item{Key: pkgredis.MakeSeedPeersKeyForPeerInManager(tc.req.Hostname, tc.req.Ip), Value: tc.cached}); err != nil {
					t.Fatal(err)
				}
			}

			resp, err := s.ListSeedPeers(context.Background(), tc.req)
			tc.expect(t, s, resp, err)
		})
	}
}

func TestManagerServerV2_UpdateSeedPeer(t *testing.T) {
	tests := []struct {
		name   string
		req    *managerv2.UpdateSeedPeerRequest
		expect func(t *testing.T, s *managerServerV2, resp *managerv2.SeedPeer, err error)
	}{
		{
			name: "creates active seed peer when no row matches",
			req: &managerv2.UpdateSeedPeerRequest{
				Hostname:          mockNewHostname,
				Ip:                mockNewIP,
				Type:              mockSeedPeerType,
				Idc:               &mockIDC,
				Location:          &mockLocation,
				Port:              mockSeedPeerPort,
				DownloadPort:      mockSeedPeerDownloadPort,
				SeedPeerClusterId: uint64(mockSeedPeerClusterID),
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.SeedPeer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var count int64
				assert.NoError(s.db.Unscoped().Model(&models.SeedPeer{}).Where(models.SeedPeer{Hostname: mockNewHostname}).Count(&count).Error)
				assert.Equal(int64(1), count)
				var seedPeer models.SeedPeer
				assert.NoError(s.db.First(&seedPeer, models.SeedPeer{Hostname: mockNewHostname}).Error)
				assert.Equal(uint64(seedPeer.ID), resp.GetId())
				assert.Equal(mockNewIP, seedPeer.IP)
				assert.Equal(mockSeedPeerType, seedPeer.Type)
				assert.Equal(mockIDC, seedPeer.IDC)
				assert.Equal(mockLocation, seedPeer.Location)
				assert.Equal(mockSeedPeerPort, seedPeer.Port)
				assert.Equal(mockSeedPeerDownloadPort, seedPeer.DownloadPort)
				assert.Equal(mockSeedPeerClusterID, seedPeer.SeedPeerClusterID)
				assert.Equal(models.SeedPeerStateActive, seedPeer.State)
				assert.Equal(models.SeedPeerStateActive, resp.GetState())
			},
		},
		{
			name: "updates matching seed peer instead of duplicating and invalidates cache",
			req: &managerv2.UpdateSeedPeerRequest{
				Hostname:          mockInactiveSeedPeerHostname,
				Ip:                mockInactiveSeedPeerIP,
				Type:              mockSeedPeerType,
				Idc:               &mockUpdatedIDC,
				Location:          &mockUpdatedLocation,
				Port:              mockSeedPeerPort,
				DownloadPort:      mockSeedPeerDownloadPort,
				SeedPeerClusterId: uint64(mockSeedPeerClusterID),
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.SeedPeer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var count int64
				assert.NoError(s.db.Unscoped().Model(&models.SeedPeer{}).Where(models.SeedPeer{Hostname: mockInactiveSeedPeerHostname}).Count(&count).Error)
				assert.Equal(int64(1), count)
				var seedPeer models.SeedPeer
				assert.NoError(s.db.First(&seedPeer, models.SeedPeer{Hostname: mockInactiveSeedPeerHostname}).Error)
				assert.Equal(uint64(seedPeer.ID), resp.GetId())
				assert.Equal(mockUpdatedIDC, seedPeer.IDC)
				assert.Equal(mockUpdatedLocation, seedPeer.Location)
				assert.Equal(models.SeedPeerStateActive, seedPeer.State)
				assert.Equal(mockUpdatedIDC, resp.GetIdc())
				assert.False(s.cache.Exists(context.Background(), pkgredis.MakeSeedPeerKeyInManager(mockSeedPeerClusterID, mockInactiveSeedPeerHostname, mockInactiveSeedPeerIP)))
			},
		},
		{
			name: "changed port updates the existing row instead of violating the unique index",
			req: &managerv2.UpdateSeedPeerRequest{
				Hostname:          mockActiveSeedPeerHostname,
				Ip:                mockActiveSeedPeerIP,
				Type:              mockSeedPeerType,
				Port:              mockUpdatedSeedPeerPort,
				DownloadPort:      mockSeedPeerDownloadPort,
				SeedPeerClusterId: uint64(mockSeedPeerClusterID),
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.SeedPeer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var count int64
				assert.NoError(s.db.Unscoped().Model(&models.SeedPeer{}).Where(models.SeedPeer{Hostname: mockActiveSeedPeerHostname}).Count(&count).Error)
				assert.Equal(int64(1), count)
				var seedPeer models.SeedPeer
				assert.NoError(s.db.First(&seedPeer, models.SeedPeer{Hostname: mockActiveSeedPeerHostname}).Error)
				assert.Equal(uint64(seedPeer.ID), resp.GetId())
				assert.Equal(mockUpdatedSeedPeerPort, seedPeer.Port)
				assert.Equal(mockUpdatedSeedPeerPort, resp.GetPort())
			},
		},
		{
			name: "unknown cluster returns internal and creates nothing",
			req: &managerv2.UpdateSeedPeerRequest{
				Hostname:          mockNewHostname,
				Ip:                mockNewIP,
				Port:              mockSeedPeerPort,
				DownloadPort:      mockSeedPeerDownloadPort,
				SeedPeerClusterId: uint64(mockUnknownClusterID),
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.SeedPeer, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.Equal(codes.Internal, status.Code(err))
				var count int64
				assert.NoError(s.db.Unscoped().Model(&models.SeedPeer{}).Where(models.SeedPeer{Hostname: mockNewHostname}).Count(&count).Error)
				assert.Equal(int64(0), count)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &managerServerV2{db: mockDB(t), cache: mockCache()}
			if err := s.cache.Set(&cachev9.Item{Key: pkgredis.MakeSeedPeerKeyInManager(uint(tc.req.SeedPeerClusterId), tc.req.Hostname, tc.req.Ip), Value: &managerv2.SeedPeer{Hostname: mockCachedHostname}}); err != nil {
				t.Fatal(err)
			}

			resp, err := s.UpdateSeedPeer(context.Background(), tc.req)
			tc.expect(t, s, resp, err)
		})
	}
}

func TestManagerServerV2_DeleteSeedPeer(t *testing.T) {
	tests := []struct {
		name   string
		req    *managerv2.DeleteSeedPeerRequest
		expect func(t *testing.T, s *managerServerV2, err error)
	}{
		{
			name: "deletes matching seed peer permanently",
			req:  &managerv2.DeleteSeedPeerRequest{Hostname: mockActiveSeedPeerHostname, Ip: mockActiveSeedPeerIP, SeedPeerClusterId: uint64(mockSeedPeerClusterID)},
			expect: func(t *testing.T, s *managerServerV2, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var count int64
				assert.NoError(s.db.Unscoped().Model(&models.SeedPeer{}).Where(models.SeedPeer{Hostname: mockActiveSeedPeerHostname}).Count(&count).Error)
				assert.Equal(int64(0), count)
				assert.NoError(s.db.Unscoped().Model(&models.SeedPeer{}).Count(&count).Error)
				assert.Equal(int64(2), count)
			},
		},
		{
			name: "seed peer in another cluster is untouched",
			req:  &managerv2.DeleteSeedPeerRequest{Hostname: mockActiveSeedPeerHostname, Ip: mockActiveSeedPeerIP, SeedPeerClusterId: uint64(mockSecondSeedPeerClusterID)},
			expect: func(t *testing.T, s *managerServerV2, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var count int64
				assert.NoError(s.db.Unscoped().Model(&models.SeedPeer{}).Count(&count).Error)
				assert.Equal(int64(3), count)
			},
		},
		{
			name: "unknown seed peer is a no-op",
			req:  &managerv2.DeleteSeedPeerRequest{Hostname: mockUnknownHostname, Ip: mockUnknownIP, SeedPeerClusterId: uint64(mockSeedPeerClusterID)},
			expect: func(t *testing.T, s *managerServerV2, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var count int64
				assert.NoError(s.db.Unscoped().Model(&models.SeedPeer{}).Count(&count).Error)
				assert.Equal(int64(3), count)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &managerServerV2{db: mockDB(t), cache: mockCache()}
			_, err := s.DeleteSeedPeer(context.Background(), tc.req)
			tc.expect(t, s, err)
		})
	}
}

func TestManagerServerV2_GetScheduler(t *testing.T) {
	tests := []struct {
		name   string
		req    *managerv2.GetSchedulerRequest
		cached *managerv2.Scheduler
		expect func(t *testing.T, s *managerServerV2, resp *managerv2.Scheduler, err error)
	}{
		{
			name: "cache miss loads scheduler with cluster configs and active seed peers",
			req:  &managerv2.GetSchedulerRequest{Hostname: mockActiveSchedulerHostname, Ip: mockActiveSchedulerIP, SchedulerClusterId: uint64(mockSchedulerClusterID)},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.Scheduler, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var scheduler models.Scheduler
				assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockActiveSchedulerHostname}).Error)
				assert.Equal(uint64(scheduler.ID), resp.GetId())
				assert.Equal(mockActiveSchedulerHostname, resp.GetHostname())
				assert.Equal(mockActiveSchedulerIP, resp.GetIp())
				assert.Equal(mockSchedulerPort, resp.GetPort())
				assert.Equal(mockIDC, resp.GetIdc())
				assert.Equal(mockLocation, resp.GetLocation())
				assert.Equal(models.SchedulerStateActive, resp.GetState())
				assert.JSONEq(mockDefaultFeaturesJSON, string(resp.GetFeatures()))
				assert.Equal(uint64(mockSchedulerClusterID), resp.GetSchedulerClusterId())
				assert.Equal(uint64(mockSchedulerClusterID), resp.GetSchedulerCluster().GetId())
				assert.Equal(mockSchedulerClusterName, resp.GetSchedulerCluster().GetName())
				assert.JSONEq(mockSchedulerClusterConfigJSON, string(resp.GetSchedulerCluster().GetConfig()))
				assert.JSONEq(mockSchedulerClusterClientConfigJSON, string(resp.GetSchedulerCluster().GetClientConfig()))
				assert.JSONEq(mockSchedulerClusterScopesJSON, string(resp.GetSchedulerCluster().GetScopes()))
				assert.Len(resp.GetSeedPeers(), 1)
				for _, seedPeer := range resp.GetSeedPeers() {
					assert.Equal(mockActiveSeedPeerHostname, seedPeer.GetHostname())
					assert.Equal(models.SeedPeerStateActive, seedPeer.GetState())
					assert.Equal(uint64(mockSeedPeerClusterID), seedPeer.GetSeedPeerCluster().GetId())
					assert.Equal(mockSeedPeerClusterName, seedPeer.GetSeedPeerCluster().GetName())
					assert.JSONEq(mockSeedPeerClusterConfigJSON, string(seedPeer.GetSeedPeerCluster().GetConfig()))
				}

				assert.True(s.cache.Exists(context.Background(), pkgredis.MakeSchedulerKeyInManager(mockSchedulerClusterID, mockActiveSchedulerHostname, mockActiveSchedulerIP)))
			},
		},
		{
			name:   "cache hit returns cached scheduler without reading db",
			req:    &managerv2.GetSchedulerRequest{Hostname: mockActiveSchedulerHostname, Ip: mockActiveSchedulerIP, SchedulerClusterId: uint64(mockSchedulerClusterID)},
			cached: &managerv2.Scheduler{Hostname: mockCachedHostname},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.Scheduler, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(mockCachedHostname, resp.GetHostname())
			},
		},
		{
			name: "unknown scheduler returns internal",
			req:  &managerv2.GetSchedulerRequest{Hostname: mockUnknownHostname, Ip: mockUnknownIP, SchedulerClusterId: uint64(mockSchedulerClusterID)},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.Scheduler, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.Equal(codes.Internal, status.Code(err))
			},
		},
		{
			name: "scheduler in another cluster returns internal",
			req:  &managerv2.GetSchedulerRequest{Hostname: mockActiveSchedulerHostname, Ip: mockActiveSchedulerIP, SchedulerClusterId: uint64(mockSecondSchedulerClusterID)},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.Scheduler, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.Equal(codes.Internal, status.Code(err))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &managerServerV2{db: mockDB(t), cache: mockCache()}
			if tc.cached != nil {
				if err := s.cache.Set(&cachev9.Item{Key: pkgredis.MakeSchedulerKeyInManager(uint(tc.req.SchedulerClusterId), tc.req.Hostname, tc.req.Ip), Value: tc.cached}); err != nil {
					t.Fatal(err)
				}
			}

			resp, err := s.GetScheduler(context.Background(), tc.req)
			tc.expect(t, s, resp, err)
		})
	}
}

func TestManagerServerV2_UpdateScheduler(t *testing.T) {
	tests := []struct {
		name   string
		req    *managerv2.UpdateSchedulerRequest
		expect func(t *testing.T, s *managerServerV2, resp *managerv2.Scheduler, err error)
	}{
		{
			name: "creates scheduler with default features when no row matches",
			req: &managerv2.UpdateSchedulerRequest{
				Hostname:           mockNewHostname,
				Ip:                 mockNewIP,
				Port:               mockSchedulerPort,
				Idc:                &mockIDC,
				Location:           &mockLocation,
				SchedulerClusterId: uint64(mockSchedulerClusterID),
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.Scheduler, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var count int64
				assert.NoError(s.db.Unscoped().Model(&models.Scheduler{}).Where(models.Scheduler{Hostname: mockNewHostname}).Count(&count).Error)
				assert.Equal(int64(1), count)
				var scheduler models.Scheduler
				assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockNewHostname}).Error)
				assert.Equal(uint64(scheduler.ID), resp.GetId())
				assert.Equal(mockNewIP, scheduler.IP)
				assert.Equal(mockSchedulerPort, scheduler.Port)
				assert.Equal(mockIDC, scheduler.IDC)
				assert.Equal(mockLocation, scheduler.Location)
				assert.Equal(mockSchedulerClusterID, scheduler.SchedulerClusterID)
				assert.Equal(models.Array(types.DefaultSchedulerFeatures), scheduler.Features)
				assert.Empty(scheduler.Config)
				assert.Equal(models.SchedulerStateInactive, scheduler.State)
				assert.WithinDuration(time.Now(), scheduler.LastKeepAliveAt, time.Minute)
				assert.JSONEq(mockDefaultFeaturesJSON, string(resp.GetFeatures()))
			},
		},
		{
			name: "creates scheduler with requested features and config",
			req: &managerv2.UpdateSchedulerRequest{
				Hostname:           mockNewHostname,
				Ip:                 mockNewIP,
				Port:               mockSchedulerPort,
				SchedulerClusterId: uint64(mockSchedulerClusterID),
				Features:           []string{types.SchedulerFeatureSchedule},
				Config:             []byte(`{"foo":"bar"}`),
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.Scheduler, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var scheduler models.Scheduler
				assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockNewHostname}).Error)
				assert.Equal(models.Array{types.SchedulerFeatureSchedule}, scheduler.Features)
				assert.Equal(models.JSONMap{"foo": "bar"}, scheduler.Config)
				assert.JSONEq(`["schedule"]`, string(resp.GetFeatures()))
			},
		},
		{
			name: "updates matching scheduler instead of duplicating and invalidates cache",
			req: &managerv2.UpdateSchedulerRequest{
				Hostname:           mockInactiveSchedulerHostname,
				Ip:                 mockInactiveSchedulerIP,
				Port:               mockSchedulerPort,
				Idc:                &mockUpdatedIDC,
				Location:           &mockUpdatedLocation,
				SchedulerClusterId: uint64(mockSchedulerClusterID),
				Features:           []string{types.SchedulerFeaturePreheat},
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.Scheduler, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var count int64
				assert.NoError(s.db.Unscoped().Model(&models.Scheduler{}).Where(models.Scheduler{Hostname: mockInactiveSchedulerHostname}).Count(&count).Error)
				assert.Equal(int64(1), count)
				var scheduler models.Scheduler
				assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockInactiveSchedulerHostname}).Error)
				assert.Equal(uint64(scheduler.ID), resp.GetId())
				assert.Equal(mockUpdatedIDC, scheduler.IDC)
				assert.Equal(mockUpdatedLocation, scheduler.Location)
				assert.Equal(models.Array{types.SchedulerFeaturePreheat}, scheduler.Features)
				assert.Equal(models.SchedulerStateInactive, scheduler.State)
				assert.WithinDuration(time.Now(), scheduler.LastKeepAliveAt, time.Minute)
				assert.JSONEq(`["preheat"]`, string(resp.GetFeatures()))
				assert.False(s.cache.Exists(context.Background(), pkgredis.MakeSchedulerKeyInManager(mockSchedulerClusterID, mockInactiveSchedulerHostname, mockInactiveSchedulerIP)))
			},
		},
		{
			name: "changed port updates the existing row instead of violating the unique index",
			req: &managerv2.UpdateSchedulerRequest{
				Hostname:           mockActiveSchedulerHostname,
				Ip:                 mockActiveSchedulerIP,
				Port:               mockUpdatedSchedulerPort,
				SchedulerClusterId: uint64(mockSchedulerClusterID),
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.Scheduler, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var count int64
				assert.NoError(s.db.Unscoped().Model(&models.Scheduler{}).Where(models.Scheduler{Hostname: mockActiveSchedulerHostname}).Count(&count).Error)
				assert.Equal(int64(1), count)
				var scheduler models.Scheduler
				assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockActiveSchedulerHostname}).Error)
				assert.Equal(uint64(scheduler.ID), resp.GetId())
				assert.Equal(mockUpdatedSchedulerPort, scheduler.Port)
				assert.Equal(mockUpdatedSchedulerPort, resp.GetPort())
			},
		},
		{
			name: "malformed config on update returns internal and leaves row unchanged",
			req: &managerv2.UpdateSchedulerRequest{
				Hostname:           mockInactiveSchedulerHostname,
				Ip:                 mockInactiveSchedulerIP,
				Port:               mockSchedulerPort,
				Idc:                &mockUpdatedIDC,
				SchedulerClusterId: uint64(mockSchedulerClusterID),
				Config:             []byte(`{`),
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.Scheduler, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.Equal(codes.Internal, status.Code(err))
				var scheduler models.Scheduler
				assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockInactiveSchedulerHostname}).Error)
				assert.Empty(scheduler.IDC)
			},
		},
		{
			name: "malformed config on create returns internal and creates nothing",
			req: &managerv2.UpdateSchedulerRequest{
				Hostname:           mockNewHostname,
				Ip:                 mockNewIP,
				Port:               mockSchedulerPort,
				SchedulerClusterId: uint64(mockSchedulerClusterID),
				Config:             []byte(`{`),
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.Scheduler, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.Equal(codes.Internal, status.Code(err))
				var count int64
				assert.NoError(s.db.Unscoped().Model(&models.Scheduler{}).Where(models.Scheduler{Hostname: mockNewHostname}).Count(&count).Error)
				assert.Equal(int64(0), count)
			},
		},
		{
			name: "unknown cluster returns internal and creates nothing",
			req: &managerv2.UpdateSchedulerRequest{
				Hostname:           mockNewHostname,
				Ip:                 mockNewIP,
				Port:               mockSchedulerPort,
				SchedulerClusterId: uint64(mockUnknownClusterID),
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.Scheduler, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.Equal(codes.Internal, status.Code(err))
				var count int64
				assert.NoError(s.db.Unscoped().Model(&models.Scheduler{}).Where(models.Scheduler{Hostname: mockNewHostname}).Count(&count).Error)
				assert.Equal(int64(0), count)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &managerServerV2{db: mockDB(t), cache: mockCache()}
			if err := s.cache.Set(&cachev9.Item{Key: pkgredis.MakeSchedulerKeyInManager(uint(tc.req.SchedulerClusterId), tc.req.Hostname, tc.req.Ip), Value: &managerv2.Scheduler{Hostname: mockCachedHostname}}); err != nil {
				t.Fatal(err)
			}

			resp, err := s.UpdateScheduler(context.Background(), tc.req)
			tc.expect(t, s, resp, err)
		})
	}
}

func TestManagerServerV2_ListSchedulers(t *testing.T) {
	req := &managerv2.ListSchedulersRequest{
		Hostname: mockPeerHostname,
		Ip:       mockPeerIP,
		Idc:      &mockIDC,
		Location: &mockLocation,
		Version:  mockVersion,
		Commit:   mockCommit,
	}
	cacheKey := pkgredis.MakeSchedulersKeyForPeerInManager(mockPeerHostname, mockPeerIP, mockVersion)
	tests := []struct {
		name   string
		cached *managerv2.ListSchedulersResponse
		mock   func(m *mocks.MockSearcherMockRecorder)
		expect func(t *testing.T, s *managerServerV2, resp *managerv2.ListSchedulersResponse, err error)
	}{
		{
			name: "returns active schedulers of the matched cluster with cluster configs and active seed peers",
			mock: func(m *mocks.MockSearcherMockRecorder) {
				m.FindSchedulerClusters(gomock.Any(), gomock.Any(), mockPeerIP, mockPeerHostname,
					map[string]string{searcher.ConditionIDC: mockIDC, searcher.ConditionLocation: mockLocation}, gomock.Any()).
					DoAndReturn(func(_ context.Context, clusters []models.SchedulerCluster, _, _ string, _ map[string]string, _ *zap.SugaredLogger) ([]models.SchedulerCluster, error) {
						for _, cluster := range clusters {
							if cluster.Name == mockSchedulerClusterName {
								return []models.SchedulerCluster{cluster}, nil
							}
						}

						return nil, errors.New("scheduler cluster not found")
					})
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.ListSchedulersResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.GetSchedulers(), 1)
				for _, scheduler := range resp.GetSchedulers() {
					assert.Equal(mockActiveSchedulerHostname, scheduler.GetHostname())
					assert.Equal(models.SchedulerStateActive, scheduler.GetState())
					assert.Equal(uint64(mockSchedulerClusterID), scheduler.GetSchedulerClusterId())
					assert.Equal(uint64(mockSchedulerClusterID), scheduler.GetSchedulerCluster().GetId())
					assert.Equal(mockSchedulerClusterName, scheduler.GetSchedulerCluster().GetName())
					assert.JSONEq(mockSchedulerClusterConfigJSON, string(scheduler.GetSchedulerCluster().GetConfig()))
					assert.JSONEq(mockSchedulerClusterClientConfigJSON, string(scheduler.GetSchedulerCluster().GetClientConfig()))
					assert.JSONEq(mockSchedulerClusterSeedClientConfigJSON, string(scheduler.GetSchedulerCluster().GetSeedClientConfig()))
					assert.JSONEq(mockSchedulerClusterScopesJSON, string(scheduler.GetSchedulerCluster().GetScopes()))
					assert.Len(scheduler.GetSeedPeers(), 1)
					for _, seedPeer := range scheduler.GetSeedPeers() {
						assert.Equal(mockActiveSeedPeerHostname, seedPeer.GetHostname())
						assert.JSONEq(mockSeedPeerClusterConfigJSON, string(seedPeer.GetSeedPeerCluster().GetConfig()))
					}
				}

				assert.True(s.cache.Exists(context.Background(), cacheKey))
			},
		},
		{
			name: "searcher only receives active schedulers with schedule feature",
			mock: func(m *mocks.MockSearcherMockRecorder) {
				m.FindSchedulerClusters(gomock.Any(), gomock.Cond(func(clusters []models.SchedulerCluster) bool {
					hostnames := map[string][]string{}
					for _, cluster := range clusters {
						for _, scheduler := range cluster.Schedulers {
							hostnames[cluster.Name] = append(hostnames[cluster.Name], scheduler.Hostname)
						}
					}

					return reflect.DeepEqual(map[string][]string{
						mockSchedulerClusterName:       {mockActiveSchedulerHostname},
						mockSecondSchedulerClusterName: {mockSecondClusterSchedulerHostname},
					}, hostnames)
				}), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, clusters []models.SchedulerCluster, _, _ string, _ map[string]string, _ *zap.SugaredLogger) ([]models.SchedulerCluster, error) {
						for _, cluster := range clusters {
							if cluster.Name == mockSecondSchedulerClusterName {
								return []models.SchedulerCluster{cluster}, nil
							}
						}

						return nil, errors.New("scheduler cluster not found")
					})
			},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.ListSchedulersResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.GetSchedulers(), 1)
				for _, scheduler := range resp.GetSchedulers() {
					assert.Equal(mockSecondClusterSchedulerHostname, scheduler.GetHostname())
					assert.Equal(uint64(mockSecondSchedulerClusterID), scheduler.GetSchedulerCluster().GetId())
					assert.Empty(scheduler.GetSeedPeers())
				}
			},
		},
		{
			name: "searcher failure falls back to active schedulers of every cluster",
			mock: func(m *mocks.MockSearcherMockRecorder) {
				m.FindSchedulerClusters(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("no matching scheduler cluster"))
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.ListSchedulersResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var hostnames []string
				for _, scheduler := range resp.GetSchedulers() {
					hostnames = append(hostnames, scheduler.GetHostname())
				}

				assert.Subset(hostnames, []string{mockActiveSchedulerHostname, mockSecondClusterSchedulerHostname})
				assert.NotContains(hostnames, mockInactiveSchedulerHostname)
				assert.True(s.cache.Exists(context.Background(), cacheKey))
			},
		},
		{
			name: "no candidate cluster returns empty response without caching",
			mock: func(m *mocks.MockSearcherMockRecorder) {
				m.FindSchedulerClusters(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]models.SchedulerCluster{}, nil)
			},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.ListSchedulersResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Empty(resp.GetSchedulers())
				assert.False(s.cache.Exists(context.Background(), cacheKey))
			},
		},
		{
			name:   "cache hit skips searcher and db",
			cached: &managerv2.ListSchedulersResponse{Schedulers: []*managerv2.Scheduler{{Hostname: mockCachedHostname}}},
			mock:   func(m *mocks.MockSearcherMockRecorder) {},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.ListSchedulersResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.GetSchedulers(), 1)
				assert.Equal(mockCachedHostname, resp.GetSchedulers()[0].GetHostname())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			ms := mocks.NewMockSearcher(ctl)
			tc.mock(ms.EXPECT())

			s := &managerServerV2{db: mockDB(t), cache: mockCache(), searcher: ms}
			if tc.cached != nil {
				if err := s.cache.Set(&cachev9.Item{Key: cacheKey, Value: tc.cached}); err != nil {
					t.Fatal(err)
				}
			}

			resp, err := s.ListSchedulers(context.Background(), req)
			tc.expect(t, s, resp, err)
		})
	}
}

func TestManagerServerV2_ListSchedulersByClusterID(t *testing.T) {
	tests := []struct {
		name   string
		req    *managerv2.ListSchedulersRequest
		cached *managerv2.ListSchedulersResponse
		expect func(t *testing.T, s *managerServerV2, resp *managerv2.ListSchedulersResponse, err error)
	}{
		{
			name: "returns schedulers of the cluster with cluster configs and caches them",
			req:  &managerv2.ListSchedulersRequest{Hostname: mockPeerHostname, Ip: mockPeerIP, SchedulerClusterId: uint64(mockSchedulerClusterID)},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.ListSchedulersResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				var hostnames []string
				for _, scheduler := range resp.GetSchedulers() {
					hostnames = append(hostnames, scheduler.GetHostname())
					assert.Equal(uint64(mockSchedulerClusterID), scheduler.GetSchedulerClusterId())
					assert.Equal(uint64(mockSchedulerClusterID), scheduler.GetSchedulerCluster().GetId())
					assert.Equal(mockSchedulerClusterName, scheduler.GetSchedulerCluster().GetName())
					assert.JSONEq(mockSchedulerClusterConfigJSON, string(scheduler.GetSchedulerCluster().GetConfig()))
					assert.JSONEq(mockSchedulerClusterClientConfigJSON, string(scheduler.GetSchedulerCluster().GetClientConfig()))
					assert.JSONEq(mockSchedulerClusterSeedClientConfigJSON, string(scheduler.GetSchedulerCluster().GetSeedClientConfig()))
					assert.JSONEq(mockSchedulerClusterScopesJSON, string(scheduler.GetSchedulerCluster().GetScopes()))
					assert.JSONEq(mockDefaultFeaturesJSON, string(scheduler.GetFeatures()))
				}

				assert.Contains(hostnames, mockActiveSchedulerHostname)
				assert.NotContains(hostnames, mockInactiveSchedulerHostname)
				assert.NotContains(hostnames, mockSecondClusterSchedulerHostname)
				assert.True(s.cache.Exists(context.Background(), pkgredis.MakeSchedulersByClusterIDKeyForPeerInManager(mockSchedulerClusterID)))
			},
		},
		{
			name: "cluster without schedulers returns empty response without caching",
			req:  &managerv2.ListSchedulersRequest{Hostname: mockPeerHostname, Ip: mockPeerIP, SchedulerClusterId: uint64(mockEmptySchedulerClusterID)},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.ListSchedulersResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Empty(resp.GetSchedulers())
				assert.False(s.cache.Exists(context.Background(), pkgredis.MakeSchedulersByClusterIDKeyForPeerInManager(mockEmptySchedulerClusterID)))
			},
		},
		{
			name: "unknown cluster returns record not found",
			req:  &managerv2.ListSchedulersRequest{Hostname: mockPeerHostname, Ip: mockPeerIP, SchedulerClusterId: uint64(mockUnknownClusterID)},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.ListSchedulersResponse, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, gorm.ErrRecordNotFound)
			},
		},
		{
			name:   "cache hit returns cached response without reading db",
			req:    &managerv2.ListSchedulersRequest{Hostname: mockPeerHostname, Ip: mockPeerIP, SchedulerClusterId: uint64(mockSchedulerClusterID)},
			cached: &managerv2.ListSchedulersResponse{Schedulers: []*managerv2.Scheduler{{Hostname: mockCachedHostname}}},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.ListSchedulersResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.GetSchedulers(), 1)
				assert.Equal(mockCachedHostname, resp.GetSchedulers()[0].GetHostname())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &managerServerV2{db: mockDB(t), cache: mockCache()}
			if tc.cached != nil {
				if err := s.cache.Set(&cachev9.Item{Key: pkgredis.MakeSchedulersByClusterIDKeyForPeerInManager(uint(tc.req.SchedulerClusterId)), Value: tc.cached}); err != nil {
					t.Fatal(err)
				}
			}

			resp, err := s.ListSchedulers(context.Background(), tc.req)
			tc.expect(t, s, resp, err)
		})
	}
}

func TestManagerServerV2_ListApplications(t *testing.T) {
	validApplication := models.Application{
		Name: "app-1",
		URL:  "https://example.com",
		BIO:  "first application",
		Priority: models.JSONMap{
			"value": 10,
			"urls":  []any{map[string]any{"regex": "^https://", "value": 20}},
		},
	}
	tests := []struct {
		name         string
		applications []models.Application
		cached       *managerv2.ListApplicationsResponse
		expect       func(t *testing.T, s *managerServerV2, resp *managerv2.ListApplicationsResponse, err error)
	}{
		{
			name: "no application returns not found and is not cached",
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.ListApplicationsResponse, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.Equal(codes.NotFound, status.Code(err))
				assert.False(s.cache.Exists(context.Background(), pkgredis.MakeApplicationsKeyInManager()))
			},
		},
		{
			name:         "returns applications with priority and url priorities and caches them",
			applications: []models.Application{validApplication},
			expect: func(t *testing.T, s *managerServerV2, resp *managerv2.ListApplicationsResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.GetApplications(), 1)
				for _, application := range resp.GetApplications() {
					assert.NotZero(application.GetId())
					assert.Equal(validApplication.Name, application.GetName())
					assert.Equal(validApplication.URL, application.GetUrl())
					assert.Equal(validApplication.BIO, application.GetBio())
					assert.Equal(commonv2.Priority(10), application.GetPriority().GetValue())
					assert.Len(application.GetPriority().GetUrls(), 1)
					for _, url := range application.GetPriority().GetUrls() {
						assert.Equal("^https://", url.GetRegex())
						assert.Equal(commonv2.Priority(20), url.GetValue())
					}
				}

				assert.True(s.cache.Exists(context.Background(), pkgredis.MakeApplicationsKeyInManager()))
			},
		},
		{
			name: "application with malformed priority is skipped",
			applications: []models.Application{
				validApplication,
				{Name: "app-2", URL: "https://example.org", Priority: models.JSONMap{"value": "high"}},
			},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.ListApplicationsResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.GetApplications(), 1)
				for _, application := range resp.GetApplications() {
					assert.Equal(validApplication.Name, application.GetName())
				}
			},
		},
		{
			name:   "cache hit returns cached response without reading db",
			cached: &managerv2.ListApplicationsResponse{Applications: []*managerv2.Application{{Name: mockCachedHostname}}},
			expect: func(t *testing.T, _ *managerServerV2, resp *managerv2.ListApplicationsResponse, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.GetApplications(), 1)
				for _, application := range resp.GetApplications() {
					assert.Equal(mockCachedHostname, application.GetName())
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &managerServerV2{db: mockDB(t), cache: mockCache()}
			for i := range tc.applications {
				application := tc.applications[i]
				if err := s.db.Create(&application).Error; err != nil {
					t.Fatal(err)
				}
			}

			if tc.cached != nil {
				if err := s.cache.Set(&cachev9.Item{Key: pkgredis.MakeApplicationsKeyInManager(), Value: tc.cached}); err != nil {
					t.Fatal(err)
				}
			}

			resp, err := s.ListApplications(context.Background(), &managerv2.ListApplicationsRequest{Hostname: mockPeerHostname, Ip: mockPeerIP})
			tc.expect(t, s, resp, err)
		})
	}
}

func TestManagerServerV2_KeepAlive(t *testing.T) {
	schedulerReq := &managerv2.KeepAliveRequest{
		SourceType: managerv2.SourceType_SCHEDULER_SOURCE,
		Hostname:   mockInactiveSchedulerHostname,
		Ip:         mockInactiveSchedulerIP,
		ClusterId:  uint64(mockSchedulerClusterID),
	}
	seedPeerReq := &managerv2.KeepAliveRequest{
		SourceType: managerv2.SourceType_SEED_PEER_SOURCE,
		Hostname:   mockInactiveSeedPeerHostname,
		Ip:         mockInactiveSeedPeerIP,
		ClusterId:  uint64(mockSeedPeerClusterID),
	}
	tests := []struct {
		name   string
		mock   func(t *testing.T, s *managerServerV2, ms *managerv2mocks.MockManager_KeepAliveServerMockRecorder, states *[]string)
		expect func(t *testing.T, s *managerServerV2, states []string, err error)
	}{
		{
			name: "scheduler turns active on first message and inactive when stream closes",
			mock: func(t *testing.T, s *managerServerV2, ms *managerv2mocks.MockManager_KeepAliveServerMockRecorder, states *[]string) {
				if err := s.cache.Set(&cachev9.Item{Key: pkgredis.MakeSchedulerKeyInManager(mockSchedulerClusterID, mockInactiveSchedulerHostname, mockInactiveSchedulerIP), Value: &managerv2.Scheduler{Hostname: mockCachedHostname}}); err != nil {
					t.Fatal(err)
				}

				if err := s.cache.Set(&cachev9.Item{Key: pkgredis.MakeSchedulersByClusterIDKeyForPeerInManager(mockSchedulerClusterID), Value: &managerv2.ListSchedulersResponse{}}); err != nil {
					t.Fatal(err)
				}

				ms.Context().Return(context.Background()).AnyTimes()
				gomock.InOrder(
					ms.Recv().DoAndReturn(func() (*managerv2.KeepAliveRequest, error) {
						assert := assert.New(t)
						var scheduler models.Scheduler
						assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockInactiveSchedulerHostname, IP: mockInactiveSchedulerIP, SchedulerClusterID: mockSchedulerClusterID}).Error)
						*states = append(*states, scheduler.State)
						return schedulerReq, nil
					}),
					ms.Recv().DoAndReturn(func() (*managerv2.KeepAliveRequest, error) {
						assert := assert.New(t)
						var scheduler models.Scheduler
						assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockInactiveSchedulerHostname, IP: mockInactiveSchedulerIP, SchedulerClusterID: mockSchedulerClusterID}).Error)
						*states = append(*states, scheduler.State)
						return schedulerReq, nil
					}),
					ms.Recv().DoAndReturn(func() (*managerv2.KeepAliveRequest, error) {
						assert := assert.New(t)
						var scheduler models.Scheduler
						assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockInactiveSchedulerHostname, IP: mockInactiveSchedulerIP, SchedulerClusterID: mockSchedulerClusterID}).Error)
						*states = append(*states, scheduler.State)
						return nil, io.EOF
					}),
				)
			},
			expect: func(t *testing.T, s *managerServerV2, states []string, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal([]string{models.SchedulerStateInactive, models.SchedulerStateActive, models.SchedulerStateActive}, states)
				var scheduler models.Scheduler
				assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockInactiveSchedulerHostname}).Error)
				assert.Equal(models.SchedulerStateInactive, scheduler.State)
				assert.WithinDuration(time.Now(), scheduler.LastKeepAliveAt, time.Minute)
				assert.False(s.cache.Exists(context.Background(), pkgredis.MakeSchedulerKeyInManager(mockSchedulerClusterID, mockInactiveSchedulerHostname, mockInactiveSchedulerIP)))
				assert.False(s.cache.Exists(context.Background(), pkgredis.MakeSchedulersByClusterIDKeyForPeerInManager(mockSchedulerClusterID)))
			},
		},
		{
			name: "seed peer turns active on first message and inactive when stream closes",
			mock: func(t *testing.T, s *managerServerV2, ms *managerv2mocks.MockManager_KeepAliveServerMockRecorder, states *[]string) {
				if err := s.cache.Set(&cachev9.Item{Key: pkgredis.MakeSeedPeerKeyInManager(mockSeedPeerClusterID, mockInactiveSeedPeerHostname, mockInactiveSeedPeerIP), Value: &managerv2.SeedPeer{Hostname: mockCachedHostname}}); err != nil {
					t.Fatal(err)
				}

				ms.Context().Return(context.Background()).AnyTimes()
				gomock.InOrder(
					ms.Recv().DoAndReturn(func() (*managerv2.KeepAliveRequest, error) {
						assert := assert.New(t)
						var seedPeer models.SeedPeer
						assert.NoError(s.db.First(&seedPeer, models.SeedPeer{Hostname: mockInactiveSeedPeerHostname, IP: mockInactiveSeedPeerIP, SeedPeerClusterID: mockSeedPeerClusterID}).Error)
						*states = append(*states, seedPeer.State)
						return seedPeerReq, nil
					}),
					ms.Recv().DoAndReturn(func() (*managerv2.KeepAliveRequest, error) {
						assert := assert.New(t)
						var seedPeer models.SeedPeer
						assert.NoError(s.db.First(&seedPeer, models.SeedPeer{Hostname: mockInactiveSeedPeerHostname, IP: mockInactiveSeedPeerIP, SeedPeerClusterID: mockSeedPeerClusterID}).Error)
						*states = append(*states, seedPeer.State)
						return nil, io.EOF
					}),
				)
			},
			expect: func(t *testing.T, s *managerServerV2, states []string, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal([]string{models.SeedPeerStateInactive, models.SeedPeerStateActive}, states)
				var seedPeer models.SeedPeer
				assert.NoError(s.db.First(&seedPeer, models.SeedPeer{Hostname: mockInactiveSeedPeerHostname}).Error)
				assert.Equal(models.SeedPeerStateInactive, seedPeer.State)
				assert.False(s.cache.Exists(context.Background(), pkgredis.MakeSeedPeerKeyInManager(mockSeedPeerClusterID, mockInactiveSeedPeerHostname, mockInactiveSeedPeerIP)))
			},
		},
		{
			name: "stream failure marks scheduler inactive and returns unknown",
			mock: func(t *testing.T, s *managerServerV2, ms *managerv2mocks.MockManager_KeepAliveServerMockRecorder, states *[]string) {
				ms.Context().Return(context.Background()).AnyTimes()
				gomock.InOrder(
					ms.Recv().DoAndReturn(func() (*managerv2.KeepAliveRequest, error) {
						assert := assert.New(t)
						var scheduler models.Scheduler
						assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockInactiveSchedulerHostname, IP: mockInactiveSchedulerIP, SchedulerClusterID: mockSchedulerClusterID}).Error)
						*states = append(*states, scheduler.State)
						return schedulerReq, nil
					}),
					ms.Recv().DoAndReturn(func() (*managerv2.KeepAliveRequest, error) {
						assert := assert.New(t)
						var scheduler models.Scheduler
						assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockInactiveSchedulerHostname, IP: mockInactiveSchedulerIP, SchedulerClusterID: mockSchedulerClusterID}).Error)
						*states = append(*states, scheduler.State)
						return nil, errors.New("connection reset")
					}),
				)
			},
			expect: func(t *testing.T, s *managerServerV2, states []string, err error) {
				assert := assert.New(t)
				assert.Equal(codes.Unknown, status.Code(err))
				assert.Equal([]string{models.SchedulerStateInactive, models.SchedulerStateActive}, states)
				var scheduler models.Scheduler
				assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockInactiveSchedulerHostname}).Error)
				assert.Equal(models.SchedulerStateInactive, scheduler.State)
			},
		},
		{
			name: "unknown scheduler returns internal",
			mock: func(_ *testing.T, _ *managerServerV2, ms *managerv2mocks.MockManager_KeepAliveServerMockRecorder, _ *[]string) {
				ms.Recv().Return(&managerv2.KeepAliveRequest{
					SourceType: managerv2.SourceType_SCHEDULER_SOURCE,
					Hostname:   mockUnknownHostname,
					Ip:         mockUnknownIP,
					ClusterId:  uint64(mockSchedulerClusterID),
				}, nil)
			},
			expect: func(t *testing.T, _ *managerServerV2, _ []string, err error) {
				assert := assert.New(t)
				assert.Equal(codes.Internal, status.Code(err))
			},
		},
		{
			name: "failing first receive returns internal without touching db",
			mock: func(_ *testing.T, _ *managerServerV2, ms *managerv2mocks.MockManager_KeepAliveServerMockRecorder, _ *[]string) {
				ms.Recv().Return(nil, errors.New("broken stream"))
			},
			expect: func(t *testing.T, s *managerServerV2, states []string, err error) {
				assert := assert.New(t)
				assert.Equal(codes.Internal, status.Code(err))
				assert.Empty(states)
				var scheduler models.Scheduler
				assert.NoError(s.db.First(&scheduler, models.Scheduler{Hostname: mockInactiveSchedulerHostname}).Error)
				assert.Equal(models.SchedulerStateInactive, scheduler.State)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			stream := managerv2mocks.NewMockManager_KeepAliveServer(ctl)
			s := &managerServerV2{db: mockDB(t), cache: mockCache()}
			var states []string
			tc.mock(t, s, stream.EXPECT(), &states)

			err := s.KeepAlive(stream)
			tc.expect(t, s, states, err)
		})
	}
}
