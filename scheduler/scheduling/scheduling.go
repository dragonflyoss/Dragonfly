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

//go:generate mockgen -destination mocks/scheduling_mock.go -source scheduling.go -package mocks

package scheduling

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "d7y.io/api/v2/pkg/apis/common/v1"
	commonv2 "d7y.io/api/v2/pkg/apis/common/v2"
	schedulerv1 "d7y.io/api/v2/pkg/apis/scheduler/v1"
	schedulerv2 "d7y.io/api/v2/pkg/apis/scheduler/v2"

	"d7y.io/dragonfly/v2/pkg/container/set"
	pkgtime "d7y.io/dragonfly/v2/pkg/time"
	"d7y.io/dragonfly/v2/pkg/types"
	"d7y.io/dragonfly/v2/scheduler/config"
	"d7y.io/dragonfly/v2/scheduler/resource/persistent"
	"d7y.io/dragonfly/v2/scheduler/resource/persistentcache"
	"d7y.io/dragonfly/v2/scheduler/resource/standard"
	"d7y.io/dragonfly/v2/scheduler/scheduling/evaluator"
)

// Scheduling defines the interface for scheduling operations in the peer-to-peer download system.
// It provides methods for selecting parents and candidates for normal and persistent cache tasks,
// supporting both v1 and v2 gRPC versions, as well as persistent cache replication.
type Scheduling interface {
	// ScheduleCandidateParents schedules candidate parents to the given normal peer for task download.
	// This method is used exclusively in the v2 gRPC version.
	ScheduleCandidateParents(context.Context, *standard.Peer, set.SafeSet[string]) error

	// ScheduleParentAndCandidateParents schedules a primary parent along with candidate parents to the given normal peer for task download.
	// This method is used exclusively in the v1 gRPC version.
	ScheduleParentAndCandidateParents(context.Context, *standard.Peer, set.SafeSet[string])

	// FindCandidateParents identifies suitable candidate parents for the given peer to download the task.
	// This method is used exclusively in the v2 gRPC version.
	FindCandidateParents(context.Context, *standard.Peer, set.SafeSet[string]) ([]*standard.Peer, bool)

	// FindParentAndCandidateParents identifies a primary parent along with suitable candidate parents for the given peer to download the task.
	// This method is used exclusively in the v1 gRPC version.
	FindParentAndCandidateParents(context.Context, *standard.Peer, set.SafeSet[string]) ([]*standard.Peer, bool)

	// FindReplicatePersistentHosts identifies replication hosts for persistent tasks. It compares the current persistent replica count
	// against the target count to select sufficient parents, returning cached replication peers, non-cached replication hosts, and a success flag.
	FindReplicatePersistentHosts(context.Context, *persistent.Task, set.SafeSet[string]) ([]*persistent.Peer, []*persistent.Host, bool)

	// FindCandidatePersistentParents identifies suitable candidate parents in the persistent for the given peer to download the task.
	FindCandidatePersistentParents(context.Context, *persistent.Peer, set.SafeSet[string]) ([]*persistent.Peer, bool)

	// FindReplicatePersistentCacheHosts identifies replication hosts for persistent cache tasks. It compares the current persistent replica count
	// against the target count to select sufficient parents, returning cached replication peers, non-cached replication hosts, and a success flag.
	FindReplicatePersistentCacheHosts(context.Context, *persistentcache.Task, set.SafeSet[string]) ([]*persistentcache.Peer, []*persistentcache.Host, bool)

	// FindCandidatePersistentCacheParents identifies suitable candidate parents in the persistent cache for the given peer to download the task.
	FindCandidatePersistentCacheParents(context.Context, *persistentcache.Peer, set.SafeSet[string]) ([]*persistentcache.Peer, bool)
}

// scheduling implements the Scheduling interface, managing peer parent selection and persistent cache replication
// using an evaluator, configuration, resource managers, and dynamic config updates.
type scheduling struct {
	// evaluator is the interface for evaluating scheduling decisions.
	evaluator evaluator.Evaluator

	// config holds the static configuration for the scheduler.
	config *config.SchedulerConfig

	// persistentResource manages resources for persistent operations.
	persistentResource persistent.Resource

	// persistentCacheResource manages resources for persistent cache operations.
	persistentCacheResource persistentcache.Resource

	// dynconfig provides access to dynamic configuration updates for the scheduler.
	dynconfig config.DynconfigInterface
}

// New creates a new scheduling instance with the given configuration and dependencies.
func New(cfg *config.SchedulerConfig, persistentResource persistent.Resource, persistentCacheResource persistentcache.Resource, dynconfig config.DynconfigInterface, pluginDir string) Scheduling {
	return &scheduling{
		evaluator:               evaluator.New(cfg.Algorithm, pluginDir),
		config:                  cfg,
		persistentResource:      persistentResource,
		persistentCacheResource: persistentCacheResource,
		dynconfig:               dynconfig,
	}
}

// ScheduleCandidateParents schedules candidate parents for the given peer.
// It retries scheduling up to RetryLimit times, falling back to source if needed.
// This method is used only in the v2 version of the gRPC protocol.
func (s *scheduling) ScheduleCandidateParents(ctx context.Context, peer *standard.Peer, blocklist set.SafeSet[string]) error {
	var n int
	for {
		select {
		case <-ctx.Done():
			peer.Log.Infof("context was done")
			return ctx.Err()
		default:
		}

		// Early return if back-to-source is required due to the peer's NeedBackToSource flag being set
		// or retry count exceeding RetryBackToSourceLimit.
		if peer.Task.CanBackToSource() {
			if peer.NeedBackToSource.Load() {
				stream, loaded := peer.LoadAnnouncePeerStream()
				if !loaded {
					peer.Log.Error("load stream failed")
					return status.Error(codes.FailedPrecondition, "load stream failed")
				}

				peer.Log.Infof("send NeedBackToSourceResponse, because of peer's NeedBackToSource is %t", peer.NeedBackToSource.Load())
				description := fmt.Sprintf("peer's NeedBackToSource is %t", peer.NeedBackToSource.Load())
				if err := stream.Send(&schedulerv2.AnnouncePeerResponse{
					Response: &schedulerv2.AnnouncePeerResponse_NeedBackToSourceResponse{
						NeedBackToSourceResponse: &schedulerv2.NeedBackToSourceResponse{
							Description: &description,
						},
					},
				}); err != nil {
					peer.Log.Error(err)
					return status.Error(codes.FailedPrecondition, err.Error())
				}

				return nil
			}

			// Check overall retry limit for back-to-source decision.
			if n >= s.config.RetryBackToSourceLimit {
				stream, loaded := peer.LoadAnnouncePeerStream()
				if !loaded {
					peer.Log.Error("load stream failed")
					return status.Error(codes.FailedPrecondition, "load stream failed")
				}

				peer.Log.Infof("send NeedBackToSourceResponse, because of scheduling exceeded RetryBackToSourceLimit %d", s.config.RetryBackToSourceLimit)
				description := "scheduling exceeded RetryBackToSourceLimit"
				if err := stream.Send(&schedulerv2.AnnouncePeerResponse{
					Response: &schedulerv2.AnnouncePeerResponse_NeedBackToSourceResponse{
						NeedBackToSourceResponse: &schedulerv2.NeedBackToSourceResponse{
							Description: &description,
						},
					},
				}); err != nil {
					peer.Log.Error(err)
					return status.Error(codes.FailedPrecondition, err.Error())
				}

				return nil
			}
		}

		// Check overall retry limit before proceeding.
		if n >= s.config.RetryLimit {
			peer.Log.Errorf("scheduling failed, because of scheduling exceeded RetryLimit %d", s.config.RetryLimit)
			return status.Error(codes.FailedPrecondition, "scheduling exceeded RetryLimit")
		}

		// Clean up any existing incoming edges for this peer to prepare for new scheduling.
		if err := peer.Task.DeletePeerInEdges(peer.ID); err != nil {
			peer.Log.Error(err)
			return status.Error(codes.Internal, err.Error())
		}

		// Attempt to find candidate parents.
		candidateParents, found := s.FindCandidateParents(ctx, peer, blocklist)
		if !found {
			n++
			peer.Log.Infof("scheduling failed in %d times, because of candidate parents not found", n)

			// Sleep with context-aware timeout to avoid blocking on cancellation.
			pkgtime.RandomDelayWithJitter(ctx, s.config.RetryInterval)
			continue
		}

		// Add edges from candidate parents to the peer in a single batch.
		addedParents := peer.Task.AddPeerEdges(candidateParents, peer)
		if len(addedParents) < len(candidateParents) {
			peer.Log.Warnf("peer adds %d of %d edges", len(addedParents), len(candidateParents))
		}

		if len(addedParents) == 0 {
			n++
			peer.Log.Infof("scheduling failed in %d times, because of no candidate parent edges could be added", n)

			// Sleep with context-aware timeout to avoid blocking on cancellation.
			pkgtime.RandomDelayWithJitter(ctx, s.config.RetryInterval)
			continue
		}

		stream, loaded := peer.LoadAnnouncePeerStream()
		if !loaded {
			if err := peer.Task.DeletePeerInEdges(peer.ID); err != nil {
				err = fmt.Errorf("peer deletes inedges failed: %w", err)
				peer.Log.Error(err)
				return status.Error(codes.Internal, err.Error())
			}

			peer.Log.Error("load stream failed")
			return status.Error(codes.FailedPrecondition, "load stream failed")
		}

		peer.Log.Info("send NormalTaskResponse")
		rawResponse := &schedulerv2.AnnouncePeerResponse{
			Response: constructSuccessNormalTaskResponse(addedParents),
		}
		clonedResponse := proto.Clone(rawResponse).(*schedulerv2.AnnouncePeerResponse)

		if err := stream.Send(clonedResponse); err != nil {
			if err := peer.Task.DeletePeerInEdges(peer.ID); err != nil {
				err = fmt.Errorf("peer deletes inedges failed: %w", err)
				peer.Log.Error(err)
				return status.Error(codes.Internal, err.Error())
			}

			peer.Log.Error(err)
			return status.Error(codes.FailedPrecondition, err.Error())
		}

		peer.Log.Infof("scheduling success in %d times", n+1)
		return nil
	}
}

// ScheduleCandidateParents schedules candidate parents for the given peer.
// It retries scheduling up to RetryLimit times, falling back to source if needed.
// This method is used only in the v1 version of the gRPC protocol.
func (s *scheduling) ScheduleParentAndCandidateParents(ctx context.Context, peer *standard.Peer, blocklist set.SafeSet[string]) {
	var n int
	for {
		select {
		case <-ctx.Done():
			peer.Log.Infof("context was done")
			return
		default:
		}

		// Early return if back-to-source is required due to the peer's NeedBackToSource flag being set
		// or retry count exceeding RetryBackToSourceLimit.
		if peer.Task.CanBackToSource() {
			if peer.NeedBackToSource.Load() {
				stream, loaded := peer.LoadReportPieceResultStream()
				if !loaded {
					peer.Log.Error("load stream failed")
					return
				}

				if err := stream.Send(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedNeedBackSource}); err != nil {
					peer.Log.Error(err)
					return
				}
				peer.Log.Infof("send Code_SchedNeedBackSource to peer, because of peer's NeedBackToSource is %t", peer.NeedBackToSource.Load())

				if err := peer.FSM.Event(ctx, standard.PeerEventDownloadBackToSource); err != nil {
					err = fmt.Errorf("peer fsm event failed: %w", err)
					peer.Log.Error(err)
					return
				}

				if peer.Task.FSM.Is(standard.TaskStateFailed) {
					if err := peer.Task.FSM.Event(ctx, standard.TaskEventDownload); err != nil {
						err = fmt.Errorf("task fsm event failed: %w", err)
						peer.Task.Log.Error(err)
						return
					}
				}

				return
			}

			// Check overall retry limit for back-to-source decision.
			if n >= s.config.RetryBackToSourceLimit {
				stream, loaded := peer.LoadReportPieceResultStream()
				if !loaded {
					peer.Log.Error("load stream failed")
					return
				}

				if err := stream.Send(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedNeedBackSource}); err != nil {
					peer.Log.Error(err)
					return
				}
				peer.Log.Infof("send Code_SchedNeedBackSource to peer, because of scheduling exceeded RetryBackToSourceLimit %d", s.config.RetryBackToSourceLimit)

				if err := peer.FSM.Event(ctx, standard.PeerEventDownloadBackToSource); err != nil {
					err = fmt.Errorf("peer fsm event failed: %w", err)
					peer.Log.Error(err)
					return
				}

				if peer.Task.FSM.Is(standard.TaskStateFailed) {
					if err := peer.Task.FSM.Event(ctx, standard.TaskEventDownload); err != nil {
						err = fmt.Errorf("task fsm event failed: %w", err)
						peer.Task.Log.Error(err)
						return
					}
				}

				return
			}
		}

		// Check overall retry limit before proceeding.
		if n >= s.config.RetryLimit {
			stream, loaded := peer.LoadReportPieceResultStream()
			if !loaded {
				peer.Log.Error("load stream failed")
				return
			}

			if err := stream.Send(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedTaskStatusError}); err != nil {
				peer.Log.Error(err)
				return
			}

			peer.Log.Errorf("send SchedulePeerFailed to peer, because of scheduling exceeded RetryLimit %d", s.config.RetryLimit)
			return
		}

		if err := peer.Task.DeletePeerInEdges(peer.ID); err != nil {
			n++
			err := fmt.Errorf("scheduling failed in %d times, because of %w", n, err)
			peer.Log.Error(err)

			// Sleep with context-aware timeout to avoid blocking on cancellation.
			pkgtime.RandomDelayWithJitter(ctx, s.config.RetryInterval)
			continue
		}

		// Attempt to find candidate parents.
		candidateParents, found := s.FindCandidateParents(ctx, peer, blocklist)
		if !found {
			n++
			peer.Log.Infof("scheduling failed in %d times, because of candidate parents not found", n)

			// Sleep with context-aware timeout to avoid blocking on cancellation.
			pkgtime.RandomDelayWithJitter(ctx, s.config.RetryInterval)
			continue
		}

		// Add edges from candidate parents to the peer in a single batch.
		addedParents := peer.Task.AddPeerEdges(candidateParents, peer)
		if len(addedParents) < len(candidateParents) {
			peer.Log.Debugf("peer adds %d of %d edges", len(addedParents), len(candidateParents))
		}

		if len(addedParents) == 0 {
			n++
			peer.Log.Infof("scheduling failed in %d times, because of no candidate parent edges could be added", n)

			// Sleep with context-aware timeout to avoid blocking on cancellation.
			pkgtime.RandomDelayWithJitter(ctx, s.config.RetryInterval)
			continue
		}

		stream, loaded := peer.LoadReportPieceResultStream()
		if !loaded {
			n++
			peer.Log.Errorf("scheduling failed in %d times, because of loading peer stream failed", n)

			if err := peer.Task.DeletePeerInEdges(peer.ID); err != nil {
				err = fmt.Errorf("peer deletes inedges failed: %w", err)
				peer.Log.Error(err)
				return
			}

			return
		}

		peer.Log.Info("send PeerPacket to peer")
		if err := stream.Send(constructSuccessPeerPacket(peer, addedParents[0], addedParents[1:])); err != nil {
			n++
			err = fmt.Errorf("send PeerPacket to peer failed in %d times, because of %w", n, err)
			peer.Log.Error(err)

			if err := peer.Task.DeletePeerInEdges(peer.ID); err != nil {
				err = fmt.Errorf("peer deletes inedges failed: %w", err)
				peer.Log.Error(err)
				return
			}

			return
		}

		peer.Log.Infof("scheduling success in %d times", n+1)
		return
	}
}

// FindCandidateParents identifies and evaluates candidate parent peers for the given peer.
// It returns a slice of selected candidate parents and a boolean indicating success.
// This method filters, evaluates, and limits candidates based on configuration.
func (s *scheduling) FindCandidateParents(ctx context.Context, peer *standard.Peer, blocklist set.SafeSet[string]) ([]*standard.Peer, bool) {
	// Validate peer state: only peers in ReceivedNormal or Running states are eligible for scheduling.
	// Other states (e.g., BackToSource) indicate prior scheduling or fallback.
	if !(peer.FSM.Is(standard.PeerStateReceivedNormal) || peer.FSM.Is(standard.PeerStateRunning)) {
		peer.Log.Infof("peer state is %s, can not schedule parent", peer.FSM.Current())
		return []*standard.Peer{}, false
	}

	// Filter potential candidate parents, excluding those in the blocklist.
	candidateParents := s.filterCandidateParents(peer, blocklist)
	if len(candidateParents) == 0 {
		peer.Log.Info("can not find candidate parents")
		return []*standard.Peer{}, false
	}

	// Evaluate and sort candidates by score (highest first) using the configured evaluator.
	candidateParents = s.evaluator.EvaluateParents(candidateParents, peer)

	// Determine the maximum number of candidate parents to select.
	// Use dynamic config if available and valid. Otherwise, fall back to default.
	candidateParentLimit := config.DefaultSchedulerCandidateParentLimit
	if config, err := s.dynconfig.GetSchedulerClusterConfig(); err == nil {
		if config.CandidateParentLimit > 0 {
			candidateParentLimit = int(config.CandidateParentLimit)
		}
	}

	// Trim the list to the configured limit if necessary
	if len(candidateParents) > candidateParentLimit {
		candidateParents = candidateParents[:candidateParentLimit]
	}

	parentIDs := make([]string, 0, len(candidateParents))
	for _, candidateParent := range candidateParents {
		parentIDs = append(parentIDs, candidateParent.ID)
	}

	peer.Log.Infof("scheduling candidate parents is %#v", parentIDs)
	return candidateParents, true
}

// FindParentAndCandidateParents identifies and evaluates candidate parent peers for the given peer.
// It returns a slice of selected candidate parents and a boolean indicating success.
// This method filters, evaluates, and limits candidates based on configuration, but requires the peer to be in Running state.
func (s *scheduling) FindParentAndCandidateParents(ctx context.Context, peer *standard.Peer, blocklist set.SafeSet[string]) ([]*standard.Peer, bool) {
	// Validate peer state: only peers in Running state are eligible for scheduling.
	// Other states indicate prior scheduling or fallback.
	if !peer.FSM.Is(standard.PeerStateRunning) {
		peer.Log.Infof("peer state is %s, can not schedule parent", peer.FSM.Current())
		return []*standard.Peer{}, false
	}

	// Filter potential candidate parents, excluding those in the blocklist.
	candidateParents := s.filterCandidateParents(peer, blocklist)
	if len(candidateParents) == 0 {
		peer.Log.Info("can not find candidate parents")
		return []*standard.Peer{}, false
	}

	// Evaluate and sort candidates by score (highest first) using the configured evaluator.
	candidateParents = s.evaluator.EvaluateParents(candidateParents, peer)

	// Determine the maximum number of candidate parents to select.
	// Use dynamic config if available and valid. Otherwise, fall back to default.
	candidateParentLimit := config.DefaultSchedulerCandidateParentLimit
	if config, err := s.dynconfig.GetSchedulerClusterConfig(); err == nil {
		if config.CandidateParentLimit > 0 {
			candidateParentLimit = int(config.CandidateParentLimit)
		}
	}

	// Trim the list to the configured limit if necessary.
	if len(candidateParents) > candidateParentLimit {
		candidateParents = candidateParents[:candidateParentLimit]
	}

	parentIDs := make([]string, 0, len(candidateParents))
	for _, candidateParent := range candidateParents {
		parentIDs = append(parentIDs, candidateParent.ID)
	}

	peer.Log.Infof("scheduling candidate parents is %#v", parentIDs)
	return candidateParents, true
}

// candidateHasRequestedPiece reports whether a candidate has at least one piece
// covered by the requesting peer's range. Full-file requests retain the existing
// optimistic candidate selection because their requested piece set is not bounded
// by a range at scheduling time.
func candidateHasRequestedPiece(peer, candidateParent *standard.Peer) bool {
	if peer.Range == nil || peer.Range.Length <= 0 || peer.Task.PieceLength == 0 || peer.Range.Start < 0 {
		return true
	}

	start := uint64(peer.Range.Start)
	length := uint64(peer.Range.Length)
	end := start + length - 1
	if end < start {
		return true
	}

	firstPiece := start / peer.Task.PieceLength
	lastPiece := end / peer.Task.PieceLength
	for pieceNumber := firstPiece; pieceNumber <= lastPiece; pieceNumber++ {
		if candidateParent.FinishedPieces.Test(uint(pieceNumber)) {
			return true
		}
		if pieceNumber == lastPiece {
			break
		}
	}

	return false
}

// filterCandidateParents selects eligible candidate parent peers from a random sample of peers.
// It applies multiple filters: blocklist exclusion, shared disable check, host uniqueness,
// DAG presence, state validation for normal hosts, bad parent exclusion, and edge feasibility.
// Returns the filtered list of candidate parents.
func (s *scheduling) filterCandidateParents(peer *standard.Peer, blocklist set.SafeSet[string]) []*standard.Peer {
	// Determine the sample size for random peer selection.
	// Use dynamic config if available and valid. Otherwise, fall back to default.
	filterParentLimit := config.DefaultSchedulerFilterParentLimit
	if config, err := s.dynconfig.GetSchedulerClusterConfig(); err == nil && config.FilterParentLimit > 0 {
		filterParentLimit = int(config.FilterParentLimit)
	}

	// Load a random sample of peers up to the filter limit.
	randomPeers := peer.Task.LoadRandomPeers(uint(filterParentLimit))
	prefilteredParents := make([]*standard.Peer, 0, len(randomPeers))
	prefilteredParentIDs := make([]string, 0, len(randomPeers))
	for _, candidateParent := range randomPeers {
		// Skip if candidate is in the blocklist.
		if blocklist.Contains(candidateParent.ID) {
			peer.Log.Debugf("parent %s host %s is not selected because it is in blocklist", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// Skip if candidate has shared disabled.
		if candidateParent.Host.DisableShared {
			peer.Log.Debugf("parent %s host %s is not selected because it is disable shared", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// Skip if candidate shares the same host as the peer (avoids circular downloads).
		if peer.Host.ID == candidateParent.Host.ID {
			peer.Log.Debugf("parent %s host %s is the same as peer host", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// Skip if candidate is not present in the DAG.
		inDegree, err := peer.Task.PeerInDegree(candidateParent.ID)
		if err != nil {
			peer.Log.Debugf("can not find parent %s host %s vertex in dag", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// For normal hosts, skip if in-degree is 0 and not in BackToSource or Succeeded state
		// (indicating the host hasn't started downloading or completed successfully).
		if candidateParent.Host.Type == types.HostTypeNormal &&
			inDegree == 0 &&
			!candidateParent.FSM.Is(standard.PeerStateBackToSource) &&
			!candidateParent.FSM.Is(standard.PeerStateSucceeded) {
			peer.Log.Debugf("parent %s host %s is not selected, because its download state is %d %d %s",
				candidateParent.ID, candidateParent.Host.ID, inDegree, int(candidateParent.Host.Type), candidateParent.FSM.Current())
			continue
		}

		// Skip candidates that do not have any piece requested by this ranged peer.
		// A peer participating in the same task is not proof that it has the
		// particular piece requested by this independent range request.
		if !candidateHasRequestedPiece(peer, candidateParent) {
			peer.Log.Debugf("parent %s host %s is not selected because it does not have a requested piece", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// Skip if candidate is deemed a bad parent by the evaluator.
		if s.evaluator.IsBadParent(candidateParent) {
			peer.Log.Debugf("parent %s host %s is not selected because it is bad node", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// Skip candidates that cannot accept another upload.
		if candidateParent.Host.FreeUploadCount() <= 0 {
			peer.Log.Debugf("parent %s host %s is not selected because its free upload is empty, upload limit is %d, upload count is %d",
				candidateParent.ID,
				candidateParent.Host.ID,
				candidateParent.Host.ConcurrentUploadLimit.Load(),
				candidateParent.Host.ConcurrentUploadCount.Load(),
			)
			continue
		}

		prefilteredParents = append(prefilteredParents, candidateParent)
		prefilteredParentIDs = append(prefilteredParentIDs, candidateParent.ID)
	}

	addableParentIDs := peer.Task.CanAddPeerEdges(prefilteredParentIDs, peer.ID)
	candidateParents := make([]*standard.Peer, 0, len(prefilteredParents))
	candidateParentIDs := make([]string, 0, len(prefilteredParents))
	for _, candidateParent := range prefilteredParents {
		// Skip if an edge cannot be added between candidate and peer.
		if _, ok := addableParentIDs[candidateParent.ID]; !ok {
			peer.Log.Debugf("can not add edge with parent %s host %s", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// All filters passed: add to candidates.
		candidateParents = append(candidateParents, candidateParent)
		candidateParentIDs = append(candidateParentIDs, candidateParent.ID)
	}

	peer.Log.Infof("filter candidate parents is %#v", candidateParentIDs)
	return candidateParents
}

// FindReplicatePersistentHosts finds replicate persistent hosts for the peer to replicate the task. It will compare the current
// persistent replica count with the persistent replica count and try to find enough parents. Then function will return the cached replicate parents,
// the replicate hosts without and found flag.
func (s *scheduling) FindReplicatePersistentHosts(ctx context.Context, task *persistent.Task, blocklist set.SafeSet[string]) ([]*persistent.Peer, []*persistent.Host, bool) {
	currentPersistentReplicaCount, err := s.persistentResource.TaskManager().LoadCurrentPersistentReplicaCount(ctx, task.ID)
	if err != nil {
		err = fmt.Errorf("load current persistent replica count failed: %w", err)
		task.Log.Error(err)
		return nil, nil, false
	}

	needPersistentReplicaCount := int(task.PersistentReplicaCount - currentPersistentReplicaCount)
	if needPersistentReplicaCount <= 0 {
		task.Log.Infof("persistent task %s has enough persistent replica count %d", task.ID, task.PersistentReplicaCount)
		return nil, nil, false
	}

	var (
		replicateHosts           []*persistent.Host
		replicateHostIDs         []string
		cachedReplicateParents   []*persistent.Peer
		cachedReplicateParentIDs []string
	)
	cachedParents := s.filterCachedReplicatePersistentParents(ctx, task, blocklist)
	cachedParentsCount := len(cachedParents)

	// If the number of cached parents is greater than or equal to the number of persistent replica count,
	// return the cached parents directly and no need to find the replicate hosts without cache.
	if cachedParentsCount >= needPersistentReplicaCount {
		for _, cachedParent := range cachedParents[:needPersistentReplicaCount] {
			cachedReplicateParents = append(cachedReplicateParents, cachedParent)
			cachedReplicateParentIDs = append(cachedReplicateParentIDs, cachedParent.ID)
		}

		task.Log.Infof("find cached parents is %#v", cachedReplicateParentIDs)
		return cachedReplicateParents, nil, true
	}

	// If cached parents are not enough, append the replicate cached parents and find the replicate hosts without cache.
	if cachedParentsCount > 0 {
		for _, cachedParent := range cachedParents {
			cachedReplicateParents = append(cachedReplicateParents, cachedParent)
			cachedReplicateParentIDs = append(cachedReplicateParentIDs, cachedParent.ID)
			blocklist.Add(cachedParent.Host.ID)
		}
	}

	// Load all current persistent peers and add them to the blocklist to avoid scheduling the same host.
	currentPersistentPeers, err := s.persistentResource.PeerManager().LoadPersistentAllByTaskID(ctx, task.ID)
	if err != nil {
		err = fmt.Errorf("load all persistent peers failed: %w", err)
		task.Log.Error(err)
		return nil, nil, false
	}

	for _, currentPersistentPeer := range currentPersistentPeers {
		blocklist.Add(currentPersistentPeer.Host.ID)
	}

	// Find the replicate hosts without cache. Calculate the number of persistent replicas needed without considering the cache.
	// Formula: Needed persistent replica count without cache = Total persistent replica count - Current persistent replica count - Cached parents count.
	needPersistentReplicaCount -= cachedParentsCount
	hosts := s.filterReplicatePersistentHosts(ctx, task, needPersistentReplicaCount, blocklist)
	for _, host := range hosts {
		replicateHosts = append(replicateHosts, host)
		replicateHostIDs = append(replicateHostIDs, host.ID)
	}

	if len(cachedReplicateParents) == 0 && len(replicateHosts) == 0 {
		task.Log.Info("can not find replicate hosts")
		return nil, nil, false
	}

	task.Log.Infof("find cached parents is %#v and hosts is %#v", cachedReplicateParentIDs, replicateHostIDs)
	return cachedReplicateParents, replicateHosts, true
}

// FindCandidatePersistentParents identifies and evaluates candidate persistent parent peers for the given peer.
// It returns a slice of selected candidate parents and a boolean indicating success.
// This method filters, evaluates, and limits candidates based on configuration for persistent tasks.
func (s *scheduling) FindCandidatePersistentParents(ctx context.Context, peer *persistent.Peer, blocklist set.SafeSet[string]) ([]*persistent.Peer, bool) {
	// Filter potential candidate parents, excluding those in the blocklist.
	candidateParents := s.filterCandidatePersistentParents(ctx, peer, blocklist)
	if len(candidateParents) == 0 {
		peer.Log.Info("can not find candidate persistent parents")
		return candidateParents, false
	}

	// Evaluate and sort candidates by score (highest first) using the configured evaluator.
	candidateParents = s.evaluator.EvaluatePersistentParents(candidateParents, peer)

	// Determine the maximum number of candidate parents to select.
	// Use dynamic config if available and valid. Otherwise, fall back to default.
	candidateParentLimit := config.DefaultSchedulerCandidateParentLimit
	if config, err := s.dynconfig.GetSchedulerClusterConfig(); err == nil {
		if config.CandidateParentLimit > 0 {
			candidateParentLimit = int(config.CandidateParentLimit)
		}
	}

	// Trim the list to the configured limit if necessary.
	if len(candidateParents) > candidateParentLimit {
		candidateParents = candidateParents[:candidateParentLimit]
	}

	parentIDs := make([]string, 0, len(candidateParents))
	for _, candidateParent := range candidateParents {
		parentIDs = append(parentIDs, candidateParent.ID)
	}

	peer.Log.Infof("scheduling candidate persistent parents is %#v", parentIDs)
	return candidateParents, true
}

// filterCandidatePersistentParents selects eligible candidate persistent parent peers from all loaded peers for the task.
// It applies filters: blocklist exclusion, host uniqueness, and bad parent exclusion.
// Returns the filtered list of candidate parents.
func (s *scheduling) filterCandidatePersistentParents(ctx context.Context, peer *persistent.Peer, blocklist set.SafeSet[string]) []*persistent.Peer {
	parents, err := s.persistentResource.PeerManager().LoadAllByTaskID(ctx, peer.Task.ID)
	if err != nil {
		err = fmt.Errorf("load all persistent parents failed: %w", err)
		peer.Log.Error(err)
		return nil
	}

	candidateParents := make([]*persistent.Peer, 0, len(parents))
	candidateParentIDs := make([]string, 0, len(parents))
	for _, candidateParent := range parents {
		// Skip if candidate is in the blocklist.
		if blocklist.Contains(candidateParent.ID) {
			peer.Log.Debugf("persistent parent %s host %s is not selected because it is in blocklist", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// Skip if candidate shares the same host as the peer (avoids intra-host transfers).
		if peer.Host.ID == candidateParent.Host.ID {
			peer.Log.Debugf("persistent parent %s host %s is the same as peer host", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// Skip if candidate is deemed a bad parent by the evaluator.
		if s.evaluator.IsBadPersistentParent(candidateParent) {
			peer.Log.Debugf("persistent parent %s host %s is not selected because it is bad node", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// All filters passed: add to candidates.
		candidateParents = append(candidateParents, candidateParent)
		candidateParentIDs = append(candidateParentIDs, candidateParent.ID)
	}

	peer.Log.Infof("filter candidate persistent parents is %#v", candidateParentIDs)
	return candidateParents
}

// filterCachedReplicatePersistentParents selects eligible cached replicate persistent parent peers from all loaded peers for the task.
// Cached parents are non-persistent, succeeded peers with shared enabled.
// It applies filters: blocklist exclusion, persistence check, state validation, and shared enablement.
// Returns the filtered list of candidate replicate parents.
func (s *scheduling) filterCachedReplicatePersistentParents(ctx context.Context, task *persistent.Task, blocklist set.SafeSet[string]) []*persistent.Peer {
	parents, err := s.persistentResource.PeerManager().LoadAllByTaskID(ctx, task.ID)
	if err != nil {
		err = fmt.Errorf("load all persistent parents failed: %w", err)
		task.Log.Error(err)
		return nil
	}

	replicateParents := make([]*persistent.Peer, 0, len(parents))
	replicateParentIDs := make([]string, 0, len(parents))
	for _, replicateParent := range parents {
		// Skip if candidate is in the blocklist.
		if blocklist.Contains(replicateParent.ID) {
			task.Log.Debugf("persistent parent %s host %s is not selected because it is in blocklist", replicateParent.ID, replicateParent.Host.ID)
			continue
		}

		// Skip if the parent is persistent (only select non-persistent cached peers for replication).
		if replicateParent.Persistent {
			task.Log.Debugf("persistent parent %s host %s is not selected because it is persistent", replicateParent.ID, replicateParent.Host.ID)
			continue
		}

		// Skip if the parent is not in Succeeded state (ensures content is fully cached).
		if !replicateParent.FSM.Is(persistent.PeerStateSucceeded) {
			task.Log.Debugf("persistent parent %s host %s is not selected because its download state is %s", replicateParent.ID, replicateParent.Host.ID, replicateParent.FSM.Current())
			continue
		}

		// Skip if the host has shared disabled.
		if replicateParent.Host.DisableShared {
			task.Log.Debugf("persistent parent %s host %s is not selected because it is disable shared", replicateParent.ID, replicateParent.Host.ID)
			continue
		}

		// All filters passed: add to candidates.
		replicateParents = append(replicateParents, replicateParent)
		replicateParentIDs = append(replicateParentIDs, replicateParent.ID)
	}

	task.Log.Infof("filter cached parents is %#v", replicateParentIDs)
	return replicateParents
}

// filterReplicatePersistentHosts selects eligible hosts for persistent cache replication from a random sample.
// It applies filters: shared enablement and sufficient free disk space (>= task content length).
// Returns the filtered list of eligible hosts.
func (s *scheduling) filterReplicatePersistentHosts(ctx context.Context, task *persistent.Task, count int, blocklist set.SafeSet[string]) []*persistent.Host {
	hosts, err := s.persistentResource.HostManager().LoadRandom(ctx, count, blocklist)
	if err != nil {
		err = fmt.Errorf("load all persistent hosts failed: %w", err)
		task.Log.Error(err)
		return nil
	}

	replicateHosts := make([]*persistent.Host, 0, len(hosts))
	replicateHostIDs := make([]string, 0, len(hosts))
	for _, host := range hosts {
		// Skip if shared is disabled.
		if host.DisableShared {
			task.Log.Debugf("persistent host %s is not selected because it is disable shared", host.ID)
			continue
		}

		// Skip if free disk space is insufficient for the task content.
		if host.Disk.Free < task.ContentLength {
			task.Log.Debugf("persistent host %s is not selected because its free disk space is not enough, free disk is %d, content length is %d",
				host.ID, host.Disk.Free, task.ContentLength)
			continue
		}

		// All filters passed: add to eligible hosts.
		replicateHosts = append(replicateHosts, host)
		replicateHostIDs = append(replicateHostIDs, host.ID)
	}

	task.Log.Infof("filter hosts is %#v", replicateHostIDs)
	return replicateHosts
}

// FindReplicatePersistentCacheHosts finds replicate persistent cache hosts for the peer to replicate the task. It will compare the current
// persistent replica count with the persistent replica count and try to find enough parents. Then function will return the cached replicate parents,
// the replicate hosts without cache and found flag.
func (s *scheduling) FindReplicatePersistentCacheHosts(ctx context.Context, task *persistentcache.Task, blocklist set.SafeSet[string]) ([]*persistentcache.Peer, []*persistentcache.Host, bool) {
	currentPersistentReplicaCount, err := s.persistentCacheResource.TaskManager().LoadCurrentPersistentReplicaCount(ctx, task.ID)
	if err != nil {
		err = fmt.Errorf("load current persistent replica count failed: %w", err)
		task.Log.Error(err)
		return nil, nil, false
	}

	needPersistentReplicaCount := int(task.PersistentReplicaCount - currentPersistentReplicaCount)
	if needPersistentReplicaCount <= 0 {
		task.Log.Infof("persistent cache task %s has enough persistent replica count %d", task.ID, task.PersistentReplicaCount)
		return nil, nil, false
	}

	var (
		replicateHosts           []*persistentcache.Host
		replicateHostIDs         []string
		cachedReplicateParents   []*persistentcache.Peer
		cachedReplicateParentIDs []string
	)
	cachedParents := s.filterCachedReplicatePersistentCacheParents(ctx, task, blocklist)
	cachedParentsCount := len(cachedParents)

	// If the number of cached parents is greater than or equal to the number of persistent replica count,
	// return the cached parents directly and no need to find the replicate hosts without cache.
	if cachedParentsCount >= needPersistentReplicaCount {
		for _, cachedParent := range cachedParents[:needPersistentReplicaCount] {
			cachedReplicateParents = append(cachedReplicateParents, cachedParent)
			cachedReplicateParentIDs = append(cachedReplicateParentIDs, cachedParent.ID)
		}

		task.Log.Infof("find cached parents is %#v", cachedReplicateParentIDs)
		return cachedReplicateParents, nil, true
	}

	// If cached parents are not enough, append the replicate cached parents and find the replicate hosts without cache.
	if cachedParentsCount > 0 {
		for _, cachedParent := range cachedParents {
			cachedReplicateParents = append(cachedReplicateParents, cachedParent)
			cachedReplicateParentIDs = append(cachedReplicateParentIDs, cachedParent.ID)
			blocklist.Add(cachedParent.Host.ID)
		}
	}

	// Load all current persistent peers and add them to the blocklist to avoid scheduling the same host.
	currentPersistentPeers, err := s.persistentCacheResource.PeerManager().LoadPersistentAllByTaskID(ctx, task.ID)
	if err != nil {
		err = fmt.Errorf("load all persistent cache peers failed: %w", err)
		task.Log.Error(err)
		return nil, nil, false
	}

	for _, currentPersistentPeer := range currentPersistentPeers {
		blocklist.Add(currentPersistentPeer.Host.ID)
	}

	// Find the replicate hosts without cache. Calculate the number of persistent replicas needed without considering the cache.
	// Formula: Needed persistent replica count without cache = Total persistent replica count - Current persistent replica count - Cached parents count.
	needPersistentReplicaCount -= cachedParentsCount
	hosts := s.filterReplicatePersistentCacheHosts(ctx, task, needPersistentReplicaCount, blocklist)
	for _, host := range hosts {
		replicateHosts = append(replicateHosts, host)
		replicateHostIDs = append(replicateHostIDs, host.ID)
	}

	if len(cachedReplicateParents) == 0 && len(replicateHosts) == 0 {
		task.Log.Info("can not find replicate hosts")
		return nil, nil, false
	}

	task.Log.Infof("find cached parents is %#v and hosts is %#v", cachedReplicateParentIDs, replicateHostIDs)
	return cachedReplicateParents, replicateHosts, true
}

// FindCandidatePersistentCacheParents identifies and evaluates candidate persistent cache parent peers for the given peer.
// It returns a slice of selected candidate parents and a boolean indicating success.
// This method filters, evaluates, and limits candidates based on configuration for persistent cache tasks.
func (s *scheduling) FindCandidatePersistentCacheParents(ctx context.Context, peer *persistentcache.Peer, blocklist set.SafeSet[string]) ([]*persistentcache.Peer, bool) {
	// Filter potential candidate parents, excluding those in the blocklist.
	candidateParents := s.filterCandidatePersistentCacheParents(ctx, peer, blocklist)
	if len(candidateParents) == 0 {
		peer.Log.Info("can not find candidate persistent cache parents")
		return candidateParents, false
	}

	// Evaluate and sort candidates by score (highest first) using the configured evaluator.
	candidateParents = s.evaluator.EvaluatePersistentCacheParents(candidateParents, peer)

	// Determine the maximum number of candidate parents to select.
	// Use dynamic config if available and valid. Otherwise, fall back to default.
	candidateParentLimit := config.DefaultSchedulerCandidateParentLimit
	if config, err := s.dynconfig.GetSchedulerClusterConfig(); err == nil {
		if config.CandidateParentLimit > 0 {
			candidateParentLimit = int(config.CandidateParentLimit)
		}
	}

	// Trim the list to the configured limit if necessary.
	if len(candidateParents) > candidateParentLimit {
		candidateParents = candidateParents[:candidateParentLimit]
	}

	parentIDs := make([]string, 0, len(candidateParents))
	for _, candidateParent := range candidateParents {
		parentIDs = append(parentIDs, candidateParent.ID)
	}

	peer.Log.Infof("scheduling candidate persistent cache parents is %#v", parentIDs)
	return candidateParents, true
}

// filterCandidatePersistentCacheParents selects eligible candidate persistent cache parent peers from all loaded peers for the task.
// It applies filters: blocklist exclusion, host uniqueness, and bad parent exclusion.
// Returns the filtered list of candidate parents.
func (s *scheduling) filterCandidatePersistentCacheParents(ctx context.Context, peer *persistentcache.Peer, blocklist set.SafeSet[string]) []*persistentcache.Peer {
	parents, err := s.persistentCacheResource.PeerManager().LoadAllByTaskID(ctx, peer.Task.ID)
	if err != nil {
		err = fmt.Errorf("load all persistent cache parents failed: %w", err)
		peer.Log.Error(err)
		return nil
	}

	candidateParents := make([]*persistentcache.Peer, 0, len(parents))
	candidateParentIDs := make([]string, 0, len(parents))
	for _, candidateParent := range parents {
		// Skip if candidate is in the blocklist.
		if blocklist.Contains(candidateParent.ID) {
			peer.Log.Debugf("persistent cache parent %s host %s is not selected because it is in blocklist", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// Skip if candidate shares the same host as the peer (avoids intra-host transfers).
		if peer.Host.ID == candidateParent.Host.ID {
			peer.Log.Debugf("persistent cache parent %s host %s is the same as peer host", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// Skip if candidate is deemed a bad parent by the evaluator.
		if s.evaluator.IsBadPersistentCacheParent(candidateParent) {
			peer.Log.Debugf("persistent cache parent %s host %s is not selected because it is bad node", candidateParent.ID, candidateParent.Host.ID)
			continue
		}

		// All filters passed: add to candidates.
		candidateParents = append(candidateParents, candidateParent)
		candidateParentIDs = append(candidateParentIDs, candidateParent.ID)
	}

	peer.Log.Infof("filter candidate persistent cache parents is %#v", candidateParentIDs)
	return candidateParents
}

// filterCachedReplicatePersistentCacheParents selects eligible cached replicate persistent cache parent peers from all loaded peers for the task.
// Cached parents are non-persistent, succeeded peers with shared enabled.
// It applies filters: blocklist exclusion, persistence check, state validation, and shared enablement.
// Returns the filtered list of candidate replicate parents.
func (s *scheduling) filterCachedReplicatePersistentCacheParents(ctx context.Context, task *persistentcache.Task, blocklist set.SafeSet[string]) []*persistentcache.Peer {
	parents, err := s.persistentCacheResource.PeerManager().LoadAllByTaskID(ctx, task.ID)
	if err != nil {
		err = fmt.Errorf("load all persistent cache parents failed: %w", err)
		task.Log.Error(err)
		return nil
	}

	replicateParents := make([]*persistentcache.Peer, 0, len(parents))
	replicateParentIDs := make([]string, 0, len(parents))
	for _, replicateParent := range parents {
		// Skip if candidate is in the blocklist.
		if blocklist.Contains(replicateParent.ID) {
			task.Log.Debugf("persistent cache parent %s host %s is not selected because it is in blocklist", replicateParent.ID, replicateParent.Host.ID)
			continue
		}

		// Skip if the parent is persistent (only select non-persistent cached peers for replication).
		if replicateParent.Persistent {
			task.Log.Debugf("persistent cache parent %s host %s is not selected because it is persistent", replicateParent.ID, replicateParent.Host.ID)
			continue
		}

		// Skip if the parent is not in Succeeded state (ensures content is fully cached).
		if !replicateParent.FSM.Is(persistentcache.PeerStateSucceeded) {
			task.Log.Debugf("persistent cache parent %s host %s is not selected because its download state is %s", replicateParent.ID, replicateParent.Host.ID, replicateParent.FSM.Current())
			continue
		}

		// Skip if the host has shared disabled.
		if replicateParent.Host.DisableShared {
			task.Log.Debugf("persistent cache parent %s host %s is not selected because it is disable shared", replicateParent.ID, replicateParent.Host.ID)
			continue
		}

		// All filters passed: add to candidates.
		replicateParents = append(replicateParents, replicateParent)
		replicateParentIDs = append(replicateParentIDs, replicateParent.ID)
	}

	task.Log.Infof("filter cached parents is %#v", replicateParentIDs)
	return replicateParents
}

// filterReplicatePersistentCacheHosts selects eligible hosts for persistent cache replication from a random sample.
// It applies filters: shared enablement and sufficient free disk space (>= task content length).
// Returns the filtered list of eligible hosts.
func (s *scheduling) filterReplicatePersistentCacheHosts(ctx context.Context, task *persistentcache.Task, count int, blocklist set.SafeSet[string]) []*persistentcache.Host {
	hosts, err := s.persistentCacheResource.HostManager().LoadRandom(ctx, count, blocklist)
	if err != nil {
		err = fmt.Errorf("load all persistent cache hosts failed: %w", err)
		task.Log.Error(err)
		return nil
	}

	replicateHosts := make([]*persistentcache.Host, 0, len(hosts))
	replicateHostIDs := make([]string, 0, len(hosts))
	for _, host := range hosts {
		// Skip if shared is disabled.
		if host.DisableShared {
			task.Log.Debugf("persistent cache host %s is not selected because it is disable shared", host.ID)
			continue
		}

		// Skip if free disk space is insufficient for the task content.
		if host.Disk.Free < task.ContentLength {
			task.Log.Debugf("persistent cache host %s is not selected because its free disk space is not enough, free disk is %d, content length is %d",
				host.ID, host.Disk.Free, task.ContentLength)
			continue
		}

		// All filters passed: add to eligible hosts.
		replicateHosts = append(replicateHosts, host)
		replicateHostIDs = append(replicateHostIDs, host.ID)
	}

	task.Log.Infof("filter hosts is %#v", replicateHostIDs)
	return replicateHosts
}

// constructSuccessNormalTaskResponse builds a successful NormalTaskResponse for gRPC v2.
// It converts the provided candidate parents into the commonv2.Peer format,
// including nested task, host, and resource details.
func constructSuccessNormalTaskResponse(candidateParents []*standard.Peer) *schedulerv2.AnnouncePeerResponse_NormalTaskResponse {
	parents := make([]*commonv2.Peer, 0, len(candidateParents))
	for _, candidateParent := range candidateParents {
		// Construct the base peer structure.
		parent := &commonv2.Peer{
			Id:                   candidateParent.ID,
			Priority:             candidateParent.Priority,
			ConcurrentPieceCount: candidateParent.ConcurrentPieceCount,
			Cost:                 durationpb.New(candidateParent.Cost.Load()),
			State:                candidateParent.FSM.Current(),
			NeedBackToSource:     candidateParent.NeedBackToSource.Load(),
			CreatedAt:            timestamppb.New(candidateParent.CreatedAt.Load()),
			UpdatedAt:            timestamppb.New(candidateParent.UpdatedAt.Load()),
		}

		// Set optional range if available.
		if candidateParent.Range != nil {
			parent.Range = &commonv2.Range{
				Start:  uint64(candidateParent.Range.Start),
				Length: uint64(candidateParent.Range.Length),
			}
		}

		// Construct the task structure.
		parent.Task = &commonv2.Task{
			Id:                  candidateParent.Task.ID,
			Type:                candidateParent.Task.Type,
			Url:                 candidateParent.Task.URL,
			Tag:                 &candidateParent.Task.Tag,
			Application:         &candidateParent.Task.Application,
			FilteredQueryParams: candidateParent.Task.FilteredQueryParams,
			RequestHeader:       candidateParent.Task.Header,
			ContentLength:       uint64(candidateParent.Task.ContentLength.Load()),
			PieceCount:          uint32(candidateParent.Task.TotalPieceCount.Load()),
			SizeScope:           candidateParent.Task.SizeScope(),
			State:               candidateParent.Task.FSM.Current(),
			PeerCount:           uint32(candidateParent.Task.PeerCount()),
			CreatedAt:           timestamppb.New(candidateParent.Task.CreatedAt.Load()),
			UpdatedAt:           timestamppb.New(candidateParent.Task.UpdatedAt.Load()),
		}

		// Set optional digest if available.
		if candidateParent.Task.Digest != nil {
			dgst := candidateParent.Task.Digest.String()
			parent.Task.Digest = &dgst
		}

		// Construct the host structure with nested resources.
		parent.Host = &commonv2.Host{
			Id:              candidateParent.Host.ID,
			Type:            uint32(candidateParent.Host.Type),
			Hostname:        candidateParent.Host.Hostname,
			Ip:              candidateParent.Host.IP,
			Port:            candidateParent.Host.Port,
			DownloadPort:    candidateParent.Host.DownloadPort,
			ProxyPort:       candidateParent.Host.ProxyPort,
			Os:              candidateParent.Host.OS,
			Platform:        candidateParent.Host.Platform,
			PlatformFamily:  candidateParent.Host.PlatformFamily,
			PlatformVersion: candidateParent.Host.PlatformVersion,
			KernelVersion:   candidateParent.Host.KernelVersion,
			Cpu: &commonv2.CPU{
				LogicalCount:   candidateParent.Host.CPU.LogicalCount,
				PhysicalCount:  candidateParent.Host.CPU.PhysicalCount,
				Percent:        candidateParent.Host.CPU.Percent,
				ProcessPercent: candidateParent.Host.CPU.ProcessPercent,
				Times: &commonv2.CPUTimes{
					User:      candidateParent.Host.CPU.Times.User,
					System:    candidateParent.Host.CPU.Times.System,
					Idle:      candidateParent.Host.CPU.Times.Idle,
					Nice:      candidateParent.Host.CPU.Times.Nice,
					Iowait:    candidateParent.Host.CPU.Times.Iowait,
					Irq:       candidateParent.Host.CPU.Times.Irq,
					Softirq:   candidateParent.Host.CPU.Times.Softirq,
					Steal:     candidateParent.Host.CPU.Times.Steal,
					Guest:     candidateParent.Host.CPU.Times.Guest,
					GuestNice: candidateParent.Host.CPU.Times.GuestNice,
				},
			},
			Memory: &commonv2.Memory{
				Total:              candidateParent.Host.Memory.Total,
				Available:          candidateParent.Host.Memory.Available,
				Used:               candidateParent.Host.Memory.Used,
				UsedPercent:        candidateParent.Host.Memory.UsedPercent,
				ProcessUsedPercent: candidateParent.Host.Memory.ProcessUsedPercent,
				Free:               candidateParent.Host.Memory.Free,
			},
			Network: &commonv2.Network{
				TcpConnectionCount:       candidateParent.Host.Network.TCPConnectionCount,
				UploadTcpConnectionCount: candidateParent.Host.Network.UploadTCPConnectionCount,
				Location:                 &candidateParent.Host.Network.Location,
				Idc:                      &candidateParent.Host.Network.IDC,
				RxBandwidth:              &candidateParent.Host.Network.RxBandwidth,
				MaxRxBandwidth:           candidateParent.Host.Network.MaxRxBandwidth,
				TxBandwidth:              &candidateParent.Host.Network.TxBandwidth,
				MaxTxBandwidth:           candidateParent.Host.Network.MaxTxBandwidth,
			},
			Disk: &commonv2.Disk{
				Total:             candidateParent.Host.Disk.Total,
				Free:              candidateParent.Host.Disk.Free,
				Used:              candidateParent.Host.Disk.Used,
				UsedPercent:       candidateParent.Host.Disk.UsedPercent,
				InodesTotal:       candidateParent.Host.Disk.InodesTotal,
				InodesUsed:        candidateParent.Host.Disk.InodesUsed,
				InodesFree:        candidateParent.Host.Disk.InodesFree,
				InodesUsedPercent: candidateParent.Host.Disk.InodesUsedPercent,
				WriteBandwidth:    candidateParent.Host.Disk.WriteBandwidth,
				ReadBandwidth:     candidateParent.Host.Disk.ReadBandwidth,
			},
			Build: &commonv2.Build{
				GitVersion: candidateParent.Host.Build.GitVersion,
				GitCommit:  &candidateParent.Host.Build.GitCommit,
				GoVersion:  &candidateParent.Host.Build.GoVersion,
				Platform:   &candidateParent.Host.Build.Platform,
			},
		}

		parents = append(parents, parent)
	}

	return &schedulerv2.AnnouncePeerResponse_NormalTaskResponse{
		NormalTaskResponse: &schedulerv2.NormalTaskResponse{
			CandidateParents: parents,
		},
	}
}

// constructSuccessPeerPacket builds a successful PeerPacket for gRPC v1.
// It includes the source peer details, main parent, and candidate parents.
// Used only in the v1 version of the gRPC protocol.
func constructSuccessPeerPacket(peer *standard.Peer, parent *standard.Peer, candidateParents []*standard.Peer) *schedulerv1.PeerPacket {
	parents := make([]*schedulerv1.PeerPacket_DestPeer, 0, len(candidateParents))
	for _, candidateParent := range candidateParents {
		parents = append(parents, &schedulerv1.PeerPacket_DestPeer{
			Ip:      candidateParent.Host.IP,
			RpcPort: candidateParent.Host.Port,
			PeerId:  candidateParent.ID,
		})
	}

	return &schedulerv1.PeerPacket{
		TaskId: peer.Task.ID,
		SrcPid: peer.ID,
		MainPeer: &schedulerv1.PeerPacket_DestPeer{
			Ip:      parent.Host.IP,
			RpcPort: parent.Host.Port,
			PeerId:  parent.ID,
		},
		CandidatePeers: parents,
		Code:           commonv1.Code_Success,
	}
}
