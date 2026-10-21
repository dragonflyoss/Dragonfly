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

package scheduling

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "d7y.io/api/v2/pkg/apis/common/v1"
	commonv2 "d7y.io/api/v2/pkg/apis/common/v2"
	schedulerv1 "d7y.io/api/v2/pkg/apis/scheduler/v1"
	schedulerv1mocks "d7y.io/api/v2/pkg/apis/scheduler/v1/mocks"
	schedulerv2 "d7y.io/api/v2/pkg/apis/scheduler/v2"
	schedulerv2mocks "d7y.io/api/v2/pkg/apis/scheduler/v2/mocks"

	"d7y.io/dragonfly/v2/manager/types"
	pkgatomic "d7y.io/dragonfly/v2/pkg/atomic"
	"d7y.io/dragonfly/v2/pkg/container/set"
	"d7y.io/dragonfly/v2/pkg/digest"
	"d7y.io/dragonfly/v2/pkg/graph/dag"
	"d7y.io/dragonfly/v2/pkg/idgen"
	nethttp "d7y.io/dragonfly/v2/pkg/net/http"
	pkgtypes "d7y.io/dragonfly/v2/pkg/types"
	"d7y.io/dragonfly/v2/scheduler/config"
	configmocks "d7y.io/dragonfly/v2/scheduler/config/mocks"
	"d7y.io/dragonfly/v2/scheduler/resource/persistent"
	"d7y.io/dragonfly/v2/scheduler/resource/persistentcache"
	"d7y.io/dragonfly/v2/scheduler/resource/standard"
	"d7y.io/dragonfly/v2/scheduler/scheduling/evaluator"
)

var (
	mockPluginDir = "bas"

	mockSchedulerConfig = &config.SchedulerConfig{
		RetryLimit:             2,
		RetryBackToSourceLimit: 1,
		RetryInterval:          10 * time.Millisecond,
		BackToSourceCount:      int(mockTaskBackToSourceLimit),
		Algorithm:              evaluator.DefaultAlgorithm,
	}

	mockRawHost = standard.Host{
		ID:              mockHostID,
		Type:            pkgtypes.HostTypeNormal,
		Hostname:        "foo",
		IP:              "127.0.0.1",
		Port:            8003,
		DownloadPort:    8001,
		ProxyPort:       8004,
		OS:              "darwin",
		Platform:        "darwin",
		PlatformFamily:  "Standalone Workstation",
		PlatformVersion: "11.1",
		KernelVersion:   "20.2.0",
		CPU:             mockCPU,
		Memory:          mockMemory,
		Network:         mockNetwork,
		Disk:            mockDisk,
		Build:           mockBuild,
		CreatedAt:       pkgatomic.NewTime(time.Now()),
		UpdatedAt:       pkgatomic.NewTime(time.Now()),
	}

	mockRawSeedHost = standard.Host{
		ID:              mockSeedHostID,
		Type:            pkgtypes.HostTypeSuperSeed,
		Hostname:        "bar",
		IP:              "127.0.0.1",
		Port:            8003,
		DownloadPort:    8001,
		ProxyPort:       8004,
		OS:              "darwin",
		Platform:        "darwin",
		PlatformFamily:  "Standalone Workstation",
		PlatformVersion: "11.1",
		KernelVersion:   "20.2.0",
		CPU:             mockCPU,
		Memory:          mockMemory,
		Network:         mockNetwork,
		Disk:            mockDisk,
		Build:           mockBuild,
		CreatedAt:       pkgatomic.NewTime(time.Now()),
		UpdatedAt:       pkgatomic.NewTime(time.Now()),
	}

	mockCPU = standard.CPU{
		LogicalCount:   4,
		PhysicalCount:  2,
		Percent:        1,
		ProcessPercent: 0.5,
		Times: standard.CPUTimes{
			User:      240662.2,
			System:    317950.1,
			Idle:      3393691.3,
			Nice:      0,
			Iowait:    0,
			Irq:       0,
			Softirq:   0,
			Steal:     0,
			Guest:     0,
			GuestNice: 0,
		},
	}

	mockMemory = standard.Memory{
		Total:              17179869184,
		Available:          5962813440,
		Used:               11217055744,
		UsedPercent:        65.291858,
		ProcessUsedPercent: 41.525125,
		Free:               2749598908,
	}

	mockNetwork = standard.Network{
		TCPConnectionCount:       10,
		UploadTCPConnectionCount: 1,
		Location:                 mockHostLocation,
		IDC:                      mockHostIDC,
		RxBandwidth:              100,
		MaxRxBandwidth:           200,
		TxBandwidth:              100,
		MaxTxBandwidth:           200,
	}

	mockDisk = standard.Disk{
		Total:             499963174912,
		Free:              37226479616,
		Used:              423809622016,
		UsedPercent:       91.92547406065952,
		InodesTotal:       4882452880,
		InodesUsed:        7835772,
		InodesFree:        4874617108,
		InodesUsedPercent: 0.1604884305611568,
	}

	mockBuild = standard.Build{
		GitVersion: "v1.0.0",
		GitCommit:  "221176b117c6d59366d68f2b34d38be50c935883",
		GoVersion:  "1.18",
		Platform:   "darwin",
	}

	mockTaskBackToSourceLimit   int32  = 200
	mockTaskURL                        = "http://example.com/foo"
	mockTaskPieceLength         uint64 = 2048
	mockTaskID                         = idgen.TaskIDV2ByURLBased(mockTaskURL, &mockTaskPieceLength, mockTaskTag, mockTaskApplication, mockTaskFilteredQueryParams, "")
	mockTaskDigest                     = digest.New(digest.AlgorithmSHA256, "c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4")
	mockTaskTag                        = "d7y"
	mockTaskApplication                = "foo"
	mockTaskFilteredQueryParams        = []string{"bar"}
	mockTaskHeader                     = map[string]string{"content-length": "100"}
	mockHostID                         = idgen.HostID("127.0.0.1", "foo", false)
	mockSeedHostID                     = idgen.HostID("127.0.0.1", "bar", true)
	mockHostLocation                   = "baz"
	mockHostIDC                        = "bas"
	mockPeerID                         = idgen.PeerID()
	mockSeedPeerID                     = idgen.PeerID()

	mockPiece = standard.Piece{
		Number:      1,
		ParentID:    "foo",
		Offset:      2,
		Length:      10,
		Digest:      digest.New(digest.AlgorithmMD5, "1f70f5a1630d608a71442c54ab706638"),
		TrafficType: commonv2.TrafficType_REMOTE_PEER,
		Cost:        1 * time.Minute,
		CreatedAt:   time.Now(),
	}
)

func TestScheduling_New(t *testing.T) {
	tests := []struct {
		name      string
		pluginDir string
		expect    func(t *testing.T, s any)
	}{
		{
			name:      "new scheduling",
			pluginDir: "bar",
			expect: func(t *testing.T, s any) {
				assert := assert.New(t)
				assert.Equal("scheduling", reflect.TypeOf(s).Elem().Name())
			},
		},
		{
			name:      "new scheduling with empty pluginDir",
			pluginDir: "",
			expect: func(t *testing.T, s any) {
				assert := assert.New(t)
				assert.Equal("scheduling", reflect.TypeOf(s).Elem().Name())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)

			tc.expect(t, New(mockSchedulerConfig, persistentResource, persistentCacheResource, dynconfig, tc.pluginDir))
		})
	}
}

func TestScheduling_ScheduleCandidateParents(t *testing.T) {
	needBackToSourceDescription := "peer's NeedBackToSource is true"
	exceededLimitDescription := "scheduling exceeded RetryBackToSourceLimit"

	tests := []struct {
		name   string
		mock   func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder)
		expect func(t *testing.T, peer *standard.Peer, err error)
	}{
		{
			name: "context was done",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				cancel()
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, context.Canceled)
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "peer needs back-to-source and peer stream load failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.NeedBackToSource.Store(true)
				peer.FSM.SetState(standard.PeerStateRunning)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "load stream failed"))
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "peer needs back-to-source and send NeedBackToSourceResponse failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.NeedBackToSource.Store(true)
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreAnnouncePeerStream(stream)

				ma.Send(gomock.Eq(&schedulerv2.AnnouncePeerResponse{
					Response: &schedulerv2.AnnouncePeerResponse_NeedBackToSourceResponse{
						NeedBackToSourceResponse: &schedulerv2.NeedBackToSourceResponse{
							Description: &needBackToSourceDescription,
						},
					},
				})).Return(errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "foo"))
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "peer needs back-to-source and send NeedBackToSourceResponse success",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.NeedBackToSource.Store(true)
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreAnnouncePeerStream(stream)

				ma.Send(gomock.Eq(&schedulerv2.AnnouncePeerResponse{
					Response: &schedulerv2.AnnouncePeerResponse_NeedBackToSourceResponse{
						NeedBackToSourceResponse: &schedulerv2.NeedBackToSourceResponse{
							Description: &needBackToSourceDescription,
						},
					},
				})).Return(nil).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule exceeds RetryBackToSourceLimit and peer stream load failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.FSM.SetState(standard.PeerStateRunning)
				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "load stream failed"))
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule exceeds RetryBackToSourceLimit and send NeedBackToSourceResponse failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreAnnouncePeerStream(stream)

				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1),
					ma.Send(gomock.Eq(&schedulerv2.AnnouncePeerResponse{
						Response: &schedulerv2.AnnouncePeerResponse_NeedBackToSourceResponse{
							NeedBackToSourceResponse: &schedulerv2.NeedBackToSourceResponse{
								Description: &exceededLimitDescription,
							},
						},
					})).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "foo"))
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule exceeds RetryBackToSourceLimit and send NeedBackToSourceResponse success",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreAnnouncePeerStream(stream)

				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1),
					ma.Send(gomock.Eq(&schedulerv2.AnnouncePeerResponse{
						Response: &schedulerv2.AnnouncePeerResponse_NeedBackToSourceResponse{
							NeedBackToSourceResponse: &schedulerv2.NeedBackToSourceResponse{
								Description: &exceededLimitDescription,
							},
						},
					})).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule exceeds RetryLimit",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.BackToSourceLimit.Store(-1)
				peer.StoreAnnouncePeerStream(stream)

				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "scheduling exceeded RetryLimit"))
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule succeeded",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				task.StorePeer(seedPeer)
				peer.FSM.SetState(standard.PeerStateRunning)
				seedPeer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreAnnouncePeerStream(stream)
				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2),
					ma.Send(gomock.Any()).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(peer.Parents(), 1)
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule failed when no edges can be added and falls back to source",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				task.StorePeer(seedPeer)
				peer.FSM.SetState(standard.PeerStateRunning)
				seedPeer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreAnnouncePeerStream(stream)

				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1),
					md.GetSchedulerClusterConfig().DoAndReturn(func() (types.SchedulerClusterConfig, error) {
						if err := task.AddPeerEdge(peer, seedPeer); err != nil {
							return types.SchedulerClusterConfig{}, err
						}

						return types.SchedulerClusterConfig{}, errors.New("foo")
					}).Times(1),
					ma.Send(gomock.Eq(&schedulerv2.AnnouncePeerResponse{
						Response: &schedulerv2.AnnouncePeerResponse_NeedBackToSourceResponse{
							NeedBackToSourceResponse: &schedulerv2.NeedBackToSourceResponse{
								Description: &exceededLimitDescription,
							},
						},
					})).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule succeeded with partially added edges",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				candidateParent := standard.NewPeer(idgen.PeerID(), task, seedPeer.Host)
				task.StorePeer(peer)
				task.StorePeer(seedPeer)
				task.StorePeer(candidateParent)
				peer.FSM.SetState(standard.PeerStateRunning)
				seedPeer.FSM.SetState(standard.PeerStateRunning)
				candidateParent.FSM.SetState(standard.PeerStateRunning)
				peer.StoreAnnouncePeerStream(stream)

				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1),
					md.GetSchedulerClusterConfig().DoAndReturn(func() (types.SchedulerClusterConfig, error) {
						if err := task.AddPeerEdge(peer, candidateParent); err != nil {
							return types.SchedulerClusterConfig{}, err
						}

						return types.SchedulerClusterConfig{}, errors.New("foo")
					}).Times(1),
					ma.Send(gomock.Any()).DoAndReturn(func(resp *schedulerv2.AnnouncePeerResponse) error {
						normalTaskResponse := resp.GetNormalTaskResponse()
						if normalTaskResponse == nil {
							return errors.New("expected NormalTaskResponse")
						}

						if len(normalTaskResponse.CandidateParents) != 1 || normalTaskResponse.CandidateParents[0].Id != seedPeer.ID {
							return fmt.Errorf("unexpected candidate parents in response")
						}

						return nil
					}).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				if assert.Len(peer.Parents(), 1) {
					assert.Equal(mockSeedPeerID, peer.Parents()[0].ID)
				}

				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "peer is not stored in task",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, dag.ErrVertexNotFound.Error()))
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
			},
		},
		{
			name: "candidate parents found but peer stream load failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				task.StorePeer(seedPeer)
				peer.FSM.SetState(standard.PeerStateRunning)
				seedPeer.FSM.SetState(standard.PeerStateRunning)
				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "load stream failed"))
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
			},
		},
		{
			name: "send NormalTaskResponse failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv2.Scheduler_AnnouncePeerServer, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				task.StorePeer(seedPeer)
				peer.FSM.SetState(standard.PeerStateRunning)
				seedPeer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreAnnouncePeerStream(stream)
				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2),
					ma.Send(gomock.Any()).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "foo"))
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			stream := schedulerv2mocks.NewMockScheduler_AnnouncePeerServer(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			ctx, cancel := context.WithCancel(context.Background())
			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			mockSeedHost := standard.NewHost(
				mockRawSeedHost.ID, mockRawSeedHost.IP, mockRawSeedHost.Name, mockRawSeedHost.Hostname,
				mockRawSeedHost.Port, mockRawSeedHost.DownloadPort, mockRawSeedHost.ProxyPort, mockRawSeedHost.Type)
			seedPeer := standard.NewPeer(mockSeedPeerID, mockTask, mockSeedHost)
			blocklist := set.NewSafeSet[string]()

			tc.mock(cancel, peer, seedPeer, blocklist, stream, stream.EXPECT(), dynconfig.EXPECT())
			scheduling := New(mockSchedulerConfig, persistentResource, persistentCacheResource, dynconfig, mockPluginDir)
			tc.expect(t, peer, scheduling.ScheduleCandidateParents(ctx, peer, blocklist))
		})
	}
}

func TestScheduling_ScheduleParentAndCandidateParents(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder)
		expect func(t *testing.T, peer *standard.Peer)
	}{
		{
			name: "context was done",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				cancel()
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "peer needs back-to-source and peer stream load failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.NeedBackToSource.Store(true)
				peer.FSM.SetState(standard.PeerStateRunning)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "peer needs back-to-source and send Code_SchedNeedBackSource failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.NeedBackToSource.Store(true)
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreReportPieceResultStream(stream)

				mr.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedNeedBackSource})).Return(errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "peer needs back-to-source and send Code_SchedNeedBackSource success",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.NeedBackToSource.Store(true)
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreReportPieceResultStream(stream)

				mr.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedNeedBackSource})).Return(nil).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateBackToSource))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "peer needs back-to-source and task state is TaskStateFailed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.NeedBackToSource.Store(true)
				peer.FSM.SetState(standard.PeerStateRunning)
				task.FSM.SetState(standard.TaskStateFailed)
				peer.StoreReportPieceResultStream(stream)

				mr.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedNeedBackSource})).Return(nil).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateBackToSource))
				assert.True(peer.Task.FSM.Is(standard.TaskStateRunning))
			},
		},
		{
			name: "schedule exceeds RetryBackToSourceLimit and peer stream load failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.FSM.SetState(standard.PeerStateRunning)
				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule exceeds RetryBackToSourceLimit and send Code_SchedNeedBackSource failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreReportPieceResultStream(stream)

				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1),
					mr.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedNeedBackSource})).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule exceeds RetryBackToSourceLimit and send Code_SchedNeedBackSource success",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreReportPieceResultStream(stream)

				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1),
					mr.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedNeedBackSource})).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateBackToSource))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule exceeds RetryBackToSourceLimit and  task state is TaskStateFailed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.FSM.SetState(standard.PeerStateRunning)
				task.FSM.SetState(standard.TaskStateFailed)
				peer.StoreReportPieceResultStream(stream)

				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1),
					mr.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedNeedBackSource})).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateBackToSource))
				assert.True(peer.Task.FSM.Is(standard.TaskStateRunning))
			},
		},
		{
			name: "schedule exceeds RetryLimit and peer stream load failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.BackToSourceLimit.Store(-1)
				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule exceeds RetryLimit and send Code_SchedTaskStatusError failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.BackToSourceLimit.Store(-1)
				peer.StoreReportPieceResultStream(stream)

				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2),
					mr.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedTaskStatusError})).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule exceeds RetryLimit and send Code_SchedTaskStatusError success",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.BackToSourceLimit.Store(-1)
				peer.StoreReportPieceResultStream(stream)

				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2),
					mr.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedTaskStatusError})).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule succeeded",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				task.StorePeer(seedPeer)
				peer.FSM.SetState(standard.PeerStateRunning)
				seedPeer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreReportPieceResultStream(stream)
				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2),
					mr.Send(gomock.Any()).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Len(peer.Parents(), 1)
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule failed when no edges can be added and falls back to source",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				task.StorePeer(seedPeer)
				peer.FSM.SetState(standard.PeerStateRunning)
				seedPeer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreReportPieceResultStream(stream)

				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1),
					md.GetSchedulerClusterConfig().DoAndReturn(func() (types.SchedulerClusterConfig, error) {
						if err := task.AddPeerEdge(peer, seedPeer); err != nil {
							return types.SchedulerClusterConfig{}, err
						}

						return types.SchedulerClusterConfig{}, errors.New("foo")
					}).Times(1),
					mr.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedNeedBackSource})).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateBackToSource))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "schedule succeeded with partially added edges",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				candidateParent := standard.NewPeer(idgen.PeerID(), task, seedPeer.Host)
				task.StorePeer(peer)
				task.StorePeer(seedPeer)
				task.StorePeer(candidateParent)
				peer.FSM.SetState(standard.PeerStateRunning)
				seedPeer.FSM.SetState(standard.PeerStateRunning)
				candidateParent.FSM.SetState(standard.PeerStateRunning)
				peer.StoreReportPieceResultStream(stream)

				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1),
					md.GetSchedulerClusterConfig().DoAndReturn(func() (types.SchedulerClusterConfig, error) {
						if err := task.AddPeerEdge(peer, candidateParent); err != nil {
							return types.SchedulerClusterConfig{}, err
						}

						return types.SchedulerClusterConfig{}, errors.New("foo")
					}).Times(1),
					mr.Send(gomock.Any()).DoAndReturn(func(packet *schedulerv1.PeerPacket) error {
						if packet.MainPeer == nil || packet.MainPeer.PeerId != seedPeer.ID {
							return errors.New("unexpected main peer in packet")
						}

						if len(packet.CandidatePeers) != 0 {
							return errors.New("unexpected candidate peers in packet")
						}

						return nil
					}).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				if assert.Len(peer.Parents(), 1) {
					assert.Equal(mockSeedPeerID, peer.Parents()[0].ID)
				}

				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "peer needs back-to-source but peer fsm event failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				peer.NeedBackToSource.Store(true)
				peer.FSM.SetState(standard.PeerStateSucceeded)
				peer.StoreReportPieceResultStream(stream)

				mr.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedNeedBackSource})).Return(nil).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.True(peer.FSM.Is(standard.PeerStateSucceeded))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "peer is not stored in task and retries until back-to-source",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreReportPieceResultStream(stream)

				mr.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedNeedBackSource})).Return(nil).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.True(peer.FSM.Is(standard.PeerStateBackToSource))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "candidate parents found but peer stream load failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				task.StorePeer(seedPeer)
				peer.FSM.SetState(standard.PeerStateRunning)
				seedPeer.FSM.SetState(standard.PeerStateRunning)
				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
		{
			name: "send PeerPacket failed",
			mock: func(cancel context.CancelFunc, peer *standard.Peer, seedPeer *standard.Peer, blocklist set.SafeSet[string], stream schedulerv1.Scheduler_ReportPieceResultServer, mr *schedulerv1mocks.MockScheduler_ReportPieceResultServerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				task := peer.Task
				task.StorePeer(peer)
				task.StorePeer(seedPeer)
				peer.FSM.SetState(standard.PeerStateRunning)
				seedPeer.FSM.SetState(standard.PeerStateRunning)
				peer.StoreReportPieceResultStream(stream)
				gomock.InOrder(
					md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2),
					mr.Send(gomock.Any()).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer) {
				assert := assert.New(t)
				assert.Empty(peer.Parents())
				assert.True(peer.FSM.Is(standard.PeerStateRunning))
				assert.True(peer.Task.FSM.Is(standard.TaskStatePending))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			stream := schedulerv1mocks.NewMockScheduler_ReportPieceResultServer(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			ctx, cancel := context.WithCancel(context.Background())
			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			mockSeedHost := standard.NewHost(
				mockRawSeedHost.ID, mockRawSeedHost.IP, mockRawSeedHost.Name, mockRawSeedHost.Hostname,
				mockRawSeedHost.Port, mockRawSeedHost.DownloadPort, mockRawSeedHost.ProxyPort, mockRawSeedHost.Type)
			seedPeer := standard.NewPeer(mockSeedPeerID, mockTask, mockSeedHost)
			blocklist := set.NewSafeSet[string]()

			tc.mock(cancel, peer, seedPeer, blocklist, stream, stream.EXPECT(), dynconfig.EXPECT())
			scheduling := New(mockSchedulerConfig, persistentResource, persistentCacheResource, dynconfig, mockPluginDir)
			scheduling.ScheduleParentAndCandidateParents(ctx, peer, blocklist)
			tc.expect(t, peer)
		})
	}
}

func TestCandidateHasRequestedPiece(t *testing.T) {
	task := standard.NewTask(
		mockTaskID,
		mockTaskURL,
		mockTaskTag,
		mockTaskApplication,
		commonv2.TaskType_STANDARD,
		mockTaskFilteredQueryParams,
		mockTaskHeader,
		mockTaskBackToSourceLimit,
		standard.WithPieceLength(4*1024*1024),
	)
	requester := standard.NewPeer(
		mockPeerID,
		task,
		standard.NewHost(mockHostID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
			mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type),
		standard.WithRange(nethttp.Range{Start: 8 * 1024 * 1024, Length: 4 * 1024 * 1024}),
	)
	candidate := standard.NewPeer(
		"candidate-parent",
		task,
		standard.NewHost("candidate-host", mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
			mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type),
	)

	assert.False(t, candidateHasRequestedPiece(requester, candidate))
	candidate.FinishedPieces.Set(1)
	assert.False(t, candidateHasRequestedPiece(requester, candidate))
	candidate.FinishedPieces.Set(2)
	assert.True(t, candidateHasRequestedPiece(requester, candidate))

	requester.Range = nil
	assert.True(t, candidateHasRequestedPiece(requester, candidate))
}

func TestScheduling_FindCandidateParents(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder)
		expect func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool)
	}{
		{
			name: "task peers state is failed",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateFailed)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.Empty(parents)
				assert.False(ok)
			},
		},
		{
			name: "task peers is empty",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "task contains only one peer and peer is itself",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateReceivedNormal)
				peer.Task.StorePeer(peer)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "peer is in blocklist",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				blocklist.Add(mockPeers[0].ID)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "peer is bad node",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateReceivedNormal)
				mockPeers[0].FSM.SetState(standard.PeerStateFailed)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "parent is peer's descendant",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				if err := peer.Task.AddPeerEdge(peer, mockPeers[0]); err != nil {
					t.Fatal(err)
				}

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "parent free upload load is zero",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateReceivedNormal)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				mockPeers[0].Host.ConcurrentUploadLimit.Store(0)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "parent is disabled share data with other peers",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateReceivedNormal)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				mockPeers[0].Host.DisableShared = true

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "find back-to-source parent",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				mockPeers[1].FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				peer.Task.BackToSourcePeers.Add(mockPeers[0].ID)
				mockPeers[0].FSM.SetState(standard.PeerStateBackToSource)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Equal(mockPeers[0].ID, parents[0].ID)
			},
		},
		{
			name: "find normal parent",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				mockPeers[1].FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				peer.Task.StorePeer(mockPeers[2])
				mockPeers[0].Host.Type = pkgtypes.HostTypeNormal
				mockPeers[1].Host.Type = pkgtypes.HostTypeSuperSeed

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Equal(mockPeers[1].ID, parents[0].ID)
			},
		},
		{
			name: "parent state is PeerStateSucceeded",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateSucceeded)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Equal(mockPeers[0].ID, parents[0].ID)
			},
		},
		{
			name: "find parent with ancestor",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				mockPeers[1].FSM.SetState(standard.PeerStateRunning)
				mockPeers[2].FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].Host.Type = pkgtypes.HostTypeNormal
				mockPeers[1].Host.Type = pkgtypes.HostTypeNormal
				mockPeers[2].Host.Type = pkgtypes.HostTypeSuperSeed
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				peer.Task.StorePeer(mockPeers[2])
				if err := peer.Task.AddPeerEdge(mockPeers[2], mockPeers[0]); err != nil {
					t.Fatal(err)
				}

				if err := peer.Task.AddPeerEdge(mockPeers[2], mockPeers[1]); err != nil {
					t.Fatal(err)
				}

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Equal(mockPeers[2].ID, parents[0].ID)
			},
		},
		{
			name: "find parent and fetch candidateParentLimit from manager dynconfig",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				mockPeers[1].FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				peer.Task.BackToSourcePeers.Add(mockPeers[0].ID)
				peer.Task.BackToSourcePeers.Add(mockPeers[1].ID)
				mockPeers[0].FSM.SetState(standard.PeerStateBackToSource)
				mockPeers[1].FSM.SetState(standard.PeerStateBackToSource)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{
					CandidateParentLimit: 3,
				}, nil).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Contains([]string{mockPeers[0].ID, mockPeers[1].ID, peer.ID}, parents[0].ID)
			},
		},
		{
			name: "candidateParents is longer than candidateParentLimit",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				mockPeers[1].FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				peer.Task.BackToSourcePeers.Add(mockPeers[0].ID)
				mockPeers[0].FSM.SetState(standard.PeerStateBackToSource)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{
					CandidateParentLimit: 1,
				}, nil).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Len(parents, 1)
				assert.Equal(mockPeers[0].ID, parents[0].ID)
			},
		},
		{
			name: "seed parent is bad node",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateFailed)
				mockPeers[0].Host.Type = pkgtypes.HostTypeSuperSeed
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "candidate parents are trimmed to candidateParentLimit",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateBackToSource)
				mockPeers[1].FSM.SetState(standard.PeerStateBackToSource)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				peer.Task.BackToSourcePeers.Add(mockPeers[0].ID)
				peer.Task.BackToSourcePeers.Add(mockPeers[1].ID)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{
					CandidateParentLimit: 1,
				}, nil).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Len(parents, 1)
				assert.Contains([]string{mockPeers[0].ID, mockPeers[1].ID}, parents[0].ID)
			},
		},
		{
			name: "find parent and fetch filterParentLimit from manager dynconfig",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateBackToSource)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.BackToSourcePeers.Add(mockPeers[0].ID)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{
					FilterParentLimit: 2,
				}, nil).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Equal([]string{mockPeers[0].ID}, []string{parents[0].ID})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)

			var mockPeers []*standard.Peer
			for range 11 {
				mockHost := standard.NewHost(
					idgen.HostID("127.0.0.1", uuid.New().String(), false), mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
					mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
				peer := standard.NewPeer(idgen.PeerID(), mockTask, mockHost)
				mockPeers = append(mockPeers, peer)
			}

			blocklist := set.NewSafeSet[string]()
			tc.mock(peer, mockPeers, blocklist, dynconfig.EXPECT())
			scheduling := New(mockSchedulerConfig, persistentResource, persistentCacheResource, dynconfig, mockPluginDir)
			parents, found := scheduling.FindCandidateParents(context.Background(), peer, blocklist)
			tc.expect(t, peer, mockPeers, parents, found)
		})
	}
}

func TestScheduling_FindParentAndCandidateParents(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder)
		expect func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool)
	}{
		{
			name: "task peers state is failed",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateFailed)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.Empty(parents)
				assert.False(ok)
			},
		},
		{
			name: "task peers is empty",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "task contains only one peer and peer is itself",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "peer is in blocklist",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				blocklist.Add(mockPeers[0].ID)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "peer is bad node",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateFailed)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "parent is peer's descendant",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				if err := peer.Task.AddPeerEdge(peer, mockPeers[0]); err != nil {
					t.Fatal(err)
				}

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "find back-to-source parent",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				mockPeers[1].FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				peer.Task.BackToSourcePeers.Add(mockPeers[0].ID)
				mockPeers[0].FSM.SetState(standard.PeerStateBackToSource)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Equal(mockPeers[0].ID, parents[0].ID)
			},
		},
		{
			name: "find normal parent",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				mockPeers[1].FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				mockPeers[0].Host.Type = pkgtypes.HostTypeSuperSeed
				mockPeers[1].Host.Type = pkgtypes.HostTypeNormal

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Equal(mockPeers[0].ID, parents[0].ID)
			},
		},
		{
			name: "parent state is PeerStateSucceeded",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateSucceeded)
				mockPeers[1].FSM.SetState(standard.PeerStateSucceeded)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				mockPeers[0].Host.Type = pkgtypes.HostTypeSuperSeed
				mockPeers[1].Host.Type = pkgtypes.HostTypeNormal

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Equal(mockPeers[1].ID, parents[0].ID)
			},
		},
		{
			name: "find parent with ancestor",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				mockPeers[1].FSM.SetState(standard.PeerStateRunning)
				mockPeers[2].FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].Host.Type = pkgtypes.HostTypeNormal
				mockPeers[1].Host.Type = pkgtypes.HostTypeNormal
				mockPeers[2].Host.Type = pkgtypes.HostTypeSuperSeed
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				peer.Task.StorePeer(mockPeers[2])
				if err := peer.Task.AddPeerEdge(mockPeers[2], mockPeers[0]); err != nil {
					t.Fatal(err)
				}

				if err := peer.Task.AddPeerEdge(mockPeers[2], mockPeers[1]); err != nil {
					t.Fatal(err)
				}

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Equal(mockPeers[2].ID, parents[0].ID)
			},
		},
		{
			name: "find parent and fetch candidateParentLimit from manager dynconfig",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				mockPeers[1].FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				peer.Task.BackToSourcePeers.Add(mockPeers[0].ID)
				peer.Task.BackToSourcePeers.Add(mockPeers[1].ID)
				mockPeers[0].FSM.SetState(standard.PeerStateBackToSource)
				mockPeers[1].FSM.SetState(standard.PeerStateBackToSource)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{
					CandidateParentLimit: 3,
				}, nil).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Contains([]string{mockPeers[0].ID, mockPeers[1].ID, peer.ID}, parents[0].ID)
			},
		},
		{
			name: "candidateParents is longer than candidateParentLimit",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateRunning)
				mockPeers[1].FSM.SetState(standard.PeerStateRunning)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				peer.Task.BackToSourcePeers.Add(mockPeers[0].ID)
				mockPeers[0].FSM.SetState(standard.PeerStateBackToSource)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{
					CandidateParentLimit: 1,
				}, nil).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Len(parents, 1)
				assert.Equal(mockPeers[0].ID, parents[0].ID)
			},
		},
		{
			name: "seed parent is bad node",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateFailed)
				mockPeers[0].Host.Type = pkgtypes.HostTypeSuperSeed
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "candidate parents are trimmed to candidateParentLimit",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateBackToSource)
				mockPeers[1].FSM.SetState(standard.PeerStateBackToSource)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.StorePeer(mockPeers[1])
				peer.Task.BackToSourcePeers.Add(mockPeers[0].ID)
				peer.Task.BackToSourcePeers.Add(mockPeers[1].ID)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{
					CandidateParentLimit: 1,
				}, nil).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Len(parents, 1)
				assert.Contains([]string{mockPeers[0].ID, mockPeers[1].ID}, parents[0].ID)
			},
		},
		{
			name: "find parent and fetch filterParentLimit from manager dynconfig",
			mock: func(peer *standard.Peer, mockPeers []*standard.Peer, blocklist set.SafeSet[string], md *configmocks.MockDynconfigInterfaceMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				mockPeers[0].FSM.SetState(standard.PeerStateBackToSource)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(mockPeers[0])
				peer.Task.BackToSourcePeers.Add(mockPeers[0].ID)

				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{
					FilterParentLimit: 2,
				}, nil).Times(2)
			},
			expect: func(t *testing.T, peer *standard.Peer, mockPeers []*standard.Peer, parents []*standard.Peer, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
				assert.Equal([]string{mockPeers[0].ID}, []string{parents[0].ID})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)

			var mockPeers []*standard.Peer
			for range 11 {
				mockHost := standard.NewHost(
					idgen.HostID("127.0.0.1", uuid.New().String(), false), mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
					mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
				peer := standard.NewPeer(idgen.PeerID(), mockTask, mockHost)
				mockPeers = append(mockPeers, peer)
			}

			blocklist := set.NewSafeSet[string]()
			tc.mock(peer, mockPeers, blocklist, dynconfig.EXPECT())
			scheduling := New(mockSchedulerConfig, persistentResource, persistentCacheResource, dynconfig, mockPluginDir)
			parents, found := scheduling.FindParentAndCandidateParents(context.Background(), peer, blocklist)
			tc.expect(t, peer, mockPeers, parents, found)
		})
	}
}

func TestScheduling_constructSuccessNormalTaskResponse(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, resp *schedulerv2.AnnouncePeerResponse_NormalTaskResponse, candidateParents []*standard.Peer)
	}{
		{
			name: "construct success normal task response",
			expect: func(t *testing.T, resp *schedulerv2.AnnouncePeerResponse_NormalTaskResponse, candidateParents []*standard.Peer) {
				dgst := candidateParents[0].Task.Digest.String()

				assert := assert.New(t)
				assert.EqualValues(&schedulerv2.AnnouncePeerResponse_NormalTaskResponse{
					NormalTaskResponse: &schedulerv2.NormalTaskResponse{
						CandidateParents: []*commonv2.Peer{
							{
								Id: candidateParents[0].ID,
								Range: &commonv2.Range{
									Start:  uint64(candidateParents[0].Range.Start),
									Length: uint64(candidateParents[0].Range.Length),
								},
								Priority:             candidateParents[0].Priority,
								ConcurrentPieceCount: candidateParents[0].ConcurrentPieceCount,
								Cost:                 durationpb.New(candidateParents[0].Cost.Load()),
								State:                candidateParents[0].FSM.Current(),
								Task: &commonv2.Task{
									Id:                  candidateParents[0].Task.ID,
									Type:                candidateParents[0].Task.Type,
									Url:                 candidateParents[0].Task.URL,
									Digest:              &dgst,
									Tag:                 &candidateParents[0].Task.Tag,
									Application:         &candidateParents[0].Task.Application,
									FilteredQueryParams: candidateParents[0].Task.FilteredQueryParams,
									RequestHeader:       candidateParents[0].Task.Header,
									ContentLength:       uint64(candidateParents[0].Task.ContentLength.Load()),
									PieceCount:          uint32(candidateParents[0].Task.TotalPieceCount.Load()),
									SizeScope:           candidateParents[0].Task.SizeScope(),
									State:               candidateParents[0].Task.FSM.Current(),
									PeerCount:           uint32(candidateParents[0].Task.PeerCount()),
									CreatedAt:           timestamppb.New(candidateParents[0].Task.CreatedAt.Load()),
									UpdatedAt:           timestamppb.New(candidateParents[0].Task.UpdatedAt.Load()),
								},
								Host: &commonv2.Host{
									Id:              candidateParents[0].Host.ID,
									Type:            uint32(candidateParents[0].Host.Type),
									Hostname:        candidateParents[0].Host.Hostname,
									Ip:              candidateParents[0].Host.IP,
									Port:            candidateParents[0].Host.Port,
									DownloadPort:    candidateParents[0].Host.DownloadPort,
									ProxyPort:       candidateParents[0].Host.ProxyPort,
									Os:              candidateParents[0].Host.OS,
									Platform:        candidateParents[0].Host.Platform,
									PlatformFamily:  candidateParents[0].Host.PlatformFamily,
									PlatformVersion: candidateParents[0].Host.PlatformVersion,
									KernelVersion:   candidateParents[0].Host.KernelVersion,
									Cpu: &commonv2.CPU{
										LogicalCount:   candidateParents[0].Host.CPU.LogicalCount,
										PhysicalCount:  candidateParents[0].Host.CPU.PhysicalCount,
										Percent:        candidateParents[0].Host.CPU.Percent,
										ProcessPercent: candidateParents[0].Host.CPU.ProcessPercent,
										Times: &commonv2.CPUTimes{
											User:      candidateParents[0].Host.CPU.Times.User,
											System:    candidateParents[0].Host.CPU.Times.System,
											Idle:      candidateParents[0].Host.CPU.Times.Idle,
											Nice:      candidateParents[0].Host.CPU.Times.Nice,
											Iowait:    candidateParents[0].Host.CPU.Times.Iowait,
											Irq:       candidateParents[0].Host.CPU.Times.Irq,
											Softirq:   candidateParents[0].Host.CPU.Times.Softirq,
											Steal:     candidateParents[0].Host.CPU.Times.Steal,
											Guest:     candidateParents[0].Host.CPU.Times.Guest,
											GuestNice: candidateParents[0].Host.CPU.Times.GuestNice,
										},
									},
									Memory: &commonv2.Memory{
										Total:              candidateParents[0].Host.Memory.Total,
										Available:          candidateParents[0].Host.Memory.Available,
										Used:               candidateParents[0].Host.Memory.Used,
										UsedPercent:        candidateParents[0].Host.Memory.UsedPercent,
										ProcessUsedPercent: candidateParents[0].Host.Memory.ProcessUsedPercent,
										Free:               candidateParents[0].Host.Memory.Free,
									},
									Network: &commonv2.Network{
										TcpConnectionCount:       candidateParents[0].Host.Network.TCPConnectionCount,
										UploadTcpConnectionCount: candidateParents[0].Host.Network.UploadTCPConnectionCount,
										Location:                 &candidateParents[0].Host.Network.Location,
										Idc:                      &candidateParents[0].Host.Network.IDC,
										RxBandwidth:              &candidateParents[0].Host.Network.RxBandwidth,
										MaxRxBandwidth:           candidateParents[0].Host.Network.MaxRxBandwidth,
										TxBandwidth:              &candidateParents[0].Host.Network.TxBandwidth,
										MaxTxBandwidth:           candidateParents[0].Host.Network.MaxTxBandwidth,
									},
									Disk: &commonv2.Disk{
										Total:             candidateParents[0].Host.Disk.Total,
										Free:              candidateParents[0].Host.Disk.Free,
										Used:              candidateParents[0].Host.Disk.Used,
										UsedPercent:       candidateParents[0].Host.Disk.UsedPercent,
										InodesTotal:       candidateParents[0].Host.Disk.InodesTotal,
										InodesUsed:        candidateParents[0].Host.Disk.InodesUsed,
										InodesFree:        candidateParents[0].Host.Disk.InodesFree,
										InodesUsedPercent: candidateParents[0].Host.Disk.InodesUsedPercent,
									},
									Build: &commonv2.Build{
										GitVersion: candidateParents[0].Host.Build.GitVersion,
										GitCommit:  &candidateParents[0].Host.Build.GitCommit,
										GoVersion:  &candidateParents[0].Host.Build.GoVersion,
										Platform:   &candidateParents[0].Host.Build.Platform,
									},
								},
								NeedBackToSource: candidateParents[0].NeedBackToSource.Load(),
								CreatedAt:        timestamppb.New(candidateParents[0].CreatedAt.Load()),
								UpdatedAt:        timestamppb.New(candidateParents[0].UpdatedAt.Load()),
							},
						},
					},
				}, resp)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			candidateParents := []*standard.Peer{standard.NewPeer(idgen.PeerID(), mockTask, mockHost, standard.WithRange(nethttp.Range{
				Start:  1,
				Length: 10,
			}))}
			candidateParents[0].Task.StorePiece(&mockPiece)

			tc.expect(t, constructSuccessNormalTaskResponse(candidateParents), candidateParents)
		})
	}
}

func TestScheduling_constructSuccessPeerPacket(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, packet *schedulerv1.PeerPacket, parent *standard.Peer, candidateParents []*standard.Peer)
	}{
		{
			name: "construct success peer packet",
			expect: func(t *testing.T, packet *schedulerv1.PeerPacket, parent *standard.Peer, candidateParents []*standard.Peer) {
				assert := assert.New(t)
				assert.EqualValues(&schedulerv1.PeerPacket{
					TaskId: mockTaskID,
					SrcPid: mockPeerID,
					MainPeer: &schedulerv1.PeerPacket_DestPeer{
						Ip:      parent.Host.IP,
						RpcPort: parent.Host.Port,
						PeerId:  parent.ID,
					},
					CandidatePeers: []*schedulerv1.PeerPacket_DestPeer{
						{
							Ip:      candidateParents[0].Host.IP,
							RpcPort: candidateParents[0].Host.Port,
							PeerId:  candidateParents[0].ID,
						},
					},
					Code: commonv1.Code_Success,
				}, packet)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))

			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			parent := standard.NewPeer(idgen.PeerID(), mockTask, mockHost)
			candidateParents := []*standard.Peer{standard.NewPeer(idgen.PeerID(), mockTask, mockHost)}

			tc.expect(t, constructSuccessPeerPacket(peer, parent, candidateParents), parent, candidateParents)
		})
	}
}

func mockPersistentHost(id string, idc string, disableShared bool, diskFree uint64) *persistent.Host {
	return persistent.NewHost(
		id, id, id, "127.0.0.1", "darwin", "darwin", "Standalone Workstation", "11.1", "20.2.0", 8003, 8001, 8004,
		1, disableShared, pkgtypes.HostTypeNormal, persistent.CPU{}, persistent.Memory{}, persistent.Network{IDC: idc},
		persistent.Disk{Free: diskFree}, persistent.Build{}, time.Second, time.Now(), time.Now(), nil)
}

func mockPersistentTask(persistentReplicaCount, contentLength uint64) *persistent.Task {
	return persistent.NewTask(mockTaskID, mockTaskURL, "", "", persistent.TaskStatePending, persistentReplicaCount, contentLength, 1, time.Hour, time.Now(), time.Now(), nil)
}

func mockPersistentPeer(state string, isPersistent bool, task *persistent.Task, host *persistent.Host) *persistent.Peer {
	return persistent.NewPeer(idgen.PeerID(), state, isPersistent, nil, nil, task, host, 0, time.Now(), time.Now(), nil)
}

func mockPersistentCacheHost(id string, idc string, disableShared bool, diskFree uint64) *persistentcache.Host {
	return persistentcache.NewHost(
		id, id, id, "127.0.0.1", "darwin", "darwin", "Standalone Workstation", "11.1", "20.2.0", 8003, 8001, 8004,
		1, disableShared, pkgtypes.HostTypeNormal, persistentcache.CPU{}, persistentcache.Memory{}, persistentcache.Network{IDC: idc},
		persistentcache.Disk{Free: diskFree}, persistentcache.Build{}, time.Second, time.Now(), time.Now(), nil)
}

func mockPersistentCacheTask(persistentReplicaCount, contentLength uint64) *persistentcache.Task {
	return persistentcache.NewTask(mockTaskID, mockTaskTag, mockTaskApplication, persistentcache.TaskStatePending, persistentReplicaCount, mockTaskPieceLength, contentLength, 1, time.Hour, time.Now(), time.Now(), nil)
}

func mockPersistentCachePeer(state string, isPersistent bool, task *persistentcache.Task, host *persistentcache.Host) *persistentcache.Peer {
	return persistentcache.NewPeer(idgen.PeerID(), state, isPersistent, nil, nil, task, host, 0, time.Now(), time.Now(), nil)
}

func TestScheduling_FindReplicatePersistentHosts(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(task *persistent.Task, blocklist set.SafeSet[string], mt *persistent.MockTaskManagerMockRecorder, mp *persistent.MockPeerManagerMockRecorder, mh *persistent.MockHostManagerMockRecorder) ([]*persistent.Peer, []*persistent.Host)
		expect func(t *testing.T, mockParents []*persistent.Peer, mockHosts []*persistent.Host, cachedParents []*persistent.Peer, hosts []*persistent.Host, blocklist set.SafeSet[string], found bool)
	}{
		{
			name: "load current persistent replica count failed",
			mock: func(task *persistent.Task, blocklist set.SafeSet[string], mt *persistent.MockTaskManagerMockRecorder, mp *persistent.MockPeerManagerMockRecorder, mh *persistent.MockHostManagerMockRecorder) ([]*persistent.Peer, []*persistent.Host) {
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(0), errors.New("foo")).Times(1)
				return nil, nil
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, mockHosts []*persistent.Host, cachedParents []*persistent.Peer, hosts []*persistent.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Nil(cachedParents)
				assert.Nil(hosts)
			},
		},
		{
			name: "task already has enough persistent replicas",
			mock: func(task *persistent.Task, blocklist set.SafeSet[string], mt *persistent.MockTaskManagerMockRecorder, mp *persistent.MockPeerManagerMockRecorder, mh *persistent.MockHostManagerMockRecorder) ([]*persistent.Peer, []*persistent.Host) {
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(task.PersistentReplicaCount, nil).Times(1)
				return nil, nil
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, mockHosts []*persistent.Host, cachedParents []*persistent.Peer, hosts []*persistent.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Nil(cachedParents)
				assert.Nil(hosts)
			},
		},
		{
			name: "cached parents satisfy the needed replica count",
			mock: func(task *persistent.Task, blocklist set.SafeSet[string], mt *persistent.MockTaskManagerMockRecorder, mp *persistent.MockPeerManagerMockRecorder, mh *persistent.MockHostManagerMockRecorder) ([]*persistent.Peer, []*persistent.Host) {
				cachedParents := []*persistent.Peer{
					mockPersistentPeer(persistent.PeerStateSucceeded, false, task, mockPersistentHost("cached1", mockHostIDC, false, 1000)),
					mockPersistentPeer(persistent.PeerStateSucceeded, false, task, mockPersistentHost("cached2", mockHostIDC, false, 1000)),
					mockPersistentPeer(persistent.PeerStateSucceeded, false, task, mockPersistentHost("cached3", mockHostIDC, false, 1000)),
				}
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(1), nil).Times(1)
				mp.LoadAllByTaskID(gomock.Any(), task.ID).Return(cachedParents, nil).Times(1)
				return cachedParents[:2], nil
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, mockHosts []*persistent.Host, cachedParents []*persistent.Peer, hosts []*persistent.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.True(found)
				assert.Equal(mockParents, cachedParents)
				assert.Nil(hosts)
				assert.Equal(uint(0), blocklist.Len())
			},
		},
		{
			name: "ineligible cached parents are filtered and replicate hosts are found",
			mock: func(task *persistent.Task, blocklist set.SafeSet[string], mt *persistent.MockTaskManagerMockRecorder, mp *persistent.MockPeerManagerMockRecorder, mh *persistent.MockHostManagerMockRecorder) ([]*persistent.Peer, []*persistent.Host) {
				blocklisted := mockPersistentPeer(persistent.PeerStateSucceeded, false, task, mockPersistentHost("blocklisted", mockHostIDC, false, 1000))
				blocklist.Add(blocklisted.ID)
				persistentPeer := mockPersistentPeer(persistent.PeerStateSucceeded, true, task, mockPersistentHost("persistent", mockHostIDC, false, 1000))
				parents := []*persistent.Peer{
					blocklisted,
					persistentPeer,
					mockPersistentPeer(persistent.PeerStateRunning, false, task, mockPersistentHost("running", mockHostIDC, false, 1000)),
					mockPersistentPeer(persistent.PeerStateSucceeded, false, task, mockPersistentHost("disableShared", mockHostIDC, true, 1000)),
				}
				hosts := []*persistent.Host{
					mockPersistentHost("shared", mockHostIDC, false, 1000),
					mockPersistentHost("hostDisableShared", mockHostIDC, true, 1000),
					mockPersistentHost("smallDisk", mockHostIDC, false, 10),
				}
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(2), nil).Times(1)
				mp.LoadAllByTaskID(gomock.Any(), task.ID).Return(parents, nil).Times(1)
				mp.LoadPersistentAllByTaskID(gomock.Any(), task.ID).Return([]*persistent.Peer{persistentPeer}, nil).Times(1)
				mh.LoadRandom(gomock.Any(), 1, gomock.Any()).Return(hosts, nil).Times(1)
				return nil, hosts[:1]
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, mockHosts []*persistent.Host, cachedParents []*persistent.Peer, hosts []*persistent.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.True(found)
				assert.Nil(cachedParents)
				assert.Equal(mockHosts, hosts)
				assert.True(blocklist.Contains("persistent"))
			},
		},
		{
			name: "load current persistent peers failed",
			mock: func(task *persistent.Task, blocklist set.SafeSet[string], mt *persistent.MockTaskManagerMockRecorder, mp *persistent.MockPeerManagerMockRecorder, mh *persistent.MockHostManagerMockRecorder) ([]*persistent.Peer, []*persistent.Host) {
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(0), nil).Times(1)
				mp.LoadAllByTaskID(gomock.Any(), task.ID).Return(nil, errors.New("foo")).Times(1)
				mp.LoadPersistentAllByTaskID(gomock.Any(), task.ID).Return(nil, errors.New("foo")).Times(1)
				return nil, nil
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, mockHosts []*persistent.Host, cachedParents []*persistent.Peer, hosts []*persistent.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Nil(cachedParents)
				assert.Nil(hosts)
			},
		},
		{
			name: "cached parents are insufficient and are completed with replicate hosts",
			mock: func(task *persistent.Task, blocklist set.SafeSet[string], mt *persistent.MockTaskManagerMockRecorder, mp *persistent.MockPeerManagerMockRecorder, mh *persistent.MockHostManagerMockRecorder) ([]*persistent.Peer, []*persistent.Host) {
				cachedParent := mockPersistentPeer(persistent.PeerStateSucceeded, false, task, mockPersistentHost("cached", mockHostIDC, false, 1000))
				persistentPeer := mockPersistentPeer(persistent.PeerStateSucceeded, true, task, mockPersistentHost("persistent", mockHostIDC, false, 1000))
				hosts := []*persistent.Host{mockPersistentHost("host1", mockHostIDC, false, 1000), mockPersistentHost("host2", mockHostIDC, false, 1000)}
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(0), nil).Times(1)
				mp.LoadAllByTaskID(gomock.Any(), task.ID).Return([]*persistent.Peer{cachedParent, persistentPeer}, nil).Times(1)
				mp.LoadPersistentAllByTaskID(gomock.Any(), task.ID).Return([]*persistent.Peer{persistentPeer}, nil).Times(1)
				mh.LoadRandom(gomock.Any(), 2, gomock.Any()).Return(hosts, nil).Times(1)
				return []*persistent.Peer{cachedParent}, hosts
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, mockHosts []*persistent.Host, cachedParents []*persistent.Peer, hosts []*persistent.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.True(found)
				assert.Equal(mockParents, cachedParents)
				assert.Equal(mockHosts, hosts)
				assert.True(blocklist.Contains("cached"))
				assert.True(blocklist.Contains("persistent"))
			},
		},
		{
			name: "load replicate hosts failed",
			mock: func(task *persistent.Task, blocklist set.SafeSet[string], mt *persistent.MockTaskManagerMockRecorder, mp *persistent.MockPeerManagerMockRecorder, mh *persistent.MockHostManagerMockRecorder) ([]*persistent.Peer, []*persistent.Host) {
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(0), nil).Times(1)
				mp.LoadAllByTaskID(gomock.Any(), task.ID).Return(nil, nil).Times(1)
				mp.LoadPersistentAllByTaskID(gomock.Any(), task.ID).Return(nil, nil).Times(1)
				mh.LoadRandom(gomock.Any(), 3, gomock.Any()).Return(nil, errors.New("foo")).Times(1)
				return nil, nil
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, mockHosts []*persistent.Host, cachedParents []*persistent.Peer, hosts []*persistent.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Nil(cachedParents)
				assert.Nil(hosts)
			},
		},
		{
			name: "no cached parent and no replicate host",
			mock: func(task *persistent.Task, blocklist set.SafeSet[string], mt *persistent.MockTaskManagerMockRecorder, mp *persistent.MockPeerManagerMockRecorder, mh *persistent.MockHostManagerMockRecorder) ([]*persistent.Peer, []*persistent.Host) {
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(0), nil).Times(1)
				mp.LoadAllByTaskID(gomock.Any(), task.ID).Return(nil, nil).Times(1)
				mp.LoadPersistentAllByTaskID(gomock.Any(), task.ID).Return(nil, nil).Times(1)
				mh.LoadRandom(gomock.Any(), 3, gomock.Any()).Return(nil, nil).Times(1)
				return nil, nil
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, mockHosts []*persistent.Host, cachedParents []*persistent.Peer, hosts []*persistent.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Nil(cachedParents)
				assert.Nil(hosts)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			taskManager := persistent.NewMockTaskManager(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)
			hostManager := persistent.NewMockHostManager(ctl)
			persistentResource.EXPECT().TaskManager().Return(taskManager).AnyTimes()
			persistentResource.EXPECT().PeerManager().Return(peerManager).AnyTimes()
			persistentResource.EXPECT().HostManager().Return(hostManager).AnyTimes()

			task := mockPersistentTask(3, 100)
			blocklist := set.NewSafeSet[string]()
			mockParents, mockHosts := tc.mock(task, blocklist, taskManager.EXPECT(), peerManager.EXPECT(), hostManager.EXPECT())
			scheduling := New(mockSchedulerConfig, persistentResource, persistentCacheResource, dynconfig, mockPluginDir)
			cachedParents, hosts, found := scheduling.FindReplicatePersistentHosts(context.Background(), task, blocklist)
			tc.expect(t, mockParents, mockHosts, cachedParents, hosts, blocklist, found)
		})
	}
}

func TestScheduling_FindCandidatePersistentParents(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(peer *persistent.Peer, blocklist set.SafeSet[string], mp *persistent.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistent.Peer
		expect func(t *testing.T, mockParents []*persistent.Peer, candidateParents []*persistent.Peer, found bool)
	}{
		{
			name: "load persistent parents failed",
			mock: func(peer *persistent.Peer, blocklist set.SafeSet[string], mp *persistent.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistent.Peer {
				mp.LoadAllByTaskID(gomock.Any(), peer.Task.ID).Return(nil, errors.New("foo")).Times(1)
				return nil
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, candidateParents []*persistent.Peer, found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Empty(candidateParents)
			},
		},
		{
			name: "parent is in blocklist",
			mock: func(peer *persistent.Peer, blocklist set.SafeSet[string], mp *persistent.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistent.Peer {
				parent := mockPersistentPeer(persistent.PeerStateSucceeded, true, peer.Task, mockPersistentHost("parent", mockHostIDC, false, 1000))
				blocklist.Add(parent.ID)
				mp.LoadAllByTaskID(gomock.Any(), peer.Task.ID).Return([]*persistent.Peer{parent}, nil).Times(1)
				return nil
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, candidateParents []*persistent.Peer, found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Empty(candidateParents)
			},
		},
		{
			name: "parent shares the peer host",
			mock: func(peer *persistent.Peer, blocklist set.SafeSet[string], mp *persistent.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistent.Peer {
				parent := mockPersistentPeer(persistent.PeerStateSucceeded, true, peer.Task, peer.Host)
				mp.LoadAllByTaskID(gomock.Any(), peer.Task.ID).Return([]*persistent.Peer{parent}, nil).Times(1)
				return nil
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, candidateParents []*persistent.Peer, found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Empty(candidateParents)
			},
		},
		{
			name: "parent is bad node",
			mock: func(peer *persistent.Peer, blocklist set.SafeSet[string], mp *persistent.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistent.Peer {
				parent := mockPersistentPeer(persistent.PeerStateFailed, true, peer.Task, mockPersistentHost("parent", mockHostIDC, false, 1000))
				mp.LoadAllByTaskID(gomock.Any(), peer.Task.ID).Return([]*persistent.Peer{parent}, nil).Times(1)
				return nil
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, candidateParents []*persistent.Peer, found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Empty(candidateParents)
			},
		},
		{
			name: "candidate parents are sorted by affinity",
			mock: func(peer *persistent.Peer, blocklist set.SafeSet[string], mp *persistent.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistent.Peer {
				far := mockPersistentPeer(persistent.PeerStateSucceeded, true, peer.Task, mockPersistentHost("far", "other", false, 1000))
				near := mockPersistentPeer(persistent.PeerStateSucceeded, true, peer.Task, mockPersistentHost("near", mockHostIDC, false, 1000))
				mp.LoadAllByTaskID(gomock.Any(), peer.Task.ID).Return([]*persistent.Peer{far, near}, nil).Times(1)
				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
				return []*persistent.Peer{near, far}
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, candidateParents []*persistent.Peer, found bool) {
				assert := assert.New(t)
				assert.True(found)
				assert.Equal(mockParents, candidateParents)
			},
		},
		{
			name: "candidate parents are trimmed to candidateParentLimit",
			mock: func(peer *persistent.Peer, blocklist set.SafeSet[string], mp *persistent.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistent.Peer {
				far := mockPersistentPeer(persistent.PeerStateSucceeded, true, peer.Task, mockPersistentHost("far", "other", false, 1000))
				near := mockPersistentPeer(persistent.PeerStateSucceeded, true, peer.Task, mockPersistentHost("near", mockHostIDC, false, 1000))
				mp.LoadAllByTaskID(gomock.Any(), peer.Task.ID).Return([]*persistent.Peer{far, near}, nil).Times(1)
				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{CandidateParentLimit: 1}, nil).Times(1)
				return []*persistent.Peer{near}
			},
			expect: func(t *testing.T, mockParents []*persistent.Peer, candidateParents []*persistent.Peer, found bool) {
				assert := assert.New(t)
				assert.True(found)
				assert.Equal(mockParents, candidateParents)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)
			persistentResource.EXPECT().PeerManager().Return(peerManager).AnyTimes()

			task := mockPersistentTask(3, 100)
			peer := mockPersistentPeer(persistent.PeerStateRunning, false, task, mockPersistentHost("peer", mockHostIDC, false, 1000))
			blocklist := set.NewSafeSet[string]()
			mockParents := tc.mock(peer, blocklist, peerManager.EXPECT(), dynconfig.EXPECT())
			scheduling := New(mockSchedulerConfig, persistentResource, persistentCacheResource, dynconfig, mockPluginDir)
			candidateParents, found := scheduling.FindCandidatePersistentParents(context.Background(), peer, blocklist)
			tc.expect(t, mockParents, candidateParents, found)
		})
	}
}

func TestScheduling_FindReplicatePersistentCacheHosts(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(task *persistentcache.Task, blocklist set.SafeSet[string], mt *persistentcache.MockTaskManagerMockRecorder, mp *persistentcache.MockPeerManagerMockRecorder, mh *persistentcache.MockHostManagerMockRecorder) ([]*persistentcache.Peer, []*persistentcache.Host)
		expect func(t *testing.T, mockParents []*persistentcache.Peer, mockHosts []*persistentcache.Host, cachedParents []*persistentcache.Peer, hosts []*persistentcache.Host, blocklist set.SafeSet[string], found bool)
	}{
		{
			name: "load current persistent replica count failed",
			mock: func(task *persistentcache.Task, blocklist set.SafeSet[string], mt *persistentcache.MockTaskManagerMockRecorder, mp *persistentcache.MockPeerManagerMockRecorder, mh *persistentcache.MockHostManagerMockRecorder) ([]*persistentcache.Peer, []*persistentcache.Host) {
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(0), errors.New("foo")).Times(1)
				return nil, nil
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, mockHosts []*persistentcache.Host, cachedParents []*persistentcache.Peer, hosts []*persistentcache.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Nil(cachedParents)
				assert.Nil(hosts)
			},
		},
		{
			name: "task already has enough persistent replicas",
			mock: func(task *persistentcache.Task, blocklist set.SafeSet[string], mt *persistentcache.MockTaskManagerMockRecorder, mp *persistentcache.MockPeerManagerMockRecorder, mh *persistentcache.MockHostManagerMockRecorder) ([]*persistentcache.Peer, []*persistentcache.Host) {
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(task.PersistentReplicaCount, nil).Times(1)
				return nil, nil
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, mockHosts []*persistentcache.Host, cachedParents []*persistentcache.Peer, hosts []*persistentcache.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Nil(cachedParents)
				assert.Nil(hosts)
			},
		},
		{
			name: "cached parents satisfy the needed replica count",
			mock: func(task *persistentcache.Task, blocklist set.SafeSet[string], mt *persistentcache.MockTaskManagerMockRecorder, mp *persistentcache.MockPeerManagerMockRecorder, mh *persistentcache.MockHostManagerMockRecorder) ([]*persistentcache.Peer, []*persistentcache.Host) {
				cachedParents := []*persistentcache.Peer{
					mockPersistentCachePeer(persistentcache.PeerStateSucceeded, false, task, mockPersistentCacheHost("cached1", mockHostIDC, false, 1000)),
					mockPersistentCachePeer(persistentcache.PeerStateSucceeded, false, task, mockPersistentCacheHost("cached2", mockHostIDC, false, 1000)),
					mockPersistentCachePeer(persistentcache.PeerStateSucceeded, false, task, mockPersistentCacheHost("cached3", mockHostIDC, false, 1000)),
				}
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(1), nil).Times(1)
				mp.LoadAllByTaskID(gomock.Any(), task.ID).Return(cachedParents, nil).Times(1)
				return cachedParents[:2], nil
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, mockHosts []*persistentcache.Host, cachedParents []*persistentcache.Peer, hosts []*persistentcache.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.True(found)
				assert.Equal(mockParents, cachedParents)
				assert.Nil(hosts)
				assert.Equal(uint(0), blocklist.Len())
			},
		},
		{
			name: "ineligible cached parents are filtered and replicate hosts are found",
			mock: func(task *persistentcache.Task, blocklist set.SafeSet[string], mt *persistentcache.MockTaskManagerMockRecorder, mp *persistentcache.MockPeerManagerMockRecorder, mh *persistentcache.MockHostManagerMockRecorder) ([]*persistentcache.Peer, []*persistentcache.Host) {
				blocklisted := mockPersistentCachePeer(persistentcache.PeerStateSucceeded, false, task, mockPersistentCacheHost("blocklisted", mockHostIDC, false, 1000))
				blocklist.Add(blocklisted.ID)
				persistentPeer := mockPersistentCachePeer(persistentcache.PeerStateSucceeded, true, task, mockPersistentCacheHost("persistent", mockHostIDC, false, 1000))
				parents := []*persistentcache.Peer{
					blocklisted,
					persistentPeer,
					mockPersistentCachePeer(persistentcache.PeerStateRunning, false, task, mockPersistentCacheHost("running", mockHostIDC, false, 1000)),
					mockPersistentCachePeer(persistentcache.PeerStateSucceeded, false, task, mockPersistentCacheHost("disableShared", mockHostIDC, true, 1000)),
				}
				hosts := []*persistentcache.Host{
					mockPersistentCacheHost("shared", mockHostIDC, false, 1000),
					mockPersistentCacheHost("hostDisableShared", mockHostIDC, true, 1000),
					mockPersistentCacheHost("smallDisk", mockHostIDC, false, 10),
				}
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(2), nil).Times(1)
				mp.LoadAllByTaskID(gomock.Any(), task.ID).Return(parents, nil).Times(1)
				mp.LoadPersistentAllByTaskID(gomock.Any(), task.ID).Return([]*persistentcache.Peer{persistentPeer}, nil).Times(1)
				mh.LoadRandom(gomock.Any(), 1, gomock.Any()).Return(hosts, nil).Times(1)
				return nil, hosts[:1]
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, mockHosts []*persistentcache.Host, cachedParents []*persistentcache.Peer, hosts []*persistentcache.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.True(found)
				assert.Nil(cachedParents)
				assert.Equal(mockHosts, hosts)
				assert.True(blocklist.Contains("persistent"))
			},
		},
		{
			name: "load current persistent peers failed",
			mock: func(task *persistentcache.Task, blocklist set.SafeSet[string], mt *persistentcache.MockTaskManagerMockRecorder, mp *persistentcache.MockPeerManagerMockRecorder, mh *persistentcache.MockHostManagerMockRecorder) ([]*persistentcache.Peer, []*persistentcache.Host) {
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(0), nil).Times(1)
				mp.LoadAllByTaskID(gomock.Any(), task.ID).Return(nil, errors.New("foo")).Times(1)
				mp.LoadPersistentAllByTaskID(gomock.Any(), task.ID).Return(nil, errors.New("foo")).Times(1)
				return nil, nil
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, mockHosts []*persistentcache.Host, cachedParents []*persistentcache.Peer, hosts []*persistentcache.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Nil(cachedParents)
				assert.Nil(hosts)
			},
		},
		{
			name: "cached parents are insufficient and are completed with replicate hosts",
			mock: func(task *persistentcache.Task, blocklist set.SafeSet[string], mt *persistentcache.MockTaskManagerMockRecorder, mp *persistentcache.MockPeerManagerMockRecorder, mh *persistentcache.MockHostManagerMockRecorder) ([]*persistentcache.Peer, []*persistentcache.Host) {
				cachedParent := mockPersistentCachePeer(persistentcache.PeerStateSucceeded, false, task, mockPersistentCacheHost("cached", mockHostIDC, false, 1000))
				persistentPeer := mockPersistentCachePeer(persistentcache.PeerStateSucceeded, true, task, mockPersistentCacheHost("persistent", mockHostIDC, false, 1000))
				hosts := []*persistentcache.Host{mockPersistentCacheHost("host1", mockHostIDC, false, 1000), mockPersistentCacheHost("host2", mockHostIDC, false, 1000)}
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(0), nil).Times(1)
				mp.LoadAllByTaskID(gomock.Any(), task.ID).Return([]*persistentcache.Peer{cachedParent, persistentPeer}, nil).Times(1)
				mp.LoadPersistentAllByTaskID(gomock.Any(), task.ID).Return([]*persistentcache.Peer{persistentPeer}, nil).Times(1)
				mh.LoadRandom(gomock.Any(), 2, gomock.Any()).Return(hosts, nil).Times(1)
				return []*persistentcache.Peer{cachedParent}, hosts
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, mockHosts []*persistentcache.Host, cachedParents []*persistentcache.Peer, hosts []*persistentcache.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.True(found)
				assert.Equal(mockParents, cachedParents)
				assert.Equal(mockHosts, hosts)
				assert.True(blocklist.Contains("cached"))
				assert.True(blocklist.Contains("persistent"))
			},
		},
		{
			name: "load replicate hosts failed",
			mock: func(task *persistentcache.Task, blocklist set.SafeSet[string], mt *persistentcache.MockTaskManagerMockRecorder, mp *persistentcache.MockPeerManagerMockRecorder, mh *persistentcache.MockHostManagerMockRecorder) ([]*persistentcache.Peer, []*persistentcache.Host) {
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(0), nil).Times(1)
				mp.LoadAllByTaskID(gomock.Any(), task.ID).Return(nil, nil).Times(1)
				mp.LoadPersistentAllByTaskID(gomock.Any(), task.ID).Return(nil, nil).Times(1)
				mh.LoadRandom(gomock.Any(), 3, gomock.Any()).Return(nil, errors.New("foo")).Times(1)
				return nil, nil
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, mockHosts []*persistentcache.Host, cachedParents []*persistentcache.Peer, hosts []*persistentcache.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Nil(cachedParents)
				assert.Nil(hosts)
			},
		},
		{
			name: "no cached parent and no replicate host",
			mock: func(task *persistentcache.Task, blocklist set.SafeSet[string], mt *persistentcache.MockTaskManagerMockRecorder, mp *persistentcache.MockPeerManagerMockRecorder, mh *persistentcache.MockHostManagerMockRecorder) ([]*persistentcache.Peer, []*persistentcache.Host) {
				mt.LoadCurrentPersistentReplicaCount(gomock.Any(), task.ID).Return(uint64(0), nil).Times(1)
				mp.LoadAllByTaskID(gomock.Any(), task.ID).Return(nil, nil).Times(1)
				mp.LoadPersistentAllByTaskID(gomock.Any(), task.ID).Return(nil, nil).Times(1)
				mh.LoadRandom(gomock.Any(), 3, gomock.Any()).Return(nil, nil).Times(1)
				return nil, nil
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, mockHosts []*persistentcache.Host, cachedParents []*persistentcache.Peer, hosts []*persistentcache.Host, blocklist set.SafeSet[string], found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Nil(cachedParents)
				assert.Nil(hosts)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			taskManager := persistentcache.NewMockTaskManager(ctl)
			peerManager := persistentcache.NewMockPeerManager(ctl)
			hostManager := persistentcache.NewMockHostManager(ctl)
			persistentCacheResource.EXPECT().TaskManager().Return(taskManager).AnyTimes()
			persistentCacheResource.EXPECT().PeerManager().Return(peerManager).AnyTimes()
			persistentCacheResource.EXPECT().HostManager().Return(hostManager).AnyTimes()

			task := mockPersistentCacheTask(3, 100)
			blocklist := set.NewSafeSet[string]()
			mockParents, mockHosts := tc.mock(task, blocklist, taskManager.EXPECT(), peerManager.EXPECT(), hostManager.EXPECT())
			scheduling := New(mockSchedulerConfig, persistentResource, persistentCacheResource, dynconfig, mockPluginDir)
			cachedParents, hosts, found := scheduling.FindReplicatePersistentCacheHosts(context.Background(), task, blocklist)
			tc.expect(t, mockParents, mockHosts, cachedParents, hosts, blocklist, found)
		})
	}
}

func TestScheduling_FindCandidatePersistentCacheParents(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(peer *persistentcache.Peer, blocklist set.SafeSet[string], mp *persistentcache.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistentcache.Peer
		expect func(t *testing.T, mockParents []*persistentcache.Peer, candidateParents []*persistentcache.Peer, found bool)
	}{
		{
			name: "load persistent cache parents failed",
			mock: func(peer *persistentcache.Peer, blocklist set.SafeSet[string], mp *persistentcache.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistentcache.Peer {
				mp.LoadAllByTaskID(gomock.Any(), peer.Task.ID).Return(nil, errors.New("foo")).Times(1)
				return nil
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, candidateParents []*persistentcache.Peer, found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Empty(candidateParents)
			},
		},
		{
			name: "parent is in blocklist",
			mock: func(peer *persistentcache.Peer, blocklist set.SafeSet[string], mp *persistentcache.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistentcache.Peer {
				parent := mockPersistentCachePeer(persistentcache.PeerStateSucceeded, true, peer.Task, mockPersistentCacheHost("parent", mockHostIDC, false, 1000))
				blocklist.Add(parent.ID)
				mp.LoadAllByTaskID(gomock.Any(), peer.Task.ID).Return([]*persistentcache.Peer{parent}, nil).Times(1)
				return nil
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, candidateParents []*persistentcache.Peer, found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Empty(candidateParents)
			},
		},
		{
			name: "parent shares the peer host",
			mock: func(peer *persistentcache.Peer, blocklist set.SafeSet[string], mp *persistentcache.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistentcache.Peer {
				parent := mockPersistentCachePeer(persistentcache.PeerStateSucceeded, true, peer.Task, peer.Host)
				mp.LoadAllByTaskID(gomock.Any(), peer.Task.ID).Return([]*persistentcache.Peer{parent}, nil).Times(1)
				return nil
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, candidateParents []*persistentcache.Peer, found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Empty(candidateParents)
			},
		},
		{
			name: "parent is bad node",
			mock: func(peer *persistentcache.Peer, blocklist set.SafeSet[string], mp *persistentcache.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistentcache.Peer {
				parent := mockPersistentCachePeer(persistentcache.PeerStateFailed, true, peer.Task, mockPersistentCacheHost("parent", mockHostIDC, false, 1000))
				mp.LoadAllByTaskID(gomock.Any(), peer.Task.ID).Return([]*persistentcache.Peer{parent}, nil).Times(1)
				return nil
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, candidateParents []*persistentcache.Peer, found bool) {
				assert := assert.New(t)
				assert.False(found)
				assert.Empty(candidateParents)
			},
		},
		{
			name: "candidate parents are sorted by affinity",
			mock: func(peer *persistentcache.Peer, blocklist set.SafeSet[string], mp *persistentcache.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistentcache.Peer {
				far := mockPersistentCachePeer(persistentcache.PeerStateSucceeded, true, peer.Task, mockPersistentCacheHost("far", "other", false, 1000))
				near := mockPersistentCachePeer(persistentcache.PeerStateSucceeded, true, peer.Task, mockPersistentCacheHost("near", mockHostIDC, false, 1000))
				mp.LoadAllByTaskID(gomock.Any(), peer.Task.ID).Return([]*persistentcache.Peer{far, near}, nil).Times(1)
				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{}, errors.New("foo")).Times(1)
				return []*persistentcache.Peer{near, far}
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, candidateParents []*persistentcache.Peer, found bool) {
				assert := assert.New(t)
				assert.True(found)
				assert.Equal(mockParents, candidateParents)
			},
		},
		{
			name: "candidate parents are trimmed to candidateParentLimit",
			mock: func(peer *persistentcache.Peer, blocklist set.SafeSet[string], mp *persistentcache.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) []*persistentcache.Peer {
				far := mockPersistentCachePeer(persistentcache.PeerStateSucceeded, true, peer.Task, mockPersistentCacheHost("far", "other", false, 1000))
				near := mockPersistentCachePeer(persistentcache.PeerStateSucceeded, true, peer.Task, mockPersistentCacheHost("near", mockHostIDC, false, 1000))
				mp.LoadAllByTaskID(gomock.Any(), peer.Task.ID).Return([]*persistentcache.Peer{far, near}, nil).Times(1)
				md.GetSchedulerClusterConfig().Return(types.SchedulerClusterConfig{CandidateParentLimit: 1}, nil).Times(1)
				return []*persistentcache.Peer{near}
			},
			expect: func(t *testing.T, mockParents []*persistentcache.Peer, candidateParents []*persistentcache.Peer, found bool) {
				assert := assert.New(t)
				assert.True(found)
				assert.Equal(mockParents, candidateParents)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			peerManager := persistentcache.NewMockPeerManager(ctl)
			persistentCacheResource.EXPECT().PeerManager().Return(peerManager).AnyTimes()

			task := mockPersistentCacheTask(3, 100)
			peer := mockPersistentCachePeer(persistentcache.PeerStateRunning, false, task, mockPersistentCacheHost("peer", mockHostIDC, false, 1000))
			blocklist := set.NewSafeSet[string]()
			mockParents := tc.mock(peer, blocklist, peerManager.EXPECT(), dynconfig.EXPECT())
			scheduling := New(mockSchedulerConfig, persistentResource, persistentCacheResource, dynconfig, mockPluginDir)
			candidateParents, found := scheduling.FindCandidatePersistentCacheParents(context.Background(), peer, blocklist)
			tc.expect(t, mockParents, candidateParents, found)
		})
	}
}
