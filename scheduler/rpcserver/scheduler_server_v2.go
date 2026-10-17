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

	"google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"

	commonv2 "d7y.io/api/v2/pkg/apis/common/v2"
	schedulerv2 "d7y.io/api/v2/pkg/apis/scheduler/v2"

	internaljob "d7y.io/dragonfly/v2/internal/job"
	"d7y.io/dragonfly/v2/scheduler/config"
	"d7y.io/dragonfly/v2/scheduler/job"
	"d7y.io/dragonfly/v2/scheduler/metrics"
	"d7y.io/dragonfly/v2/scheduler/resource/persistent"
	"d7y.io/dragonfly/v2/scheduler/resource/persistentcache"
	"d7y.io/dragonfly/v2/scheduler/resource/standard"
	"d7y.io/dragonfly/v2/scheduler/scheduling"
	"d7y.io/dragonfly/v2/scheduler/service"
)

// schedulerServerV2 is v2 version of the scheduler grpc server.
type schedulerServerV2 struct {
	// Service interface.
	service *service.V2
}

// newSchedulerServerV2 returns v2 version of the scheduler server.
func newSchedulerServerV2(
	cfg *config.Config,
	resource standard.Resource,
	persistentResource persistent.Resource,
	persistentCacheResource persistentcache.Resource,
	scheduling scheduling.Scheduling,
	job job.Job,
	dynconfig config.DynconfigInterface,
	peerDialOptions []grpc.DialOption,
) schedulerv2.SchedulerServer {
	return &schedulerServerV2{service.NewV2(cfg, resource, persistentResource, persistentCacheResource, scheduling, job, internaljob.NewImage(), dynconfig,
		service.WithPeerDialOptions(peerDialOptions...))}
}

// AnnouncePeer announces peer to scheduler.
func (s *schedulerServerV2) AnnouncePeer(stream schedulerv2.Scheduler_AnnouncePeerServer) error {
	// Collect ConcurrentScheduleGauge metrics.
	metrics.ConcurrentScheduleGauge.Inc()
	defer metrics.ConcurrentScheduleGauge.Dec()

	// Collect AnnouncePeerCount metrics.
	metrics.AnnouncePeerCount.Inc()
	if err := s.service.AnnouncePeer(stream); err != nil {
		// Collect AnnouncePeerFailureCount metrics.
		metrics.AnnouncePeerFailureCount.Inc()
		return err
	}

	return nil
}

// StatPeer checks information of peer.
func (s *schedulerServerV2) StatPeer(ctx context.Context, req *schedulerv2.StatPeerRequest) (*commonv2.Peer, error) {
	// Collect StatPeerCount metrics.
	metrics.StatPeerCount.Inc()
	resp, err := s.service.StatPeer(ctx, req)
	if err != nil {
		// Collect StatPeerFailureCount metrics.
		metrics.StatPeerFailureCount.Inc()
		return nil, err
	}

	return resp, nil
}

// DeletePeer releases peer in scheduler.
func (s *schedulerServerV2) DeletePeer(ctx context.Context, req *schedulerv2.DeletePeerRequest) (*emptypb.Empty, error) {
	// Collect LeavePeerCount metrics.
	metrics.LeavePeerCount.Inc()
	if err := s.service.DeletePeer(ctx, req); err != nil {
		// Collect LeavePeerFailureCount metrics.
		metrics.LeavePeerFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// StatTask checks information of task.
func (s *schedulerServerV2) StatTask(ctx context.Context, req *schedulerv2.StatTaskRequest) (*commonv2.Task, error) {
	// Collect StatTaskCount metrics.
	metrics.StatTaskCount.Inc()
	resp, err := s.service.StatTask(ctx, req)
	if err != nil {
		// Collect StatTaskFailureCount metrics.
		metrics.StatTaskFailureCount.Inc()
		return nil, err
	}

	return resp, nil
}

// DeleteTask releases task in scheduler.
func (s *schedulerServerV2) DeleteTask(ctx context.Context, req *schedulerv2.DeleteTaskRequest) (*emptypb.Empty, error) {
	// Collect LeaseTaskCount metrics.
	metrics.LeaveTaskCount.Inc()
	if err := s.service.DeleteTask(ctx, req); err != nil {
		// Collect LeaseTaskFailureCount metrics.
		metrics.LeaveTaskFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// AnnounceHost announces host to scheduler.
func (s *schedulerServerV2) AnnounceHost(ctx context.Context, req *schedulerv2.AnnounceHostRequest) (*emptypb.Empty, error) {
	// Collect AnnounceHostCount metrics.
	metrics.AnnounceHostCount.WithLabelValues(req.Host.GetOs(), req.Host.GetPlatform(), req.Host.GetPlatformFamily(), req.Host.GetPlatformFamily(),
		req.Host.GetKernelVersion(), req.Host.Build.GetGitVersion(), req.Host.Build.GetGitCommit(), req.Host.Build.GetGoVersion(), req.Host.Build.GetPlatform()).Inc()
	if err := s.service.AnnounceHost(ctx, req); err != nil {
		// Collect AnnounceHostFailureCount metrics.
		metrics.AnnounceHostFailureCount.WithLabelValues(req.Host.GetOs(), req.Host.GetPlatform(), req.Host.GetPlatformFamily(), req.Host.GetPlatformFamily(),
			req.Host.GetKernelVersion(), req.Host.Build.GetGitVersion(), req.Host.Build.GetGitCommit(), req.Host.Build.GetGoVersion(), req.Host.Build.GetPlatform()).Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// ListHosts lists hosts in scheduler.
func (s *schedulerServerV2) ListHosts(ctx context.Context, req *schedulerv2.ListHostsRequest) (*schedulerv2.ListHostsResponse, error) {
	// Collect ListHostsCount metrics.
	metrics.ListHostsCount.Inc()
	resp, err := s.service.ListHosts(ctx, req)
	if err != nil {
		// Collect ListHostsFailureCount metrics.
		metrics.ListHostsCountFailureCount.Inc()
		return nil, err
	}

	return resp, nil
}

// DeleteHost releases host in scheduler.
func (s *schedulerServerV2) DeleteHost(ctx context.Context, req *schedulerv2.DeleteHostRequest) (*emptypb.Empty, error) {
	// Collect LeaveHostCount metrics.
	metrics.LeaveHostCount.Inc()
	if err := s.service.DeleteHost(ctx, req); err != nil {
		// Collect LeaveHostFailureCount metrics.
		metrics.LeaveHostFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// TODO(fu220): Implement the following methods.
// AnnounceCachePeer announces cache peer to scheduler.
func (s *schedulerServerV2) AnnounceCachePeer(stream schedulerv2.Scheduler_AnnounceCachePeerServer) error {
	return nil
}

// TODO(fu220): Implement the following methods.
// Checks information of cache peer.
func (s *schedulerServerV2) StatCachePeer(ctx context.Context, req *schedulerv2.StatCachePeerRequest) (*commonv2.CachePeer, error) {
	return nil, nil
}

// TODO(fu220): Implement the following methods.
// DeleteCachePeer releases cache peer in scheduler.
func (s *schedulerServerV2) DeleteCachePeer(ctx context.Context, req *schedulerv2.DeleteCachePeerRequest) (*emptypb.Empty, error) {
	return new(emptypb.Empty), nil
}

// TODO(fu220): Implement the following methods.
// Checks information of cache task.
func (s *schedulerServerV2) StatCacheTask(ctx context.Context, req *schedulerv2.StatCacheTaskRequest) (*commonv2.CacheTask, error) {
	return nil, nil
}

// TODO(fu220): Implement the following methods.
// DeleteCacheTask releases cache task in scheduler.
func (s *schedulerServerV2) DeleteCacheTask(ctx context.Context, req *schedulerv2.DeleteCacheTaskRequest) (*emptypb.Empty, error) {
	return new(emptypb.Empty), nil
}

// AnnouncePersistentPeer announces persistent peer to scheduler.
func (s *schedulerServerV2) AnnouncePersistentPeer(stream schedulerv2.Scheduler_AnnouncePersistentPeerServer) error {
	// Collect AnnouncePersistentPeerCount metrics.
	metrics.AnnouncePersistentPeerCount.Inc()
	if err := s.service.AnnouncePersistentPeer(stream); err != nil {
		// Collect AnnouncePersistentPeerFailureCount metrics.
		metrics.AnnouncePersistentPeerFailureCount.Inc()
		return err
	}

	return nil
}

// StatPersistentPeer checks information of persistent peer.
func (s *schedulerServerV2) StatPersistentPeer(ctx context.Context, req *schedulerv2.StatPersistentPeerRequest) (*commonv2.PersistentPeer, error) {
	// Collect StatPersistentPeerCount metrics.
	metrics.StatPersistentPeerCount.Inc()
	resp, err := s.service.StatPersistentPeer(ctx, req)
	if err != nil {
		// Collect StatPersistentPeerFailureCount metrics.
		metrics.StatPersistentPeerFailureCount.Inc()
		return nil, err
	}

	return resp, nil
}

// DeletePersistentPeer releases persistent peer in scheduler.
func (s *schedulerServerV2) DeletePersistentPeer(ctx context.Context, req *schedulerv2.DeletePersistentPeerRequest) (*emptypb.Empty, error) {
	// Collect DeletePersistentPeerCount metrics.
	metrics.DeletePersistentPeerCount.Inc()
	if err := s.service.DeletePersistentPeer(ctx, req); err != nil {
		// Collect DeletePersistentPeerFailureCount metrics.
		metrics.DeletePersistentPeerFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// UploadPersistentTaskStarted uploads the metadata of the persistent task started.
func (s *schedulerServerV2) UploadPersistentTaskStarted(ctx context.Context, req *schedulerv2.UploadPersistentTaskStartedRequest) (*emptypb.Empty, error) {
	// Collect UploadPersistentTaskStartedCount metrics.
	metrics.UploadPersistentTaskStartedCount.Inc()
	if err := s.service.UploadPersistentTaskStarted(ctx, req); err != nil {
		// Collect UploadPersistentTaskStartedFailureCount metrics.
		metrics.UploadPersistentTaskStartedFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// UploadPersistentTaskFinished uploads the metadata of the persistent task finished.
func (s *schedulerServerV2) UploadPersistentTaskFinished(ctx context.Context, req *schedulerv2.UploadPersistentTaskFinishedRequest) (*commonv2.PersistentTask, error) {
	// Collect UploadPersistentTaskFinishedCount metrics.
	metrics.UploadPersistentTaskFinishedCount.Inc()
	resp, err := s.service.UploadPersistentTaskFinished(ctx, req)
	if err != nil {
		// Collect UploadPersistentTaskFinishedFailureCount metrics.
		metrics.UploadPersistentTaskFinishedFailureCount.Inc()
		return nil, err
	}

	return resp, nil
}

// UploadPersistentTaskFailed uploads the metadata of the persistent task failed.
func (s *schedulerServerV2) UploadPersistentTaskFailed(ctx context.Context, req *schedulerv2.UploadPersistentTaskFailedRequest) (*emptypb.Empty, error) {
	// Collect UploadPersistentTaskFailedCount metrics.
	metrics.UploadPersistentTaskFailedCount.Inc()
	if err := s.service.UploadPersistentTaskFailed(ctx, req); err != nil {
		// Collect UploadPersistentTaskFailedFailureCount metrics.
		metrics.UploadPersistentTaskFailedFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// StatPersistentTask checks information of persistent task.
func (s *schedulerServerV2) StatPersistentTask(ctx context.Context, req *schedulerv2.StatPersistentTaskRequest) (*commonv2.PersistentTask, error) {
	// Collect StatPersistentTaskCount metrics.
	metrics.StatPersistentTaskCount.Inc()
	resp, err := s.service.StatPersistentTask(ctx, req)
	if err != nil {
		// Collect StatPersistentTaskFailureCount metrics.
		metrics.StatPersistentTaskFailureCount.Inc()
		return nil, err
	}

	return resp, nil
}

// DeletePersistentTask releases persistent task in scheduler.
func (s *schedulerServerV2) DeletePersistentTask(ctx context.Context, req *schedulerv2.DeletePersistentTaskRequest) (*emptypb.Empty, error) {
	// Collect DeletePersistentTaskCount metrics.
	metrics.DeletePersistentTaskCount.Inc()
	if err := s.service.DeletePersistentTask(ctx, req); err != nil {
		// Collect DeletePersistentTaskFailureCount metrics.
		metrics.DeletePersistentTaskFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// AnnouncePersistentCachePeer announces persistent cache peer to scheduler.
func (s *schedulerServerV2) AnnouncePersistentCachePeer(stream schedulerv2.Scheduler_AnnouncePersistentCachePeerServer) error {
	// Collect AnnouncePersistentCachePeerCount metrics.
	metrics.AnnouncePersistentCachePeerCount.Inc()
	if err := s.service.AnnouncePersistentCachePeer(stream); err != nil {
		// Collect AnnouncePersistentCachePeerFailureCount metrics.
		metrics.AnnouncePersistentCachePeerFailureCount.Inc()
		return err
	}

	return nil
}

// StatPersistentCachePeer checks information of persistent cache peer.
func (s *schedulerServerV2) StatPersistentCachePeer(ctx context.Context, req *schedulerv2.StatPersistentCachePeerRequest) (*commonv2.PersistentCachePeer, error) {
	// Collect StatPersistentCachePeerCount metrics.
	metrics.StatPersistentCachePeerCount.Inc()
	resp, err := s.service.StatPersistentCachePeer(ctx, req)
	if err != nil {
		// Collect StatPersistentCachePeerFailureCount metrics.
		metrics.StatPersistentCachePeerFailureCount.Inc()
		return nil, err
	}

	return resp, nil
}

// DeletePersistentCachePeer releases persistent cache peer in scheduler.
func (s *schedulerServerV2) DeletePersistentCachePeer(ctx context.Context, req *schedulerv2.DeletePersistentCachePeerRequest) (*emptypb.Empty, error) {
	// Collect DeletePersistentCachePeerCount metrics.
	metrics.DeletePersistentCachePeerCount.Inc()
	if err := s.service.DeletePersistentCachePeer(ctx, req); err != nil {
		// Collect DeletePersistentCachePeerFailureCount metrics.
		metrics.DeletePersistentCachePeerFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// UploadPersistentCacheTaskStarted uploads the metadata of the persistent cache task started.
func (s *schedulerServerV2) UploadPersistentCacheTaskStarted(ctx context.Context, req *schedulerv2.UploadPersistentCacheTaskStartedRequest) (*emptypb.Empty, error) {
	// Collect UploadPersistentCacheTaskStartedCount metrics.
	metrics.UploadPersistentCacheTaskStartedCount.Inc()
	if err := s.service.UploadPersistentCacheTaskStarted(ctx, req); err != nil {
		// Collect UploadPersistentCacheTaskStartedFailureCount metrics.
		metrics.UploadPersistentCacheTaskStartedFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// UploadPersistentCacheTaskFinished uploads the metadata of the persistent cache task finished.
func (s *schedulerServerV2) UploadPersistentCacheTaskFinished(ctx context.Context, req *schedulerv2.UploadPersistentCacheTaskFinishedRequest) (*commonv2.PersistentCacheTask, error) {
	// Collect UploadPersistentCacheTaskFinishedCount metrics.
	metrics.UploadPersistentCacheTaskFinishedCount.Inc()
	resp, err := s.service.UploadPersistentCacheTaskFinished(ctx, req)
	if err != nil {
		// Collect UploadPersistentCacheTaskFinishedFailureCount metrics.
		metrics.UploadPersistentCacheTaskFinishedFailureCount.Inc()
		return nil, err
	}

	return resp, nil
}

// UploadPersistentCacheTaskFailed uploads the metadata of the persistent cache task failed.
func (s *schedulerServerV2) UploadPersistentCacheTaskFailed(ctx context.Context, req *schedulerv2.UploadPersistentCacheTaskFailedRequest) (*emptypb.Empty, error) {
	// Collect UploadPersistentCacheTaskFailedCount metrics.
	metrics.UploadPersistentCacheTaskFailedCount.Inc()
	if err := s.service.UploadPersistentCacheTaskFailed(ctx, req); err != nil {
		// Collect UploadPersistentCacheTaskFailedFailureCount metrics.
		metrics.UploadPersistentCacheTaskFailedFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// StatPersistentCacheTask checks information of persistent cache task.
func (s *schedulerServerV2) StatPersistentCacheTask(ctx context.Context, req *schedulerv2.StatPersistentCacheTaskRequest) (*commonv2.PersistentCacheTask, error) {
	// Collect StatPersistentCacheTaskCount metrics.
	metrics.StatPersistentCacheTaskCount.Inc()
	resp, err := s.service.StatPersistentCacheTask(ctx, req)
	if err != nil {
		// Collect StatPersistentCacheTaskFailureCount metrics.
		metrics.StatPersistentCacheTaskFailureCount.Inc()
		return nil, err
	}

	return resp, nil
}

// DeletePersistentCacheTask releases persistent cache task in scheduler.
func (s *schedulerServerV2) DeletePersistentCacheTask(ctx context.Context, req *schedulerv2.DeletePersistentCacheTaskRequest) (*emptypb.Empty, error) {
	// Collect DeletePersistentCacheTaskCount metrics.
	metrics.DeletePersistentCacheTaskCount.Inc()
	if err := s.service.DeletePersistentCacheTask(ctx, req); err != nil {
		// Collect DeletePersistentCacheTaskFailureCount metrics.
		metrics.DeletePersistentCacheTaskFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// PreheatImage synchronously resolves an image manifest and triggers an asynchronous preheat task.
//
// This is a blocking call. The RPC will not return until the server has completed the
// initial synchronous work: resolving the image manifest and preparing all layer URLs.
//
// After this call successfully returns, a scheduler on the server begins the actual
// preheating process, instructing peers to download the layers in the background.
//
// A successful response (google.protobuf.Empty) confirms that the preparation is complete
// and the asynchronous download task has been scheduled.
func (s *schedulerServerV2) PreheatImage(ctx context.Context, req *schedulerv2.PreheatImageRequest) (*emptypb.Empty, error) {
	// Collect PreheatImageCount metrics.
	metrics.PreheatImageCount.Inc()
	if err := s.service.PreheatImage(ctx, req); err != nil {
		// Collect PreheatImageFailureCount metrics.
		metrics.PreheatImageFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// StatImage provides detailed status for a container image's distribution in peers.
//
// This is a blocking call that first resolves the image manifest and then queries
// all peers to collect the image's download state across the network.
// The response includes both layer information and the status on each peer.
func (s *schedulerServerV2) StatImage(ctx context.Context, req *schedulerv2.StatImageRequest) (*schedulerv2.StatImageResponse, error) {
	// Collect StatImageCount metrics.
	metrics.StatImageCount.Inc()
	resp, err := s.service.StatImage(ctx, req)
	if err != nil {
		// Collect StatImageFailureCount metrics.
		metrics.StatImageFailureCount.Inc()
		return nil, err
	}

	return resp, nil
}

// PreheatFile synchronously triggers an asynchronous preheat task for a file.
//
// This is a blocking call. The RPC will not return until the server has completed the
// initial synchronous work: preparing the file URL.
//
// After this call successfully returns, a scheduler on the server begins the actual
// preheating process, instructing peers to download the file in the background.
//
// A successful response (google.protobuf.Empty) confirms that the preparation is complete
// and the asynchronous download task has been scheduled.
func (s *schedulerServerV2) PreheatFile(ctx context.Context, req *schedulerv2.PreheatFileRequest) (*emptypb.Empty, error) {
	// Collect PreheatFileCount metrics.
	metrics.PreheatFileCount.Inc()
	if err := s.service.PreheatFile(ctx, req); err != nil {
		// Collect PreheatFileFailureCount metrics.
		metrics.PreheatFileFailureCount.Inc()
		return nil, err
	}

	return new(emptypb.Empty), nil
}

// StatFile provides detailed status for files distribution in peers.
//
// This is a blocking call that first queries the file/dir entries and then queries
// all peers to collect the file's download state across the network.
// The response includes the file status on each peer.
func (s *schedulerServerV2) StatFile(ctx context.Context, req *schedulerv2.StatFileRequest) (*schedulerv2.StatFileResponse, error) {
	// Collect StatFileCount metrics.
	metrics.StatFileCount.Inc()
	resp, err := s.service.StatFile(ctx, req)
	if err != nil {
		// Collect StatFileFailureCount metrics.
		metrics.StatFileFailureCount.Inc()
		return nil, err
	}

	return resp, nil
}
