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

//go:generate mockgen -destination mocks/sync_peers_mock.go -source sync_peers.go -package mocks

package job

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	machineryv1tasks "github.com/dragonflyoss/machinery/v1/tasks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	internaljob "d7y.io/dragonfly/v2/internal/job"
	"d7y.io/dragonfly/v2/manager/config"
	"d7y.io/dragonfly/v2/manager/models"
	"d7y.io/dragonfly/v2/pkg/idgen"
	pkgtypes "d7y.io/dragonfly/v2/pkg/types"
	resource "d7y.io/dragonfly/v2/scheduler/resource/standard"
)

// SyncPeers is an interface for sync peers. It is only supported in Rust client,
// refer to https://github.com/dragonflyoss/client.
type SyncPeers interface {
	// CreateSyncPeers creates sync peers job, and merge the sync peer results with the data
	// in the peer table in the database. It is a synchronous operation, and it will returns
	// an error if the sync peers job is failed.
	CreateSyncPeers(context.Context, []models.Scheduler) error

	// Serve started sync peers server.
	Serve()

	// Stop sync peers server.
	Stop()
}

// syncPeers is an implementation of SyncPeers.
type syncPeers struct {
	config *config.Config
	job    *internaljob.Job
	db     *gorm.DB
	mu     *sync.Mutex
	done   chan struct{}
}

// newSyncPeers returns a new SyncPeers.
func newSyncPeers(cfg *config.Config, job *internaljob.Job, gdb *gorm.DB) (SyncPeers, error) {
	return &syncPeers{
		config: cfg,
		job:    job,
		db:     gdb,
		mu:     &sync.Mutex{},
		done:   make(chan struct{}),
	}, nil
}

// CreateSyncPeers creates sync peers job.
func (s *syncPeers) CreateSyncPeers(ctx context.Context, schedulers []models.Scheduler) error {
	// Avoid running multiple sync peers jobs at the same time.
	if !s.mu.TryLock() {
		return errors.New("sync peers job is running")
	}
	defer s.mu.Unlock()

	// Send sync peer requests to all available schedulers, and merge the sync peer results
	// with the data in the peer table in the database.
	for _, scheduler := range schedulers {
		log := logger.WithScheduler(scheduler.Hostname, scheduler.IP, uint64(scheduler.SchedulerClusterID))

		// Send sync peer request to scheduler.
		results, err := s.createSyncPeers(ctx, scheduler)
		if err != nil {
			log.Error(err)
			continue
		}
		log.Infof("[sync-peers] sync peers count is %d", len(results))

		// Merge sync peer results with the data in the peer table.
		s.mergePeers(ctx, scheduler, results, log)
	}

	return nil
}

// Serve started sync peers server.
func (s *syncPeers) Serve() {
	tick := time.NewTicker(s.config.Job.SyncPeers.Interval)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			ctx, cancel := context.WithTimeout(context.Background(), s.config.Job.SyncPeers.Timeout)
			defer cancel()

			// Find all of the scheduler clusters that has active schedulers.
			var schedulerClusters []models.SchedulerCluster
			if err := s.db.WithContext(ctx).Find(&schedulerClusters).Error; err != nil {
				logger.Errorf("[sync-peers] sync peers find scheduler clusters failed: %v", err)
			}

			// Find all of the schedulers that has active scheduler cluster.
			var schedulers []models.Scheduler
			for _, schedulerCluster := range schedulerClusters {
				var scheduler models.Scheduler
				if err := s.db.WithContext(ctx).Preload("SchedulerCluster").First(&scheduler, models.Scheduler{
					SchedulerClusterID: schedulerCluster.ID,
					State:              models.SchedulerStateActive,
				}).Error; err != nil {
					continue
				}

				logger.Infof("[sync-peers] sync peers find scheduler cluster %s", schedulerCluster.Name)
				schedulers = append(schedulers, scheduler)
			}
			logger.Infof("[sync-peers] sync peers find schedulers count is %d", len(schedulers))

			if err := s.CreateSyncPeers(ctx, schedulers); err != nil {
				logger.Errorf("[sync-peers] sync peers failed: %v", err)
			}
		case <-s.done:
			return
		}
	}
}

// Stop sync peers server.
func (s *syncPeers) Stop() {
	close(s.done)
}

// createSyncPeers creates sync peers.
func (s *syncPeers) createSyncPeers(ctx context.Context, scheduler models.Scheduler) ([]*resource.Host, error) {
	var span trace.Span
	ctx, span = tracer.Start(ctx, config.SpanSyncPeers, trace.WithSpanKind(trace.SpanKindProducer))
	defer span.End()

	// Initialize queue.
	queue, err := getSchedulerQueue(scheduler)
	if err != nil {
		return nil, err
	}

	var tasks []*machineryv1tasks.Signature
	for i := 0; i < s.config.Job.SyncPeers.SyncBatchSize; i++ {
		// Initialize task signature.
		task := &machineryv1tasks.Signature{
			UUID:       fmt.Sprintf("task_%s", uuid.New().String()),
			Name:       internaljob.SyncPeersJob,
			RoutingKey: queue.String(),
		}
		tasks = append(tasks, task)
	}
	if len(tasks) <= 0 {
		return []*resource.Host{}, nil
	}
	group, err := machineryv1tasks.NewGroup(tasks...)
	if err != nil {
		return nil, err
	}
	for i, signature := range group.Tasks {
		// Set signature args.
		args, err := internaljob.MarshalRequest(internaljob.SyncPeerRequest{
			GroupUUID:      group.GroupUUID,
			GroupTaskCount: i,
			TotalGroupTask: len(group.Tasks),
		})
		if err != nil {
			logger.Errorf("[sync-peers] sync peer task marshal request: %v, error: %v", args, err)
			return nil, err
		}
		signature.Args = args
	}
	// Send sync peer task to worker.
	logger.Infof("[sync-peers] create sync peers in queue %v, tasksNumber %d, taskUUID %v", queue, len(tasks), tasks[0].UUID)
	asyncResults, err := s.job.Server.SendGroupWithContext(ctx, group, 0)
	if err != nil {
		logger.Errorf("[sync-peers] create sync peers in queue %v failed", queue, err)
		return nil, err
	}

	// Get sync peer task result.
	results := make([]reflect.Value, 0, len(asyncResults))
	for _, asyncResult := range asyncResults {
		result, err := asyncResult.GetWithTimeout(s.config.Job.SyncPeers.Timeout, DefaultTaskPollingInterval)
		if err != nil {
			return nil, err
		}
		results = append(results, result...)
	}

	// Unmarshal sync peer task result.
	var hosts []*resource.Host
	if err := internaljob.UnmarshalResponse(results, &hosts); err != nil {
		return nil, err
	}

	return hosts, nil
}

// Merge sync peer results with the data in the peer table.
func (s *syncPeers) mergePeers(ctx context.Context, scheduler models.Scheduler, results []*resource.Host, log *logger.SugaredLoggerOnWith) {
	// Convert sync peer results from slice to map.
	syncPeers := make(map[string]*resource.Host, len(results))
	for _, result := range results {
		// Skip the sync peer that does not belong to the scheduler cluster,
		// it is only supported in Rust client. The golang client lacks the
		// SchedulerClusterID field.
		if result.SchedulerClusterID == 0 {
			continue
		}

		syncPeers[result.ID] = result
	}

	oldPeers := make([]*models.Peer, 0, s.config.Job.SyncPeers.BatchSize)
	if err := s.db.WithContext(ctx).Model(&models.Peer{}).Where("scheduler_cluster_id = ?", scheduler.SchedulerClusterID).FindInBatches(&oldPeers, s.config.Job.SyncPeers.BatchSize, func(tx *gorm.DB, batch int) error {
		peers := make([]*models.Peer, 0, s.config.Job.SyncPeers.BatchSize)
		for _, oldPeer := range oldPeers {
			// If the peer exists in the sync peer results, update the peer data in the database with
			// the sync peer results and delete the sync peer from the sync peers map.
			isSeedPeer := pkgtypes.ParseHostType(oldPeer.Type) != pkgtypes.HostTypeNormal
			id := idgen.HostID(oldPeer.IP, oldPeer.Hostname, isSeedPeer)
			if syncPeer, ok := syncPeers[id]; ok {
				peers = append(peers, &models.Peer{
					Hostname:           syncPeer.Hostname,
					Type:               syncPeer.Type.Name(),
					IDC:                syncPeer.Network.IDC,
					Location:           syncPeer.Network.Location,
					IP:                 syncPeer.IP,
					Port:               syncPeer.Port,
					DownloadPort:       syncPeer.DownloadPort,
					ProxyPort:          syncPeer.ProxyPort,
					ObjectStoragePort:  syncPeer.ObjectStoragePort,
					State:              models.PeerStateActive,
					OS:                 syncPeer.OS,
					Platform:           syncPeer.Platform,
					PlatformFamily:     syncPeer.PlatformFamily,
					PlatformVersion:    syncPeer.PlatformVersion,
					KernelVersion:      syncPeer.KernelVersion,
					GitVersion:         syncPeer.Build.GitVersion,
					GitCommit:          syncPeer.Build.GitCommit,
					BuildPlatform:      syncPeer.Build.Platform,
					SchedulerClusterID: uint(syncPeer.SchedulerClusterID),
				})

				// Delete the sync peer from the sync peers map.
				delete(syncPeers, id)
			} else {
				// If the peer does not exist in the sync peer results, delete the peer in the database.
				if err := tx.Unscoped().Delete(&models.Peer{}, oldPeer.ID).Error; err != nil {
					log.Error(err)
				}
			}
		}

		// Avoid save empty slice.
		if len(peers) > 0 {
			tx.Save(&peers)
		}

		return nil
	}).Error; err != nil {
		log.Error(err)
		return
	}

	// Insert the sync peers that do not exist in the database into the peer table.
	peers := make([]*models.Peer, 0, len(syncPeers))
	for _, syncPeer := range syncPeers {
		peers = append(peers, &models.Peer{
			Hostname:           syncPeer.Hostname,
			Type:               syncPeer.Type.Name(),
			IDC:                syncPeer.Network.IDC,
			Location:           syncPeer.Network.Location,
			IP:                 syncPeer.IP,
			Port:               syncPeer.Port,
			DownloadPort:       syncPeer.DownloadPort,
			ProxyPort:          syncPeer.ProxyPort,
			ObjectStoragePort:  syncPeer.ObjectStoragePort,
			State:              models.PeerStateActive,
			OS:                 syncPeer.OS,
			Platform:           syncPeer.Platform,
			PlatformFamily:     syncPeer.PlatformFamily,
			PlatformVersion:    syncPeer.PlatformVersion,
			KernelVersion:      syncPeer.KernelVersion,
			GitVersion:         syncPeer.Build.GitVersion,
			GitCommit:          syncPeer.Build.GitCommit,
			BuildPlatform:      syncPeer.Build.Platform,
			SchedulerClusterID: uint(syncPeer.SchedulerClusterID),
		})
	}

	// Avoid save empty slice.
	if len(peers) > 0 {
		if err := s.db.WithContext(ctx).CreateInBatches(peers, len(peers)).Error; err != nil {
			log.Error(err)
		}
	}
}
