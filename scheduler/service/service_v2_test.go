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

package service

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/bits-and-blooms/bitset"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv2 "d7y.io/api/v2/pkg/apis/common/v2"
	dfdaemonv2 "d7y.io/api/v2/pkg/apis/dfdaemon/v2"
	dfdaemonv2mocks "d7y.io/api/v2/pkg/apis/dfdaemon/v2/mocks"
	managerv2 "d7y.io/api/v2/pkg/apis/manager/v2"
	schedulerv2 "d7y.io/api/v2/pkg/apis/scheduler/v2"
	schedulerv2mocks "d7y.io/api/v2/pkg/apis/scheduler/v2/mocks"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	internaljob "d7y.io/dragonfly/v2/internal/job"
	internaljobmocks "d7y.io/dragonfly/v2/internal/job/mocks"
	managertypes "d7y.io/dragonfly/v2/manager/types"
	pkgatomic "d7y.io/dragonfly/v2/pkg/atomic"
	"d7y.io/dragonfly/v2/pkg/container/set"
	nethttp "d7y.io/dragonfly/v2/pkg/net/http"
	dfdaemonclientmocks "d7y.io/dragonfly/v2/pkg/rpc/dfdaemon/client/mocks"
	pkgtypes "d7y.io/dragonfly/v2/pkg/types"
	"d7y.io/dragonfly/v2/scheduler/config"
	configmocks "d7y.io/dragonfly/v2/scheduler/config/mocks"
	jobmocks "d7y.io/dragonfly/v2/scheduler/job/mocks"
	"d7y.io/dragonfly/v2/scheduler/resource/persistent"
	"d7y.io/dragonfly/v2/scheduler/resource/persistentcache"
	"d7y.io/dragonfly/v2/scheduler/resource/standard"
	schedulingmocks "d7y.io/dragonfly/v2/scheduler/scheduling/mocks"
)

var (
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

	mockInterval                    = durationpb.New(5 * time.Minute).AsDuration()
	mockConcurrentPieceCount uint32 = 5

	mockRawPersistentHost = persistent.Host{
		ID:                 mockHostID,
		Type:               pkgtypes.HostTypeNormal,
		Hostname:           "foo",
		IP:                 "127.0.0.1",
		Port:               8003,
		DownloadPort:       8001,
		ProxyPort:          8004,
		OS:                 "darwin",
		Platform:           "darwin",
		PlatformFamily:     "Standalone Workstation",
		PlatformVersion:    "11.1",
		KernelVersion:      "20.2.0",
		CPU:                mockPersistentCPU,
		Memory:             mockPersistentMemory,
		Network:            mockPersistentNetwork,
		Disk:               mockPersistentDisk,
		Build:              mockPersistentBuild,
		SchedulerClusterID: 1,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	mockPersistentCPU = persistent.CPU{
		LogicalCount:   4,
		PhysicalCount:  2,
		Percent:        1,
		ProcessPercent: 0.5,
		Times: persistent.CPUTimes{
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

	mockPersistentMemory = persistent.Memory{
		Total:              17179869184,
		Available:          5962813440,
		Used:               11217055744,
		UsedPercent:        65.291858,
		ProcessUsedPercent: 41.525125,
		Free:               2749598908,
	}

	mockPersistentNetwork = persistent.Network{
		TCPConnectionCount:       10,
		UploadTCPConnectionCount: 1,
		Location:                 mockHostLocation,
		IDC:                      mockHostIDC,
	}

	mockPersistentDisk = persistent.Disk{
		Total:             499963174912,
		Free:              37226479616,
		Used:              423809622016,
		UsedPercent:       91.92547406065952,
		InodesTotal:       4882452880,
		InodesUsed:        7835772,
		InodesFree:        4874617108,
		InodesUsedPercent: 0.1604884305611568,
	}

	mockPersistentBuild = persistent.Build{
		GitVersion: "v1.0.0",
		GitCommit:  "221176b117c6d59366d68f2b34d38be50c935883",
		GoVersion:  "1.18",
		Platform:   "darwin",
	}

	mockPersistentInterval = durationpb.New(5 * time.Minute).AsDuration()

	mockRawPersistentCacheHost = persistentcache.Host{
		ID:                 mockHostID,
		Type:               pkgtypes.HostTypeNormal,
		Hostname:           "foo",
		IP:                 "127.0.0.1",
		Port:               8003,
		DownloadPort:       8001,
		ProxyPort:          8004,
		OS:                 "darwin",
		Platform:           "darwin",
		PlatformFamily:     "Standalone Workstation",
		PlatformVersion:    "11.1",
		KernelVersion:      "20.2.0",
		CPU:                mockPersistentCacheCPU,
		Memory:             mockPersistentCacheMemory,
		Network:            mockPersistentCacheNetwork,
		Disk:               mockPersistentCacheDisk,
		Build:              mockPersistentCacheBuild,
		SchedulerClusterID: 1,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	mockPersistentCacheCPU = persistentcache.CPU{
		LogicalCount:   4,
		PhysicalCount:  2,
		Percent:        1,
		ProcessPercent: 0.5,
		Times: persistentcache.CPUTimes{
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

	mockPersistentCacheMemory = persistentcache.Memory{
		Total:              17179869184,
		Available:          5962813440,
		Used:               11217055744,
		UsedPercent:        65.291858,
		ProcessUsedPercent: 41.525125,
		Free:               2749598908,
	}

	mockPersistentCacheNetwork = persistentcache.Network{
		TCPConnectionCount:       10,
		UploadTCPConnectionCount: 1,
		Location:                 mockHostLocation,
		IDC:                      mockHostIDC,
	}

	mockPersistentCacheDisk = persistentcache.Disk{
		Total:             499963174912,
		Free:              37226479616,
		Used:              423809622016,
		UsedPercent:       91.92547406065952,
		InodesTotal:       4882452880,
		InodesUsed:        7835772,
		InodesFree:        4874617108,
		InodesUsedPercent: 0.1604884305611568,
	}

	mockPersistentCacheBuild = persistentcache.Build{
		GitVersion: "v1.0.0",
		GitCommit:  "221176b117c6d59366d68f2b34d38be50c935883",
		GoVersion:  "1.18",
		Platform:   "darwin",
	}

	mockPersistentCacheInterval = durationpb.New(5 * time.Minute).AsDuration()
)

func mockPersistentHost() *persistent.Host {
	return persistent.NewHost(
		mockRawPersistentHost.ID, mockRawPersistentHost.Name, mockRawPersistentHost.Hostname, mockRawPersistentHost.IP,
		mockRawPersistentHost.OS, mockRawPersistentHost.Platform, mockRawPersistentHost.PlatformFamily, mockRawPersistentHost.PlatformVersion, mockRawPersistentHost.KernelVersion,
		mockRawPersistentHost.Port, mockRawPersistentHost.DownloadPort, mockRawPersistentHost.ProxyPort, mockRawPersistentHost.SchedulerClusterID, mockRawPersistentHost.DisableShared, mockRawPersistentHost.Type,
		mockRawPersistentHost.CPU, mockRawPersistentHost.Memory, mockRawPersistentHost.Network, mockRawPersistentHost.Disk,
		mockRawPersistentHost.Build, mockRawPersistentHost.AnnounceInterval, mockRawPersistentHost.CreatedAt, mockRawPersistentHost.UpdatedAt, mockRawHost.Log)
}

func mockPersistentTask(state string) *persistent.Task {
	return persistent.NewTask(mockTaskID, mockTaskURL, "", "", state, 2, 1024, 2, time.Hour, time.Now(), time.Now(), logger.WithTaskID(mockTaskID))
}

func mockPersistentPeer(id, state string, task *persistent.Task, host *persistent.Host) *persistent.Peer {
	return persistent.NewPeer(id, state, true, bitset.New(2), []string{}, task, host, 0, time.Now().Add(-time.Minute), time.Now().Add(-time.Minute), logger.WithPeer(host.ID, task.ID, id))
}

func mockPersistentCacheHost() *persistentcache.Host {
	return persistentcache.NewHost(
		mockRawPersistentCacheHost.ID, mockRawPersistentCacheHost.Name, mockRawPersistentCacheHost.Hostname, mockRawPersistentCacheHost.IP,
		mockRawPersistentCacheHost.OS, mockRawPersistentCacheHost.Platform, mockRawPersistentCacheHost.PlatformFamily, mockRawPersistentCacheHost.PlatformVersion, mockRawPersistentCacheHost.KernelVersion,
		mockRawPersistentCacheHost.Port, mockRawPersistentCacheHost.DownloadPort, mockRawPersistentCacheHost.ProxyPort, mockRawPersistentCacheHost.SchedulerClusterID, mockRawPersistentCacheHost.DisableShared, mockRawPersistentCacheHost.Type,
		mockRawPersistentCacheHost.CPU, mockRawPersistentCacheHost.Memory, mockRawPersistentCacheHost.Network, mockRawPersistentCacheHost.Disk,
		mockRawPersistentCacheHost.Build, mockRawPersistentCacheHost.AnnounceInterval, mockRawPersistentCacheHost.CreatedAt, mockRawPersistentCacheHost.UpdatedAt, mockRawHost.Log)
}

func mockPersistentCacheTask(state string) *persistentcache.Task {
	return persistentcache.NewTask(mockTaskID, mockTaskTag, mockTaskApplication, state, 2, 512, 1024, 2, time.Hour, time.Now(), time.Now(), logger.WithTaskID(mockTaskID))
}

func mockPersistentCachePeer(id, state string, task *persistentcache.Task, host *persistentcache.Host) *persistentcache.Peer {
	return persistentcache.NewPeer(id, state, true, bitset.New(2), []string{}, task, host, 0, time.Now().Add(-time.Minute), time.Now().Add(-time.Minute), logger.WithPeer(host.ID, task.ID, id))
}

func TestService_NewV2(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, s any)
	}{
		{
			name: "new service",
			expect: func(t *testing.T, s any) {
				assert := assert.New(t)
				assert.Equal("V2", reflect.TypeOf(s).Elem().Name())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			tc.expect(t, NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig))
		})
	}
}

func TestServiceV2_AnnouncePeer(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder)
		expect func(t *testing.T, peer *standard.Peer, err error)
	}{
		{
			name: "context was done",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				stream.EXPECT().Context().Return(ctx).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, context.Canceled)
			},
		},
		{
			name: "receive error",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(nil, errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "receive EOF",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "register failed marks peer failed and releases announce stream",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.StoreAnnouncePeerStream(stream)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_RegisterPeerRequest{
							RegisterPeerRequest: &schedulerv2.RegisterPeerRequest{Download: &commonv2.Download{}},
						},
					}, nil).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(mockHostID)).Return(nil, false).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "host %s not found", mockHostID))
				assert.Equal(standard.PeerStateFailed, peer.FSM.Current())
				_, loaded := peer.LoadAnnouncePeerStream()
				assert.False(loaded)
			},
		},
		{
			name: "register failed and peer for failure handling not found",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_RegisterPeerRequest{
							RegisterPeerRequest: &schedulerv2.RegisterPeerRequest{Download: &commonv2.Download{}},
						},
					}, nil).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(mockHostID)).Return(nil, false).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "peer %s not found", mockPeerID))
			},
		},
		{
			name: "download started then stream closed",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(standard.PeerStateReceivedNormal)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_DownloadPeerStartedRequest{},
					}, nil).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(standard.PeerStateRunning, peer.FSM.Current())
			},
		},
		{
			name: "download back-to-source started failed and peer marked failed with task failed",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(standard.PeerStatePending)
				peer.Task.FSM.SetState(standard.TaskStateRunning)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_DownloadPeerBackToSourceStartedRequest{},
					}, nil).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "event DownloadBackToSource inappropriate in current state Pending"))
				assert.Equal(standard.PeerStateFailed, peer.FSM.Current())
				assert.Equal(standard.TaskStateFailed, peer.Task.FSM.Current())
			},
		},
		{
			name: "reschedule failed and peer marked failed",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_ReschedulePeerRequest{
							ReschedulePeerRequest: &schedulerv2.ReschedulePeerRequest{CandidateParents: []*commonv2.Peer{{Id: mockSeedPeerID}}},
						},
					}, nil).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					ms.ScheduleCandidateParents(gomock.Any(), gomock.Eq(peer), gomock.Cond(func(blocklist set.SafeSet[string]) bool { return blocklist.Contains(mockSeedPeerID) })).Return(errors.New("foo")).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "foo"))
				assert.Equal(standard.PeerStateFailed, peer.FSM.Current())
			},
		},
		{
			name: "download finished closes the stream",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_DownloadPeerFinishedRequest{},
					}, nil).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(standard.PeerStateSucceeded, peer.FSM.Current())
			},
		},
		{
			name: "download back-to-source finished failed and peer marked failed",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.FSM.SetState(standard.TaskStatePending)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_DownloadPeerBackToSourceFinishedRequest{},
					}, nil).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "event DownloadFailed inappropriate in current state Pending"))
				assert.Equal(standard.PeerStateFailed, peer.FSM.Current())
			},
		},
		{
			name: "download failed closes the stream",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(standard.PeerStateRunning)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_DownloadPeerFailedRequest{DownloadPeerFailedRequest: &schedulerv2.DownloadPeerFailedRequest{}},
					}, nil).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(standard.PeerStateFailed, peer.FSM.Current())
			},
		},
		{
			name: "download back-to-source failed closes the stream and resets task",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(standard.PeerStateBackToSource)
				peer.Task.FSM.SetState(standard.TaskStateRunning)
				peer.Task.ContentLength.Store(1)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_DownloadPeerBackToSourceFailedRequest{DownloadPeerBackToSourceFailedRequest: &schedulerv2.DownloadPeerBackToSourceFailedRequest{}},
					}, nil).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(standard.PeerStateFailed, peer.FSM.Current())
				assert.Equal(standard.TaskStateFailed, peer.Task.FSM.Current())
				assert.Equal(int64(-1), peer.Task.ContentLength.Load())
			},
		},
		{
			name: "piece events are handled and errors are only logged",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_DownloadPieceFinishedRequest{
							DownloadPieceFinishedRequest: &schedulerv2.DownloadPieceFinishedRequest{Piece: &commonv2.Piece{Number: 1, ParentId: &mockSeedPeerID}},
						},
					}, nil).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockSeedPeerID)).Return(nil, false).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_DownloadPieceBackToSourceFinishedRequest{
							DownloadPieceBackToSourceFinishedRequest: &schedulerv2.DownloadPieceBackToSourceFinishedRequest{Piece: &commonv2.Piece{Number: 2}},
						},
					}, nil).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_DownloadPieceFailedRequest{
							DownloadPieceFailedRequest: &schedulerv2.DownloadPieceFailedRequest{ParentId: mockSeedPeerID},
						},
					}, nil).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePeerRequest_DownloadPieceBackToSourceFailedRequest{
							DownloadPieceBackToSourceFailedRequest: &schedulerv2.DownloadPieceBackToSourceFailedRequest{},
						},
					}, nil).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(uint(2), peer.FinishedPieces.Count())
				assert.True(peer.FinishedPieces.Test(1))
				assert.True(peer.FinishedPieces.Test(2))
				assert.Len(peer.PieceCosts(), 2)
				assert.False(peer.BlockParents.Contains(mockSeedPeerID))
			},
		},
		{
			name: "unknown request",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, hostManager standard.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePeerRequest{HostId: mockHostID, TaskId: mockTaskID, PeerId: mockPeerID}, nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.Equal(codes.FailedPrecondition, status.Code(err))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			hostManager := standard.NewMockHostManager(ctl)
			stream := schedulerv2mocks.NewMockScheduler_AnnouncePeerServer(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost, standard.WithPriority(commonv2.Priority_LEVEL6))
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(peer, peerManager, hostManager, stream, resource.EXPECT(), peerManager.EXPECT(), hostManager.EXPECT(), scheduling.EXPECT())
			tc.expect(t, peer, svc.AnnouncePeer(stream))
		})
	}
}

func TestServiceV2_StatPeer(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder)
		expect func(t *testing.T, peer *standard.Peer, resp *commonv2.Peer, err error)
	}{
		{
			name: "peer not found",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Any()).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, resp *commonv2.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "peer %s not found", mockPeerID))
			},
		},
		{
			name: "peer has been loaded",
			mock: func(peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				peer.Task.StorePiece(&mockPiece)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Any()).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, resp *commonv2.Peer, err error) {
				assert := assert.New(t)
				dgst := peer.Task.Digest.String()

				assert.EqualValues(&commonv2.Peer{
					Id: peer.ID,
					Range: &commonv2.Range{
						Start:  uint64(peer.Range.Start),
						Length: uint64(peer.Range.Length),
					},
					Priority:             peer.Priority,
					ConcurrentPieceCount: mockConcurrentPieceCount,
					Cost:                 durationpb.New(peer.Cost.Load()),
					State:                peer.FSM.Current(),
					Task: &commonv2.Task{
						Id:                  peer.Task.ID,
						Type:                peer.Task.Type,
						Url:                 peer.Task.URL,
						Digest:              &dgst,
						Tag:                 &peer.Task.Tag,
						Application:         &peer.Task.Application,
						FilteredQueryParams: peer.Task.FilteredQueryParams,
						RequestHeader:       peer.Task.Header,
						ContentLength:       uint64(peer.Task.ContentLength.Load()),
						PieceCount:          uint32(peer.Task.TotalPieceCount.Load()),
						SizeScope:           peer.Task.SizeScope(),
						Pieces: []*commonv2.Piece{
							{
								Number:      uint32(mockPiece.Number),
								ParentId:    &mockPiece.ParentID,
								Offset:      mockPiece.Offset,
								Length:      mockPiece.Length,
								Digest:      mockPiece.Digest.String(),
								TrafficType: &mockPiece.TrafficType,
								Cost:        durationpb.New(mockPiece.Cost),
								CreatedAt:   timestamppb.New(mockPiece.CreatedAt),
							},
						},
						State:     peer.Task.FSM.Current(),
						PeerCount: uint32(peer.Task.PeerCount()),
						CreatedAt: timestamppb.New(peer.Task.CreatedAt.Load()),
						UpdatedAt: timestamppb.New(peer.Task.UpdatedAt.Load()),
					},
					Host: &commonv2.Host{
						Id:              peer.Host.ID,
						Type:            uint32(peer.Host.Type),
						Hostname:        peer.Host.Hostname,
						Ip:              peer.Host.IP,
						Port:            peer.Host.Port,
						DownloadPort:    peer.Host.DownloadPort,
						ProxyPort:       peer.Host.ProxyPort,
						Os:              peer.Host.OS,
						Platform:        peer.Host.Platform,
						PlatformFamily:  peer.Host.PlatformFamily,
						PlatformVersion: peer.Host.PlatformVersion,
						KernelVersion:   peer.Host.KernelVersion,
						Cpu: &commonv2.CPU{
							LogicalCount:   peer.Host.CPU.LogicalCount,
							PhysicalCount:  peer.Host.CPU.PhysicalCount,
							Percent:        peer.Host.CPU.Percent,
							ProcessPercent: peer.Host.CPU.ProcessPercent,
							Times: &commonv2.CPUTimes{
								User:      peer.Host.CPU.Times.User,
								System:    peer.Host.CPU.Times.System,
								Idle:      peer.Host.CPU.Times.Idle,
								Nice:      peer.Host.CPU.Times.Nice,
								Iowait:    peer.Host.CPU.Times.Iowait,
								Irq:       peer.Host.CPU.Times.Irq,
								Softirq:   peer.Host.CPU.Times.Softirq,
								Steal:     peer.Host.CPU.Times.Steal,
								Guest:     peer.Host.CPU.Times.Guest,
								GuestNice: peer.Host.CPU.Times.GuestNice,
							},
						},
						Memory: &commonv2.Memory{
							Total:              peer.Host.Memory.Total,
							Available:          peer.Host.Memory.Available,
							Used:               peer.Host.Memory.Used,
							UsedPercent:        peer.Host.Memory.UsedPercent,
							ProcessUsedPercent: peer.Host.Memory.ProcessUsedPercent,
							Free:               peer.Host.Memory.Free,
						},
						Network: &commonv2.Network{
							TcpConnectionCount:       peer.Host.Network.TCPConnectionCount,
							UploadTcpConnectionCount: peer.Host.Network.UploadTCPConnectionCount,
							Location:                 &peer.Host.Network.Location,
							Idc:                      &peer.Host.Network.IDC,
							RxBandwidth:              &peer.Host.Network.RxBandwidth,
							MaxRxBandwidth:           peer.Host.Network.MaxRxBandwidth,
							TxBandwidth:              &peer.Host.Network.TxBandwidth,
							MaxTxBandwidth:           peer.Host.Network.MaxTxBandwidth,
						},
						Disk: &commonv2.Disk{
							Total:             peer.Host.Disk.Total,
							Free:              peer.Host.Disk.Free,
							Used:              peer.Host.Disk.Used,
							UsedPercent:       peer.Host.Disk.UsedPercent,
							InodesTotal:       peer.Host.Disk.InodesTotal,
							InodesUsed:        peer.Host.Disk.InodesUsed,
							InodesFree:        peer.Host.Disk.InodesFree,
							InodesUsedPercent: peer.Host.Disk.InodesUsedPercent,
						},
						Build: &commonv2.Build{
							GitVersion: peer.Host.Build.GitVersion,
							GitCommit:  &peer.Host.Build.GitCommit,
							GoVersion:  &peer.Host.Build.GoVersion,
							Platform:   &peer.Host.Build.Platform,
						},
					},
					NeedBackToSource: peer.NeedBackToSource.Load(),
					CreatedAt:        timestamppb.New(peer.CreatedAt.Load()),
					UpdatedAt:        timestamppb.New(peer.UpdatedAt.Load()),
				}, resp)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			peerManager := standard.NewMockPeerManager(ctl)
			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockSeedPeerID, mockTask, mockHost, standard.WithRange(mockPeerRange), standard.WithConcurrentPieceCount(mockConcurrentPieceCount))
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig, Metrics: config.MetricsConfig{EnableHost: true}}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(peer, peerManager, resource.EXPECT(), peerManager.EXPECT())
			resp, err := svc.StatPeer(context.Background(), &schedulerv2.StatPeerRequest{TaskId: mockTaskID, PeerId: mockPeerID})
			tc.expect(t, peer, resp, err)
		})
	}
}

func TestServiceV2_StatTask(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(task *standard.Task, taskManager standard.TaskManager, mr *standard.MockResourceMockRecorder, mt *standard.MockTaskManagerMockRecorder)
		expect func(t *testing.T, task *standard.Task, resp *commonv2.Task, err error)
	}{
		{
			name: "task not found",
			mock: func(task *standard.Task, taskManager standard.TaskManager, mr *standard.MockResourceMockRecorder, mt *standard.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Any()).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, task *standard.Task, resp *commonv2.Task, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "task %s not found", mockTaskID))
			},
		},
		{
			name: "task has been loaded",
			mock: func(task *standard.Task, taskManager standard.TaskManager, mr *standard.MockResourceMockRecorder, mt *standard.MockTaskManagerMockRecorder) {
				task.StorePiece(&mockPiece)
				gomock.InOrder(
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Any()).Return(task, true).Times(1),
				)
			},
			expect: func(t *testing.T, task *standard.Task, resp *commonv2.Task, err error) {
				assert := assert.New(t)
				dgst := task.Digest.String()

				assert.EqualValues(&commonv2.Task{
					Id:                  task.ID,
					Type:                task.Type,
					Url:                 task.URL,
					Digest:              &dgst,
					Tag:                 &task.Tag,
					Application:         &task.Application,
					FilteredQueryParams: task.FilteredQueryParams,
					RequestHeader:       task.Header,
					ContentLength:       uint64(task.ContentLength.Load()),
					PieceCount:          uint32(task.TotalPieceCount.Load()),
					SizeScope:           task.SizeScope(),
					Pieces: []*commonv2.Piece{
						{
							Number:      uint32(mockPiece.Number),
							ParentId:    &mockPiece.ParentID,
							Offset:      mockPiece.Offset,
							Length:      mockPiece.Length,
							Digest:      mockPiece.Digest.String(),
							TrafficType: &mockPiece.TrafficType,
							Cost:        durationpb.New(mockPiece.Cost),
							CreatedAt:   timestamppb.New(mockPiece.CreatedAt),
						},
					},
					State:     task.FSM.Current(),
					PeerCount: uint32(task.PeerCount()),
					CreatedAt: timestamppb.New(task.CreatedAt.Load()),
					UpdatedAt: timestamppb.New(task.UpdatedAt.Load()),
				}, resp)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			taskManager := standard.NewMockTaskManager(ctl)
			task := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig, Metrics: config.MetricsConfig{EnableHost: true}}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(task, taskManager, resource.EXPECT(), taskManager.EXPECT())
			resp, err := svc.StatTask(context.Background(), &schedulerv2.StatTaskRequest{TaskId: mockTaskID})
			tc.expect(t, task, resp, err)
		})
	}
}

func mockAnnounceHostRequest(hostType pkgtypes.HostType) *schedulerv2.AnnounceHostRequest {
	return &schedulerv2.AnnounceHostRequest{
		Host: &commonv2.Host{
			Id:              mockHostID,
			Type:            uint32(hostType),
			Hostname:        "hostname",
			Ip:              "127.0.0.1",
			Port:            8003,
			DownloadPort:    8001,
			Os:              "darwin",
			Platform:        "darwin",
			PlatformFamily:  "Standalone Workstation",
			PlatformVersion: "11.1",
			KernelVersion:   "20.2.0",
			Cpu: &commonv2.CPU{
				LogicalCount:   mockCPU.LogicalCount,
				PhysicalCount:  mockCPU.PhysicalCount,
				Percent:        mockCPU.Percent,
				ProcessPercent: mockCPU.ProcessPercent,
				Times: &commonv2.CPUTimes{
					User:      mockCPU.Times.User,
					System:    mockCPU.Times.System,
					Idle:      mockCPU.Times.Idle,
					Nice:      mockCPU.Times.Nice,
					Iowait:    mockCPU.Times.Iowait,
					Irq:       mockCPU.Times.Irq,
					Softirq:   mockCPU.Times.Softirq,
					Steal:     mockCPU.Times.Steal,
					Guest:     mockCPU.Times.Guest,
					GuestNice: mockCPU.Times.GuestNice,
				},
			},
			Memory: &commonv2.Memory{
				Total:              mockMemory.Total,
				Available:          mockMemory.Available,
				Used:               mockMemory.Used,
				UsedPercent:        mockMemory.UsedPercent,
				ProcessUsedPercent: mockMemory.ProcessUsedPercent,
				Free:               mockMemory.Free,
			},
			Network: &commonv2.Network{
				TcpConnectionCount:       mockNetwork.TCPConnectionCount,
				UploadTcpConnectionCount: mockNetwork.UploadTCPConnectionCount,
				Location:                 &mockNetwork.Location,
				Idc:                      &mockNetwork.IDC,
				RxBandwidth:              &mockNetwork.RxBandwidth,
				MaxRxBandwidth:           mockNetwork.MaxRxBandwidth,
				TxBandwidth:              &mockNetwork.TxBandwidth,
				MaxTxBandwidth:           mockNetwork.MaxTxBandwidth,
			},
			Disk: &commonv2.Disk{
				Total:             mockDisk.Total,
				Free:              mockDisk.Free,
				Used:              mockDisk.Used,
				UsedPercent:       mockDisk.UsedPercent,
				InodesTotal:       mockDisk.InodesTotal,
				InodesUsed:        mockDisk.InodesUsed,
				InodesFree:        mockDisk.InodesFree,
				InodesUsedPercent: mockDisk.InodesUsedPercent,
			},
			Build: &commonv2.Build{
				GitVersion: mockBuild.GitVersion,
				GitCommit:  &mockBuild.GitCommit,
				GoVersion:  &mockBuild.GoVersion,
				Platform:   &mockBuild.Platform,
			},
		},
		Interval: durationpb.New(5 * time.Minute),
	}
}

func TestServiceV2_AnnounceHost(t *testing.T) {
	tests := []struct {
		name string
		req  *schedulerv2.AnnounceHostRequest
		run  func(t *testing.T, svc *V2, req *schedulerv2.AnnounceHostRequest, host *standard.Host, persistentHost *persistent.Host, persistentCacheHost *persistentcache.Host, hostManager standard.HostManager, persistentHostManager persistent.HostManager, persistentCacheHostManager persistentcache.HostManager, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpcr *persistentcache.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder)
	}{
		{
			name: "host not found",
			req: &schedulerv2.AnnounceHostRequest{
				Host: &commonv2.Host{
					Id:              mockHostID,
					Type:            uint32(pkgtypes.HostTypeNormal),
					Hostname:        "hostname",
					Ip:              "127.0.0.1",
					Port:            8003,
					DownloadPort:    8001,
					DisableShared:   true,
					Os:              "darwin",
					Platform:        "darwin",
					PlatformFamily:  "Standalone Workstation",
					PlatformVersion: "11.1",
					KernelVersion:   "20.2.0",
					Cpu: &commonv2.CPU{
						LogicalCount:   mockCPU.LogicalCount,
						PhysicalCount:  mockCPU.PhysicalCount,
						Percent:        mockCPU.Percent,
						ProcessPercent: mockCPU.ProcessPercent,
						Times: &commonv2.CPUTimes{
							User:      mockCPU.Times.User,
							System:    mockCPU.Times.System,
							Idle:      mockCPU.Times.Idle,
							Nice:      mockCPU.Times.Nice,
							Iowait:    mockCPU.Times.Iowait,
							Irq:       mockCPU.Times.Irq,
							Softirq:   mockCPU.Times.Softirq,
							Steal:     mockCPU.Times.Steal,
							Guest:     mockCPU.Times.Guest,
							GuestNice: mockCPU.Times.GuestNice,
						},
					},
					Memory: &commonv2.Memory{
						Total:              mockMemory.Total,
						Available:          mockMemory.Available,
						Used:               mockMemory.Used,
						UsedPercent:        mockMemory.UsedPercent,
						ProcessUsedPercent: mockMemory.ProcessUsedPercent,
						Free:               mockMemory.Free,
					},
					Network: &commonv2.Network{
						TcpConnectionCount:       mockNetwork.TCPConnectionCount,
						UploadTcpConnectionCount: mockNetwork.UploadTCPConnectionCount,
						Location:                 &mockNetwork.Location,
						Idc:                      &mockNetwork.IDC,
						RxBandwidth:              &mockNetwork.RxBandwidth,
						MaxRxBandwidth:           mockNetwork.MaxRxBandwidth,
						TxBandwidth:              &mockNetwork.TxBandwidth,
						MaxTxBandwidth:           mockNetwork.MaxTxBandwidth,
					},
					Disk: &commonv2.Disk{
						Total:             mockDisk.Total,
						Free:              mockDisk.Free,
						Used:              mockDisk.Used,
						UsedPercent:       mockDisk.UsedPercent,
						InodesTotal:       mockDisk.InodesTotal,
						InodesUsed:        mockDisk.InodesUsed,
						InodesFree:        mockDisk.InodesFree,
						InodesUsedPercent: mockDisk.InodesUsedPercent,
					},
					Build: &commonv2.Build{
						GitVersion: mockBuild.GitVersion,
						GitCommit:  &mockBuild.GitCommit,
						GoVersion:  &mockBuild.GoVersion,
						Platform:   &mockBuild.Platform,
					},
				},
				Interval: durationpb.New(5 * time.Minute),
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.AnnounceHostRequest, host *standard.Host, persistentHost *persistent.Host, persistentCacheHost *persistentcache.Host, hostManager standard.HostManager, persistentHostManager persistent.HostManager, persistentCacheHostManager persistentcache.HostManager, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpcr *persistentcache.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					md.GetSchedulerClusterClientConfig().Return(managertypes.SchedulerClusterClientConfig{LoadLimit: 10}, nil).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(nil, false).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Store(gomock.Any()).Do(func(host *standard.Host) {
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockCPU, host.CPU)
						assert.EqualValues(mockMemory, host.Memory)
						assert.EqualValues(mockNetwork, host.Network)
						assert.EqualValues(mockDisk, host.Disk)
						assert.EqualValues(mockBuild, host.Build)
						assert.EqualValues(mockInterval, host.AnnounceInterval)
						assert.Equal(int32(10), host.ConcurrentUploadLimit.Load())
						assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
						assert.Equal(int64(0), host.UploadCount.Load())
						assert.Equal(int64(0), host.UploadFailedCount.Load())
						assert.NotNil(host.Peers)
						assert.Equal(int32(0), host.PeerCount.Load())
						assert.NotEqual(0, host.CreatedAt.Load().Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Load().Nanosecond())
						assert.NotNil(host.Log)
					}).Return().Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Any()).Return(nil, false).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Store(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, host *persistent.Host) {
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockPersistentCPU, host.CPU)
						assert.EqualValues(mockPersistentMemory, host.Memory)
						assert.EqualValues(mockPersistentNetwork, host.Network)
						assert.EqualValues(mockPersistentDisk, host.Disk)
						assert.EqualValues(mockPersistentBuild, host.Build)
						assert.EqualValues(mockPersistentInterval, host.AnnounceInterval)
						assert.NotEqual(0, host.CreatedAt.Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Nanosecond())
						assert.NotNil(host.Log)
					}).Return(nil).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Load(gomock.Any(), gomock.Any()).Return(nil, false).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Store(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, host *persistentcache.Host) {
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockPersistentCacheCPU, host.CPU)
						assert.EqualValues(mockPersistentCacheMemory, host.Memory)
						assert.EqualValues(mockPersistentCacheNetwork, host.Network)
						assert.EqualValues(mockPersistentCacheDisk, host.Disk)
						assert.EqualValues(mockPersistentCacheBuild, host.Build)
						assert.EqualValues(mockPersistentCacheInterval, host.AnnounceInterval)
						assert.NotEqual(0, host.CreatedAt.Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Nanosecond())
						assert.NotNil(host.Log)
					}).Return(nil).Times(1),
				)

				assert.NoError(svc.AnnounceHost(context.Background(), req))
			},
		},
		{
			name: "host not found, dynconfig returns error",
			req: &schedulerv2.AnnounceHostRequest{
				Host: &commonv2.Host{
					Id:              mockHostID,
					Type:            uint32(pkgtypes.HostTypeNormal),
					Hostname:        "hostname",
					Ip:              "127.0.0.1",
					Port:            8003,
					DownloadPort:    8001,
					DisableShared:   false,
					Os:              "darwin",
					Platform:        "darwin",
					PlatformFamily:  "Standalone Workstation",
					PlatformVersion: "11.1",
					KernelVersion:   "20.2.0",
					Cpu: &commonv2.CPU{
						LogicalCount:   mockCPU.LogicalCount,
						PhysicalCount:  mockCPU.PhysicalCount,
						Percent:        mockCPU.Percent,
						ProcessPercent: mockCPU.ProcessPercent,
						Times: &commonv2.CPUTimes{
							User:      mockCPU.Times.User,
							System:    mockCPU.Times.System,
							Idle:      mockCPU.Times.Idle,
							Nice:      mockCPU.Times.Nice,
							Iowait:    mockCPU.Times.Iowait,
							Irq:       mockCPU.Times.Irq,
							Softirq:   mockCPU.Times.Softirq,
							Steal:     mockCPU.Times.Steal,
							Guest:     mockCPU.Times.Guest,
							GuestNice: mockCPU.Times.GuestNice,
						},
					},
					Memory: &commonv2.Memory{
						Total:              mockMemory.Total,
						Available:          mockMemory.Available,
						Used:               mockMemory.Used,
						UsedPercent:        mockMemory.UsedPercent,
						ProcessUsedPercent: mockMemory.ProcessUsedPercent,
						Free:               mockMemory.Free,
					},
					Network: &commonv2.Network{
						TcpConnectionCount:       mockNetwork.TCPConnectionCount,
						UploadTcpConnectionCount: mockNetwork.UploadTCPConnectionCount,
						Location:                 &mockNetwork.Location,
						Idc:                      &mockNetwork.IDC,
						RxBandwidth:              &mockNetwork.RxBandwidth,
						MaxRxBandwidth:           mockNetwork.MaxRxBandwidth,
						TxBandwidth:              &mockNetwork.TxBandwidth,
						MaxTxBandwidth:           mockNetwork.MaxTxBandwidth,
					},
					Disk: &commonv2.Disk{
						Total:             mockDisk.Total,
						Free:              mockDisk.Free,
						Used:              mockDisk.Used,
						UsedPercent:       mockDisk.UsedPercent,
						InodesTotal:       mockDisk.InodesTotal,
						InodesUsed:        mockDisk.InodesUsed,
						InodesFree:        mockDisk.InodesFree,
						InodesUsedPercent: mockDisk.InodesUsedPercent,
					},
					Build: &commonv2.Build{
						GitVersion: mockBuild.GitVersion,
						GitCommit:  &mockBuild.GitCommit,
						GoVersion:  &mockBuild.GoVersion,
						Platform:   &mockBuild.Platform,
					},
				},
				Interval: durationpb.New(5 * time.Minute),
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.AnnounceHostRequest, host *standard.Host, persistentHost *persistent.Host, persistentCacheHost *persistentcache.Host, hostManager standard.HostManager, persistentHostManager persistent.HostManager, persistentCacheHostManager persistentcache.HostManager, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpcr *persistentcache.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					md.GetSchedulerClusterClientConfig().Return(managertypes.SchedulerClusterClientConfig{}, errors.New("foo")).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(nil, false).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Store(gomock.Any()).Do(func(host *standard.Host) {
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockCPU, host.CPU)
						assert.EqualValues(mockMemory, host.Memory)
						assert.EqualValues(mockNetwork, host.Network)
						assert.EqualValues(mockDisk, host.Disk)
						assert.EqualValues(mockBuild, host.Build)
						assert.EqualValues(mockInterval, host.AnnounceInterval)
						assert.Equal(int32(200), host.ConcurrentUploadLimit.Load())
						assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
						assert.Equal(int64(0), host.UploadCount.Load())
						assert.Equal(int64(0), host.UploadFailedCount.Load())
						assert.NotNil(host.Peers)
						assert.Equal(int32(0), host.PeerCount.Load())
						assert.NotEqual(0, host.CreatedAt.Load().Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Load().Nanosecond())
						assert.NotNil(host.Log)
					}).Return().Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Any()).Return(nil, false).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Store(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, host *persistent.Host) {
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockPersistentCPU, host.CPU)
						assert.EqualValues(mockPersistentMemory, host.Memory)
						assert.EqualValues(mockPersistentNetwork, host.Network)
						assert.EqualValues(mockPersistentDisk, host.Disk)
						assert.EqualValues(mockPersistentBuild, host.Build)
						assert.EqualValues(mockPersistentInterval, host.AnnounceInterval)
						assert.NotEqual(0, host.CreatedAt.Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Nanosecond())
						assert.NotNil(host.Log)
					}).Return(nil).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Load(gomock.Any(), gomock.Any()).Return(nil, false).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Store(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, host *persistentcache.Host) {
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockPersistentCacheCPU, host.CPU)
						assert.EqualValues(mockPersistentCacheMemory, host.Memory)
						assert.EqualValues(mockPersistentCacheNetwork, host.Network)
						assert.EqualValues(mockPersistentCacheDisk, host.Disk)
						assert.EqualValues(mockPersistentCacheBuild, host.Build)
						assert.EqualValues(mockPersistentCacheInterval, host.AnnounceInterval)
						assert.NotEqual(0, host.CreatedAt.Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Nanosecond())
						assert.NotNil(host.Log)
					}).Return(nil).Times(1),
				)

				assert.NoError(svc.AnnounceHost(context.Background(), req))
			},
		},
		{
			name: "host already exists",
			req: &schedulerv2.AnnounceHostRequest{
				Host: &commonv2.Host{
					Id:              mockHostID,
					Type:            uint32(pkgtypes.HostTypeNormal),
					Hostname:        "foo",
					Ip:              "127.0.0.1",
					Port:            8003,
					DownloadPort:    8001,
					DisableShared:   true,
					Os:              "darwin",
					Platform:        "darwin",
					PlatformFamily:  "Standalone Workstation",
					PlatformVersion: "11.1",
					KernelVersion:   "20.2.0",
					Cpu: &commonv2.CPU{
						LogicalCount:   mockCPU.LogicalCount,
						PhysicalCount:  mockCPU.PhysicalCount,
						Percent:        mockCPU.Percent,
						ProcessPercent: mockCPU.ProcessPercent,
						Times: &commonv2.CPUTimes{
							User:      mockCPU.Times.User,
							System:    mockCPU.Times.System,
							Idle:      mockCPU.Times.Idle,
							Nice:      mockCPU.Times.Nice,
							Iowait:    mockCPU.Times.Iowait,
							Irq:       mockCPU.Times.Irq,
							Softirq:   mockCPU.Times.Softirq,
							Steal:     mockCPU.Times.Steal,
							Guest:     mockCPU.Times.Guest,
							GuestNice: mockCPU.Times.GuestNice,
						},
					},
					Memory: &commonv2.Memory{
						Total:              mockMemory.Total,
						Available:          mockMemory.Available,
						Used:               mockMemory.Used,
						UsedPercent:        mockMemory.UsedPercent,
						ProcessUsedPercent: mockMemory.ProcessUsedPercent,
						Free:               mockMemory.Free,
					},
					Network: &commonv2.Network{
						TcpConnectionCount:       mockNetwork.TCPConnectionCount,
						UploadTcpConnectionCount: mockNetwork.UploadTCPConnectionCount,
						Location:                 &mockNetwork.Location,
						Idc:                      &mockNetwork.IDC,
						RxBandwidth:              &mockNetwork.RxBandwidth,
						MaxRxBandwidth:           mockNetwork.MaxRxBandwidth,
						TxBandwidth:              &mockNetwork.TxBandwidth,
						MaxTxBandwidth:           mockNetwork.MaxTxBandwidth,
					},
					Disk: &commonv2.Disk{
						Total:             mockDisk.Total,
						Free:              mockDisk.Free,
						Used:              mockDisk.Used,
						UsedPercent:       mockDisk.UsedPercent,
						InodesTotal:       mockDisk.InodesTotal,
						InodesUsed:        mockDisk.InodesUsed,
						InodesFree:        mockDisk.InodesFree,
						InodesUsedPercent: mockDisk.InodesUsedPercent,
					},
					Build: &commonv2.Build{
						GitVersion: mockBuild.GitVersion,
						GitCommit:  &mockBuild.GitCommit,
						GoVersion:  &mockBuild.GoVersion,
						Platform:   &mockBuild.Platform,
					},
				},
				Interval: durationpb.New(5 * time.Minute),
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.AnnounceHostRequest, host *standard.Host, persistentHost *persistent.Host, persistentCacheHost *persistentcache.Host, hostManager standard.HostManager, persistentHostManager persistent.HostManager, persistentCacheHostManager persistentcache.HostManager, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpcr *persistentcache.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					md.GetSchedulerClusterClientConfig().Return(managertypes.SchedulerClusterClientConfig{LoadLimit: 10}, nil).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(host, true).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Any()).Return(persistentHost, true).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Store(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, host *persistent.Host) {
						assert := assert.New(t)
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockPersistentCPU, host.CPU)
						assert.EqualValues(mockPersistentMemory, host.Memory)
						assert.EqualValues(mockPersistentNetwork, host.Network)
						assert.EqualValues(mockPersistentDisk, host.Disk)
						assert.EqualValues(mockPersistentBuild, host.Build)
						assert.EqualValues(mockPersistentInterval, host.AnnounceInterval)
						assert.NotEqual(0, host.CreatedAt.Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Nanosecond())
						assert.NotNil(host.Log)
					}).Return(nil).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Load(gomock.Any(), gomock.Any()).Return(persistentCacheHost, true).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Store(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, host *persistentcache.Host) {
						assert := assert.New(t)
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockPersistentCacheCPU, host.CPU)
						assert.EqualValues(mockPersistentCacheMemory, host.Memory)
						assert.EqualValues(mockPersistentCacheNetwork, host.Network)
						assert.EqualValues(mockPersistentCacheDisk, host.Disk)
						assert.EqualValues(mockPersistentCacheBuild, host.Build)
						assert.EqualValues(mockPersistentCacheInterval, host.AnnounceInterval)
						assert.NotEqual(0, host.CreatedAt.Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Nanosecond())
						assert.NotNil(host.Log)
					}).Return(nil).Times(1),
				)

				assert := assert.New(t)
				assert.NoError(svc.AnnounceHost(context.Background(), req))
				assert.Equal(req.Host.Id, host.ID)
				assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
				assert.Equal(req.Host.Hostname, host.Hostname)
				assert.Equal(req.Host.Ip, host.IP)
				assert.Equal(req.Host.Port, host.Port)
				assert.Equal(req.Host.DownloadPort, host.DownloadPort)
				assert.Equal(req.Host.DisableShared, host.DisableShared)
				assert.Equal(req.Host.Os, host.OS)
				assert.Equal(req.Host.Platform, host.Platform)
				assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
				assert.Equal(req.Host.KernelVersion, host.KernelVersion)
				assert.EqualValues(mockCPU, host.CPU)
				assert.EqualValues(mockMemory, host.Memory)
				assert.EqualValues(mockNetwork, host.Network)
				assert.EqualValues(mockDisk, host.Disk)
				assert.EqualValues(mockBuild, host.Build)
				assert.EqualValues(mockInterval, host.AnnounceInterval)
				assert.Equal(int32(10), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEqual(0, host.CreatedAt.Load().Nanosecond())
				assert.NotEqual(0, host.UpdatedAt.Load().Nanosecond())
				assert.NotNil(host.Log)
			},
		},
		{
			name: "host already exists, dynconfig returns error",
			req: &schedulerv2.AnnounceHostRequest{
				Host: &commonv2.Host{
					Id:              mockHostID,
					Type:            uint32(pkgtypes.HostTypeNormal),
					Hostname:        "foo",
					Ip:              "127.0.0.1",
					Port:            8003,
					DownloadPort:    8001,
					DisableShared:   false,
					Os:              "darwin",
					Platform:        "darwin",
					PlatformFamily:  "Standalone Workstation",
					PlatformVersion: "11.1",
					KernelVersion:   "20.2.0",
					Cpu: &commonv2.CPU{
						LogicalCount:   mockCPU.LogicalCount,
						PhysicalCount:  mockCPU.PhysicalCount,
						Percent:        mockCPU.Percent,
						ProcessPercent: mockCPU.ProcessPercent,
						Times: &commonv2.CPUTimes{
							User:      mockCPU.Times.User,
							System:    mockCPU.Times.System,
							Idle:      mockCPU.Times.Idle,
							Nice:      mockCPU.Times.Nice,
							Iowait:    mockCPU.Times.Iowait,
							Irq:       mockCPU.Times.Irq,
							Softirq:   mockCPU.Times.Softirq,
							Steal:     mockCPU.Times.Steal,
							Guest:     mockCPU.Times.Guest,
							GuestNice: mockCPU.Times.GuestNice,
						},
					},
					Memory: &commonv2.Memory{
						Total:              mockMemory.Total,
						Available:          mockMemory.Available,
						Used:               mockMemory.Used,
						UsedPercent:        mockMemory.UsedPercent,
						ProcessUsedPercent: mockMemory.ProcessUsedPercent,
						Free:               mockMemory.Free,
					},
					Network: &commonv2.Network{
						TcpConnectionCount:       mockNetwork.TCPConnectionCount,
						UploadTcpConnectionCount: mockNetwork.UploadTCPConnectionCount,
						Location:                 &mockNetwork.Location,
						Idc:                      &mockNetwork.IDC,
						RxBandwidth:              &mockNetwork.RxBandwidth,
						MaxRxBandwidth:           mockNetwork.MaxRxBandwidth,
						TxBandwidth:              &mockNetwork.TxBandwidth,
						MaxTxBandwidth:           mockNetwork.MaxTxBandwidth,
					},
					Disk: &commonv2.Disk{
						Total:             mockDisk.Total,
						Free:              mockDisk.Free,
						Used:              mockDisk.Used,
						UsedPercent:       mockDisk.UsedPercent,
						InodesTotal:       mockDisk.InodesTotal,
						InodesUsed:        mockDisk.InodesUsed,
						InodesFree:        mockDisk.InodesFree,
						InodesUsedPercent: mockDisk.InodesUsedPercent,
					},
					Build: &commonv2.Build{
						GitVersion: mockBuild.GitVersion,
						GitCommit:  &mockBuild.GitCommit,
						GoVersion:  &mockBuild.GoVersion,
						Platform:   &mockBuild.Platform,
					},
				},
				Interval: durationpb.New(5 * time.Minute),
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.AnnounceHostRequest, host *standard.Host, persistentHost *persistent.Host, persistentCacheHost *persistentcache.Host, hostManager standard.HostManager, persistentHostManager persistent.HostManager, persistentCacheHostManager persistentcache.HostManager, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpcr *persistentcache.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					md.GetSchedulerClusterClientConfig().Return(managertypes.SchedulerClusterClientConfig{}, errors.New("foo")).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(host, true).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Any()).Return(persistentHost, true).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Store(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, host *persistent.Host) {
						assert := assert.New(t)
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockPersistentCPU, host.CPU)
						assert.EqualValues(mockPersistentMemory, host.Memory)
						assert.EqualValues(mockPersistentNetwork, host.Network)
						assert.EqualValues(mockPersistentDisk, host.Disk)
						assert.EqualValues(mockPersistentBuild, host.Build)
						assert.EqualValues(mockPersistentInterval, host.AnnounceInterval)
						assert.NotEqual(0, host.CreatedAt.Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Nanosecond())
						assert.NotNil(host.Log)
					}).Return(nil).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Load(gomock.Any(), gomock.Any()).Return(persistentCacheHost, true).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Store(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, host *persistentcache.Host) {
						assert := assert.New(t)
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockPersistentCacheCPU, host.CPU)
						assert.EqualValues(mockPersistentCacheMemory, host.Memory)
						assert.EqualValues(mockPersistentCacheNetwork, host.Network)
						assert.EqualValues(mockPersistentCacheDisk, host.Disk)
						assert.EqualValues(mockPersistentCacheBuild, host.Build)
						assert.EqualValues(mockPersistentCacheInterval, host.AnnounceInterval)
						assert.NotEqual(0, host.CreatedAt.Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Nanosecond())
						assert.NotNil(host.Log)
					}).Return(nil).Times(1),
				)

				assert := assert.New(t)
				assert.NoError(svc.AnnounceHost(context.Background(), req))
				assert.Equal(req.Host.Id, host.ID)
				assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
				assert.Equal(req.Host.Hostname, host.Hostname)
				assert.Equal(req.Host.Ip, host.IP)
				assert.Equal(req.Host.Port, host.Port)
				assert.Equal(req.Host.DownloadPort, host.DownloadPort)
				assert.Equal(req.Host.DisableShared, host.DisableShared)
				assert.Equal(req.Host.Os, host.OS)
				assert.Equal(req.Host.Platform, host.Platform)
				assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
				assert.Equal(req.Host.KernelVersion, host.KernelVersion)
				assert.EqualValues(mockCPU, host.CPU)
				assert.EqualValues(mockMemory, host.Memory)
				assert.EqualValues(mockNetwork, host.Network)
				assert.EqualValues(mockDisk, host.Disk)
				assert.EqualValues(mockBuild, host.Build)
				assert.EqualValues(mockInterval, host.AnnounceInterval)
				assert.Equal(int32(200), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEqual(0, host.CreatedAt.Load().Nanosecond())
				assert.NotEqual(0, host.UpdatedAt.Load().Nanosecond())
				assert.NotNil(host.Log)
			},
		},
		{
			name: "host not found, store persistent host failed",
			req: &schedulerv2.AnnounceHostRequest{
				Host: &commonv2.Host{
					Id:              mockHostID,
					Type:            uint32(pkgtypes.HostTypeNormal),
					Hostname:        "hostname",
					Ip:              "127.0.0.1",
					Port:            8003,
					DownloadPort:    8001,
					DisableShared:   true,
					Os:              "darwin",
					Platform:        "darwin",
					PlatformFamily:  "Standalone Workstation",
					PlatformVersion: "11.1",
					KernelVersion:   "20.2.0",
					Cpu: &commonv2.CPU{
						LogicalCount:   mockCPU.LogicalCount,
						PhysicalCount:  mockCPU.PhysicalCount,
						Percent:        mockCPU.Percent,
						ProcessPercent: mockCPU.ProcessPercent,
						Times: &commonv2.CPUTimes{
							User:      mockCPU.Times.User,
							System:    mockCPU.Times.System,
							Idle:      mockCPU.Times.Idle,
							Nice:      mockCPU.Times.Nice,
							Iowait:    mockCPU.Times.Iowait,
							Irq:       mockCPU.Times.Irq,
							Softirq:   mockCPU.Times.Softirq,
							Steal:     mockCPU.Times.Steal,
							Guest:     mockCPU.Times.Guest,
							GuestNice: mockCPU.Times.GuestNice,
						},
					},
					Memory: &commonv2.Memory{
						Total:              mockMemory.Total,
						Available:          mockMemory.Available,
						Used:               mockMemory.Used,
						UsedPercent:        mockMemory.UsedPercent,
						ProcessUsedPercent: mockMemory.ProcessUsedPercent,
						Free:               mockMemory.Free,
					},
					Network: &commonv2.Network{
						TcpConnectionCount:       mockNetwork.TCPConnectionCount,
						UploadTcpConnectionCount: mockNetwork.UploadTCPConnectionCount,
						Location:                 &mockNetwork.Location,
						Idc:                      &mockNetwork.IDC,
						RxBandwidth:              &mockNetwork.RxBandwidth,
						MaxRxBandwidth:           mockNetwork.MaxRxBandwidth,
						TxBandwidth:              &mockNetwork.TxBandwidth,
						MaxTxBandwidth:           mockNetwork.MaxTxBandwidth,
					},
					Disk: &commonv2.Disk{
						Total:             mockDisk.Total,
						Free:              mockDisk.Free,
						Used:              mockDisk.Used,
						UsedPercent:       mockDisk.UsedPercent,
						InodesTotal:       mockDisk.InodesTotal,
						InodesUsed:        mockDisk.InodesUsed,
						InodesFree:        mockDisk.InodesFree,
						InodesUsedPercent: mockDisk.InodesUsedPercent,
					},
					Build: &commonv2.Build{
						GitVersion: mockBuild.GitVersion,
						GitCommit:  &mockBuild.GitCommit,
						GoVersion:  &mockBuild.GoVersion,
						Platform:   &mockBuild.Platform,
					},
				},
				Interval: durationpb.New(5 * time.Minute),
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.AnnounceHostRequest, host *standard.Host, persistentHost *persistent.Host, persistentCacheHost *persistentcache.Host, hostManager standard.HostManager, persistentHostManager persistent.HostManager, persistentCacheHostManager persistentcache.HostManager, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpcr *persistentcache.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					md.GetSchedulerClusterClientConfig().Return(managertypes.SchedulerClusterClientConfig{LoadLimit: 10}, nil).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(nil, false).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Store(gomock.Any()).Do(func(host *standard.Host) {
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockCPU, host.CPU)
						assert.EqualValues(mockMemory, host.Memory)
						assert.EqualValues(mockNetwork, host.Network)
						assert.EqualValues(mockDisk, host.Disk)
						assert.EqualValues(mockBuild, host.Build)
						assert.EqualValues(mockInterval, host.AnnounceInterval)
						assert.Equal(int32(10), host.ConcurrentUploadLimit.Load())
						assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
						assert.Equal(int64(0), host.UploadCount.Load())
						assert.Equal(int64(0), host.UploadFailedCount.Load())
						assert.NotNil(host.Peers)
						assert.Equal(int32(0), host.PeerCount.Load())
						assert.NotEqual(0, host.CreatedAt.Load().Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Load().Nanosecond())
						assert.NotNil(host.Log)
					}).Return().Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Any()).Return(nil, false).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Store(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, host *persistent.Host) {
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockPersistentCPU, host.CPU)
						assert.EqualValues(mockPersistentMemory, host.Memory)
						assert.EqualValues(mockPersistentNetwork, host.Network)
						assert.EqualValues(mockPersistentDisk, host.Disk)
						assert.EqualValues(mockPersistentBuild, host.Build)
						assert.EqualValues(mockPersistentInterval, host.AnnounceInterval)
						assert.NotEqual(0, host.CreatedAt.Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Nanosecond())
						assert.NotNil(host.Log)
					}).Return(errors.New("bar")).Times(1),
				)

				assert.Error(svc.AnnounceHost(context.Background(), req))
			},
		},
		{
			name: "host already exists, store persistent cache host failed",
			req: &schedulerv2.AnnounceHostRequest{
				Host: &commonv2.Host{
					Id:              mockHostID,
					Type:            uint32(pkgtypes.HostTypeNormal),
					Hostname:        "foo",
					Ip:              "127.0.0.1",
					Port:            8003,
					DownloadPort:    8001,
					ProxyPort:       8004,
					DisableShared:   false,
					Os:              "darwin",
					Platform:        "darwin",
					PlatformFamily:  "Standalone Workstation",
					PlatformVersion: "11.1",
					KernelVersion:   "20.2.0",
					Cpu: &commonv2.CPU{
						LogicalCount:   mockCPU.LogicalCount,
						PhysicalCount:  mockCPU.PhysicalCount,
						Percent:        mockCPU.Percent,
						ProcessPercent: mockCPU.ProcessPercent,
						Times: &commonv2.CPUTimes{
							User:      mockCPU.Times.User,
							System:    mockCPU.Times.System,
							Idle:      mockCPU.Times.Idle,
							Nice:      mockCPU.Times.Nice,
							Iowait:    mockCPU.Times.Iowait,
							Irq:       mockCPU.Times.Irq,
							Softirq:   mockCPU.Times.Softirq,
							Steal:     mockCPU.Times.Steal,
							Guest:     mockCPU.Times.Guest,
							GuestNice: mockCPU.Times.GuestNice,
						},
					},
					Memory: &commonv2.Memory{
						Total:              mockMemory.Total,
						Available:          mockMemory.Available,
						Used:               mockMemory.Used,
						UsedPercent:        mockMemory.UsedPercent,
						ProcessUsedPercent: mockMemory.ProcessUsedPercent,
						Free:               mockMemory.Free,
					},
					Network: &commonv2.Network{
						TcpConnectionCount:       mockNetwork.TCPConnectionCount,
						UploadTcpConnectionCount: mockNetwork.UploadTCPConnectionCount,
						Location:                 &mockNetwork.Location,
						Idc:                      &mockNetwork.IDC,
						RxBandwidth:              &mockNetwork.RxBandwidth,
						MaxRxBandwidth:           mockNetwork.MaxRxBandwidth,
						TxBandwidth:              &mockNetwork.TxBandwidth,
						MaxTxBandwidth:           mockNetwork.MaxTxBandwidth,
					},
					Disk: &commonv2.Disk{
						Total:             mockDisk.Total,
						Free:              mockDisk.Free,
						Used:              mockDisk.Used,
						UsedPercent:       mockDisk.UsedPercent,
						InodesTotal:       mockDisk.InodesTotal,
						InodesUsed:        mockDisk.InodesUsed,
						InodesFree:        mockDisk.InodesFree,
						InodesUsedPercent: mockDisk.InodesUsedPercent,
					},
					Build: &commonv2.Build{
						GitVersion: mockBuild.GitVersion,
						GitCommit:  &mockBuild.GitCommit,
						GoVersion:  &mockBuild.GoVersion,
						Platform:   &mockBuild.Platform,
					},
				},
				Interval: durationpb.New(5 * time.Minute),
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.AnnounceHostRequest, host *standard.Host, persistentHost *persistent.Host, persistentCacheHost *persistentcache.Host, hostManager standard.HostManager, persistentHostManager persistent.HostManager, persistentCacheHostManager persistentcache.HostManager, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpcr *persistentcache.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					md.GetSchedulerClusterClientConfig().Return(managertypes.SchedulerClusterClientConfig{}, errors.New("foo")).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(host, true).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Any()).Return(persistentHost, true).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Store(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, host *persistent.Host) {
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockPersistentCPU, host.CPU)
						assert.EqualValues(mockPersistentMemory, host.Memory)
						assert.EqualValues(mockPersistentNetwork, host.Network)
						assert.EqualValues(mockPersistentDisk, host.Disk)
						assert.EqualValues(mockPersistentBuild, host.Build)
						assert.EqualValues(mockPersistentInterval, host.AnnounceInterval)
						assert.NotEqual(0, host.CreatedAt.Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Nanosecond())
						assert.NotNil(host.Log)
					}).Return(nil).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Load(gomock.Any(), gomock.Any()).Return(persistentCacheHost, true).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Store(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, host *persistentcache.Host) {
						assert.Equal(req.Host.Id, host.ID)
						assert.Equal(pkgtypes.HostType(req.Host.Type), host.Type)
						assert.Equal(req.Host.Hostname, host.Hostname)
						assert.Equal(req.Host.Ip, host.IP)
						assert.Equal(req.Host.Port, host.Port)
						assert.Equal(req.Host.DownloadPort, host.DownloadPort)
						assert.Equal(req.Host.ProxyPort, host.ProxyPort)
						assert.Equal(req.Host.DisableShared, host.DisableShared)
						assert.Equal(req.Host.Os, host.OS)
						assert.Equal(req.Host.Platform, host.Platform)
						assert.Equal(req.Host.PlatformVersion, host.PlatformVersion)
						assert.Equal(req.Host.KernelVersion, host.KernelVersion)
						assert.EqualValues(mockPersistentCacheCPU, host.CPU)
						assert.EqualValues(mockPersistentCacheMemory, host.Memory)
						assert.EqualValues(mockPersistentCacheNetwork, host.Network)
						assert.EqualValues(mockPersistentCacheDisk, host.Disk)
						assert.EqualValues(mockPersistentCacheBuild, host.Build)
						assert.EqualValues(mockPersistentCacheInterval, host.AnnounceInterval)
						assert.NotEqual(0, host.CreatedAt.Nanosecond())
						assert.NotEqual(0, host.UpdatedAt.Nanosecond())
						assert.NotNil(host.Log)
					}).Return(errors.New("bar")).Times(1),
				)

				assert.Error(svc.AnnounceHost(context.Background(), req))
			},
		},

		{
			name: "host not found and host type is HostTypeSuperSeed uses seed peer cluster load limit",
			req:  mockAnnounceHostRequest(pkgtypes.HostTypeSuperSeed),
			run: func(t *testing.T, svc *V2, req *schedulerv2.AnnounceHostRequest, host *standard.Host, persistentHost *persistent.Host, persistentCacheHost *persistentcache.Host, hostManager standard.HostManager, persistentHostManager persistent.HostManager, persistentCacheHostManager persistentcache.HostManager, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpcr *persistentcache.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					md.GetSeedPeerClusterConfig().Return(managertypes.SeedPeerClusterConfig{LoadLimit: 20}, nil).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(nil, false).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Store(gomock.Cond(func(host *standard.Host) bool {
						return host.Type == pkgtypes.HostTypeSuperSeed && host.ConcurrentUploadLimit.Load() == 20
					})).Return().Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Any()).Return(nil, false).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Store(gomock.Any(), gomock.Cond(func(host *persistent.Host) bool { return host.Type == pkgtypes.HostTypeSuperSeed })).Return(nil).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Load(gomock.Any(), gomock.Any()).Return(nil, false).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Store(gomock.Any(), gomock.Cond(func(host *persistentcache.Host) bool { return host.Type == pkgtypes.HostTypeSuperSeed })).Return(nil).Times(1),
				)

				assert.NoError(svc.AnnounceHost(context.Background(), req))
			},
		},
		{
			name: "host already exists and host type is HostTypeSuperSeed with seed peer cluster config error keeps default limit",
			req:  mockAnnounceHostRequest(pkgtypes.HostTypeSuperSeed),
			run: func(t *testing.T, svc *V2, req *schedulerv2.AnnounceHostRequest, host *standard.Host, persistentHost *persistent.Host, persistentCacheHost *persistentcache.Host, hostManager standard.HostManager, persistentHostManager persistent.HostManager, persistentCacheHostManager persistentcache.HostManager, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpcr *persistentcache.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					md.GetSeedPeerClusterConfig().Return(managertypes.SeedPeerClusterConfig{}, errors.New("foo")).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(host, true).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Any()).Return(persistentHost, true).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Store(gomock.Any(), gomock.Eq(persistentHost)).Return(nil).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Load(gomock.Any(), gomock.Any()).Return(persistentCacheHost, true).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Store(gomock.Any(), gomock.Eq(persistentCacheHost)).Return(nil).Times(1),
				)

				assert := assert.New(t)
				assert.NoError(svc.AnnounceHost(context.Background(), req))
				assert.Equal(pkgtypes.HostTypeSuperSeed, host.Type)
				assert.Equal(int32(200), host.ConcurrentUploadLimit.Load())
				assert.Equal(pkgtypes.HostTypeSuperSeed, persistentHost.Type)
				assert.Equal(pkgtypes.HostTypeSuperSeed, persistentCacheHost.Type)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			hostManager := standard.NewMockHostManager(ctl)
			persistentHostManager := persistent.NewMockHostManager(ctl)
			persistentCacheHostManager := persistentcache.NewMockHostManager(ctl)
			host := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			persistentHost := persistent.NewHost(
				mockRawPersistentHost.ID, mockRawPersistentHost.Name, mockRawPersistentHost.Hostname, mockRawPersistentHost.IP,
				mockRawPersistentHost.OS, mockRawPersistentHost.Platform, mockRawPersistentHost.PlatformFamily, mockRawPersistentHost.PlatformVersion, mockRawPersistentHost.KernelVersion,
				mockRawPersistentHost.Port, mockRawPersistentHost.DownloadPort, mockRawPersistentHost.ProxyPort, mockRawPersistentHost.SchedulerClusterID, mockRawPersistentHost.DisableShared, pkgtypes.HostType(mockRawPersistentHost.Type),
				mockRawPersistentHost.CPU, mockRawPersistentHost.Memory, mockRawPersistentHost.Network, mockRawPersistentHost.Disk,
				mockRawPersistentHost.Build, mockRawPersistentHost.AnnounceInterval, mockRawPersistentHost.CreatedAt, mockRawPersistentHost.UpdatedAt, mockRawHost.Log)
			persistentCacheHost := persistentcache.NewHost(
				mockRawPersistentCacheHost.ID, mockRawPersistentCacheHost.Name, mockRawPersistentCacheHost.Hostname, mockRawPersistentCacheHost.IP,
				mockRawPersistentCacheHost.OS, mockRawPersistentCacheHost.Platform, mockRawPersistentCacheHost.PlatformFamily, mockRawPersistentCacheHost.PlatformVersion, mockRawPersistentCacheHost.KernelVersion,
				mockRawPersistentCacheHost.Port, mockRawPersistentCacheHost.DownloadPort, mockRawPersistentCacheHost.ProxyPort, mockRawPersistentCacheHost.SchedulerClusterID, mockRawPersistentCacheHost.DisableShared, pkgtypes.HostType(mockRawPersistentCacheHost.Type),
				mockRawPersistentCacheHost.CPU, mockRawPersistentCacheHost.Memory, mockRawPersistentCacheHost.Network, mockRawPersistentCacheHost.Disk,
				mockRawPersistentCacheHost.Build, mockRawPersistentCacheHost.AnnounceInterval, mockRawPersistentCacheHost.CreatedAt, mockRawPersistentCacheHost.UpdatedAt, mockRawHost.Log)

			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig, Metrics: config.MetricsConfig{EnableHost: true}}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, tc.req, host, persistentHost, persistentCacheHost, hostManager, persistentHostManager, persistentCacheHostManager, resource.EXPECT(), persistentResource.EXPECT(), persistentCacheResource.EXPECT(), hostManager.EXPECT(), persistentHostManager.EXPECT(), persistentCacheHostManager.EXPECT(), dynconfig.EXPECT())
		})
	}
}

func TestServiceV2_ListHosts(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(host *standard.Host, hostManager standard.HostManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder)
		expect func(t *testing.T, host *standard.Host, resp []*commonv2.Host, err error)
	}{
		{
			name: "host manager is empty",
			mock: func(host *standard.Host, hostManager standard.HostManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Range(gomock.Any()).Do(func(f func(key, value any) bool) {
						f(nil, nil)
					}).Return().Times(1),
				)
			},
			expect: func(t *testing.T, host *standard.Host, resp []*commonv2.Host, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp, 0)
			},
		},
		{
			name: "host manager is not empty",
			mock: func(host *standard.Host, hostManager standard.HostManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Range(gomock.Any()).Do(func(f func(key, value any) bool) {
						f(nil, host)
					}).Return().Times(1),
				)
			},
			expect: func(t *testing.T, host *standard.Host, resp []*commonv2.Host, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp, 1)
				assert.EqualValues(&commonv2.Host{
					Id:           mockHostID,
					Type:         uint32(pkgtypes.HostTypeNormal),
					Hostname:     "foo",
					Ip:           "127.0.0.1",
					Port:         8003,
					DownloadPort: mockRawHost.DownloadPort,
					ProxyPort:    mockRawHost.ProxyPort,
					Cpu: &commonv2.CPU{
						LogicalCount:   mockCPU.LogicalCount,
						PhysicalCount:  mockCPU.PhysicalCount,
						Percent:        mockCPU.Percent,
						ProcessPercent: mockCPU.ProcessPercent,
						Times: &commonv2.CPUTimes{
							User:      mockCPU.Times.User,
							System:    mockCPU.Times.System,
							Idle:      mockCPU.Times.Idle,
							Nice:      mockCPU.Times.Nice,
							Iowait:    mockCPU.Times.Iowait,
							Irq:       mockCPU.Times.Irq,
							Softirq:   mockCPU.Times.Softirq,
							Steal:     mockCPU.Times.Steal,
							Guest:     mockCPU.Times.Guest,
							GuestNice: mockCPU.Times.GuestNice,
						},
					},
					Memory: &commonv2.Memory{
						Total:              mockMemory.Total,
						Available:          mockMemory.Available,
						Used:               mockMemory.Used,
						UsedPercent:        mockMemory.UsedPercent,
						ProcessUsedPercent: mockMemory.ProcessUsedPercent,
						Free:               mockMemory.Free,
					},
					Network: &commonv2.Network{
						TcpConnectionCount:       mockNetwork.TCPConnectionCount,
						UploadTcpConnectionCount: mockNetwork.UploadTCPConnectionCount,
						Location:                 &mockNetwork.Location,
						Idc:                      &mockNetwork.IDC,
						RxBandwidth:              &mockNetwork.RxBandwidth,
						MaxRxBandwidth:           mockNetwork.MaxRxBandwidth,
						TxBandwidth:              &mockNetwork.TxBandwidth,
						MaxTxBandwidth:           mockNetwork.MaxTxBandwidth,
					},
					Disk: &commonv2.Disk{
						Total:             mockDisk.Total,
						Free:              mockDisk.Free,
						Used:              mockDisk.Used,
						UsedPercent:       mockDisk.UsedPercent,
						InodesTotal:       mockDisk.InodesTotal,
						InodesUsed:        mockDisk.InodesUsed,
						InodesFree:        mockDisk.InodesFree,
						InodesUsedPercent: mockDisk.InodesUsedPercent,
					},
					Build: &commonv2.Build{
						GitVersion: mockBuild.GitVersion,
						GitCommit:  &mockBuild.GitCommit,
						GoVersion:  &mockBuild.GoVersion,
						Platform:   &mockBuild.Platform,
					},
				}, resp[0])
			},
		},

		{
			name: "host manager contains an invalid value",
			mock: func(host *standard.Host, hostManager standard.HostManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Range(gomock.Any()).Do(func(f func(key, value any) bool) {
						f(nil, "foo")
						f(nil, host)
					}).Return().Times(1),
				)
			},
			expect: func(t *testing.T, host *standard.Host, resp []*commonv2.Host, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp, 1)
				assert.Equal(mockHostID, resp[0].Id)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			hostManager := standard.NewMockHostManager(ctl)
			host := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname, mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type,
				standard.WithCPU(mockCPU), standard.WithMemory(mockMemory), standard.WithNetwork(mockNetwork), standard.WithDisk(mockDisk), standard.WithBuild(mockBuild))
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig, Metrics: config.MetricsConfig{EnableHost: true}}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(host, hostManager, resource.EXPECT(), hostManager.EXPECT())
			resp, err := svc.ListHosts(context.Background(), &schedulerv2.ListHostsRequest{})
			tc.expect(t, host, resp.Hosts, err)
		})
	}
}

func TestServiceV2_DeletePeer(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder)
		expect func(t *testing.T, err error)
	}{
		{
			name: "delete peer by id",
			mock: func(peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Delete(gomock.Eq(mockPeerID)).Return().Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, nil, nil, scheduling, job, internalJobImage, dynconfig)

			tc.mock(peerManager, resource.EXPECT(), peerManager.EXPECT())
			tc.expect(t, svc.DeletePeer(context.Background(), &schedulerv2.DeletePeerRequest{HostId: mockHostID, TaskId: mockTaskID, PeerId: mockPeerID}))
		})
	}
}

func TestServiceV2_DeleteTask(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(host *standard.Host, peer *standard.Peer, otherPeer *standard.Peer, hostManager standard.HostManager, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder)
		expect func(t *testing.T, err error)
	}{
		{
			name: "host not found",
			mock: func(host *standard.Host, peer *standard.Peer, otherPeer *standard.Peer, hostManager standard.HostManager, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(mockHostID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "host %s not found", mockHostID))
			},
		},
		{
			name: "host has no peers",
			mock: func(host *standard.Host, peer *standard.Peer, otherPeer *standard.Peer, hostManager standard.HostManager, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(mockHostID)).Return(host, true).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "only peers of the requested task are deleted",
			mock: func(host *standard.Host, peer *standard.Peer, otherPeer *standard.Peer, hostManager standard.HostManager, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				host.StorePeer(peer)
				host.StorePeer(otherPeer)
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Delete(gomock.Eq(peer.ID)).Return().Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			hostManager := standard.NewMockHostManager(ctl)
			peerManager := standard.NewMockPeerManager(ctl)

			host := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			otherTask := standard.NewTask("bar", mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			peer := standard.NewPeer(mockPeerID, task, host)
			otherPeer := standard.NewPeer(mockSeedPeerID, otherTask, host)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, nil, nil, scheduling, job, internalJobImage, dynconfig)

			tc.mock(host, peer, otherPeer, hostManager, peerManager, resource.EXPECT(), hostManager.EXPECT(), peerManager.EXPECT())
			tc.expect(t, svc.DeleteTask(context.Background(), &schedulerv2.DeleteTaskRequest{HostId: mockHostID, TaskId: mockTaskID}))
		})
	}
}

func TestServiceV2_DeleteHost(t *testing.T) {
	tests := []struct {
		name string
		mock func(wg *sync.WaitGroup, host *standard.Host, mockPeer *standard.Peer, persistentPeer *persistent.Peer, persistentCachePeer *persistentcache.Peer,
			peerManager standard.PeerManager, hostManager standard.HostManager, persistentPeerManager persistent.PeerManager, persistentHostManager persistent.HostManager,
			persistentCachePeerManager persistentcache.PeerManager, persistentCacheHostManager persistentcache.HostManager,
			mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder,
			mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder,
			mpcr *persistentcache.MockResourceMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder,
			ms *schedulingmocks.MockSchedulingMockRecorder)
		expect func(t *testing.T, peer *standard.Peer, err error)
	}{
		{
			name: "host not found",
			mock: func(wg *sync.WaitGroup, host *standard.Host, mockPeer *standard.Peer, persistentPeer *persistent.Peer, persistentCachePeer *persistentcache.Peer,
				peerManager standard.PeerManager, hostManager standard.HostManager, persistentPeerManager persistent.PeerManager, persistentHostManager persistent.HostManager,
				persistentCachePeerManager persistentcache.PeerManager, persistentCacheHostManager persistentcache.HostManager,
				mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder,
				mpcr *persistentcache.MockResourceMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder,
				ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "host %s not found", mockHostID))
			},
		},
		{
			name: "host has not peers",
			mock: func(wg *sync.WaitGroup, host *standard.Host, mockPeer *standard.Peer, persistentPeer *persistent.Peer, persistentCachePeer *persistentcache.Peer,
				peerManager standard.PeerManager, hostManager standard.HostManager, persistentPeerManager persistent.PeerManager, persistentHostManager persistent.HostManager,
				persistentCachePeerManager persistentcache.PeerManager, persistentCacheHostManager persistentcache.HostManager,
				mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder,
				mpcr *persistentcache.MockResourceMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder,
				ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(host, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.DeleteAllByHostID(gomock.Any()).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Delete(gomock.Any()).Return().Times(1),
					mpr.PeerManager().Return(persistentPeerManager).Times(1),
					mpp.LoadAllByHostID(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, nil).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Delete(gomock.Any(), gomock.Eq(mockHostID)).Return(nil).Times(1),
					mpcr.PeerManager().Return(persistentCachePeerManager).Times(1),
					mpcp.LoadAllByHostID(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, nil).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Delete(gomock.Any(), gomock.Eq(mockHostID)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "peer leaves succeeded",
			mock: func(wg *sync.WaitGroup, host *standard.Host, mockPeer *standard.Peer, persistentPeer *persistent.Peer, persistentCachePeer *persistentcache.Peer,
				peerManager standard.PeerManager, hostManager standard.HostManager, persistentPeerManager persistent.PeerManager, persistentHostManager persistent.HostManager,
				persistentCachePeerManager persistentcache.PeerManager, persistentCacheHostManager persistentcache.HostManager,
				mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder,
				mpcr *persistentcache.MockResourceMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder,
				ms *schedulingmocks.MockSchedulingMockRecorder) {
				host.Peers.Store(mockPeer.ID, mockPeer)
				mockPeer.FSM.SetState(standard.PeerStatePending)
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(host, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.DeleteAllByHostID(gomock.Any()).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Delete(gomock.Any()).Return().Times(1),
					mpr.PeerManager().Return(persistentPeerManager).Times(1),
					mpp.LoadAllByHostID(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, nil).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Delete(gomock.Any(), gomock.Eq(mockHostID)).Return(nil).Times(1),
					mpcr.PeerManager().Return(persistentCachePeerManager).Times(1),
					mpcp.LoadAllByHostID(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, nil).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Delete(gomock.Any(), gomock.Eq(mockHostID)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "load persistent peers failed is tolerated and succeeded persistent peer is replicated",
			mock: func(wg *sync.WaitGroup, host *standard.Host, mockPeer *standard.Peer, persistentPeer *persistent.Peer, persistentCachePeer *persistentcache.Peer,
				peerManager standard.PeerManager, hostManager standard.HostManager, persistentPeerManager persistent.PeerManager, persistentHostManager persistent.HostManager,
				persistentCachePeerManager persistentcache.PeerManager, persistentCacheHostManager persistentcache.HostManager,
				mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder,
				mpcr *persistentcache.MockResourceMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder,
				ms *schedulingmocks.MockSchedulingMockRecorder) {
				wg.Add(2)
				persistentPeer.FSM.SetState(persistent.PeerStateSucceeded)
				persistentCachePeer.FSM.SetState(persistentcache.PeerStateSucceeded)
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(host, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.DeleteAllByHostID(gomock.Any()).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Delete(gomock.Any()).Return().Times(1),
					mpr.PeerManager().Return(persistentPeerManager).Times(1),
					mpp.LoadAllByHostID(gomock.Any(), gomock.Eq(mockHostID)).Return([]*persistent.Peer{persistentPeer}, errors.New("foo")).Times(1),
					mpr.PeerManager().Return(persistentPeerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(persistentPeer.ID)).Return(errors.New("bar")).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Delete(gomock.Any(), gomock.Eq(mockHostID)).Return(nil).Times(1),
					mpcr.PeerManager().Return(persistentCachePeerManager).Times(1),
					mpcp.LoadAllByHostID(gomock.Any(), gomock.Eq(mockHostID)).Return([]*persistentcache.Peer{persistentCachePeer}, nil).Times(1),
					mpcr.PeerManager().Return(persistentCachePeerManager).Times(1),
					mpcp.Delete(gomock.Any(), gomock.Eq(persistentCachePeer.ID)).Return(nil).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Delete(gomock.Any(), gomock.Eq(mockHostID)).Return(nil).Times(1),
				)
				ms.FindReplicatePersistentHosts(gomock.Any(), gomock.Eq(persistentPeer.Task), gomock.Cond(func(blocklist set.SafeSet[string]) bool { return blocklist.Contains(mockHostID) })).Do(func(context.Context, *persistent.Task, set.SafeSet[string]) { wg.Done() }).Return(nil, nil, false).Times(1)
				ms.FindReplicatePersistentCacheHosts(gomock.Any(), gomock.Eq(persistentCachePeer.Task), gomock.Cond(func(blocklist set.SafeSet[string]) bool { return blocklist.Contains(mockHostID) })).Do(func(context.Context, *persistentcache.Task, set.SafeSet[string]) { wg.Done() }).Return(nil, nil, false).Times(1)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "delete persistent host failed",
			mock: func(wg *sync.WaitGroup, host *standard.Host, mockPeer *standard.Peer, persistentPeer *persistent.Peer, persistentCachePeer *persistentcache.Peer,
				peerManager standard.PeerManager, hostManager standard.HostManager, persistentPeerManager persistent.PeerManager, persistentHostManager persistent.HostManager,
				persistentCachePeerManager persistentcache.PeerManager, persistentCacheHostManager persistentcache.HostManager,
				mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder,
				mpcr *persistentcache.MockResourceMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder,
				ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(host, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.DeleteAllByHostID(gomock.Any()).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Delete(gomock.Any()).Return().Times(1),
					mpr.PeerManager().Return(persistentPeerManager).Times(1),
					mpp.LoadAllByHostID(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, nil).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Delete(gomock.Any(), gomock.Eq(mockHostID)).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "delete persistent cache host failed",
			mock: func(wg *sync.WaitGroup, host *standard.Host, mockPeer *standard.Peer, persistentPeer *persistent.Peer, persistentCachePeer *persistentcache.Peer,
				peerManager standard.PeerManager, hostManager standard.HostManager, persistentPeerManager persistent.PeerManager, persistentHostManager persistent.HostManager,
				persistentCachePeerManager persistentcache.PeerManager, persistentCacheHostManager persistentcache.HostManager,
				mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder,
				mpcr *persistentcache.MockResourceMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder,
				ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Any()).Return(host, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.DeleteAllByHostID(gomock.Any()).Times(1),
					mr.HostManager().Return(hostManager).Times(1),
					mh.Delete(gomock.Any()).Return().Times(1),
					mpr.PeerManager().Return(persistentPeerManager).Times(1),
					mpp.LoadAllByHostID(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, nil).Times(1),
					mpr.HostManager().Return(persistentHostManager).Times(1),
					mph.Delete(gomock.Any(), gomock.Eq(mockHostID)).Return(nil).Times(1),
					mpcr.PeerManager().Return(persistentCachePeerManager).Times(1),
					mpcp.LoadAllByHostID(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, nil).Times(1),
					mpcr.HostManager().Return(persistentCacheHostManager).Times(1),
					mpch.Delete(gomock.Any(), gomock.Eq(mockHostID)).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *standard.Peer, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			hostManager := standard.NewMockHostManager(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			persistentHostManager := persistent.NewMockHostManager(ctl)
			persistentPeerManager := persistent.NewMockPeerManager(ctl)
			persistentCacheHostManager := persistentcache.NewMockHostManager(ctl)
			persistentCachePeerManager := persistentcache.NewMockPeerManager(ctl)
			host := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			mockPeer := standard.NewPeer(mockSeedPeerID, mockTask, host)
			persistentHost := mockPersistentHost()
			persistentPeer := mockPersistentPeer(mockPeerID, persistent.PeerStatePending, mockPersistentTask(persistent.TaskStateSucceeded), persistentHost)
			persistentCacheHost := mockPersistentCacheHost()
			persistentCachePeer := mockPersistentCachePeer(mockPeerID, persistentcache.PeerStatePending, mockPersistentCacheTask(persistentcache.TaskStateSucceeded), persistentCacheHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig, Metrics: config.MetricsConfig{EnableHost: true}}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			var wg sync.WaitGroup
			tc.mock(&wg, host, mockPeer, persistentPeer, persistentCachePeer, peerManager, hostManager, persistentPeerManager, persistentHostManager, persistentCachePeerManager, persistentCacheHostManager,
				resource.EXPECT(), peerManager.EXPECT(), hostManager.EXPECT(), persistentResource.EXPECT(), persistentPeerManager.EXPECT(), persistentHostManager.EXPECT(),
				persistentCacheResource.EXPECT(), persistentCachePeerManager.EXPECT(), persistentCacheHostManager.EXPECT(), scheduling.EXPECT())
			err := svc.DeleteHost(context.Background(), &schedulerv2.DeleteHostRequest{HostId: mockHostID})
			wg.Wait()
			tc.expect(t, mockPeer, err)
		})
	}
}

func TestServiceV2_handleRegisterPeerRequest(t *testing.T) {
	dgst := mockTaskDigest.String()

	tests := []struct {
		name string
		req  *schedulerv2.RegisterPeerRequest
		run  func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
			peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
			mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder)
	}{
		{
			name: "host not found",
			req:  &schedulerv2.RegisterPeerRequest{},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(nil, false).Times(1),
				)

				assert.ErrorIs(svc.handleRegisterPeerRequest(context.Background(), stream, peer.Host.ID, peer.Task.ID, peer.ID, req),
					status.Errorf(codes.NotFound, "host %s not found", peer.Host.ID))
			},
		},
		{
			name: "can not found available peer and download task failed",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest: &dgst,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.SeedPeer().Return(seedPeerManager).Times(1),
					mc.HasAvailable().Return(true).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL1
				assert.ErrorIs(svc.handleRegisterPeerRequest(context.Background(), stream, peer.Host.ID, peer.Task.ID, peer.ID, req),
					status.Errorf(codes.FailedPrecondition, "%s peer is forbidden", commonv2.Priority_LEVEL1.String()))
			},
		},
		{
			name: "task state is TaskStateFailed and download task failed",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest: &dgst,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.SeedPeer().Return(seedPeerManager).Times(1),
					mc.HasAvailable().Return(true).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL1
				peer.Task.FSM.SetState(standard.TaskStateFailed)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(seedPeer)
				seedPeer.FSM.SetState(standard.PeerStateRunning)

				assert.ErrorIs(svc.handleRegisterPeerRequest(context.Background(), stream, peer.Host.ID, peer.Task.ID, peer.ID, req),
					status.Errorf(codes.FailedPrecondition, "%s peer is forbidden", commonv2.Priority_LEVEL1.String()))
			},
		},
		{
			name: "size scope is SizeScope_EMPTY and load AnnouncePeerStream failed",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest: &dgst,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.SeedPeer().Return(seedPeerManager).Times(1),
					mc.HasAvailable().Return(false).Times(1),
				)

				peer.Task.ContentLength.Store(0)
				peer.Priority = commonv2.Priority_LEVEL6

				assert := assert.New(t)
				assert.ErrorIs(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req),
					status.Error(codes.NotFound, "AnnouncePeerStream not found"))
				assert.Equal(standard.PeerStatePending, peer.FSM.Current())
				assert.Equal(standard.TaskStateRunning, peer.Task.FSM.Current())
			},
		},
		{
			name: "size scope is SizeScope_EMPTY and event PeerEventRegisterEmpty failed",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest: &dgst,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.SeedPeer().Return(seedPeerManager).Times(1),
					mc.HasAvailable().Return(false).Times(1),
				)

				peer.Task.ContentLength.Store(0)
				peer.Priority = commonv2.Priority_LEVEL6
				peer.StoreAnnouncePeerStream(stream)
				peer.FSM.SetState(standard.PeerStateReceivedEmpty)

				assert := assert.New(t)
				assert.ErrorIs(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req),
					status.Errorf(codes.Internal, "event RegisterEmpty inappropriate in current state ReceivedEmpty"))
				assert.Equal(standard.TaskStateRunning, peer.Task.FSM.Current())
			},
		},
		{
			name: "size scope is SizeScope_EMPTY and send EmptyTaskResponse failed",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest: &dgst,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.SeedPeer().Return(seedPeerManager).Times(1),
					mc.HasAvailable().Return(false).Times(1),
					ma.Send(gomock.Eq(&schedulerv2.AnnouncePeerResponse{
						Response: &schedulerv2.AnnouncePeerResponse_EmptyTaskResponse{
							EmptyTaskResponse: &schedulerv2.EmptyTaskResponse{},
						},
					})).Return(errors.New("foo")).Times(1),
				)

				peer.Task.ContentLength.Store(0)
				peer.Priority = commonv2.Priority_LEVEL6
				peer.StoreAnnouncePeerStream(stream)

				assert := assert.New(t)
				assert.ErrorIs(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req),
					status.Errorf(codes.Internal, "foo"))
				assert.Equal(standard.PeerStateReceivedEmpty, peer.FSM.Current())
				assert.Equal(standard.TaskStateRunning, peer.Task.FSM.Current())
			},
		},
		{
			name: "size scope is SizeScope_NORMAL and need back-to-source",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest:           &dgst,
					NeedBackToSource: true,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					ms.ScheduleCandidateParents(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1),
				)

				peer.Task.ContentLength.Store(129)
				peer.Task.TotalPieceCount.Store(2)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(seedPeer)
				peer.Priority = commonv2.Priority_LEVEL6
				peer.NeedBackToSource.Store(true)
				peer.StoreAnnouncePeerStream(stream)

				assert := assert.New(t)
				assert.NoError(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req))
				assert.Equal(standard.PeerStateReceivedNormal, peer.FSM.Current())
				assert.True(peer.NeedBackToSource.Load())
				assert.Equal(standard.TaskStateRunning, peer.Task.FSM.Current())
			},
		},
		{
			name: "size scope is SizeScope_NORMAL",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest: &dgst,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.SeedPeer().Return(seedPeerManager).Times(1),
					mc.HasAvailable().Return(false).Times(1),
					ms.ScheduleCandidateParents(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1),
				)

				peer.Task.ContentLength.Store(129)
				peer.Task.TotalPieceCount.Store(2)
				peer.Task.StorePeer(peer)
				peer.Task.StorePeer(seedPeer)
				peer.Priority = commonv2.Priority_LEVEL6
				peer.StoreAnnouncePeerStream(stream)

				assert := assert.New(t)
				assert.NoError(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req))
				assert.Equal(standard.PeerStateReceivedNormal, peer.FSM.Current())
				assert.Equal(standard.TaskStateRunning, peer.Task.FSM.Current())
			},
		},
		{
			name: "size scope is SizeScope_UNKNOW",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest: &dgst,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.SeedPeer().Return(seedPeerManager).Times(1),
					mc.HasAvailable().Return(false).Times(1),
					ms.ScheduleCandidateParents(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL6

				assert := assert.New(t)
				assert.NoError(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req))
				assert.Equal(standard.PeerStateReceivedNormal, peer.FSM.Current())
				assert.Equal(standard.TaskStateRunning, peer.Task.FSM.Current())
			},
		},

		{
			name: "download is metadata only and load AnnouncePeerStream failed",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest:       &dgst,
					MetadataOnly: true,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL6

				assert := assert.New(t)
				assert.ErrorIs(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req),
					status.Error(codes.NotFound, "AnnouncePeerStream not found"))
				assert.Equal(standard.PeerStatePending, peer.FSM.Current())
				assert.Equal(standard.TaskStateRunning, peer.Task.FSM.Current())
			},
		},
		{
			name: "download is metadata only and send MetadataOnlyResponse failed",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest:       &dgst,
					MetadataOnly: true,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					ma.Send(gomock.Eq(&schedulerv2.AnnouncePeerResponse{
						Response: &schedulerv2.AnnouncePeerResponse_MetadataOnlyResponse{
							MetadataOnlyResponse: &schedulerv2.MetadataOnlyResponse{},
						},
					})).Return(errors.New("foo")).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL6
				peer.Task.FSM.SetState(standard.TaskStateRunning)
				peer.StoreAnnouncePeerStream(stream)

				assert := assert.New(t)
				assert.ErrorIs(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req),
					status.Error(codes.Internal, "foo"))
				assert.Equal(standard.PeerStateReceivedNormal, peer.FSM.Current())
			},
		},
		{
			name: "download is metadata only and send MetadataOnlyResponse succeeded",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest:       &dgst,
					MetadataOnly: true,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					ma.Send(gomock.Eq(&schedulerv2.AnnouncePeerResponse{
						Response: &schedulerv2.AnnouncePeerResponse_MetadataOnlyResponse{
							MetadataOnlyResponse: &schedulerv2.MetadataOnlyResponse{},
						},
					})).Return(nil).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL6
				peer.StoreAnnouncePeerStream(stream)

				assert := assert.New(t)
				assert.NoError(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req))
				assert.Equal(standard.PeerStateReceivedNormal, peer.FSM.Current())
				assert.Equal(standard.TaskStateRunning, peer.Task.FSM.Current())
				assert.False(peer.NeedBackToSource.Load())
			},
		},
		{
			name: "host type is HostTypeSuperSeed and task state is TaskStatePending",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest: &dgst,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					ms.ScheduleCandidateParents(gomock.Any(), gomock.Eq(peer), gomock.Any()).Return(nil).Times(1),
				)

				peer.Host.Type = pkgtypes.HostTypeSuperSeed
				peer.Priority = commonv2.Priority_LEVEL6

				assert := assert.New(t)
				assert.NoError(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req))
				assert.True(peer.NeedBackToSource.Load())
				assert.Equal(standard.PeerStateReceivedNormal, peer.FSM.Current())
				assert.Equal(standard.TaskStateRunning, peer.Task.FSM.Current())
			},
		},
		{
			name: "task state is TaskStatePending and seed peer triggers download task",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest: &dgst,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.SeedPeer().Return(seedPeerManager).Times(1),
					mc.HasAvailable().Return(true).Times(1),
				)
				mr.SeedPeer().Return(seedPeerManager).Times(1)
				mc.TriggerDownloadTask(gomock.Any(), gomock.Eq(peer.Task.ID), gomock.Cond(func(req *dfdaemonv2.DownloadTaskRequest) bool {
					return req.Download.GetNeedBackToSource() && req.Download.OutputPath == nil
				})).Do(func(context.Context, string, *dfdaemonv2.DownloadTaskRequest) { wg.Done() }).Return(nil).Times(1)
				ms.ScheduleCandidateParents(gomock.Any(), gomock.Eq(peer), gomock.Any()).Return(nil).Times(1)

				peer.Priority = commonv2.Priority_LEVEL6

				assert := assert.New(t)
				assert.NoError(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req))
				assert.False(peer.NeedBackToSource.Load())
				assert.Equal(standard.PeerStateReceivedNormal, peer.FSM.Current())
				assert.Equal(standard.TaskStateRunning, peer.Task.FSM.Current())
			},
		},
		{
			name: "size scope is SizeScope_NORMAL and schedule candidate parents failed",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest: &dgst,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.SeedPeer().Return(seedPeerManager).Times(1),
					mc.HasAvailable().Return(false).Times(1),
					ms.ScheduleCandidateParents(gomock.Any(), gomock.Eq(peer), gomock.Any()).Return(errors.New("foo")).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL6

				assert := assert.New(t)
				assert.ErrorIs(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req),
					status.Error(codes.FailedPrecondition, "foo"))
				assert.Equal(standard.PeerStateReceivedNormal, peer.FSM.Current())
				assert.True(peer.BlockParents.Contains(peer.ID))
			},
		},
		{
			name: "size scope is SizeScope_NORMAL and event PeerEventRegisterNormal failed",
			req: &schedulerv2.RegisterPeerRequest{
				Download: &commonv2.Download{
					Digest: &dgst,
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.RegisterPeerRequest, peer *standard.Peer, seedPeer *standard.Peer, seedPeerManager standard.SeedPeer, hostManager standard.HostManager, taskManager standard.TaskManager,
				peerManager standard.PeerManager, stream schedulerv2.Scheduler_AnnouncePeerServer, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder, mc *standard.MockSeedPeerMockRecorder, ma *schedulerv2mocks.MockScheduler_AnnouncePeerServerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(peer.Host.ID)).Return(peer.Host, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(peer.Task.ID)).Return(peer.Task, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.SeedPeer().Return(seedPeerManager).Times(1),
					mc.HasAvailable().Return(false).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL6
				peer.FSM.SetState(standard.PeerStateRunning)

				assert := assert.New(t)
				assert.ErrorIs(svc.handleRegisterPeerRequest(context.Background(), nil, peer.Host.ID, peer.Task.ID, peer.ID, req),
					status.Error(codes.Internal, "event RegisterNormal inappropriate in current state Running"))
				assert.Equal(standard.PeerStateRunning, peer.FSM.Current())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			hostManager := standard.NewMockHostManager(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			taskManager := standard.NewMockTaskManager(ctl)
			seedPeerManager := standard.NewMockSeedPeer(ctl)
			stream := schedulerv2mocks.NewMockScheduler_AnnouncePeerServer(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			seedPeer := standard.NewPeer(mockSeedPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, tc.req, peer, seedPeer, seedPeerManager, hostManager, taskManager, peerManager, stream, resource.EXPECT(), hostManager.EXPECT(), taskManager.EXPECT(), peerManager.EXPECT(), seedPeerManager.EXPECT(), stream.EXPECT(), scheduling.EXPECT())
		})
	}
}

func TestServiceV2_handleDownloadPeerStartedRequest(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder)
	}{
		{
			name: "peer can not be loaded",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(nil, false).Times(1),
				)

				assert.ErrorIs(svc.handleDownloadPeerStartedRequest(context.Background(), peer.ID), status.Errorf(codes.NotFound, "peer %s not found", peer.ID))
			},
		},
		{
			name: "peer state is PeerStateRunning",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateRunning)

				assert.NoError(svc.handleDownloadPeerStartedRequest(context.Background(), peer.ID))
			},
		},
		{
			name: "task state is TaskStateRunning",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateReceivedNormal)
				peer.Task.FSM.SetState(standard.TaskStateRunning)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPeerStartedRequest(context.Background(), peer.ID))
				assert.NotEqual(0, peer.UpdatedAt.Load())
				assert.NotEqual(0, peer.Task.UpdatedAt.Load())
			},
		},
		{
			name: "task state is TaskStatePending",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateReceivedNormal)
				peer.Task.FSM.SetState(standard.TaskStatePending)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPeerStartedRequest(context.Background(), peer.ID))
				assert.NotEqual(0, peer.UpdatedAt.Load())
				assert.NotEqual(0, peer.Task.UpdatedAt.Load())
			},
		},

		{
			name: "peer state is PeerStatePending",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStatePending)

				assert := assert.New(t)
				assert.ErrorIs(svc.handleDownloadPeerStartedRequest(context.Background(), peer.ID), status.Error(codes.Internal, "event Download inappropriate in current state Pending"))
				assert.Equal(standard.PeerStatePending, peer.FSM.Current())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, peer, peerManager, resource.EXPECT(), peerManager.EXPECT(), dynconfig.EXPECT())
		})
	}
}

func TestServiceV2_handleDownloadPeerBackToSourceStartedRequest(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder)
	}{
		{
			name: "peer can not be loaded",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(nil, false).Times(1),
				)

				assert.ErrorIs(svc.handleDownloadPeerBackToSourceStartedRequest(context.Background(), peer.ID), status.Errorf(codes.NotFound, "peer %s not found", peer.ID))
			},
		},
		{
			name: "peer state is PeerStateRunning",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateBackToSource)

				assert.ErrorIs(svc.handleDownloadPeerBackToSourceStartedRequest(context.Background(), peer.ID), status.Error(codes.Internal, "event DownloadBackToSource inappropriate in current state BackToSource"))
			},
		},
		{
			name: "task state is TaskStateRunning",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateReceivedNormal)
				peer.Task.FSM.SetState(standard.TaskStateRunning)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPeerBackToSourceStartedRequest(context.Background(), peer.ID))
				assert.NotEqual(0, peer.UpdatedAt.Load())
				assert.NotEqual(0, peer.Task.UpdatedAt.Load())
			},
		},
		{
			name: "task state is TaskStatePending",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateReceivedNormal)
				peer.Task.FSM.SetState(standard.TaskStatePending)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPeerBackToSourceStartedRequest(context.Background(), peer.ID))
				assert.NotEqual(0, peer.UpdatedAt.Load())
				assert.NotEqual(0, peer.Task.UpdatedAt.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, peer, peerManager, resource.EXPECT(), peerManager.EXPECT(), dynconfig.EXPECT())
		})
	}
}

func TestServiceV2_handleRescheduleRequest(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
			mp *standard.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder)
	}{
		{
			name: "peer can not be loaded",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(nil, false).Times(1),
				)

				assert.ErrorIs(svc.handleReschedulePeerRequest(context.Background(), peer.ID, []*commonv2.Peer{}), status.Errorf(codes.NotFound, "peer %s not found", peer.ID))
			},
		},
		{
			name: "reschedule failed",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					ms.ScheduleCandidateParents(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("foo")).Times(1),
				)

				assert.ErrorIs(svc.handleReschedulePeerRequest(context.Background(), peer.ID, []*commonv2.Peer{}), status.Error(codes.FailedPrecondition, "foo"))
			},
		},
		{
			name: "reschedule succeeded",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					ms.ScheduleCandidateParents(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1),
				)

				assert.NoError(svc.handleReschedulePeerRequest(context.Background(), peer.ID, []*commonv2.Peer{}))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, peer, peerManager, resource.EXPECT(), peerManager.EXPECT(), scheduling.EXPECT())
		})
	}
}

func TestServiceV2_handleDownloadPeerTerminalRequestAfterLeave(t *testing.T) {
	tests := []struct {
		name string
		run  func(*V2, context.Context, string) error
	}{
		{name: "finished", run: (*V2).handleDownloadPeerFinishedRequest},
		{name: "back-to-source finished", run: (*V2).handleDownloadPeerBackToSourceFinishedRequest},
		{name: "failed", run: (*V2).handleDownloadPeerFailedRequest},
		{name: "back-to-source failed", run: (*V2).handleDownloadPeerBackToSourceFailedRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			resource := standard.NewMockResource(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			host := standard.NewHost(mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD,
				mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			peer := standard.NewPeer(mockPeerID, task, host)
			task.StorePeer(peer)
			host.StorePeer(peer)
			peer.FSM.SetState(standard.PeerStateBackToSource)
			task.FSM.SetState(standard.TaskStateSucceeded)
			task.ContentLength.Store(1024)
			task.TotalPieceCount.Store(1)
			task.DirectPiece = []byte{1}

			// Host cleanup can leave a peer while its announce stream is still open.
			host.LeavePeers()
			assert.Equal(t, standard.PeerStateLeave, peer.FSM.Current())
			updatedAt := peer.UpdatedAt.Load()
			resource.EXPECT().PeerManager().Return(peerManager).Times(2)
			peerManager.EXPECT().Load(peer.ID).Return(peer, true).Times(2)
			svc := &V2{resource: resource}

			for i := 0; i < 2; i++ {
				assert.NoError(t, tc.run(svc, context.Background(), peer.ID))
				assert.Equal(t, standard.PeerStateLeave, peer.FSM.Current())
				assert.Equal(t, time.Duration(0), peer.Cost.Load())
				assert.Equal(t, updatedAt, peer.UpdatedAt.Load())
				assert.Equal(t, standard.TaskStateSucceeded, task.FSM.Current())
				assert.Equal(t, int64(1024), task.ContentLength.Load())
				assert.Equal(t, int32(1), task.TotalPieceCount.Load())
				assert.Equal(t, []byte{1}, task.DirectPiece)
			}
		})
	}
}

func TestHandlePeerTerminalEventUsesNonCancelableContext(t *testing.T) {
	host := standard.NewHost(mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
		mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
	task := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD,
		mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
	peer := standard.NewPeer(mockPeerID, task, host)
	peer.FSM.SetState(standard.PeerStateRunning)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	transitioned, err := handlePeerTerminalEvent(ctx, peer, standard.PeerEventDownloadSucceeded)
	assert.NoError(t, err)
	assert.True(t, transitioned)
	assert.Equal(t, standard.PeerStateSucceeded, peer.FSM.Current())

	// A canceled request must not leave the FSM in an incomplete transition.
	assert.NoError(t, peer.FSM.Event(context.Background(), standard.PeerEventLeave))
	assert.Equal(t, standard.PeerStateLeave, peer.FSM.Current())
}

func TestHandlePeerTerminalEventConcurrentWithLeave(t *testing.T) {
	for i := 0; i < 100; i++ {
		host := standard.NewHost(mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
			mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
		task := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD,
			mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
		peer := standard.NewPeer(mockPeerID, task, host)
		peer.FSM.SetState(standard.PeerStateRunning)

		start := make(chan struct{})
		var wg sync.WaitGroup
		var terminalErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, terminalErr = handlePeerTerminalEvent(context.Background(), peer, standard.PeerEventDownloadSucceeded)
		}()
		go func() {
			defer wg.Done()
			<-start
			host.LeavePeers()
		}()
		close(start)
		wg.Wait()

		assert.NoError(t, terminalErr)
		assert.Contains(t, []string{standard.PeerStateSucceeded, standard.PeerStateLeave}, peer.FSM.Current())
	}
}

func TestServiceV2_handleDownloadPeerFinishedRequest(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder)
	}{
		{
			name: "peer can not be loaded",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(nil, false).Times(1),
				)

				assert.ErrorIs(svc.handleDownloadPeerFinishedRequest(context.Background(), peer.ID), status.Errorf(codes.NotFound, "peer %s not found", peer.ID))
			},
		},
		{
			name: "peer state is PeerStateSucceeded",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateSucceeded)

				assert := assert.New(t)
				assert.ErrorIs(svc.handleDownloadPeerFinishedRequest(context.Background(), peer.ID), status.Error(codes.Internal, "event DownloadSucceeded inappropriate in current state Succeeded"))
				assert.NotEqual(0, peer.Cost.Load())
			},
		},
		{
			name: "peer state is PeerStateRunning",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateRunning)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPeerFinishedRequest(context.Background(), peer.ID))
				assert.Equal(standard.PeerStateSucceeded, peer.FSM.Current())
				assert.NotEqual(0, peer.Cost.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, peer, peerManager, resource.EXPECT(), peerManager.EXPECT(), dynconfig.EXPECT())
		})
	}
}

func TestServiceV2_handleDownloadPeerBackToSourceFinishedRequest(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte{1}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer s.Close()

	tests := []struct {
		name string
		run  func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
			mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder)
	}{
		{
			name: "peer can not be loaded",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(nil, false).Times(1),
				)

				assert := assert.New(t)
				assert.ErrorIs(svc.handleDownloadPeerBackToSourceFinishedRequest(context.Background(), peer.ID), status.Errorf(codes.NotFound, "peer %s not found", peer.ID))
				assert.Equal(standard.TaskStatePending, peer.Task.FSM.Current())
			},
		},
		{
			name: "peer state is PeerStateSucceeded",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateSucceeded)

				assert := assert.New(t)
				assert.ErrorIs(svc.handleDownloadPeerBackToSourceFinishedRequest(context.Background(), peer.ID), status.Error(codes.Internal, "event DownloadSucceeded inappropriate in current state Succeeded"))
				assert.NotEqual(0, peer.Cost.Load())
				assert.Equal(standard.TaskStatePending, peer.Task.FSM.Current())
			},
		},
		{
			name: "peer has range",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Range = &nethttp.Range{}

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPeerBackToSourceFinishedRequest(context.Background(), peer.ID))
				assert.NotEqual(0, peer.Cost.Load())
				assert.Equal(standard.PeerStateSucceeded, peer.FSM.Current())
				assert.Equal(standard.TaskStatePending, peer.Task.FSM.Current())
			},
		},
		{
			name: "task state is TaskStateSucceeded",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.FSM.SetState(standard.TaskStateSucceeded)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPeerBackToSourceFinishedRequest(context.Background(), peer.ID))
				assert.NotEqual(0, peer.Cost.Load())
				assert.Equal(standard.PeerStateSucceeded, peer.FSM.Current())
				assert.Equal(standard.TaskStateSucceeded, peer.Task.FSM.Current())
			},
		},
		{
			name: "task state is TaskStatePending",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.FSM.SetState(standard.TaskStatePending)

				assert := assert.New(t)
				assert.ErrorIs(svc.handleDownloadPeerBackToSourceFinishedRequest(context.Background(), peer.ID), status.Error(codes.Internal, "event DownloadSucceeded inappropriate in current state Pending"))
				assert.NotEqual(0, peer.Cost.Load())
				assert.Equal(standard.PeerStateSucceeded, peer.FSM.Current())
				assert.Equal(standard.TaskStatePending, peer.Task.FSM.Current())
			},
		},
		{
			name: "task state is TaskStateRunning",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.FSM.SetState(standard.TaskStateRunning)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPeerBackToSourceFinishedRequest(context.Background(), peer.ID))
				assert.NotEqual(0, peer.Cost.Load())
				assert.Equal(standard.PeerStateSucceeded, peer.FSM.Current())
				assert.Equal(standard.TaskStateSucceeded, peer.Task.FSM.Current())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			url, err := url.Parse(s.URL)
			if err != nil {
				t.Fatal(err)
			}

			ip, rawPort, err := net.SplitHostPort(url.Host)
			if err != nil {
				t.Fatal(err)
			}

			port, err := strconv.ParseInt(rawPort, 10, 32)
			if err != nil {
				t.Fatal(err)
			}

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockHost.IP = ip
			mockHost.DownloadPort = int32(port)

			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, peer, peerManager, resource.EXPECT(), peerManager.EXPECT(), dynconfig.EXPECT())
		})
	}
}

func TestServiceV2_handleDownloadPeerFailedRequest(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder)
	}{
		{
			name: "peer can not be loaded",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(nil, false).Times(1),
				)

				assert.ErrorIs(svc.handleDownloadPeerFailedRequest(context.Background(), peer.ID), status.Errorf(codes.NotFound, "peer %s not found", peer.ID))
			},
		},
		{
			name: "peer state is PeerEventDownloadFailed",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
				)

				peer.FSM.SetState(standard.PeerEventDownloadFailed)

				assert.ErrorIs(svc.handleDownloadPeerFailedRequest(context.Background(), peer.ID), status.Error(codes.Internal, "event DownloadFailed inappropriate in current state DownloadFailed"))
			},
		},
		{
			name: "peer state is PeerStateRunning",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateRunning)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPeerFailedRequest(context.Background(), peer.ID))
				assert.Equal(standard.PeerStateFailed, peer.FSM.Current())
				assert.NotEqual(0, peer.UpdatedAt.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, peer, peerManager, resource.EXPECT(), peerManager.EXPECT(), dynconfig.EXPECT())
		})
	}
}

func TestServiceV2_handleDownloadPeerBackToSourceFailedRequest(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder)
	}{
		{
			name: "peer can not be loaded",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(nil, false).Times(1),
				)

				peer.Task.ContentLength.Store(1)
				peer.Task.TotalPieceCount.Store(1)
				peer.Task.DirectPiece = []byte{1}

				assert := assert.New(t)
				assert.ErrorIs(svc.handleDownloadPeerBackToSourceFailedRequest(context.Background(), peer.ID), status.Errorf(codes.NotFound, "peer %s not found", peer.ID))
				assert.Equal(standard.PeerStatePending, peer.FSM.Current())
				assert.Equal(standard.TaskStatePending, peer.Task.FSM.Current())
				assert.Equal(int64(1), peer.Task.ContentLength.Load())
				assert.Equal(int32(1), peer.Task.TotalPieceCount.Load())
				assert.Equal([]byte{1}, peer.Task.DirectPiece)
			},
		},
		{
			name: "peer state is PeerStateFailed",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateFailed)
				peer.Task.ContentLength.Store(1)
				peer.Task.TotalPieceCount.Store(1)
				peer.Task.DirectPiece = []byte{1}

				assert := assert.New(t)
				assert.ErrorIs(svc.handleDownloadPeerBackToSourceFailedRequest(context.Background(), peer.ID), status.Error(codes.Internal, "event DownloadFailed inappropriate in current state Failed"))
				assert.Equal(standard.PeerStateFailed, peer.FSM.Current())
				assert.Equal(standard.TaskStatePending, peer.Task.FSM.Current())
				assert.Equal(int64(1), peer.Task.ContentLength.Load())
				assert.Equal(int32(1), peer.Task.TotalPieceCount.Load())
				assert.Equal([]byte{1}, peer.Task.DirectPiece)
			},
		},
		{
			name: "task state is TaskStateFailed",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.FSM.SetState(standard.TaskStateFailed)
				peer.Task.ContentLength.Store(1)
				peer.Task.TotalPieceCount.Store(1)
				peer.Task.DirectPiece = []byte{1}

				assert := assert.New(t)
				assert.ErrorIs(svc.handleDownloadPeerBackToSourceFailedRequest(context.Background(), peer.ID), status.Error(codes.Internal, "event DownloadFailed inappropriate in current state Failed"))
				assert.Equal(standard.PeerStateFailed, peer.FSM.Current())
				assert.Equal(standard.TaskStateFailed, peer.Task.FSM.Current())
				assert.Equal(int64(-1), peer.Task.ContentLength.Load())
				assert.Equal(int32(0), peer.Task.TotalPieceCount.Load())
				assert.Equal([]byte{}, peer.Task.DirectPiece)
			},
		},
		{
			name: "task state is TaskStateRunning",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder, md *configmocks.MockDynconfigInterfaceMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					md.GetApplications().Return([]*managerv2.Application{}, nil).Times(1),
				)

				peer.FSM.SetState(standard.PeerStateRunning)
				peer.Task.FSM.SetState(standard.TaskStateRunning)
				peer.Task.ContentLength.Store(1)
				peer.Task.TotalPieceCount.Store(1)
				peer.Task.DirectPiece = []byte{1}

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPeerBackToSourceFailedRequest(context.Background(), peer.ID))
				assert.Equal(standard.PeerStateFailed, peer.FSM.Current())
				assert.Equal(standard.TaskStateFailed, peer.Task.FSM.Current())
				assert.Equal(int64(-1), peer.Task.ContentLength.Load())
				assert.Equal(int32(0), peer.Task.TotalPieceCount.Load())
				assert.Equal([]byte{}, peer.Task.DirectPiece)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, peer, peerManager, resource.EXPECT(), peerManager.EXPECT(), dynconfig.EXPECT())
		})
	}
}

func TestServiceV2_handleDownloadPieceFinishedRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *schedulerv2.DownloadPieceFinishedRequest
		run  func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceFinishedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder)
	}{
		{
			name: "invalid digest",
			req: &schedulerv2.DownloadPieceFinishedRequest{
				Piece: &commonv2.Piece{
					Number:      uint32(mockPiece.Number),
					ParentId:    &mockPiece.ParentID,
					Offset:      mockPiece.Offset,
					Length:      mockPiece.Length,
					Digest:      "foo",
					TrafficType: &mockPiece.TrafficType,
					Cost:        durationpb.New(mockPiece.Cost),
					CreatedAt:   timestamppb.New(mockPiece.CreatedAt),
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceFinishedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				assert := assert.New(t)
				assert.ErrorIs(svc.handleDownloadPieceFinishedRequest(peer.ID, req), status.Error(codes.InvalidArgument, "invalid digest"))
			},
		},
		{
			name: "peer can not be loaded",
			req: &schedulerv2.DownloadPieceFinishedRequest{
				Piece: &commonv2.Piece{
					Number:      uint32(mockPiece.Number),
					ParentId:    &mockPiece.ParentID,
					Offset:      mockPiece.Offset,
					Length:      mockPiece.Length,
					Digest:      mockPiece.Digest.String(),
					TrafficType: &mockPiece.TrafficType,
					Cost:        durationpb.New(mockPiece.Cost),
					CreatedAt:   timestamppb.New(mockPiece.CreatedAt),
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceFinishedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(nil, false).Times(1),
				)

				assert.ErrorIs(svc.handleDownloadPieceFinishedRequest(peer.ID, req), status.Errorf(codes.NotFound, "peer %s not found", peer.ID))
			},
		},
		{
			name: "parent can not be loaded",
			req: &schedulerv2.DownloadPieceFinishedRequest{
				Piece: &commonv2.Piece{
					Number:      uint32(mockPiece.Number),
					ParentId:    &mockPiece.ParentID,
					Offset:      mockPiece.Offset,
					Length:      mockPiece.Length,
					Digest:      mockPiece.Digest.String(),
					TrafficType: &mockPiece.TrafficType,
					Cost:        durationpb.New(mockPiece.Cost),
					CreatedAt:   timestamppb.New(mockPiece.CreatedAt),
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceFinishedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(req.Piece.GetParentId())).Return(nil, false).Times(1),
				)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPieceFinishedRequest(peer.ID, req))

				assert.Equal(uint(1), peer.FinishedPieces.Count())
				assert.Len(peer.PieceCosts(), 1)
				assert.NotEqual(0, peer.PieceUpdatedAt.Load())
				assert.NotEqual(0, peer.UpdatedAt.Load())
				assert.NotEqual(0, peer.Task.UpdatedAt.Load())
			},
		},
		{
			name: "parent can be loaded",
			req: &schedulerv2.DownloadPieceFinishedRequest{
				Piece: &commonv2.Piece{
					Number:      uint32(mockPiece.Number),
					ParentId:    &mockPiece.ParentID,
					Offset:      mockPiece.Offset,
					Length:      mockPiece.Length,
					Digest:      mockPiece.Digest.String(),
					TrafficType: &mockPiece.TrafficType,
					Cost:        durationpb.New(mockPiece.Cost),
					CreatedAt:   timestamppb.New(mockPiece.CreatedAt),
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceFinishedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(req.Piece.GetParentId())).Return(peer, true).Times(1),
				)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPieceFinishedRequest(peer.ID, req))

				assert.Equal(uint(1), peer.FinishedPieces.Count())
				assert.Len(peer.PieceCosts(), 1)
				assert.NotEqual(0, peer.PieceUpdatedAt.Load())
				assert.NotEqual(0, peer.UpdatedAt.Load())
				assert.NotEqual(0, peer.Task.UpdatedAt.Load())
				assert.NotEqual(0, peer.Host.UpdatedAt.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, tc.req, peer, peerManager, resource.EXPECT(), peerManager.EXPECT())
		})
	}
}

func TestServiceV2_handleDownloadPieceBackToSourceFinishedRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *schedulerv2.DownloadPieceBackToSourceFinishedRequest
		run  func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceBackToSourceFinishedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder)
	}{
		{
			name: "invalid digest",
			req: &schedulerv2.DownloadPieceBackToSourceFinishedRequest{
				Piece: &commonv2.Piece{
					Number:      uint32(mockPiece.Number),
					ParentId:    &mockPiece.ParentID,
					Offset:      mockPiece.Offset,
					Length:      mockPiece.Length,
					Digest:      "foo",
					TrafficType: &mockPiece.TrafficType,
					Cost:        durationpb.New(mockPiece.Cost),
					CreatedAt:   timestamppb.New(mockPiece.CreatedAt),
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceBackToSourceFinishedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				assert := assert.New(t)
				assert.ErrorIs(svc.handleDownloadPieceBackToSourceFinishedRequest(context.Background(), peer.ID, req), status.Error(codes.InvalidArgument, "invalid digest"))
			},
		},
		{
			name: "peer can not be loaded",
			req: &schedulerv2.DownloadPieceBackToSourceFinishedRequest{
				Piece: &commonv2.Piece{
					Number:      uint32(mockPiece.Number),
					ParentId:    &mockPiece.ParentID,
					Offset:      mockPiece.Offset,
					Length:      mockPiece.Length,
					Digest:      mockPiece.Digest.String(),
					TrafficType: &mockPiece.TrafficType,
					Cost:        durationpb.New(mockPiece.Cost),
					CreatedAt:   timestamppb.New(mockPiece.CreatedAt),
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceBackToSourceFinishedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(nil, false).Times(1),
				)

				assert.ErrorIs(svc.handleDownloadPieceBackToSourceFinishedRequest(context.Background(), peer.ID, req), status.Errorf(codes.NotFound, "peer %s not found", peer.ID))
			},
		},
		{
			name: "peer can be loaded",
			req: &schedulerv2.DownloadPieceBackToSourceFinishedRequest{
				Piece: &commonv2.Piece{
					Number:      uint32(mockPiece.Number),
					ParentId:    &mockPiece.ParentID,
					Offset:      mockPiece.Offset,
					Length:      mockPiece.Length,
					Digest:      mockPiece.Digest.String(),
					TrafficType: &mockPiece.TrafficType,
					Cost:        durationpb.New(mockPiece.Cost),
					CreatedAt:   timestamppb.New(mockPiece.CreatedAt),
				},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceBackToSourceFinishedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
				)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPieceBackToSourceFinishedRequest(context.Background(), peer.ID, req))

				assert.Equal(uint(1), peer.FinishedPieces.Count())
				assert.Len(peer.PieceCosts(), 1)
				assert.NotEqual(0, peer.PieceUpdatedAt.Load())
				assert.NotEqual(0, peer.UpdatedAt.Load())
				assert.NotEqual(0, peer.Task.UpdatedAt.Load())
				assert.NotEqual(0, peer.Host.UpdatedAt.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, tc.req, peer, peerManager, resource.EXPECT(), peerManager.EXPECT())
		})
	}
}

func TestServiceV2_handleDownloadPieceFailedRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *schedulerv2.DownloadPieceFailedRequest
		run  func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceFailedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
			mp *standard.MockPeerManagerMockRecorder)
	}{
		{
			name: "peer can not be loaded",
			req: &schedulerv2.DownloadPieceFailedRequest{
				ParentId:  mockSeedPeerID,
				Temporary: true,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceFailedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(nil, false).Times(1),
				)

				assert.ErrorIs(svc.handleDownloadPieceFailedRequest(context.Background(), peer.ID, req), status.Errorf(codes.NotFound, "peer %s not found", peer.ID))
			},
		},
		{
			name: "temporary is false",
			req: &schedulerv2.DownloadPieceFailedRequest{
				ParentId:  mockSeedPeerID,
				Temporary: false,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceFailedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
				)

				assert.ErrorIs(svc.handleDownloadPieceFailedRequest(context.Background(), peer.ID, req), status.Error(codes.FailedPrecondition, "download piece failed"))
			},
		},
		{
			name: "parent can not be loaded",
			req: &schedulerv2.DownloadPieceFailedRequest{
				ParentId:  mockSeedPeerID,
				Temporary: true,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceFailedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(req.GetParentId())).Return(nil, false).Times(1),
				)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPieceFailedRequest(context.Background(), peer.ID, req))
				assert.NotEqual(0, peer.UpdatedAt.Load())
				assert.True(peer.BlockParents.Contains(req.GetParentId()))
				assert.NotEqual(0, peer.Task.UpdatedAt.Load())
			},
		},
		{
			name: "parent can be loaded",
			req: &schedulerv2.DownloadPieceFailedRequest{
				ParentId:  mockSeedPeerID,
				Temporary: true,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceFailedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(req.GetParentId())).Return(peer, true).Times(1),
				)

				assert := assert.New(t)
				assert.NoError(svc.handleDownloadPieceFailedRequest(context.Background(), peer.ID, req))
				assert.NotEqual(0, peer.UpdatedAt.Load())
				assert.True(peer.BlockParents.Contains(req.GetParentId()))
				assert.NotEqual(0, peer.Task.UpdatedAt.Load())
				assert.Equal(int64(1), peer.Host.UploadFailedCount.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, tc.req, peer, peerManager, resource.EXPECT(), peerManager.EXPECT())
		})
	}
}

func TestServiceV2_handleDownloadPieceBackToSourceFailedRequest(t *testing.T) {
	mockPieceNumber := uint32(mockPiece.Number)

	tests := []struct {
		name string
		req  *schedulerv2.DownloadPieceBackToSourceFailedRequest
		run  func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceBackToSourceFailedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
			mp *standard.MockPeerManagerMockRecorder)
	}{
		{
			name: "peer can not be loaded",
			req:  &schedulerv2.DownloadPieceBackToSourceFailedRequest{},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceBackToSourceFailedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(nil, false).Times(1),
				)

				assert.ErrorIs(svc.handleDownloadPieceBackToSourceFailedRequest(context.Background(), peer.ID, req), status.Errorf(codes.NotFound, "peer %s not found", peer.ID))
			},
		},
		{
			name: "peer can be loaded",
			req: &schedulerv2.DownloadPieceBackToSourceFailedRequest{
				PieceNumber: &mockPieceNumber,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.DownloadPieceBackToSourceFailedRequest, peer *standard.Peer, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder,
				mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(peer.ID)).Return(peer, true).Times(1),
				)

				assert := assert.New(t)
				assert.ErrorIs(svc.handleDownloadPieceBackToSourceFailedRequest(context.Background(), peer.ID, req), status.Error(codes.Internal, "download piece from source failed"))
				assert.NotEqual(0, peer.UpdatedAt.Load())
				assert.NotEqual(0, peer.Task.UpdatedAt.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, tc.req, peer, peerManager, resource.EXPECT(), peerManager.EXPECT())
		})
	}
}

func TestServiceV2_handleResource(t *testing.T) {
	dgst := mockTaskDigest.String()
	mismatchDgst := "foo"

	tests := []struct {
		name     string
		download *commonv2.Download
		run      func(t *testing.T, svc *V2, download *commonv2.Download, stream schedulerv2.Scheduler_AnnouncePeerServer, mockHost *standard.Host, mockTask *standard.Task, mockPeer *standard.Peer,
			hostManager standard.HostManager, taskManager standard.TaskManager, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
			mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder)
	}{
		{
			name:     "host can not be loaded",
			download: &commonv2.Download{},
			run: func(t *testing.T, svc *V2, download *commonv2.Download, stream schedulerv2.Scheduler_AnnouncePeerServer, mockHost *standard.Host, mockTask *standard.Task, mockPeer *standard.Peer,
				hostManager standard.HostManager, taskManager standard.TaskManager, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(mockHost.ID)).Return(nil, false).Times(1),
				)

				_, _, _, err := svc.handleResource(context.Background(), stream, mockHost.ID, mockTask.ID, mockPeer.ID, download)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "host %s not found", mockHost.ID))
			},
		},
		{
			name: "task can be loaded",
			download: &commonv2.Download{
				Url:                 "foo",
				FilteredQueryParams: []string{"bar"},
				RequestHeader:       map[string]string{"baz": "bas"},
			},
			run: func(t *testing.T, svc *V2, download *commonv2.Download, stream schedulerv2.Scheduler_AnnouncePeerServer, mockHost *standard.Host, mockTask *standard.Task, mockPeer *standard.Peer,
				hostManager standard.HostManager, taskManager standard.TaskManager, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(mockHost.ID)).Return(mockHost, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(mockTask.ID)).Return(mockTask, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeer.ID)).Return(mockPeer, true).Times(1),
				)

				assert := assert.New(t)
				host, task, _, err := svc.handleResource(context.Background(), stream, mockHost.ID, mockTask.ID, mockPeer.ID, download)
				assert.NoError(err)
				assert.EqualValues(mockHost, host)
				assert.Equal(mockTask.ID, task.ID)
				assert.Equal(download.Url, task.URL)
				assert.EqualValues(download.FilteredQueryParams, task.FilteredQueryParams)
				assert.EqualValues(download.RequestHeader, task.Header)
			},
		},
		{
			name: "task can not be loaded",
			download: &commonv2.Download{
				Url:                 "foo",
				FilteredQueryParams: []string{"bar"},
				RequestHeader:       map[string]string{"baz": "bas"},
				Digest:              &dgst,
			},
			run: func(t *testing.T, svc *V2, download *commonv2.Download, stream schedulerv2.Scheduler_AnnouncePeerServer, mockHost *standard.Host, mockTask *standard.Task, mockPeer *standard.Peer,
				hostManager standard.HostManager, taskManager standard.TaskManager, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(mockHost.ID)).Return(mockHost, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(mockTask.ID)).Return(nil, false).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Store(gomock.Any()).Return().Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeer.ID)).Return(mockPeer, true).Times(1),
				)

				assert := assert.New(t)
				host, task, _, err := svc.handleResource(context.Background(), stream, mockHost.ID, mockTask.ID, mockPeer.ID, download)
				assert.NoError(err)
				assert.EqualValues(mockHost, host)
				assert.Equal(mockTask.ID, task.ID)
				assert.Equal(download.GetDigest(), task.Digest.String())
				assert.Equal(download.GetUrl(), task.URL)
				assert.EqualValues(download.GetFilteredQueryParams(), task.FilteredQueryParams)
				assert.EqualValues(download.RequestHeader, task.Header)
			},
		},
		{
			name: "invalid digest",
			download: &commonv2.Download{
				Digest: &mismatchDgst,
			},
			run: func(t *testing.T, svc *V2, download *commonv2.Download, stream schedulerv2.Scheduler_AnnouncePeerServer, mockHost *standard.Host, mockTask *standard.Task, mockPeer *standard.Peer,
				hostManager standard.HostManager, taskManager standard.TaskManager, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				assert := assert.New(t)
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(mockHost.ID)).Return(mockHost, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(mockTask.ID)).Return(nil, false).Times(1),
				)

				_, _, _, err := svc.handleResource(context.Background(), stream, mockHost.ID, mockTask.ID, mockPeer.ID, download)
				assert.ErrorIs(err, status.Error(codes.InvalidArgument, "invalid digest"))
			},
		},
		{
			name: "peer can be loaded",
			download: &commonv2.Download{
				Url:                 "foo",
				FilteredQueryParams: []string{"bar"},
				RequestHeader:       map[string]string{"baz": "bas"},
				Digest:              &dgst,
			},
			run: func(t *testing.T, svc *V2, download *commonv2.Download, stream schedulerv2.Scheduler_AnnouncePeerServer, mockHost *standard.Host, mockTask *standard.Task, mockPeer *standard.Peer,
				hostManager standard.HostManager, taskManager standard.TaskManager, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(mockHost.ID)).Return(mockHost, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(mockTask.ID)).Return(mockTask, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeer.ID)).Return(mockPeer, true).Times(1),
				)

				assert := assert.New(t)
				host, task, peer, err := svc.handleResource(context.Background(), stream, mockHost.ID, mockTask.ID, mockPeer.ID, download)
				assert.NoError(err)
				assert.EqualValues(mockHost, host)
				assert.Equal(mockTask.ID, task.ID)
				assert.Equal(download.GetDigest(), task.Digest.String())
				assert.Equal(download.GetUrl(), task.URL)
				assert.EqualValues(download.GetFilteredQueryParams(), task.FilteredQueryParams)
				assert.EqualValues(download.RequestHeader, task.Header)
				assert.EqualValues(mockPeer, peer)
			},
		},
		{
			name: "peer can not be loaded",
			download: &commonv2.Download{
				Url:                 "foo",
				FilteredQueryParams: []string{"bar"},
				RequestHeader:       map[string]string{"baz": "bas"},
				Digest:              &dgst,
				Priority:            commonv2.Priority_LEVEL1,
				Range: &commonv2.Range{
					Start:  uint64(mockPeerRange.Start),
					Length: uint64(mockPeerRange.Length),
				},
			},
			run: func(t *testing.T, svc *V2, download *commonv2.Download, stream schedulerv2.Scheduler_AnnouncePeerServer, mockHost *standard.Host, mockTask *standard.Task, mockPeer *standard.Peer,
				hostManager standard.HostManager, taskManager standard.TaskManager, peerManager standard.PeerManager, mr *standard.MockResourceMockRecorder, mh *standard.MockHostManagerMockRecorder,
				mt *standard.MockTaskManagerMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.HostManager().Return(hostManager).Times(1),
					mh.Load(gomock.Eq(mockHost.ID)).Return(mockHost, true).Times(1),
					mr.TaskManager().Return(taskManager).Times(1),
					mt.Load(gomock.Eq(mockTask.ID)).Return(mockTask, true).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Load(gomock.Eq(mockPeer.ID)).Return(nil, false).Times(1),
					mr.PeerManager().Return(peerManager).Times(1),
					mp.Store(gomock.Any()).Return().Times(1),
				)

				assert := assert.New(t)
				host, task, peer, err := svc.handleResource(context.Background(), stream, mockHost.ID, mockTask.ID, mockPeer.ID, download)
				assert.NoError(err)
				assert.EqualValues(mockHost, host)
				assert.Equal(mockTask.ID, task.ID)
				assert.Equal(download.GetDigest(), task.Digest.String())
				assert.Equal(download.GetUrl(), task.URL)
				assert.EqualValues(download.GetFilteredQueryParams(), task.FilteredQueryParams)
				assert.EqualValues(download.RequestHeader, task.Header)
				assert.Equal(mockPeer.ID, peer.ID)
				assert.Equal(download.Priority, peer.Priority)
				assert.Equal(int64(download.Range.Start), peer.Range.Start)
				assert.Equal(int64(download.Range.Length), peer.Range.Length)
				assert.NotNil(peer.AnnouncePeerStream)
				assert.EqualValues(mockHost, peer.Host)
				assert.EqualValues(mockTask, peer.Task)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			hostManager := standard.NewMockHostManager(ctl)
			taskManager := standard.NewMockTaskManager(ctl)
			peerManager := standard.NewMockPeerManager(ctl)
			stream := schedulerv2mocks.NewMockScheduler_AnnouncePeerServer(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			mockPeer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, tc.download, stream, mockHost, mockTask, mockPeer, hostManager, taskManager, peerManager, resource.EXPECT(), hostManager.EXPECT(), taskManager.EXPECT(), peerManager.EXPECT())
		})
	}
}

func TestServiceV2_downloadTaskBySeedPeer(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, svc *V2, peer *standard.Peer, seedPeerClient standard.SeedPeer, mr *standard.MockResourceMockRecorder, ms *standard.MockSeedPeerMockRecorder)
	}{
		{
			name: "priority is Priority_LEVEL6",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, seedPeerClient standard.SeedPeer, mr *standard.MockResourceMockRecorder, ms *standard.MockSeedPeerMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mr.SeedPeer().Return(seedPeerClient).Times(1),
					ms.TriggerDownloadTask(gomock.All(), gomock.Any(), gomock.Any()).Do(func(context.Context, string, *dfdaemonv2.DownloadTaskRequest) { wg.Done() }).Return(nil).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL6

				assert := assert.New(t)
				assert.NoError(svc.downloadTaskBySeedPeer(context.Background(), mockTaskID, &commonv2.Download{}, peer))
				assert.False(peer.NeedBackToSource.Load())
			},
		},
		{
			name: "priority is Priority_LEVEL6 and download task failed",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, seedPeerClient standard.SeedPeer, mr *standard.MockResourceMockRecorder, ms *standard.MockSeedPeerMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mr.SeedPeer().Return(seedPeerClient).Times(1),
					ms.TriggerDownloadTask(gomock.All(), gomock.Any(), gomock.Any()).Do(func(context.Context, string, *dfdaemonv2.DownloadTaskRequest) { wg.Done() }).Return(errors.New("foo")).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL6

				assert := assert.New(t)
				assert.NoError(svc.downloadTaskBySeedPeer(context.Background(), mockTaskID, &commonv2.Download{}, peer))
				assert.False(peer.NeedBackToSource.Load())
			},
		},
		{
			name: "priority is Priority_LEVEL5",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, seedPeerClient standard.SeedPeer, mr *standard.MockResourceMockRecorder, ms *standard.MockSeedPeerMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mr.SeedPeer().Return(seedPeerClient).Times(1),
					ms.TriggerDownloadTask(gomock.All(), gomock.Any(), gomock.Any()).Do(func(context.Context, string, *dfdaemonv2.DownloadTaskRequest) { wg.Done() }).Return(nil).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL5

				assert := assert.New(t)
				assert.NoError(svc.downloadTaskBySeedPeer(context.Background(), mockTaskID, &commonv2.Download{}, peer))
				assert.False(peer.NeedBackToSource.Load())
			},
		},
		{
			name: "priority is Priority_LEVEL5 and download task failed",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, seedPeerClient standard.SeedPeer, mr *standard.MockResourceMockRecorder, ms *standard.MockSeedPeerMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mr.SeedPeer().Return(seedPeerClient).Times(1),
					ms.TriggerDownloadTask(gomock.All(), gomock.Any(), gomock.Any()).Do(func(context.Context, string, *dfdaemonv2.DownloadTaskRequest) { wg.Done() }).Return(errors.New("foo")).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL5

				assert := assert.New(t)
				assert.NoError(svc.downloadTaskBySeedPeer(context.Background(), mockTaskID, &commonv2.Download{}, peer))
				assert.False(peer.NeedBackToSource.Load())
			},
		},
		{
			name: "priority is Priority_LEVEL4",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, seedPeerClient standard.SeedPeer, mr *standard.MockResourceMockRecorder, ms *standard.MockSeedPeerMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mr.SeedPeer().Return(seedPeerClient).Times(1),
					ms.TriggerDownloadTask(gomock.All(), gomock.Any(), gomock.Any()).Do(func(context.Context, string, *dfdaemonv2.DownloadTaskRequest) { wg.Done() }).Return(nil).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL4

				assert := assert.New(t)
				assert.NoError(svc.downloadTaskBySeedPeer(context.Background(), mockTaskID, &commonv2.Download{}, peer))
				assert.False(peer.NeedBackToSource.Load())
			},
		},
		{
			name: "priority is Priority_LEVEL4 and download task failed",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, seedPeerClient standard.SeedPeer, mr *standard.MockResourceMockRecorder, ms *standard.MockSeedPeerMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mr.SeedPeer().Return(seedPeerClient).Times(1),
					ms.TriggerDownloadTask(gomock.All(), gomock.Any(), gomock.Any()).Do(func(context.Context, string, *dfdaemonv2.DownloadTaskRequest) { wg.Done() }).Return(errors.New("foo")).Times(1),
				)

				peer.Priority = commonv2.Priority_LEVEL4

				assert := assert.New(t)
				assert.NoError(svc.downloadTaskBySeedPeer(context.Background(), mockTaskID, &commonv2.Download{}, peer))
				assert.False(peer.NeedBackToSource.Load())
			},
		},
		{
			name: "priority is Priority_LEVEL3",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, seedPeerClient standard.SeedPeer, mr *standard.MockResourceMockRecorder, ms *standard.MockSeedPeerMockRecorder) {
				peer.Priority = commonv2.Priority_LEVEL3

				assert := assert.New(t)
				assert.NoError(svc.downloadTaskBySeedPeer(context.Background(), mockTaskID, &commonv2.Download{}, peer))
				assert.True(peer.NeedBackToSource.Load())
			},
		},
		{
			name: "priority is Priority_LEVEL2",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, seedPeerClient standard.SeedPeer, mr *standard.MockResourceMockRecorder, ms *standard.MockSeedPeerMockRecorder) {
				assert := assert.New(t)
				peer.Priority = commonv2.Priority_LEVEL2

				assert.ErrorIs(svc.downloadTaskBySeedPeer(context.Background(), mockTaskID, &commonv2.Download{}, peer), status.Errorf(codes.NotFound, "%s peer not found candidate peers", commonv2.Priority_LEVEL2.String()))
			},
		},
		{
			name: "priority is Priority_LEVEL1",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, seedPeerClient standard.SeedPeer, mr *standard.MockResourceMockRecorder, ms *standard.MockSeedPeerMockRecorder) {
				assert := assert.New(t)
				peer.Priority = commonv2.Priority_LEVEL1

				assert.ErrorIs(svc.downloadTaskBySeedPeer(context.Background(), mockTaskID, &commonv2.Download{}, peer), status.Errorf(codes.FailedPrecondition, "%s peer is forbidden", commonv2.Priority_LEVEL1.String()))
			},
		},
		{
			name: "priority is Priority_LEVEL0",
			run: func(t *testing.T, svc *V2, peer *standard.Peer, seedPeerClient standard.SeedPeer, mr *standard.MockResourceMockRecorder, ms *standard.MockSeedPeerMockRecorder) {
				assert := assert.New(t)
				peer.Priority = commonv2.Priority(100)

				assert.ErrorIs(svc.downloadTaskBySeedPeer(context.Background(), mockTaskID, &commonv2.Download{}, peer), status.Errorf(codes.InvalidArgument, "invalid priority %#v", peer.Priority))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			seedPeerClient := standard.NewMockSeedPeer(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			peer := standard.NewPeer(mockPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, peer, seedPeerClient, resource.EXPECT(), seedPeerClient.EXPECT())
		})
	}
}

func TestServiceV2_AnnouncePersistentPeer(t *testing.T) {
	tests := []struct {
		name           string
		disablePersist bool
		mock           func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder)
		expect         func(t *testing.T, peer *persistent.Peer, err error)
	}{
		{
			name:           "redis is not enabled",
			disablePersist: true,
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "context was done",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				stream.EXPECT().Context().Return(ctx).Times(1)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, context.Canceled)
			},
		},
		{
			name: "receive error",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(nil, errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "receive EOF",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "register failed and peer for failure handling not found",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentPeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentPeerRequest_RegisterPersistentPeerRequest{
							RegisterPersistentPeerRequest: &schedulerv2.RegisterPersistentPeerRequest{},
						},
					}, nil).Times(1),
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, false).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "peer %s not found", mockPeerID))
			},
		},
		{
			name: "register failed and peer is marked failed",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentPeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentPeerRequest_RegisterPersistentPeerRequest{
							RegisterPersistentPeerRequest: &schedulerv2.RegisterPersistentPeerRequest{},
						},
					}, nil).Times(1),
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, false).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "host %s not found", mockHostID))
				assert.Equal(persistent.PeerStateFailed, peer.FSM.Current())
			},
		},
		{
			name: "download started then stream closed",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				peer.FSM.SetState(persistent.PeerStateReceivedNormal)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentPeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePersistentPeerRequest_DownloadPersistentPeerStartedRequest{},
					}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(persistent.PeerStateRunning, peer.FSM.Current())
			},
		},
		{
			name: "download back-to-source started failed and peer is marked failed",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				peer.FSM.SetState(persistent.PeerStatePending)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentPeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePersistentPeerRequest_DownloadPersistentPeerBackToSourceStartedRequest{},
					}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "event Download inappropriate in current state Pending"))
				assert.Equal(persistent.PeerStateFailed, peer.FSM.Current())
			},
		},
		{
			name: "download finished closes the stream",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				peer.FSM.SetState(persistent.PeerStateRunning)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentPeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePersistentPeerRequest_DownloadPersistentPeerFinishedRequest{},
					}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(persistent.PeerStateSucceeded, peer.FSM.Current())
				assert.NotZero(peer.Cost)
			},
		},
		{
			name: "download failed closes the stream",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				peer.FSM.SetState(persistent.PeerStateRunning)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentPeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePersistentPeerRequest_DownloadPersistentPeerBackToSourceFailedRequest{},
					}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(persistent.PeerStateFailed, peer.FSM.Current())
			},
		},
		{
			name: "download piece finished error is logged and stream continues",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentPeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentPeerRequest_DownloadPieceFinishedRequest{
							DownloadPieceFinishedRequest: &schedulerv2.DownloadPieceFinishedRequest{Piece: &commonv2.Piece{Number: 1}},
						},
					}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "download piece back-to-source finished updates finished pieces",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentPeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentPeerRequest_DownloadPieceBackToSourceFinishedRequest{
							DownloadPieceBackToSourceFinishedRequest: &schedulerv2.DownloadPieceBackToSourceFinishedRequest{Piece: &commonv2.Piece{Number: 1}},
						},
					}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.True(peer.FinishedPieces.Test(1))
				assert.Equal(uint(1), peer.FinishedPieces.Count())
			},
		},
		{
			name: "download piece failed with temporary error blocks parent",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentPeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentPeerRequest_DownloadPieceFailedRequest{
							DownloadPieceFailedRequest: &schedulerv2.DownloadPieceFailedRequest{ParentId: mockSeedPeerID, Temporary: true},
						},
					}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(standardPeerManager).Times(1),
					mp.Load(gomock.Eq(mockSeedPeerID)).Return(nil, false).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.ElementsMatch([]string{mockPeerID, mockSeedPeerID}, peer.BlockParents)
			},
		},
		{
			name: "download piece back-to-source failed is logged and stream continues",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentPeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentPeerRequest_DownloadPieceBackToSourceFailedRequest{
							DownloadPieceBackToSourceFailedRequest: &schedulerv2.DownloadPieceBackToSourceFailedRequest{},
						},
					}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "unknown request",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, hostManager persistent.HostManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mph *persistent.MockHostManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentPeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
					}, nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.Equal(codes.FailedPrecondition, status.Code(err))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)
			hostManager := persistent.NewMockHostManager(ctl)
			stream := schedulerv2mocks.NewMockScheduler_AnnouncePersistentPeerServer(ctl)
			standardPeerManager := standard.NewMockPeerManager(ctl)

			peer := mockPersistentPeer(mockPeerID, persistent.PeerStateReceivedNormal, mockPersistentTask(persistent.TaskStateSucceeded), mockPersistentHost())
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersist {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, nil, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			}

			tc.mock(peer, peerManager, hostManager, stream, standardPeerManager, persistentResource.EXPECT(), peerManager.EXPECT(), hostManager.EXPECT(), resource.EXPECT(), standardPeerManager.EXPECT())
			tc.expect(t, peer, svc.AnnouncePersistentPeer(stream))
		})
	}
}

func TestServiceV2_handleRegisterPersistentPeerRequest(t *testing.T) {
	concurrentPieceCount := uint32(7)

	tests := []struct {
		name                 string
		needBackToSource     bool
		concurrentPieceCount *uint32
		mock                 func(host *persistent.Host, task *persistent.Task, parent *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder)
		expect               func(t *testing.T, err error)
	}{
		{
			name: "host not found",
			mock: func(host *persistent.Host, task *persistent.Task, parent *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "host %s not found", mockHostID))
			},
		},
		{
			name: "task not found",
			mock: func(host *persistent.Host, task *persistent.Task, parent *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "task %s not found", mockTaskID))
			},
		},
		{
			name: "load available peer ids failed",
			mock: func(host *persistent.Host, task *persistent.Task, parent *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllIDsByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "load persistent peers failed"))
			},
		},
		{
			name:                 "need back-to-source and store new peer with concurrent piece count",
			needBackToSource:     true,
			concurrentPieceCount: &concurrentPieceCount,
			mock: func(host *persistent.Host, task *persistent.Task, parent *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllIDsByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]string{mockSeedPeerID}, nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(task)).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Cond(func(peer *persistent.Peer) bool {
						return peer.ID == mockPeerID && peer.Persistent && peer.ConcurrentPieceCount == 7 && peer.FSM.Is(persistent.PeerStateReceivedNormal) && peer.Task == task && peer.Host == host
					})).Return(nil).Times(1),
					stream.EXPECT().Send(gomock.Eq(&schedulerv2.AnnouncePersistentPeerResponse{
						Response: &schedulerv2.AnnouncePersistentPeerResponse_NeedBackToSourceResponse{
							NeedBackToSourceResponse: &schedulerv2.NeedBackToSourceResponse{},
						},
					})).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "task state is TaskStatePending triggers back-to-source and send failed",
			mock: func(host *persistent.Host, task *persistent.Task, parent *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				task.FSM.SetState(persistent.TaskStatePending)
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllIDsByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]string{mockSeedPeerID}, nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(task)).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Any()).Return(nil).Times(1),
					stream.EXPECT().Send(gomock.Any()).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "task state is TaskStateSucceeded without available peers and store task failed",
			mock: func(host *persistent.Host, task *persistent.Task, parent *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllIDsByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]string{}, nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(task)).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
			},
		},
		{
			name: "size scope is SizeScope_EMPTY",
			mock: func(host *persistent.Host, task *persistent.Task, parent *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				task.ContentLength = 0
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllIDsByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]string{mockSeedPeerID}, nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(task)).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Cond(func(peer *persistent.Peer) bool { return peer.FSM.Is(persistent.PeerStateReceivedEmpty) })).Return(nil).Times(1),
					stream.EXPECT().Send(gomock.Eq(&schedulerv2.AnnouncePersistentPeerResponse{
						Response: &schedulerv2.AnnouncePersistentPeerResponse_EmptyPersistentTaskResponse{
							EmptyPersistentTaskResponse: &schedulerv2.EmptyPersistentTaskResponse{},
						},
					})).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "size scope is SizeScope_EMPTY and store peer failed",
			mock: func(host *persistent.Host, task *persistent.Task, parent *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				task.ContentLength = 0
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllIDsByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]string{mockSeedPeerID}, nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(task)).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Any()).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "size scope is SizeScope_NORMAL and no candidate parents found",
			mock: func(host *persistent.Host, task *persistent.Task, parent *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllIDsByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]string{mockSeedPeerID}, nil).Times(1),
					ms.FindCandidatePersistentParents(gomock.Any(), gomock.Any(), gomock.Cond(func(blocklist set.SafeSet[string]) bool { return blocklist.Contains(mockPeerID) })).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "no candidate parents found"))
			},
		},
		{
			name: "size scope is SizeScope_NORMAL and load current replica count failed",
			mock: func(host *persistent.Host, task *persistent.Task, parent *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllIDsByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]string{mockSeedPeerID}, nil).Times(1),
					ms.FindCandidatePersistentParents(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*persistent.Peer{parent}, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "size scope is SizeScope_NORMAL and candidate parents are sent",
			mock: func(host *persistent.Host, task *persistent.Task, parent *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllIDsByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]string{mockSeedPeerID}, nil).Times(1),
					ms.FindCandidatePersistentParents(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*persistent.Peer{parent}, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(2), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(task)).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Cond(func(peer *persistent.Peer) bool {
						return peer.ID == mockPeerID && peer.FSM.Is(persistent.PeerStateReceivedNormal)
					})).Return(nil).Times(1),
					stream.EXPECT().Send(gomock.Cond(func(resp *schedulerv2.AnnouncePersistentPeerResponse) bool {
						normal, ok := resp.GetResponse().(*schedulerv2.AnnouncePersistentPeerResponse_NormalPersistentTaskResponse)
						if !ok || len(normal.NormalPersistentTaskResponse.CandidateParents) != 1 {
							return false
						}

						candidate := normal.NormalPersistentTaskResponse.CandidateParents[0]
						return candidate.Id == mockSeedPeerID && candidate.Task.Id == mockTaskID && candidate.Task.CurrentPersistentReplicaCount == 1 && candidate.Task.CurrentReplicaCount == 2 && candidate.Host.Id == mockHostID
					})).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			hostManager := persistent.NewMockHostManager(ctl)
			taskManager := persistent.NewMockTaskManager(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)
			stream := schedulerv2mocks.NewMockScheduler_AnnouncePersistentPeerServer(ctl)

			host := mockPersistentHost()
			task := mockPersistentTask(persistent.TaskStateSucceeded)
			parent := mockPersistentPeer(mockSeedPeerID, persistent.PeerStateSucceeded, task, host)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(host, task, parent, hostManager, taskManager, peerManager, stream, persistentResource.EXPECT(), hostManager.EXPECT(), taskManager.EXPECT(), peerManager.EXPECT(), scheduling.EXPECT())
			tc.expect(t, svc.handleRegisterPersistentPeerRequest(context.Background(), stream, mockHostID, mockTaskID, mockPeerID, true, tc.concurrentPieceCount, tc.needBackToSource))
		})
	}
}

func TestServiceV2_handleReschedulePersistentPeerRequest(t *testing.T) {
	tests := []struct {
		name   string
		req    *schedulerv2.ReschedulePersistentPeerRequest
		mock   func(peer *persistent.Peer, parent *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder)
		expect func(t *testing.T, peer *persistent.Peer, err error)
	}{
		{
			name: "peer not found",
			req:  &schedulerv2.ReschedulePersistentPeerRequest{},
			mock: func(peer *persistent.Peer, parent *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "peer %s not found", mockPeerID))
			},
		},
		{
			name: "no candidate parents found and reported parents are blocked",
			req: &schedulerv2.ReschedulePersistentPeerRequest{
				CandidateParents: []*commonv2.PersistentPeer{{Id: "bar"}},
			},
			mock: func(peer *persistent.Peer, parent *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.BlockParents = []string{"foo"}
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					ms.FindCandidatePersistentParents(gomock.Any(), gomock.Eq(peer), gomock.Cond(func(blocklist set.SafeSet[string]) bool {
						return blocklist.Contains(mockPeerID, "foo", "bar") && blocklist.Len() == 3
					})).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "no candidate parents found"))
				assert.ElementsMatch([]string{mockPeerID, "foo", "bar"}, peer.BlockParents)
			},
		},
		{
			name: "load current persistent replica count failed",
			req:  &schedulerv2.ReschedulePersistentPeerRequest{},
			mock: func(peer *persistent.Peer, parent *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					ms.FindCandidatePersistentParents(gomock.Any(), gomock.Eq(peer), gomock.Any()).Return([]*persistent.Peer{parent}, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "store peer failed",
			req:  &schedulerv2.ReschedulePersistentPeerRequest{},
			mock: func(peer *persistent.Peer, parent *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					ms.FindCandidatePersistentParents(gomock.Any(), gomock.Eq(peer), gomock.Any()).Return([]*persistent.Peer{parent}, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(2), nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
			},
		},
		{
			name: "candidate parents are sent",
			req:  &schedulerv2.ReschedulePersistentPeerRequest{},
			mock: func(peer *persistent.Peer, parent *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentPeerServer, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					ms.FindCandidatePersistentParents(gomock.Any(), gomock.Eq(peer), gomock.Any()).Return([]*persistent.Peer{parent}, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(2), nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					stream.EXPECT().Send(gomock.Cond(func(resp *schedulerv2.AnnouncePersistentPeerResponse) bool {
						normal, ok := resp.GetResponse().(*schedulerv2.AnnouncePersistentPeerResponse_NormalPersistentTaskResponse)
						return ok && len(normal.NormalPersistentTaskResponse.CandidateParents) == 1 && normal.NormalPersistentTaskResponse.CandidateParents[0].Id == mockSeedPeerID
					})).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.ElementsMatch([]string{mockPeerID}, peer.BlockParents)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			taskManager := persistent.NewMockTaskManager(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)
			stream := schedulerv2mocks.NewMockScheduler_AnnouncePersistentPeerServer(ctl)

			host := mockPersistentHost()
			task := mockPersistentTask(persistent.TaskStateSucceeded)
			peer := mockPersistentPeer(mockPeerID, persistent.PeerStateRunning, task, host)
			parent := mockPersistentPeer(mockSeedPeerID, persistent.PeerStateSucceeded, task, host)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(peer, parent, taskManager, peerManager, stream, persistentResource.EXPECT(), taskManager.EXPECT(), peerManager.EXPECT(), scheduling.EXPECT())
			tc.expect(t, peer, svc.handleReschedulePersistentPeerRequest(context.Background(), stream, mockTaskID, mockPeerID, tc.req))
		})
	}
}

func TestServiceV2_handleDownloadPersistentPieceFinishedRequest(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(peer *persistent.Peer, parent *persistent.Peer, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder)
		expect func(t *testing.T, peer *persistent.Peer, parent *persistent.Peer, err error)
	}{
		{
			name: "peer not found",
			mock: func(peer *persistent.Peer, parent *persistent.Peer, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, parent *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "peer %s not found", mockPeerID))
			},
		},
		{
			name: "store peer failed",
			mock: func(peer *persistent.Peer, parent *persistent.Peer, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, parent *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
				assert.True(peer.FinishedPieces.Test(1))
			},
		},
		{
			name: "parent not found",
			mock: func(peer *persistent.Peer, parent *persistent.Peer, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockSeedPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, parent *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "parent peer %s not found", mockSeedPeerID))
			},
		},
		{
			name: "store parent failed",
			mock: func(peer *persistent.Peer, parent *persistent.Peer, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockSeedPeerID)).Return(parent, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(parent)).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, parent *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
			},
		},
		{
			name: "peer and parent are updated",
			mock: func(peer *persistent.Peer, parent *persistent.Peer, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockSeedPeerID)).Return(parent, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(parent)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, parent *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.True(peer.FinishedPieces.Test(1))
				assert.Equal(uint(1), peer.FinishedPieces.Count())
				assert.True(parent.UpdatedAt.After(parent.CreatedAt))
				assert.True(parent.Host.UpdatedAt.After(parent.Host.CreatedAt))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)

			host := mockPersistentHost()
			task := mockPersistentTask(persistent.TaskStateSucceeded)
			peer := mockPersistentPeer(mockPeerID, persistent.PeerStateRunning, task, host)
			parent := mockPersistentPeer(mockSeedPeerID, persistent.PeerStateSucceeded, task, host)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(peer, parent, peerManager, persistentResource.EXPECT(), peerManager.EXPECT())
			tc.expect(t, peer, parent, svc.handleDownloadPersistentPieceFinishedRequest(context.Background(), mockPeerID, &schedulerv2.DownloadPieceFinishedRequest{
				Piece: &commonv2.Piece{Number: 1, ParentId: &mockSeedPeerID},
			}))
		})
	}
}

func TestServiceV2_handleDownloadPersistentPieceFailedRequest(t *testing.T) {
	tests := []struct {
		name   string
		req    *schedulerv2.DownloadPieceFailedRequest
		mock   func(peer *persistent.Peer, parent *standard.Peer, peerManager persistent.PeerManager, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder)
		expect func(t *testing.T, peer *persistent.Peer, parent *standard.Peer, err error)
	}{
		{
			name: "peer not found",
			req:  &schedulerv2.DownloadPieceFailedRequest{ParentId: mockSeedPeerID, Temporary: true},
			mock: func(peer *persistent.Peer, parent *standard.Peer, peerManager persistent.PeerManager, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, parent *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "peer %s not found", mockPeerID))
			},
		},
		{
			name: "temporary is false",
			req:  &schedulerv2.DownloadPieceFailedRequest{ParentId: mockSeedPeerID, Temporary: false},
			mock: func(peer *persistent.Peer, parent *standard.Peer, peerManager persistent.PeerManager, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, parent *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "download piece failed"))
				assert.Empty(peer.BlockParents)
			},
		},
		{
			name: "temporary is true and parent is a standard peer",
			req:  &schedulerv2.DownloadPieceFailedRequest{ParentId: mockSeedPeerID, Temporary: true},
			mock: func(peer *persistent.Peer, parent *standard.Peer, peerManager persistent.PeerManager, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				peer.BlockParents = []string{"foo"}
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(standardPeerManager).Times(1),
					mp.Load(gomock.Eq(mockSeedPeerID)).Return(parent, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, parent *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.ElementsMatch([]string{mockPeerID, "foo", mockSeedPeerID}, peer.BlockParents)
				assert.Equal(int64(1), parent.Host.UploadFailedCount.Load())
			},
		},
		{
			name: "temporary is true and store peer failed",
			req:  &schedulerv2.DownloadPieceFailedRequest{ParentId: mockSeedPeerID, Temporary: true},
			mock: func(peer *persistent.Peer, parent *standard.Peer, peerManager persistent.PeerManager, standardPeerManager standard.PeerManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(standardPeerManager).Times(1),
					mp.Load(gomock.Eq(mockSeedPeerID)).Return(nil, false).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, parent *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
				assert.Equal(int64(0), parent.Host.UploadFailedCount.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)
			standardPeerManager := standard.NewMockPeerManager(ctl)

			peer := mockPersistentPeer(mockPeerID, persistent.PeerStateRunning, mockPersistentTask(persistent.TaskStateSucceeded), mockPersistentHost())
			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			parent := standard.NewPeer(mockSeedPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(peer, parent, peerManager, standardPeerManager, persistentResource.EXPECT(), peerManager.EXPECT(), resource.EXPECT(), standardPeerManager.EXPECT())
			tc.expect(t, peer, parent, svc.handleDownloadPersistentPieceFailedRequest(context.Background(), mockPeerID, tc.req))
		})
	}
}

func TestServiceV2_StatPersistentPeer(t *testing.T) {
	tests := []struct {
		name           string
		disablePersist bool
		mock           func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder)
		expect         func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentPeer, err error)
	}{
		{
			name:           "redis is not enabled",
			disablePersist: true,
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentPeer, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "peer not found",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentPeer, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "persistent peer %s not found", mockPeerID))
			},
		},
		{
			name: "load current persistent replica count failed",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentPeer, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "load current replica count failed",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentPeer, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
			},
		},
		{
			name: "peer has been loaded",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(3), nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentPeer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(mockPeerID, resp.Id)
				assert.True(resp.Persistent)
				assert.Equal(persistent.PeerStateSucceeded, resp.State)
				assert.Equal(mockTaskID, resp.Task.Id)
				assert.Equal(uint64(2), resp.Task.PersistentReplicaCount)
				assert.Equal(uint64(1), resp.Task.CurrentPersistentReplicaCount)
				assert.Equal(uint64(3), resp.Task.CurrentReplicaCount)
				assert.Equal(uint64(1024), resp.Task.ContentLength)
				assert.Equal(uint32(2), resp.Task.PieceCount)
				assert.Equal(persistent.TaskStateSucceeded, resp.Task.State)
				assert.Equal(mockHostID, resp.Host.Id)
				assert.Equal(mockRawPersistentHost.IP, resp.Host.Ip)
				assert.Equal(mockRawPersistentHost.DownloadPort, resp.Host.DownloadPort)
				assert.Equal(uint64(1), resp.Host.SchedulerClusterId)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)
			taskManager := persistent.NewMockTaskManager(ctl)

			peer := mockPersistentPeer(mockPeerID, persistent.PeerStateSucceeded, mockPersistentTask(persistent.TaskStateSucceeded), mockPersistentHost())
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig, Manager: config.ManagerConfig{SchedulerClusterID: 1}}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersist {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, nil, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			}

			tc.mock(peer, peerManager, taskManager, persistentResource.EXPECT(), peerManager.EXPECT(), taskManager.EXPECT())
			resp, err := svc.StatPersistentPeer(context.Background(), &schedulerv2.StatPersistentPeerRequest{HostId: mockHostID, TaskId: mockTaskID, PeerId: mockPeerID})
			tc.expect(t, peer, resp, err)
		})
	}
}

func TestServiceV2_DeletePersistentPeer(t *testing.T) {
	tests := []struct {
		name           string
		disablePersist bool
		mock           func(wg *sync.WaitGroup, peer *persistent.Peer, peerManager persistent.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder)
		expect         func(t *testing.T, err error)
	}{
		{
			name:           "redis is not enabled",
			disablePersist: true,
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, peerManager persistent.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "peer not found",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, peerManager persistent.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "persistent peer %s not found", mockPeerID))
			},
		},
		{
			name: "delete peer failed",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, peerManager persistent.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "get dfdaemon client failed",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, peerManager persistent.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil).Times(1),
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(nil, errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "delete persistent task from peer failed is tolerated and task is replicated",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, peerManager persistent.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				wg.Add(1)
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil).Times(1),
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().DeletePersistentTask(gomock.Any(), gomock.Cond(func(req *dfdaemonv2.DeletePersistentTaskRequest) bool { return req.TaskId == mockTaskID })).Return(errors.New("baz")).Times(1),
					ms.FindReplicatePersistentHosts(gomock.Any(), gomock.Eq(peer.Task), gomock.Cond(func(blocklist set.SafeSet[string]) bool { return blocklist.Contains(mockHostID) })).Do(func(context.Context, *persistent.Task, set.SafeSet[string]) { wg.Done() }).Return(nil, nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)
			pool := dfdaemonclientmocks.NewMockPool(ctl)
			client := dfdaemonclientmocks.NewMockClient(ctl)

			peer := mockPersistentPeer(mockPeerID, persistent.PeerStateSucceeded, mockPersistentTask(persistent.TaskStateSucceeded), mockPersistentHost())
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersist {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, nil, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			}

			var wg sync.WaitGroup
			tc.mock(&wg, peer, peerManager, pool, client, resource.EXPECT(), persistentResource.EXPECT(), peerManager.EXPECT(), scheduling.EXPECT())
			err := svc.DeletePersistentPeer(context.Background(), &schedulerv2.DeletePersistentPeerRequest{HostId: mockHostID, TaskId: mockTaskID, PeerId: mockPeerID})
			wg.Wait()
			tc.expect(t, err)
		})
	}
}

func TestServiceV2_UploadPersistentTaskStarted(t *testing.T) {
	tests := []struct {
		name           string
		disablePersist bool
		mock           func(host *persistent.Host, task *persistent.Task, peer *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder)
		expect         func(t *testing.T, err error)
	}{
		{
			name:           "redis is not enabled",
			disablePersist: true,
			mock: func(host *persistent.Host, task *persistent.Task, peer *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "host not found",
			mock: func(host *persistent.Host, task *persistent.Task, peer *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "host %s not found", mockHostID))
			},
		},
		{
			name: "task already exists and is uploading",
			mock: func(host *persistent.Host, task *persistent.Task, peer *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				task.FSM.SetState(persistent.TaskStateUploading)
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.AlreadyExists, "persistent task %s is %s cannot upload", mockTaskID, persistent.TaskStateUploading))
			},
		},
		{
			name: "task already exists and is failed can be uploaded again",
			mock: func(host *persistent.Host, task *persistent.Task, peer *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				task.FSM.SetState(persistent.TaskStateFailed)
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Cond(func(task *persistent.Task) bool {
						return task.ID == mockTaskID && task.FSM.Is(persistent.TaskStateUploading) && task.URL == mockTaskURL && task.PersistentReplicaCount == 3 && task.ContentLength == 4096 && task.TotalPieceCount == 4 && task.TTL == time.Hour
					})).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Cond(func(peer *persistent.Peer) bool {
						return peer.ID == mockPeerID && peer.Persistent && peer.FSM.Is(persistent.PeerStateUploading) && peer.Task.ID == mockTaskID && peer.Host.ID == mockHostID && peer.FinishedPieces.Len() == 4
					})).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "store task failed",
			mock: func(host *persistent.Host, task *persistent.Task, peer *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, false).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Any()).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "peer already exists",
			mock: func(host *persistent.Host, task *persistent.Task, peer *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, false).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Any()).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.AlreadyExists, "persistent peer %s already exists", mockPeerID))
			},
		},
		{
			name: "store peer failed",
			mock: func(host *persistent.Host, task *persistent.Task, peer *persistent.Peer, hostManager persistent.HostManager, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mph *persistent.MockHostManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, false).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Any()).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Any()).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			hostManager := persistent.NewMockHostManager(ctl)
			taskManager := persistent.NewMockTaskManager(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)

			host := mockPersistentHost()
			task := mockPersistentTask(persistent.TaskStatePending)
			peer := mockPersistentPeer(mockPeerID, persistent.PeerStatePending, task, host)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersist {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, nil, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			}

			tc.mock(host, task, peer, hostManager, taskManager, peerManager, persistentResource.EXPECT(), hostManager.EXPECT(), taskManager.EXPECT(), peerManager.EXPECT())
			tc.expect(t, svc.UploadPersistentTaskStarted(context.Background(), &schedulerv2.UploadPersistentTaskStartedRequest{
				HostId:                 mockHostID,
				TaskId:                 mockTaskID,
				PeerId:                 mockPeerID,
				Url:                    mockTaskURL,
				PersistentReplicaCount: 3,
				ContentLength:          4096,
				PieceCount:             4,
				Ttl:                    durationpb.New(time.Hour),
			}))
		})
	}
}

func TestServiceV2_UploadPersistentTaskFinished(t *testing.T) {
	tests := []struct {
		name           string
		disablePersist bool
		mock           func(wg *sync.WaitGroup, peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder)
		expect         func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentTask, err error)
	}{
		{
			name:           "redis is not enabled",
			disablePersist: true,
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "peer not found",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "persistent peer %s not found", mockPeerID))
			},
		},
		{
			name: "peer state is PeerStateSucceeded",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(persistent.PeerStateSucceeded)
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "event Succeeded inappropriate in current state Succeeded"))
			},
		},
		{
			name: "store peer failed",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
				assert.Equal(persistent.PeerStateSucceeded, peer.FSM.Current())
				assert.True(peer.FinishedPieces.All())
			},
		},
		{
			name: "task state is TaskStateSucceeded",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.Task.FSM.SetState(persistent.TaskStateSucceeded)
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "event Succeeded inappropriate in current state Succeeded"))
			},
		},
		{
			name: "store task failed",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(peer.Task)).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
				assert.Equal(persistent.TaskStateSucceeded, peer.Task.FSM.Current())
			},
		},
		{
			name: "load current persistent replica count failed",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(peer.Task)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("baz")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "baz"))
			},
		},
		{
			name: "load current replica count failed",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(peer.Task)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("qux")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "qux"))
			},
		},
		{
			name: "upload finished and task is replicated",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				wg.Add(1)
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(peer.Task)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					ms.FindReplicatePersistentHosts(gomock.Any(), gomock.Eq(peer.Task), gomock.Cond(func(blocklist set.SafeSet[string]) bool { return blocklist.Contains(mockHostID) })).Do(func(context.Context, *persistent.Task, set.SafeSet[string]) { wg.Done() }).Return(nil, nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(persistent.PeerStateSucceeded, peer.FSM.Current())
				assert.Equal(persistent.TaskStateSucceeded, peer.Task.FSM.Current())
				assert.True(peer.FinishedPieces.All())
				assert.NotZero(peer.Cost)
				assert.Equal(mockTaskID, resp.Id)
				assert.Equal(uint64(2), resp.PersistentReplicaCount)
				assert.Equal(uint64(1), resp.CurrentPersistentReplicaCount)
				assert.Equal(uint64(1), resp.CurrentReplicaCount)
				assert.Equal(uint64(1024), resp.ContentLength)
				assert.Equal(uint32(2), resp.PieceCount)
				assert.Equal(persistent.TaskStateSucceeded, resp.State)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			taskManager := persistent.NewMockTaskManager(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)

			peer := mockPersistentPeer(mockPeerID, persistent.PeerStateUploading, mockPersistentTask(persistent.TaskStateUploading), mockPersistentHost())
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersist {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, nil, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			}

			var wg sync.WaitGroup
			tc.mock(&wg, peer, taskManager, peerManager, persistentResource.EXPECT(), taskManager.EXPECT(), peerManager.EXPECT(), scheduling.EXPECT())
			resp, err := svc.UploadPersistentTaskFinished(context.Background(), &schedulerv2.UploadPersistentTaskFinishedRequest{HostId: mockHostID, TaskId: mockTaskID, PeerId: mockPeerID})
			wg.Wait()
			tc.expect(t, peer, resp, err)
		})
	}
}

func TestServiceV2_UploadPersistentTaskFailed(t *testing.T) {
	tests := []struct {
		name           string
		disablePersist bool
		mock           func(peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder)
		expect         func(t *testing.T, peer *persistent.Peer, err error)
	}{
		{
			name:           "redis is not enabled",
			disablePersist: true,
			mock: func(peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "peer not found",
			mock: func(peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "persistent peer %s not found", mockPeerID))
			},
		},
		{
			name: "peer state is PeerStateSucceeded",
			mock: func(peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				peer.FSM.SetState(persistent.PeerStateSucceeded)
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "event Failed inappropriate in current state Succeeded"))
				assert.Equal(persistent.PeerStateSucceeded, peer.FSM.Current())
			},
		},
		{
			name: "store peer failed",
			mock: func(peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
				assert.Equal(persistent.PeerStateFailed, peer.FSM.Current())
			},
		},
		{
			name: "task state is TaskStatePending",
			mock: func(peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				peer.Task.FSM.SetState(persistent.TaskStatePending)
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "event Failed inappropriate in current state Pending"))
				assert.Equal(persistent.PeerStateFailed, peer.FSM.Current())
				assert.Equal(persistent.TaskStatePending, peer.Task.FSM.Current())
			},
		},
		{
			name: "store task failed",
			mock: func(peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(peer.Task)).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
				assert.Equal(persistent.TaskStateFailed, peer.Task.FSM.Current())
			},
		},
		{
			name: "peer failed and task marked failed",
			mock: func(peer *persistent.Peer, taskManager persistent.TaskManager, peerManager persistent.PeerManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(peer.Task)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(persistent.PeerStateFailed, peer.FSM.Current())
				assert.Equal(persistent.TaskStateFailed, peer.Task.FSM.Current())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			taskManager := persistent.NewMockTaskManager(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)

			peer := mockPersistentPeer(mockPeerID, persistent.PeerStateUploading, mockPersistentTask(persistent.TaskStateUploading), mockPersistentHost())
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersist {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, nil, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			}

			tc.mock(peer, taskManager, peerManager, persistentResource.EXPECT(), taskManager.EXPECT(), peerManager.EXPECT())
			tc.expect(t, peer, svc.UploadPersistentTaskFailed(context.Background(), &schedulerv2.UploadPersistentTaskFailedRequest{HostId: mockHostID, TaskId: mockTaskID, PeerId: mockPeerID}))
		})
	}
}

func TestServiceV2_StatPersistentTask(t *testing.T) {
	tests := []struct {
		name           string
		disablePersist bool
		mock           func(task *persistent.Task, taskManager persistent.TaskManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder)
		expect         func(t *testing.T, resp *commonv2.PersistentTask, err error)
	}{
		{
			name:           "redis is not enabled",
			disablePersist: true,
			mock: func(task *persistent.Task, taskManager persistent.TaskManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
			},
			expect: func(t *testing.T, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "task not found",
			mock: func(task *persistent.Task, taskManager persistent.TaskManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "persistent task %s not found", mockTaskID))
			},
		},
		{
			name: "load current persistent replica count failed",
			mock: func(task *persistent.Task, taskManager persistent.TaskManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "load current replica count failed",
			mock: func(task *persistent.Task, taskManager persistent.TaskManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
			},
		},
		{
			name: "task has been loaded",
			mock: func(task *persistent.Task, taskManager persistent.TaskManager, mpr *persistent.MockResourceMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(3), nil).Times(1),
				)
			},
			expect: func(t *testing.T, resp *commonv2.PersistentTask, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(mockTaskID, resp.Id)
				assert.Equal(uint64(2), resp.PersistentReplicaCount)
				assert.Equal(uint64(1), resp.CurrentPersistentReplicaCount)
				assert.Equal(uint64(3), resp.CurrentReplicaCount)
				assert.Equal(uint64(1024), resp.ContentLength)
				assert.Equal(uint32(2), resp.PieceCount)
				assert.Equal(persistent.TaskStateSucceeded, resp.State)
				assert.Equal(time.Hour, resp.Ttl.AsDuration())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			taskManager := persistent.NewMockTaskManager(ctl)

			task := mockPersistentTask(persistent.TaskStateSucceeded)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersist {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, nil, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			}

			tc.mock(task, taskManager, persistentResource.EXPECT(), taskManager.EXPECT())
			resp, err := svc.StatPersistentTask(context.Background(), &schedulerv2.StatPersistentTaskRequest{HostId: mockHostID, TaskId: mockTaskID})
			tc.expect(t, resp, err)
		})
	}
}

func TestServiceV2_DeletePersistentTask(t *testing.T) {
	tests := []struct {
		name           string
		disablePersist bool
		mock           func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder)
		expect         func(t *testing.T, err error)
	}{
		{
			name:           "redis is not enabled",
			disablePersist: true,
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "load peers by task failed",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "delete peer failed is skipped and delete task failed",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]*persistent.Peer{peer}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(errors.New("bar")).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Delete(gomock.Any(), gomock.Eq(mockTaskID)).Return(errors.New("baz")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "baz"))
			},
		},
		{
			name: "get dfdaemon client failed is skipped",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]*persistent.Peer{peer}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil).Times(1),
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(nil, errors.New("foo")).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Delete(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "delete persistent task from peer failed is skipped",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]*persistent.Peer{peer}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil).Times(1),
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().DeletePersistentTask(gomock.Any(), gomock.Any()).Return(errors.New("foo")).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Delete(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "delete task and peers succeeded",
			mock: func(peer *persistent.Peer, peerManager persistent.PeerManager, taskManager persistent.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder, mpt *persistent.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]*persistent.Peer{peer}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil).Times(1),
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().DeletePersistentTask(gomock.Any(), gomock.Cond(func(req *dfdaemonv2.DeletePersistentTaskRequest) bool { return req.TaskId == mockTaskID })).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Delete(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)
			taskManager := persistent.NewMockTaskManager(ctl)
			pool := dfdaemonclientmocks.NewMockPool(ctl)
			client := dfdaemonclientmocks.NewMockClient(ctl)

			peer := mockPersistentPeer(mockPeerID, persistent.PeerStateSucceeded, mockPersistentTask(persistent.TaskStateSucceeded), mockPersistentHost())
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersist {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, nil, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			}

			tc.mock(peer, peerManager, taskManager, pool, client, resource.EXPECT(), persistentResource.EXPECT(), peerManager.EXPECT(), taskManager.EXPECT())
			tc.expect(t, svc.DeletePersistentTask(context.Background(), &schedulerv2.DeletePersistentTaskRequest{HostId: mockHostID, TaskId: mockTaskID}))
		})
	}
}

func TestServiceV2_replicatePersistentTask(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(wg *sync.WaitGroup, peer *persistent.Peer, cachedParent *persistent.Peer, host *persistent.Host, pool *dfdaemonclientmocks.MockPool, mr *standard.MockResourceMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder)
		expect func(t *testing.T, err error)
	}{
		{
			name: "no replicate hosts found",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, cachedParent *persistent.Peer, host *persistent.Host, pool *dfdaemonclientmocks.MockPool, mr *standard.MockResourceMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				ms.FindReplicatePersistentHosts(gomock.Any(), gomock.Eq(peer.Task), gomock.Any()).Return(nil, nil, false).Times(1)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "replicate to cached parent and host both dial dfdaemon",
			mock: func(wg *sync.WaitGroup, peer *persistent.Peer, cachedParent *persistent.Peer, host *persistent.Host, pool *dfdaemonclientmocks.MockPool, mr *standard.MockResourceMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				wg.Add(2)
				ms.FindReplicatePersistentHosts(gomock.Any(), gomock.Eq(peer.Task), gomock.Any()).Return([]*persistent.Peer{cachedParent}, []*persistent.Host{host}, true).Times(1)
				mr.PeerClientPool().Return(pool).Times(2)
				pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Do(func(string, ...grpc.DialOption) { wg.Done() }).Return(nil, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			pool := dfdaemonclientmocks.NewMockPool(ctl)

			host := mockPersistentHost()
			task := mockPersistentTask(persistent.TaskStateSucceeded)
			peer := mockPersistentPeer(mockPeerID, persistent.PeerStateSucceeded, task, host)
			cachedParent := mockPersistentPeer(mockSeedPeerID, persistent.PeerStateSucceeded, task, host)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			var wg sync.WaitGroup
			tc.mock(&wg, peer, cachedParent, host, pool, resource.EXPECT(), scheduling.EXPECT())
			err := svc.replicatePersistentTask(context.Background(), peer, set.NewSafeSet[string]())
			wg.Wait()
			tc.expect(t, err)
		})
	}
}

func TestServiceV2_downloadPersistentTaskByPeer(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(task *persistent.Task, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadPersistentTaskClient, mr *standard.MockResourceMockRecorder)
		expect func(t *testing.T, err error)
	}{
		{
			name: "get dfdaemon client failed",
			mock: func(task *persistent.Task, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadPersistentTaskClient, mr *standard.MockResourceMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(nil, errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "download persistent task failed",
			mock: func(task *persistent.Task, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadPersistentTaskClient, mr *standard.MockResourceMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().DownloadPersistentTask(gomock.Any(), gomock.Cond(func(req *dfdaemonv2.DownloadPersistentTaskRequest) bool {
						return req.Url == mockTaskURL && req.Persistent && !req.NeedPieceContent
					})).Return(nil, errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "stream receive failed",
			mock: func(task *persistent.Task, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadPersistentTaskClient, mr *standard.MockResourceMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().DownloadPersistentTask(gomock.Any(), gomock.Any()).Return(stream, nil).Times(1),
					stream.EXPECT().Recv().Return(&dfdaemonv2.DownloadPersistentTaskResponse{}, nil).Times(1),
					stream.EXPECT().Recv().Return(nil, errors.New("baz")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "stream finished with EOF",
			mock: func(task *persistent.Task, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadPersistentTaskClient, mr *standard.MockResourceMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().DownloadPersistentTask(gomock.Any(), gomock.Any()).Return(stream, nil).Times(1),
					stream.EXPECT().Recv().Return(&dfdaemonv2.DownloadPersistentTaskResponse{}, nil).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			pool := dfdaemonclientmocks.NewMockPool(ctl)
			client := dfdaemonclientmocks.NewMockClient(ctl)
			stream := dfdaemonv2mocks.NewMockDfdaemonUpload_DownloadPersistentTaskClient(ctl)

			task := mockPersistentTask(persistent.TaskStateSucceeded)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(task, pool, client, stream, resource.EXPECT())
			tc.expect(t, svc.downloadPersistentTaskByPeer(context.Background(), task, mockPersistentHost()))
		})
	}
}

func TestServiceV2_persistPersistentTaskByPeer(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(cachedParent *persistent.Peer, peerManager persistent.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder)
		expect func(t *testing.T, cachedParent *persistent.Peer, err error)
	}{
		{
			name: "get dfdaemon client failed",
			mock: func(cachedParent *persistent.Peer, peerManager persistent.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(nil, errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, cachedParent *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.False(cachedParent.Persistent)
			},
		},
		{
			name: "update persistent task failed",
			mock: func(cachedParent *persistent.Peer, peerManager persistent.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().UpdatePersistentTask(gomock.Any(), gomock.Cond(func(req *dfdaemonv2.UpdatePersistentTaskRequest) bool {
						return req.TaskId == mockTaskID && req.Persistent
					})).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, cachedParent *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.False(cachedParent.Persistent)
			},
		},
		{
			name: "store cached parent failed",
			mock: func(cachedParent *persistent.Peer, peerManager persistent.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().UpdatePersistentTask(gomock.Any(), gomock.Any()).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(cachedParent)).Return(errors.New("baz")).Times(1),
				)
			},
			expect: func(t *testing.T, cachedParent *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.True(cachedParent.Persistent)
			},
		},
		{
			name: "cached parent becomes persistent",
			mock: func(cachedParent *persistent.Peer, peerManager persistent.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistent.MockResourceMockRecorder, mpp *persistent.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().UpdatePersistentTask(gomock.Any(), gomock.Any()).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(cachedParent)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, cachedParent *persistent.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.True(cachedParent.Persistent)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := persistent.NewMockPeerManager(ctl)
			pool := dfdaemonclientmocks.NewMockPool(ctl)
			client := dfdaemonclientmocks.NewMockClient(ctl)

			host := mockPersistentHost()
			task := mockPersistentTask(persistent.TaskStateSucceeded)
			peer := mockPersistentPeer(mockPeerID, persistent.PeerStateSucceeded, task, host)
			cachedParent := mockPersistentPeer(mockSeedPeerID, persistent.PeerStateSucceeded, task, host)
			cachedParent.Persistent = false
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(cachedParent, peerManager, pool, client, resource.EXPECT(), persistentResource.EXPECT(), peerManager.EXPECT())
			tc.expect(t, cachedParent, svc.persistPersistentTaskByPeer(context.Background(), peer, cachedParent))
		})
	}
}

func TestServiceV2_AnnouncePersistentCachePeer(t *testing.T) {
	tests := []struct {
		name                string
		disablePersistCache bool
		mock                func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder)
		expect              func(t *testing.T, peer *persistentcache.Peer, err error)
	}{
		{
			name:                "redis is not enabled",
			disablePersistCache: true,
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "receive error",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(nil, errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "register host not found and running peer marked failed",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(persistentcache.PeerStateRunning)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_RegisterPersistentCachePeerRequest{
							RegisterPersistentCachePeerRequest: &schedulerv2.RegisterPersistentCachePeerRequest{},
						},
					}, nil).Times(1),
					mpcr.HostManager().Return(hostManager).Times(1),
					mpch.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, false).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "host %s not found", mockHostID))
				assert.Equal(persistentcache.PeerStateFailed, peer.FSM.Current())
			},
		},
		{
			name: "register task not found and peer for failure handling not found",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_RegisterPersistentCachePeerRequest{
							RegisterPersistentCachePeerRequest: &schedulerv2.RegisterPersistentCachePeerRequest{},
						},
					}, nil).Times(1),
					mpcr.HostManager().Return(hostManager).Times(1),
					mpch.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpcr.TaskManager().Return(taskManager).Times(1),
					mpct.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, false).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "peer %s not found", mockPeerID))
			},
		},
		{
			name: "register size scope is SizeScope_EMPTY sends EmptyPersistentCacheTaskResponse",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				task.ContentLength = 0
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_RegisterPersistentCachePeerRequest{
							RegisterPersistentCachePeerRequest: &schedulerv2.RegisterPersistentCachePeerRequest{Persistent: true},
						},
					}, nil).Times(1),
					mpcr.HostManager().Return(hostManager).Times(1),
					mpch.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpcr.TaskManager().Return(taskManager).Times(1),
					mpct.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpcr.TaskManager().Return(taskManager).Times(1),
					mpct.Store(gomock.Any(), gomock.Eq(task)).Return(nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Cond(func(peer *persistentcache.Peer) bool {
						return peer.ID == mockPeerID && peer.Persistent && peer.FSM.Is(persistentcache.PeerStateReceivedEmpty)
					})).Return(nil).Times(1),
					stream.EXPECT().Send(gomock.Eq(&schedulerv2.AnnouncePersistentCachePeerResponse{
						Response: &schedulerv2.AnnouncePersistentCachePeerResponse_EmptyPersistentCacheTaskResponse{
							EmptyPersistentCacheTaskResponse: &schedulerv2.EmptyPersistentCacheTaskResponse{},
						},
					})).Return(nil).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "register size scope is SizeScope_NORMAL and no candidate parents found",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(persistentcache.PeerStateRunning)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_RegisterPersistentCachePeerRequest{
							RegisterPersistentCachePeerRequest: &schedulerv2.RegisterPersistentCachePeerRequest{},
						},
					}, nil).Times(1),
					mpcr.HostManager().Return(hostManager).Times(1),
					mpch.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpcr.TaskManager().Return(taskManager).Times(1),
					mpct.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					ms.FindCandidatePersistentCacheParents(gomock.Any(), gomock.Any(), gomock.Cond(func(blocklist set.SafeSet[string]) bool { return blocklist.Contains(mockPeerID) })).Return(nil, false).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "no candidate parents found"))
				assert.Equal(persistentcache.PeerStateFailed, peer.FSM.Current())
			},
		},
		{
			name: "register size scope is SizeScope_NORMAL sends candidate parents with tag and application",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				concurrentPieceCount := uint32(9)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_RegisterPersistentCachePeerRequest{
							RegisterPersistentCachePeerRequest: &schedulerv2.RegisterPersistentCachePeerRequest{ConcurrentPieceCount: &concurrentPieceCount},
						},
					}, nil).Times(1),
					mpcr.HostManager().Return(hostManager).Times(1),
					mpch.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpcr.TaskManager().Return(taskManager).Times(1),
					mpct.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					ms.FindCandidatePersistentCacheParents(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*persistentcache.Peer{parent}, true).Times(1),
					mpcr.TaskManager().Return(taskManager).Times(1),
					mpct.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpcr.TaskManager().Return(taskManager).Times(1),
					mpct.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(2), nil).Times(1),
					mpcr.TaskManager().Return(taskManager).Times(1),
					mpct.Store(gomock.Any(), gomock.Eq(task)).Return(nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Cond(func(peer *persistentcache.Peer) bool {
						return peer.ID == mockPeerID && !peer.Persistent && peer.ConcurrentPieceCount == 9 && peer.FSM.Is(persistentcache.PeerStateReceivedNormal)
					})).Return(nil).Times(1),
					stream.EXPECT().Send(gomock.Cond(func(resp *schedulerv2.AnnouncePersistentCachePeerResponse) bool {
						normal, ok := resp.GetResponse().(*schedulerv2.AnnouncePersistentCachePeerResponse_NormalPersistentCacheTaskResponse)
						if !ok || len(normal.NormalPersistentCacheTaskResponse.CandidateParents) != 1 {
							return false
						}

						candidate := normal.NormalPersistentCacheTaskResponse.CandidateParents[0]
						return candidate.Id == mockSeedPeerID && candidate.Task.GetTag() == mockTaskTag && candidate.Task.GetApplication() == mockTaskApplication && candidate.Task.PieceLength == 512 && candidate.Task.CurrentReplicaCount == 2
					})).Return(nil).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "download started failed and failure handling rejected in state Pending",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(persistentcache.PeerStatePending)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_DownloadPersistentCachePeerStartedRequest{},
					}, nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "event Failed inappropriate in current state Pending"))
				assert.Equal(persistentcache.PeerStatePending, peer.FSM.Current())
			},
		},
		{
			name: "download started then reschedule sends candidate parents",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_DownloadPersistentCachePeerStartedRequest{},
					}, nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_ReschedulePersistentCachePeerRequest{
							ReschedulePersistentCachePeerRequest: &schedulerv2.ReschedulePersistentCachePeerRequest{CandidateParents: []*commonv2.PersistentCachePeer{{Id: "foo"}}},
						},
					}, nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					ms.FindCandidatePersistentCacheParents(gomock.Any(), gomock.Eq(peer), gomock.Cond(func(blocklist set.SafeSet[string]) bool { return blocklist.Contains(mockPeerID, "foo") })).Return([]*persistentcache.Peer{parent}, true).Times(1),
					mpcr.TaskManager().Return(taskManager).Times(1),
					mpct.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpcr.TaskManager().Return(taskManager).Times(1),
					mpct.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(2), nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					stream.EXPECT().Send(gomock.Cond(func(resp *schedulerv2.AnnouncePersistentCachePeerResponse) bool {
						normal, ok := resp.GetResponse().(*schedulerv2.AnnouncePersistentCachePeerResponse_NormalPersistentCacheTaskResponse)
						return ok && len(normal.NormalPersistentCacheTaskResponse.CandidateParents) == 1 && normal.NormalPersistentCacheTaskResponse.CandidateParents[0].Id == mockSeedPeerID
					})).Return(nil).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(persistentcache.PeerStateRunning, peer.FSM.Current())
				assert.ElementsMatch([]string{mockPeerID, "foo"}, peer.BlockParents)
			},
		},
		{
			name: "reschedule peer not found and failure handling store failed",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(persistentcache.PeerStateRunning)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_ReschedulePersistentCachePeerRequest{
							ReschedulePersistentCachePeerRequest: &schedulerv2.ReschedulePersistentCachePeerRequest{},
						},
					}, nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Eq(peer)).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
				assert.Equal(persistentcache.PeerStateFailed, peer.FSM.Current())
			},
		},
		{
			name: "download finished closes the stream",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(persistentcache.PeerStateRunning)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_DownloadPersistentCachePeerFinishedRequest{},
					}, nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(persistentcache.PeerStateSucceeded, peer.FSM.Current())
				assert.NotZero(peer.Cost)
			},
		},
		{
			name: "download finished failed and failure handling rejected in state ReceivedNormal",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(persistentcache.PeerStateReceivedNormal)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_DownloadPersistentCachePeerFinishedRequest{},
					}, nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "event Failed inappropriate in current state ReceivedNormal"))
				assert.Equal(persistentcache.PeerStateReceivedNormal, peer.FSM.Current())
			},
		},
		{
			name: "download failed closes the stream",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(persistentcache.PeerStateRunning)
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId:  mockHostID,
						TaskId:  mockTaskID,
						PeerId:  mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_DownloadPersistentCachePeerFailedRequest{},
					}, nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(persistentcache.PeerStateFailed, peer.FSM.Current())
			},
		},
		{
			name: "download piece finished updates peer and parent",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_DownloadPieceFinishedRequest{
							DownloadPieceFinishedRequest: &schedulerv2.DownloadPieceFinishedRequest{Piece: &commonv2.Piece{Number: 1, ParentId: &mockSeedPeerID}},
						},
					}, nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockSeedPeerID)).Return(parent, true).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Eq(parent)).Return(nil).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.True(peer.FinishedPieces.Test(1))
			},
		},
		{
			name: "download piece finished parent not found is logged and stream continues",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_DownloadPieceFinishedRequest{
							DownloadPieceFinishedRequest: &schedulerv2.DownloadPieceFinishedRequest{Piece: &commonv2.Piece{Number: 1, ParentId: &mockSeedPeerID}},
						},
					}, nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockSeedPeerID)).Return(nil, false).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "download piece failed with non temporary error is logged and stream continues",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{
						HostId: mockHostID,
						TaskId: mockTaskID,
						PeerId: mockPeerID,
						Request: &schedulerv2.AnnouncePersistentCachePeerRequest_DownloadPieceFailedRequest{
							DownloadPieceFailedRequest: &schedulerv2.DownloadPieceFailedRequest{ParentId: mockSeedPeerID},
						},
					}, nil).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Empty(peer.BlockParents)
			},
		},
		{
			name: "unknown request",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, parent *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, stream *schedulerv2mocks.MockScheduler_AnnouncePersistentCachePeerServer, mpcr *persistentcache.MockResourceMockRecorder, mpch *persistentcache.MockHostManagerMockRecorder, mpct *persistentcache.MockTaskManagerMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					stream.EXPECT().Context().Return(context.Background()).Times(1),
					stream.EXPECT().Recv().Return(&schedulerv2.AnnouncePersistentCachePeerRequest{HostId: mockHostID, TaskId: mockTaskID, PeerId: mockPeerID}, nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.Equal(codes.FailedPrecondition, status.Code(err))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			hostManager := persistentcache.NewMockHostManager(ctl)
			taskManager := persistentcache.NewMockTaskManager(ctl)
			peerManager := persistentcache.NewMockPeerManager(ctl)
			stream := schedulerv2mocks.NewMockScheduler_AnnouncePersistentCachePeerServer(ctl)

			host := mockPersistentCacheHost()
			task := mockPersistentCacheTask(persistentcache.TaskStateSucceeded)
			peer := mockPersistentCachePeer(mockPeerID, persistentcache.PeerStateReceivedNormal, task, host)
			parent := mockPersistentCachePeer(mockSeedPeerID, persistentcache.PeerStateSucceeded, task, host)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersistCache {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, nil, scheduling, job, internalJobImage, dynconfig)
			}

			tc.mock(host, task, peer, parent, hostManager, taskManager, peerManager, stream, persistentCacheResource.EXPECT(), hostManager.EXPECT(), taskManager.EXPECT(), peerManager.EXPECT(), scheduling.EXPECT())
			tc.expect(t, peer, svc.AnnouncePersistentCachePeer(stream))
		})
	}
}

func TestServiceV2_handleDownloadPersistentCachePieceFailedRequest(t *testing.T) {
	tests := []struct {
		name   string
		req    *schedulerv2.DownloadPieceFailedRequest
		mock   func(peer *persistentcache.Peer, parent *standard.Peer, peerManager persistentcache.PeerManager, standardPeerManager standard.PeerManager, mpcr *persistentcache.MockResourceMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder)
		expect func(t *testing.T, peer *persistentcache.Peer, parent *standard.Peer, err error)
	}{
		{
			name: "peer not found",
			req:  &schedulerv2.DownloadPieceFailedRequest{ParentId: mockSeedPeerID, Temporary: true},
			mock: func(peer *persistentcache.Peer, parent *standard.Peer, peerManager persistentcache.PeerManager, standardPeerManager standard.PeerManager, mpcr *persistentcache.MockResourceMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, parent *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "peer %s not found", mockPeerID))
			},
		},
		{
			name: "temporary is true and parent upload failure is counted",
			req:  &schedulerv2.DownloadPieceFailedRequest{ParentId: mockSeedPeerID, Temporary: true},
			mock: func(peer *persistentcache.Peer, parent *standard.Peer, peerManager persistentcache.PeerManager, standardPeerManager standard.PeerManager, mpcr *persistentcache.MockResourceMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(standardPeerManager).Times(1),
					mp.Load(gomock.Eq(mockSeedPeerID)).Return(parent, true).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, parent *standard.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.ElementsMatch([]string{mockPeerID, mockSeedPeerID}, peer.BlockParents)
				assert.Equal(int64(1), parent.Host.UploadFailedCount.Load())
			},
		},
		{
			name: "temporary is true and store peer failed",
			req:  &schedulerv2.DownloadPieceFailedRequest{ParentId: mockSeedPeerID, Temporary: true},
			mock: func(peer *persistentcache.Peer, parent *standard.Peer, peerManager persistentcache.PeerManager, standardPeerManager standard.PeerManager, mpcr *persistentcache.MockResourceMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mr.PeerManager().Return(standardPeerManager).Times(1),
					mp.Load(gomock.Eq(mockSeedPeerID)).Return(nil, false).Times(1),
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Store(gomock.Any(), gomock.Eq(peer)).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, parent *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
				assert.Equal(int64(0), parent.Host.UploadFailedCount.Load())
			},
		},
		{
			name: "temporary is false",
			req:  &schedulerv2.DownloadPieceFailedRequest{ParentId: mockSeedPeerID},
			mock: func(peer *persistentcache.Peer, parent *standard.Peer, peerManager persistentcache.PeerManager, standardPeerManager standard.PeerManager, mpcr *persistentcache.MockResourceMockRecorder, mpcp *persistentcache.MockPeerManagerMockRecorder, mr *standard.MockResourceMockRecorder, mp *standard.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpcr.PeerManager().Return(peerManager).Times(1),
					mpcp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, parent *standard.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "download piece failed"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := persistentcache.NewMockPeerManager(ctl)
			standardPeerManager := standard.NewMockPeerManager(ctl)

			peer := mockPersistentCachePeer(mockPeerID, persistentcache.PeerStateRunning, mockPersistentCacheTask(persistentcache.TaskStateSucceeded), mockPersistentCacheHost())
			mockHost := standard.NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := standard.NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, standard.WithDigest(mockTaskDigest))
			parent := standard.NewPeer(mockSeedPeerID, mockTask, mockHost)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(peer, parent, peerManager, standardPeerManager, persistentCacheResource.EXPECT(), peerManager.EXPECT(), resource.EXPECT(), standardPeerManager.EXPECT())
			tc.expect(t, peer, parent, svc.handleDownloadPersistentCachePieceFailedRequest(context.Background(), mockPeerID, tc.req))
		})
	}
}

func TestServiceV2_StatPersistentCachePeer(t *testing.T) {
	tests := []struct {
		name                string
		disablePersistCache bool
		mock                func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder)
		expect              func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCachePeer, err error)
	}{
		{
			name:                "redis is not enabled",
			disablePersistCache: true,
			mock: func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCachePeer, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "peer not found",
			mock: func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCachePeer, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "persistent cache peer %s not found", mockPeerID))
			},
		},
		{
			name: "load current persistent replica count failed",
			mock: func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCachePeer, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "load current replica count failed",
			mock: func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCachePeer, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
			},
		},
		{
			name: "peer has been loaded",
			mock: func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(3), nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCachePeer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(mockPeerID, resp.Id)
				assert.True(resp.Persistent)
				assert.Equal(persistentcache.PeerStateSucceeded, resp.State)
				assert.Equal(mockTaskID, resp.Task.Id)
				assert.Equal(uint64(2), resp.Task.PersistentReplicaCount)
				assert.Equal(uint64(1), resp.Task.CurrentPersistentReplicaCount)
				assert.Equal(uint64(3), resp.Task.CurrentReplicaCount)
				assert.Equal(mockTaskTag, resp.Task.GetTag())
				assert.Equal(mockTaskApplication, resp.Task.GetApplication())
				assert.Equal(uint64(512), resp.Task.PieceLength)
				assert.Equal(uint64(1024), resp.Task.ContentLength)
				assert.Equal(uint32(2), resp.Task.PieceCount)
				assert.Equal(persistentcache.TaskStateSucceeded, resp.Task.State)
				assert.Equal(mockHostID, resp.Host.Id)
				assert.Equal(mockRawPersistentCacheHost.IP, resp.Host.Ip)
				assert.Equal(mockRawPersistentCacheHost.DownloadPort, resp.Host.DownloadPort)
				assert.Equal(uint64(1), resp.Host.SchedulerClusterId)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := persistentcache.NewMockPeerManager(ctl)
			taskManager := persistentcache.NewMockTaskManager(ctl)

			peer := mockPersistentCachePeer(mockPeerID, persistentcache.PeerStateSucceeded, mockPersistentCacheTask(persistentcache.TaskStateSucceeded), mockPersistentCacheHost())
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig, Manager: config.ManagerConfig{SchedulerClusterID: 1}}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersistCache {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, nil, scheduling, job, internalJobImage, dynconfig)
			}

			tc.mock(peer, peerManager, taskManager, persistentCacheResource.EXPECT(), peerManager.EXPECT(), taskManager.EXPECT())
			resp, err := svc.StatPersistentCachePeer(context.Background(), &schedulerv2.StatPersistentCachePeerRequest{HostId: mockHostID, TaskId: mockTaskID, PeerId: mockPeerID})
			tc.expect(t, peer, resp, err)
		})
	}
}

func TestServiceV2_DeletePersistentCachePeer(t *testing.T) {
	tests := []struct {
		name                string
		disablePersistCache bool
		mock                func(wg *sync.WaitGroup, peer *persistentcache.Peer, peerManager persistentcache.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder)
		expect              func(t *testing.T, err error)
	}{
		{
			name:                "redis is not enabled",
			disablePersistCache: true,
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, peerManager persistentcache.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "peer not found",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, peerManager persistentcache.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "persistent cache peer %s not found", mockPeerID))
			},
		},
		{
			name: "delete peer failed",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, peerManager persistentcache.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "get dfdaemon client failed",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, peerManager persistentcache.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil).Times(1),
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(nil, errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "delete persistent task from peer failed is tolerated and task is replicated",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, peerManager persistentcache.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				wg.Add(1)
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil).Times(1),
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().DeletePersistentCacheTask(gomock.Any(), gomock.Cond(func(req *dfdaemonv2.DeletePersistentCacheTaskRequest) bool { return req.TaskId == mockTaskID })).Return(errors.New("baz")).Times(1),
					ms.FindReplicatePersistentCacheHosts(gomock.Any(), gomock.Eq(peer.Task), gomock.Cond(func(blocklist set.SafeSet[string]) bool { return blocklist.Contains(mockHostID) })).Do(func(context.Context, *persistentcache.Task, set.SafeSet[string]) { wg.Done() }).Return(nil, nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := persistentcache.NewMockPeerManager(ctl)
			pool := dfdaemonclientmocks.NewMockPool(ctl)
			client := dfdaemonclientmocks.NewMockClient(ctl)

			peer := mockPersistentCachePeer(mockPeerID, persistentcache.PeerStateSucceeded, mockPersistentCacheTask(persistentcache.TaskStateSucceeded), mockPersistentCacheHost())
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersistCache {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, nil, scheduling, job, internalJobImage, dynconfig)
			}

			var wg sync.WaitGroup
			tc.mock(&wg, peer, peerManager, pool, client, resource.EXPECT(), persistentCacheResource.EXPECT(), peerManager.EXPECT(), scheduling.EXPECT())
			err := svc.DeletePersistentCachePeer(context.Background(), &schedulerv2.DeletePersistentCachePeerRequest{HostId: mockHostID, TaskId: mockTaskID, PeerId: mockPeerID})
			wg.Wait()
			tc.expect(t, err)
		})
	}
}

func TestServiceV2_StatPersistentCacheTask(t *testing.T) {
	tests := []struct {
		name                string
		disablePersistCache bool
		mock                func(task *persistentcache.Task, taskManager persistentcache.TaskManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder)
		expect              func(t *testing.T, resp *commonv2.PersistentCacheTask, err error)
	}{
		{
			name:                "redis is not enabled",
			disablePersistCache: true,
			mock: func(task *persistentcache.Task, taskManager persistentcache.TaskManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
			},
			expect: func(t *testing.T, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "task not found",
			mock: func(task *persistentcache.Task, taskManager persistentcache.TaskManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "persistent cache task %s not found", mockTaskID))
			},
		},
		{
			name: "load current persistent replica count failed",
			mock: func(task *persistentcache.Task, taskManager persistentcache.TaskManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "load current replica count failed",
			mock: func(task *persistentcache.Task, taskManager persistentcache.TaskManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
			},
		},
		{
			name: "task has been loaded",
			mock: func(task *persistentcache.Task, taskManager persistentcache.TaskManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(3), nil).Times(1),
				)
			},
			expect: func(t *testing.T, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(mockTaskID, resp.Id)
				assert.Equal(uint64(2), resp.PersistentReplicaCount)
				assert.Equal(uint64(1), resp.CurrentPersistentReplicaCount)
				assert.Equal(uint64(3), resp.CurrentReplicaCount)
				assert.Equal(mockTaskTag, resp.GetTag())
				assert.Equal(mockTaskApplication, resp.GetApplication())
				assert.Equal(uint64(512), resp.PieceLength)
				assert.Equal(uint64(1024), resp.ContentLength)
				assert.Equal(uint32(2), resp.PieceCount)
				assert.Equal(persistentcache.TaskStateSucceeded, resp.State)
				assert.Equal(time.Hour, resp.Ttl.AsDuration())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			taskManager := persistentcache.NewMockTaskManager(ctl)

			task := mockPersistentCacheTask(persistentcache.TaskStateSucceeded)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersistCache {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, nil, scheduling, job, internalJobImage, dynconfig)
			}

			tc.mock(task, taskManager, persistentCacheResource.EXPECT(), taskManager.EXPECT())
			resp, err := svc.StatPersistentCacheTask(context.Background(), &schedulerv2.StatPersistentCacheTaskRequest{HostId: mockHostID, TaskId: mockTaskID})
			tc.expect(t, resp, err)
		})
	}
}

func TestServiceV2_DeletePersistentCacheTask(t *testing.T) {
	tests := []struct {
		name                string
		disablePersistCache bool
		mock                func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder)
		expect              func(t *testing.T, err error)
	}{
		{
			name:                "redis is not enabled",
			disablePersistCache: true,
			mock: func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "load peers by task failed",
			mock: func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "delete peer failed is skipped and delete task failed",
			mock: func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]*persistentcache.Peer{peer}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(errors.New("bar")).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Delete(gomock.Any(), gomock.Eq(mockTaskID)).Return(errors.New("baz")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "baz"))
			},
		},
		{
			name: "get dfdaemon client failed is skipped",
			mock: func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]*persistentcache.Peer{peer}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil).Times(1),
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(nil, errors.New("foo")).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Delete(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "delete persistent task from peer failed is skipped",
			mock: func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]*persistentcache.Peer{peer}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil).Times(1),
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().DeletePersistentCacheTask(gomock.Any(), gomock.Any()).Return(errors.New("foo")).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Delete(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "delete task and peers succeeded",
			mock: func(peer *persistentcache.Peer, peerManager persistentcache.PeerManager, taskManager persistentcache.TaskManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.LoadAllByTaskID(gomock.Any(), gomock.Eq(mockTaskID)).Return([]*persistentcache.Peer{peer}, nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Delete(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil).Times(1),
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().DeletePersistentCacheTask(gomock.Any(), gomock.Cond(func(req *dfdaemonv2.DeletePersistentCacheTaskRequest) bool { return req.TaskId == mockTaskID })).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Delete(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := persistentcache.NewMockPeerManager(ctl)
			taskManager := persistentcache.NewMockTaskManager(ctl)
			pool := dfdaemonclientmocks.NewMockPool(ctl)
			client := dfdaemonclientmocks.NewMockClient(ctl)

			peer := mockPersistentCachePeer(mockPeerID, persistentcache.PeerStateSucceeded, mockPersistentCacheTask(persistentcache.TaskStateSucceeded), mockPersistentCacheHost())
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersistCache {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, nil, scheduling, job, internalJobImage, dynconfig)
			}

			tc.mock(peer, peerManager, taskManager, pool, client, resource.EXPECT(), persistentCacheResource.EXPECT(), peerManager.EXPECT(), taskManager.EXPECT())
			tc.expect(t, svc.DeletePersistentCacheTask(context.Background(), &schedulerv2.DeletePersistentCacheTaskRequest{HostId: mockHostID, TaskId: mockTaskID}))
		})
	}
}
func TestServiceV2_UploadPersistentCacheTaskStarted(t *testing.T) {
	tests := []struct {
		name                string
		disablePersistCache bool
		mock                func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mph *persistentcache.MockHostManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder)
		expect              func(t *testing.T, err error)
	}{
		{
			name:                "redis is not enabled",
			disablePersistCache: true,
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mph *persistentcache.MockHostManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "host not found",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mph *persistentcache.MockHostManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "host %s not found", mockHostID))
			},
		},
		{
			name: "task already exists and is uploading",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mph *persistentcache.MockHostManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				task.FSM.SetState(persistentcache.TaskStateUploading)
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.AlreadyExists, "persistent cache task %s is %s cannot upload", mockTaskID, persistentcache.TaskStateUploading))
			},
		},
		{
			name: "task already exists and is failed can be uploaded again",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mph *persistentcache.MockHostManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				task.FSM.SetState(persistentcache.TaskStateFailed)
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(task, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Cond(func(task *persistentcache.Task) bool {
						return task.ID == mockTaskID && task.FSM.Is(persistentcache.TaskStateUploading) && task.Tag == mockTaskTag && task.Application == mockTaskApplication && task.PieceLength == 1024 && task.PersistentReplicaCount == 3 && task.ContentLength == 4096 && task.TotalPieceCount == 4 && task.TTL == time.Hour
					})).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Cond(func(peer *persistentcache.Peer) bool {
						return peer.ID == mockPeerID && peer.Persistent && peer.FSM.Is(persistentcache.PeerStateUploading) && peer.Task.ID == mockTaskID && peer.Host.ID == mockHostID && peer.FinishedPieces.Len() == 4
					})).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "store task failed",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mph *persistentcache.MockHostManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, false).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Any()).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
			},
		},
		{
			name: "peer already exists",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mph *persistentcache.MockHostManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, false).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Any()).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.AlreadyExists, "persistent cache peer %s already exists", mockPeerID))
			},
		},
		{
			name: "store peer failed",
			mock: func(host *persistentcache.Host, task *persistentcache.Task, peer *persistentcache.Peer, hostManager persistentcache.HostManager, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mph *persistentcache.MockHostManagerMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.HostManager().Return(hostManager).Times(1),
					mph.Load(gomock.Any(), gomock.Eq(mockHostID)).Return(host, true).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Load(gomock.Any(), gomock.Eq(mockTaskID)).Return(nil, false).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Any()).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Any()).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			hostManager := persistentcache.NewMockHostManager(ctl)
			taskManager := persistentcache.NewMockTaskManager(ctl)
			peerManager := persistentcache.NewMockPeerManager(ctl)

			host := mockPersistentCacheHost()
			task := mockPersistentCacheTask(persistentcache.TaskStatePending)
			peer := mockPersistentCachePeer(mockPeerID, persistentcache.PeerStatePending, task, host)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersistCache {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, nil, scheduling, job, internalJobImage, dynconfig)
			}

			tc.mock(host, task, peer, hostManager, taskManager, peerManager, persistentCacheResource.EXPECT(), hostManager.EXPECT(), taskManager.EXPECT(), peerManager.EXPECT())
			tc.expect(t, svc.UploadPersistentCacheTaskStarted(context.Background(), &schedulerv2.UploadPersistentCacheTaskStartedRequest{
				HostId:                 mockHostID,
				TaskId:                 mockTaskID,
				PeerId:                 mockPeerID,
				PersistentReplicaCount: 3,
				Tag:                    &mockTaskTag,
				Application:            &mockTaskApplication,
				PieceLength:            1024,
				ContentLength:          4096,
				PieceCount:             4,
				Ttl:                    durationpb.New(time.Hour),
			}))
		})
	}
}

func TestServiceV2_UploadPersistentCacheTaskFinished(t *testing.T) {
	tests := []struct {
		name                string
		disablePersistCache bool
		mock                func(wg *sync.WaitGroup, peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder)
		expect              func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCacheTask, err error)
	}{
		{
			name:                "redis is not enabled",
			disablePersistCache: true,
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "peer not found",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "persistent cache peer %s not found", mockPeerID))
			},
		},
		{
			name: "peer state is PeerStateSucceeded",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.FSM.SetState(persistentcache.PeerStateSucceeded)
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "event Succeeded inappropriate in current state Succeeded"))
			},
		},
		{
			name: "store peer failed",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
				assert.Equal(persistentcache.PeerStateSucceeded, peer.FSM.Current())
				assert.True(peer.FinishedPieces.All())
			},
		},
		{
			name: "task state is TaskStateSucceeded",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				peer.Task.FSM.SetState(persistentcache.TaskStateSucceeded)
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "event Succeeded inappropriate in current state Succeeded"))
			},
		},
		{
			name: "store task failed",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(peer.Task)).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
				assert.Equal(persistentcache.TaskStateSucceeded, peer.Task.FSM.Current())
			},
		},
		{
			name: "load current persistent replica count failed",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(peer.Task)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("baz")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "baz"))
			},
		},
		{
			name: "load current replica count failed",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(peer.Task)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(0), errors.New("qux")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Error(codes.Internal, "qux"))
			},
		},
		{
			name: "upload finished and task is replicated",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				wg.Add(1)
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(peer.Task)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentPersistentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.LoadCurrentReplicaCount(gomock.Any(), gomock.Eq(mockTaskID)).Return(uint64(1), nil).Times(1),
					ms.FindReplicatePersistentCacheHosts(gomock.Any(), gomock.Eq(peer.Task), gomock.Cond(func(blocklist set.SafeSet[string]) bool { return blocklist.Contains(mockHostID) })).Do(func(context.Context, *persistentcache.Task, set.SafeSet[string]) { wg.Done() }).Return(nil, nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, resp *commonv2.PersistentCacheTask, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(persistentcache.PeerStateSucceeded, peer.FSM.Current())
				assert.Equal(persistentcache.TaskStateSucceeded, peer.Task.FSM.Current())
				assert.True(peer.FinishedPieces.All())
				assert.NotZero(peer.Cost)
				assert.Equal(mockTaskID, resp.Id)
				assert.Equal(uint64(2), resp.PersistentReplicaCount)
				assert.Equal(uint64(1), resp.CurrentPersistentReplicaCount)
				assert.Equal(uint64(1), resp.CurrentReplicaCount)
				assert.Equal(mockTaskTag, resp.GetTag())
				assert.Equal(mockTaskApplication, resp.GetApplication())
				assert.Equal(uint64(512), resp.PieceLength)
				assert.Equal(uint64(1024), resp.ContentLength)
				assert.Equal(uint32(2), resp.PieceCount)
				assert.Equal(persistentcache.TaskStateSucceeded, resp.State)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			taskManager := persistentcache.NewMockTaskManager(ctl)
			peerManager := persistentcache.NewMockPeerManager(ctl)

			peer := mockPersistentCachePeer(mockPeerID, persistentcache.PeerStateUploading, mockPersistentCacheTask(persistentcache.TaskStateUploading), mockPersistentCacheHost())
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersistCache {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, nil, scheduling, job, internalJobImage, dynconfig)
			}

			var wg sync.WaitGroup
			tc.mock(&wg, peer, taskManager, peerManager, persistentCacheResource.EXPECT(), taskManager.EXPECT(), peerManager.EXPECT(), scheduling.EXPECT())
			resp, err := svc.UploadPersistentCacheTaskFinished(context.Background(), &schedulerv2.UploadPersistentCacheTaskFinishedRequest{HostId: mockHostID, TaskId: mockTaskID, PeerId: mockPeerID})
			wg.Wait()
			tc.expect(t, peer, resp, err)
		})
	}
}

func TestServiceV2_UploadPersistentCacheTaskFailed(t *testing.T) {
	tests := []struct {
		name                string
		disablePersistCache bool
		mock                func(peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder)
		expect              func(t *testing.T, peer *persistentcache.Peer, err error)
	}{
		{
			name:                "redis is not enabled",
			disablePersistCache: true,
			mock: func(peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.FailedPrecondition, "redis is not enabled"))
			},
		},
		{
			name: "peer not found",
			mock: func(peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(nil, false).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Errorf(codes.NotFound, "persistent cache peer %s not found", mockPeerID))
			},
		},
		{
			name: "peer state is PeerStateSucceeded",
			mock: func(peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				peer.FSM.SetState(persistentcache.PeerStateSucceeded)
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "event Failed inappropriate in current state Succeeded"))
				assert.Equal(persistentcache.PeerStateSucceeded, peer.FSM.Current())
			},
		},
		{
			name: "store peer failed",
			mock: func(peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "foo"))
				assert.Equal(persistentcache.PeerStateFailed, peer.FSM.Current())
			},
		},
		{
			name: "task state is TaskStatePending",
			mock: func(peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				peer.Task.FSM.SetState(persistentcache.TaskStatePending)
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "event Failed inappropriate in current state Pending"))
				assert.Equal(persistentcache.PeerStateFailed, peer.FSM.Current())
				assert.Equal(persistentcache.TaskStatePending, peer.Task.FSM.Current())
			},
		},
		{
			name: "store task failed",
			mock: func(peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(peer.Task)).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, status.Error(codes.Internal, "bar"))
				assert.Equal(persistentcache.TaskStateFailed, peer.Task.FSM.Current())
			},
		},
		{
			name: "peer failed and task marked failed",
			mock: func(peer *persistentcache.Peer, taskManager persistentcache.TaskManager, peerManager persistentcache.PeerManager, mpr *persistentcache.MockResourceMockRecorder, mpt *persistentcache.MockTaskManagerMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Load(gomock.Any(), gomock.Eq(mockPeerID)).Return(peer, true).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(peer)).Return(nil).Times(1),
					mpr.TaskManager().Return(taskManager).Times(1),
					mpt.Store(gomock.Any(), gomock.Eq(peer.Task)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, peer *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(persistentcache.PeerStateFailed, peer.FSM.Current())
				assert.Equal(persistentcache.TaskStateFailed, peer.Task.FSM.Current())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			taskManager := persistentcache.NewMockTaskManager(ctl)
			peerManager := persistentcache.NewMockPeerManager(ctl)

			peer := mockPersistentCachePeer(mockPeerID, persistentcache.PeerStateUploading, mockPersistentCacheTask(persistentcache.TaskStateUploading), mockPersistentCacheHost())
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)
			if tc.disablePersistCache {
				svc = NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, nil, scheduling, job, internalJobImage, dynconfig)
			}

			tc.mock(peer, taskManager, peerManager, persistentCacheResource.EXPECT(), taskManager.EXPECT(), peerManager.EXPECT())
			tc.expect(t, peer, svc.UploadPersistentCacheTaskFailed(context.Background(), &schedulerv2.UploadPersistentCacheTaskFailedRequest{HostId: mockHostID, TaskId: mockTaskID, PeerId: mockPeerID}))
		})
	}
}
func TestServiceV2_replicatePersistentCacheTask(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(wg *sync.WaitGroup, peer *persistentcache.Peer, cachedParent *persistentcache.Peer, host *persistentcache.Host, pool *dfdaemonclientmocks.MockPool, mr *standard.MockResourceMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder)
		expect func(t *testing.T, err error)
	}{
		{
			name: "no replicate hosts found",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, cachedParent *persistentcache.Peer, host *persistentcache.Host, pool *dfdaemonclientmocks.MockPool, mr *standard.MockResourceMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				ms.FindReplicatePersistentCacheHosts(gomock.Any(), gomock.Eq(peer.Task), gomock.Any()).Return(nil, nil, false).Times(1)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "replicate to cached parent and host both dial dfdaemon",
			mock: func(wg *sync.WaitGroup, peer *persistentcache.Peer, cachedParent *persistentcache.Peer, host *persistentcache.Host, pool *dfdaemonclientmocks.MockPool, mr *standard.MockResourceMockRecorder, ms *schedulingmocks.MockSchedulingMockRecorder) {
				wg.Add(2)
				ms.FindReplicatePersistentCacheHosts(gomock.Any(), gomock.Eq(peer.Task), gomock.Any()).Return([]*persistentcache.Peer{cachedParent}, []*persistentcache.Host{host}, true).Times(1)
				mr.PeerClientPool().Return(pool).Times(2)
				pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Do(func(string, ...grpc.DialOption) { wg.Done() }).Return(nil, errors.New("foo")).Times(2)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			pool := dfdaemonclientmocks.NewMockPool(ctl)

			host := mockPersistentCacheHost()
			task := mockPersistentCacheTask(persistentcache.TaskStateSucceeded)
			peer := mockPersistentCachePeer(mockPeerID, persistentcache.PeerStateSucceeded, task, host)
			cachedParent := mockPersistentCachePeer(mockSeedPeerID, persistentcache.PeerStateSucceeded, task, host)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			var wg sync.WaitGroup
			tc.mock(&wg, peer, cachedParent, host, pool, resource.EXPECT(), scheduling.EXPECT())
			err := svc.replicatePersistentCacheTask(context.Background(), peer, set.NewSafeSet[string]())
			wg.Wait()
			tc.expect(t, err)
		})
	}
}

func TestServiceV2_downloadPersistentCacheTaskByPeer(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(task *persistentcache.Task, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadPersistentCacheTaskClient, mr *standard.MockResourceMockRecorder)
		expect func(t *testing.T, err error)
	}{
		{
			name: "get dfdaemon client failed",
			mock: func(task *persistentcache.Task, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadPersistentCacheTaskClient, mr *standard.MockResourceMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(nil, errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "download persistent task failed",
			mock: func(task *persistentcache.Task, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadPersistentCacheTaskClient, mr *standard.MockResourceMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().DownloadPersistentCacheTask(gomock.Any(), gomock.Cond(func(req *dfdaemonv2.DownloadPersistentCacheTaskRequest) bool {
						return req.TaskId == mockTaskID && req.Persistent && req.GetTag() == mockTaskTag && req.GetApplication() == mockTaskApplication
					})).Return(nil, errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "stream receive failed",
			mock: func(task *persistentcache.Task, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadPersistentCacheTaskClient, mr *standard.MockResourceMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().DownloadPersistentCacheTask(gomock.Any(), gomock.Any()).Return(stream, nil).Times(1),
					stream.EXPECT().Recv().Return(&dfdaemonv2.DownloadPersistentCacheTaskResponse{}, nil).Times(1),
					stream.EXPECT().Recv().Return(nil, errors.New("baz")).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "stream finished with EOF",
			mock: func(task *persistentcache.Task, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadPersistentCacheTaskClient, mr *standard.MockResourceMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().DownloadPersistentCacheTask(gomock.Any(), gomock.Any()).Return(stream, nil).Times(1),
					stream.EXPECT().Recv().Return(&dfdaemonv2.DownloadPersistentCacheTaskResponse{}, nil).Times(1),
					stream.EXPECT().Recv().Return(nil, io.EOF).Times(1),
				)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			pool := dfdaemonclientmocks.NewMockPool(ctl)
			client := dfdaemonclientmocks.NewMockClient(ctl)
			stream := dfdaemonv2mocks.NewMockDfdaemonUpload_DownloadPersistentCacheTaskClient(ctl)

			task := mockPersistentCacheTask(persistentcache.TaskStateSucceeded)
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(task, pool, client, stream, resource.EXPECT())
			tc.expect(t, svc.downloadPersistentCacheTaskByPeer(context.Background(), task, mockPersistentCacheHost()))
		})
	}
}

func TestServiceV2_persistPersistentCacheTaskByPeer(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(cachedParent *persistentcache.Peer, peerManager persistentcache.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder)
		expect func(t *testing.T, cachedParent *persistentcache.Peer, err error)
	}{
		{
			name: "get dfdaemon client failed",
			mock: func(cachedParent *persistentcache.Peer, peerManager persistentcache.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(nil, errors.New("foo")).Times(1),
				)
			},
			expect: func(t *testing.T, cachedParent *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.False(cachedParent.Persistent)
			},
		},
		{
			name: "update persistent task failed",
			mock: func(cachedParent *persistentcache.Peer, peerManager persistentcache.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().UpdatePersistentCacheTask(gomock.Any(), gomock.Cond(func(req *dfdaemonv2.UpdatePersistentCacheTaskRequest) bool {
						return req.TaskId == mockTaskID && req.Persistent
					})).Return(errors.New("bar")).Times(1),
				)
			},
			expect: func(t *testing.T, cachedParent *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.False(cachedParent.Persistent)
			},
		},
		{
			name: "store cached parent failed",
			mock: func(cachedParent *persistentcache.Peer, peerManager persistentcache.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().UpdatePersistentCacheTask(gomock.Any(), gomock.Any()).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(cachedParent)).Return(errors.New("baz")).Times(1),
				)
			},
			expect: func(t *testing.T, cachedParent *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.True(cachedParent.Persistent)
			},
		},
		{
			name: "cached parent becomes persistent",
			mock: func(cachedParent *persistentcache.Peer, peerManager persistentcache.PeerManager, pool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, mr *standard.MockResourceMockRecorder, mpr *persistentcache.MockResourceMockRecorder, mpp *persistentcache.MockPeerManagerMockRecorder) {
				gomock.InOrder(
					mr.PeerClientPool().Return(pool).Times(1),
					pool.EXPECT().Get(gomock.Eq("127.0.0.1:8001"), gomock.Any()).Return(client, nil).Times(1),
					client.EXPECT().UpdatePersistentCacheTask(gomock.Any(), gomock.Any()).Return(nil).Times(1),
					mpr.PeerManager().Return(peerManager).Times(1),
					mpp.Store(gomock.Any(), gomock.Eq(cachedParent)).Return(nil).Times(1),
				)
			},
			expect: func(t *testing.T, cachedParent *persistentcache.Peer, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.True(cachedParent.Persistent)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)
			peerManager := persistentcache.NewMockPeerManager(ctl)
			pool := dfdaemonclientmocks.NewMockPool(ctl)
			client := dfdaemonclientmocks.NewMockClient(ctl)

			host := mockPersistentCacheHost()
			task := mockPersistentCacheTask(persistentcache.TaskStateSucceeded)
			peer := mockPersistentCachePeer(mockPeerID, persistentcache.PeerStateSucceeded, task, host)
			cachedParent := mockPersistentCachePeer(mockSeedPeerID, persistentcache.PeerStateSucceeded, task, host)
			cachedParent.Persistent = false
			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.mock(cachedParent, peerManager, pool, client, resource.EXPECT(), persistentCacheResource.EXPECT(), peerManager.EXPECT())
			tc.expect(t, cachedParent, svc.persistPersistentCacheTaskByPeer(context.Background(), peer, cachedParent))
		})
	}
}

func TestServiceV2_PreheatImage(t *testing.T) {
	tests := []struct {
		name string
		req  *schedulerv2.PreheatImageRequest
		run  func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder)
	}{
		{
			name: "invalid certificate chain",
			req: &schedulerv2.PreheatImageRequest{
				Url:              "https://example.com/v2/image/manifesat/latest",
				CertificateChain: [][]byte{{0x01, 0x02, 0x03}},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				assert := assert.New(t)
				assert.ErrorIs(svc.PreheatImage(context.Background(), req), status.Errorf(codes.InvalidArgument, "failed to parse certificate chain: x509: malformed certificate"))
			},
		},
		{
			name: "parse access url failed",
			req: &schedulerv2.PreheatImageRequest{
				Url: "https://example.com/image:latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				assert := assert.New(t)
				mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return(nil, errors.New("parse access url failed")).Times(1)

				assert.ErrorIs(svc.PreheatImage(context.Background(), req), status.Errorf(codes.InvalidArgument, "failed to resolve manifests: parse access url failed"))
			},
		},
		{
			name: "length of the layers is zero",
			req: &schedulerv2.PreheatImageRequest{
				Url: "https://example.com/v2/image/manifesat/latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				assert := assert.New(t)
				mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

				assert.ErrorIs(svc.PreheatImage(context.Background(), req), status.Errorf(codes.InvalidArgument, "expected exactly one layer, got 0"))
			},
		},
		{
			name: "unsupported preheat scope",
			req: &schedulerv2.PreheatImageRequest{
				Url:   "https://example.com/v2/image/manifesat/latest",
				Scope: "invalid-scope",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				assert := assert.New(t)
				mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1)

				assert.ErrorIs(svc.PreheatImage(context.Background(), req), status.Errorf(codes.InvalidArgument, "unsupported preheat scope: invalid-scope"))
			},
		},
		{
			name: "preheat scope is empty",
			req: &schedulerv2.PreheatImageRequest{
				Url: "https://example.com/v2/image/manifesat/latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.PreheatSingleSeedPeer(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, nil).Times(1),
				)

				assert.NoError(svc.PreheatImage(context.Background(), req))
			},
		},
		{
			name: "preheat single_seed_peer",
			req: &schedulerv2.PreheatImageRequest{
				Url:   "https://example.com/v2/image/manifesat/latest",
				Scope: managertypes.SingleSeedPeerScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.PreheatSingleSeedPeer(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, nil).Times(1),
				)

				assert.NoError(svc.PreheatImage(context.Background(), req))
			},
		},
		{
			name: "preheat single_seed_peer with task id based blob digest",
			req: &schedulerv2.PreheatImageRequest{
				Url:                         "https://example.com/v2/image/manifesat/latest",
				Scope:                       managertypes.SingleSeedPeerScope,
				EnableTaskIdBasedBlobDigest: true,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.PreheatSingleSeedPeer(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, preheatRequest *internaljob.PreheatRequest, _ *logger.SugaredLoggerOnWith) {
						assert.True(preheatRequest.EnableTaskIDBasedBlobDigest)
						wg.Done()
					}).Return(nil, nil).Times(1),
				)

				assert.NoError(svc.PreheatImage(context.Background(), req))
			},
		},
		{
			name: "preheat single_seed_peer failed",
			req: &schedulerv2.PreheatImageRequest{
				Url:   "https://example.com/v2/image/manifesat/latest",
				Scope: managertypes.SingleSeedPeerScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.PreheatSingleSeedPeer(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, errors.New("foo")).Times(1),
				)

				assert.NoError(svc.PreheatImage(context.Background(), req))
			},
		},
		{
			name: "preheat all_seed_peers",
			req: &schedulerv2.PreheatImageRequest{
				Url:   "https://example.com/v2/image/manifesat/latest",
				Scope: managertypes.AllSeedPeersScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.PreheatAllSeedPeers(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.PreheatResponse{SuccessTasks: make([]*internaljob.PreheatSuccessTask, 0), FailureTasks: make([]*internaljob.PreheatFailureTask, 0)}, nil).Times(1),
				)

				assert.NoError(svc.PreheatImage(context.Background(), req))
			},
		},
		{
			name: "preheat all_seed_peers failed",
			req: &schedulerv2.PreheatImageRequest{
				Url:   "https://example.com/v2/image/manifesat/latest",
				Scope: managertypes.AllSeedPeersScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.PreheatAllSeedPeers(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, errors.New("foo")).Times(1),
				)

				assert.NoError(svc.PreheatImage(context.Background(), req))
			},
		},
		{
			name: "preheat all_peers",
			req: &schedulerv2.PreheatImageRequest{
				Url:   "https://example.com/v2/image/manifesat/latest",
				Scope: managertypes.AllPeersScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.PreheatAllPeers(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.PreheatResponse{SuccessTasks: make([]*internaljob.PreheatSuccessTask, 0), FailureTasks: make([]*internaljob.PreheatFailureTask, 0)}, nil).Times(1),
				)

				assert.NoError(svc.PreheatImage(context.Background(), req))
			},
		},
		{
			name: "preheat all_peers failed",
			req: &schedulerv2.PreheatImageRequest{
				Url:   "https://example.com/v2/image/manifesat/latest",
				Scope: managertypes.AllPeersScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.PreheatAllPeers(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, errors.New("foo")).Times(1),
				)

				assert.NoError(svc.PreheatImage(context.Background(), req))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig, Metrics: config.MetricsConfig{EnableHost: true}}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, tc.req, job.EXPECT(), internalJobImage.EXPECT())
		})
	}
}

func TestServiceV2_StatImage(t *testing.T) {
	tests := []struct {
		name string
		req  *schedulerv2.StatImageRequest
		run  func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder)
	}{
		{
			name: "invalid certificate chain",
			req: &schedulerv2.StatImageRequest{
				Url:              "https://example.com/v2/image/manifests/latest",
				CertificateChain: [][]byte{{0x01, 0x02, 0x03}},
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				resp, err := svc.StatImage(context.Background(), req)

				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Errorf(codes.InvalidArgument, "failed to parse certificate chain: x509: malformed certificate"))
			},
		},
		{
			name: "parse access url failed",
			req: &schedulerv2.StatImageRequest{
				Url: "https://example.com/image:latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return(nil, errors.New("parse access url failed")).Times(1)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Errorf(codes.InvalidArgument, "failed to resolve manifests: parse access url failed"))
			},
		},
		{
			name: "length of the layers is zero",
			req: &schedulerv2.StatImageRequest{
				Url: "https://example.com/v2/image/manifests/latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Errorf(codes.InvalidArgument, "expected exactly one layer, got 0"))
			},
		},
		{
			name: "stat layer by peer",
			req: &schedulerv2.StatImageRequest{
				Url: "https://example.com/v2/image/manifests/latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1"}}}, nil).Times(1),
				)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Image.Layers, 1)
				assert.Len(resp.Peers, 1)
			},
		},
		{
			name: "stat layer by seed peers with all_seed_peers scope",
			req: &schedulerv2.StatImageRequest{
				Url:   "https://example.com/v2/image/manifests/latest",
				Scope: managertypes.AllSeedPeersScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, getTaskRequest *internaljob.GetTaskRequest, _ *logger.SugaredLoggerOnWith) {
						assert := assert.New(t)
						assert.Equal(managertypes.AllSeedPeersScope, getTaskRequest.Scope)
						wg.Done()
					}).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "seed-peer-1"}}}, nil).Times(1),
				)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Image.Layers, 1)
				assert.Len(resp.Peers, 1)
			},
		},
		{
			name: "stat layer by peer with default scope",
			req: &schedulerv2.StatImageRequest{
				Url: "https://example.com/v2/image/manifests/latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, getTaskRequest *internaljob.GetTaskRequest, _ *logger.SugaredLoggerOnWith) {
						assert := assert.New(t)
						assert.Equal(managertypes.AllPeersScope, getTaskRequest.Scope)
						wg.Done()
					}).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1"}}}, nil).Times(1),
				)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Image.Layers, 1)
				assert.Len(resp.Peers, 1)
			},
		},
		{
			name: "stat layer by peer with task id based blob digest",
			req: &schedulerv2.StatImageRequest{
				Url:                         "https://example.com/v2/image/manifests/latest",
				EnableTaskIdBasedBlobDigest: true,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, getTaskRequest *internaljob.GetTaskRequest, _ *logger.SugaredLoggerOnWith) {
						assert := assert.New(t)
						assert.Equal("b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d", getTaskRequest.TaskID)
						wg.Done()
					}).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "seed-peer-1"}}}, nil).Times(1),
				)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Image.Layers, 1)
				assert.Len(resp.Peers, 1)
			},
		},
		{
			name: "stat layer by peer with disabled task id based blob digest",
			req: &schedulerv2.StatImageRequest{
				Url:                         "https://example.com/v2/image/manifests/latest",
				EnableTaskIdBasedBlobDigest: false,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, getTaskRequest *internaljob.GetTaskRequest, _ *logger.SugaredLoggerOnWith) {
						assert := assert.New(t)
						assert.Equal("d7f6227f463bd06d64c4e69fb2de589e0a587a687e1938d2a6321b66632d11e2", getTaskRequest.TaskID)
						wg.Done()
					}).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "seed-peer-1"}}}, nil).Times(1),
				)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Image.Layers, 1)
				assert.Len(resp.Peers, 1)
			},
		},
		{
			name: "stat layer by peer with invalid blob digest",
			req: &schedulerv2.StatImageRequest{
				Url:                         "https://example.com/v2/image/manifests/latest",
				EnableTaskIdBasedBlobDigest: true,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/md5:8a04994a666b4e4b20a2fd9e5a44f44c"}}}, nil).Times(1)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.Nil(resp)
				assert.Error(err)
				assert.Equal(codes.InvalidArgument, status.Code(err))
			},
		},
		{
			name: "stat layer by peer failed",
			req: &schedulerv2.StatImageRequest{
				Url: "https://example.com/v2/image/manifests/latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, errors.New("foo")).Times(1),
				)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Image.Layers, 1)
				assert.Len(resp.Peers, 0)
			},
		},
		{
			name: "stat multi layers by different peer",
			req: &schedulerv2.StatImageRequest{
				Url: "https://example.com/v2/image/manifests/latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(2)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d", "https://example.com/v2/image/latest/blobs/sha256:150b7321c0794448817b19fab51e415ff406ac8663c4f53d64c3590454dee201"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-2"}}}, nil).Times(1),
				)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Image.Layers, 2)
				assert.Len(resp.Peers, 2)
				assert.Len(resp.Peers[0].CachedLayers, 1)
				assert.Len(resp.Peers[1].CachedLayers, 1)
			},
		},
		{
			name: "stat multi layers by peers",
			req: &schedulerv2.StatImageRequest{
				Url: "https://example.com/v2/image/manifests/latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(2)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d", "https://example.com/v2/image/latest/blobs/sha256:150b7321c0794448817b19fab51e415ff406ac8663c4f53d64c3590454dee201"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1"}, {IP: "127.0.0.1", Hostname: "peer-2"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-2"}, {IP: "127.0.0.1", Hostname: "peer-1"}}}, nil).Times(1),
				)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Image.Layers, 2)
				assert.Len(resp.Peers, 2)
				assert.Len(resp.Peers[0].CachedLayers, 2)
				assert.Len(resp.Peers[1].CachedLayers, 2)
			},
		},
		{
			name: "stat multi layers by peers, but one of the get task failed",
			req: &schedulerv2.StatImageRequest{
				Url: "https://example.com/v2/image/manifests/latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(2)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d", "https://example.com/v2/image/latest/blobs/sha256:150b7321c0794448817b19fab51e415ff406ac8663c4f53d64c3590454dee201"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1"}, {IP: "127.0.0.1", Hostname: "peer-2"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, errors.New("foo")).Times(1),
				)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Image.Layers, 2)
				assert.Len(resp.Peers, 2)
				assert.Len(resp.Peers[0].CachedLayers, 1)
				assert.Len(resp.Peers[1].CachedLayers, 1)
			},
		},
		{
			name: "stat multi layers by peers, but the get task failed",
			req: &schedulerv2.StatImageRequest{
				Url: "https://example.com/v2/image/manifests/latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(2)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d", "https://example.com/v2/image/latest/blobs/sha256:150b7321c0794448817b19fab51e415ff406ac8663c4f53d64c3590454dee201"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, errors.New("foo")).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, errors.New("foo")).Times(1),
				)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Image.Layers, 2)
				assert.Len(resp.Peers, 0)
			},
		},
		{
			name: "stat multi layers by peers all finished",
			req: &schedulerv2.StatImageRequest{
				Url: "https://example.com/v2/image/manifests/latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(2)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d", "https://example.com/v2/image/latest/blobs/sha256:150b7321c0794448817b19fab51e415ff406ac8663c4f53d64c3590454dee201"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1", IsFinished: true}, {IP: "127.0.0.1", Hostname: "peer-2", IsFinished: true}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-2", IsFinished: true}, {IP: "127.0.0.1", Hostname: "peer-1", IsFinished: true}}}, nil).Times(1),
				)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Image.Layers, 2)
				assert.Len(resp.Peers, 2)
				assert.Len(resp.Peers[0].CachedLayers, 2)
				assert.Len(resp.Peers[1].CachedLayers, 2)
				assert.True(*resp.Peers[0].CachedLayers[0].IsFinished)
				assert.True(*resp.Peers[0].CachedLayers[1].IsFinished)
				assert.True(*resp.Peers[1].CachedLayers[0].IsFinished)
				assert.True(*resp.Peers[1].CachedLayers[1].IsFinished)
			},
		},
		{
			name: "stat multi layers by peers no finished",
			req: &schedulerv2.StatImageRequest{
				Url: "https://example.com/v2/image/manifests/latest",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatImageRequest, mj *jobmocks.MockJobMockRecorder, mi *internaljobmocks.MockImageMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(2)
				defer wg.Wait()

				gomock.InOrder(
					mi.CreatePreheatRequestsByManifestURL(gomock.Any(), gomock.Any()).Return([]*internaljob.PreheatRequest{{URLs: []string{"https://example.com/v2/image/latest/blobs/sha256:b5f4dfca35398b36f61baa60e2bf2c242401c9d7db3de9168dcf780a2feedd2d", "https://example.com/v2/image/latest/blobs/sha256:150b7321c0794448817b19fab51e415ff406ac8663c4f53d64c3590454dee201"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1", IsFinished: false}, {IP: "127.0.0.1", Hostname: "peer-2", IsFinished: false}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-2", IsFinished: false}, {IP: "127.0.0.1", Hostname: "peer-1", IsFinished: false}}}, nil).Times(1),
				)

				resp, err := svc.StatImage(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Image.Layers, 2)
				assert.Len(resp.Peers, 2)
				assert.Len(resp.Peers[0].CachedLayers, 2)
				assert.Len(resp.Peers[1].CachedLayers, 2)
				assert.False(*resp.Peers[0].CachedLayers[0].IsFinished)
				assert.False(*resp.Peers[0].CachedLayers[1].IsFinished)
				assert.False(*resp.Peers[1].CachedLayers[0].IsFinished)
				assert.False(*resp.Peers[1].CachedLayers[1].IsFinished)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)
			internalJobImage := internaljobmocks.NewMockImage(ctl)

			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig, Metrics: config.MetricsConfig{EnableHost: true}}, resource, persistentResource, persistentCacheResource, scheduling, job, internalJobImage, dynconfig)

			tc.run(t, svc, tc.req, job.EXPECT(), internalJobImage.EXPECT())
		})
	}
}

func TestServiceV2_PreheatFile(t *testing.T) {
	tests := []struct {
		name string
		req  *schedulerv2.PreheatFileRequest
		run  func(t *testing.T, svc *V2, req *schedulerv2.PreheatFileRequest, mj *jobmocks.MockJobMockRecorder)
	}{
		{
			name: "unsupported preheat scope",
			req: &schedulerv2.PreheatFileRequest{
				Url:   "https://example.com/file.txt",
				Scope: "invalid-scope",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				assert := assert.New(t)
				assert.ErrorIs(svc.PreheatFile(context.Background(), req), status.Errorf(codes.InvalidArgument, "unsupported preheat scope: invalid-scope"))
			},
		},
		{
			name: "preheat scope is empty",
			req: &schedulerv2.PreheatFileRequest{
				Url: "https://example.com/file.txt",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mj.PreheatSingleSeedPeer(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, nil).Times(1),
				)

				assert.NoError(svc.PreheatFile(context.Background(), req))
			},
		},
		{
			name: "preheat single_seed_peer",
			req: &schedulerv2.PreheatFileRequest{
				Url:   "https://example.com/file.txt",
				Scope: managertypes.SingleSeedPeerScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mj.PreheatSingleSeedPeer(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, nil).Times(1),
				)

				assert.NoError(svc.PreheatFile(context.Background(), req))
			},
		},
		{
			name: "preheat single_seed_peer failed",
			req: &schedulerv2.PreheatFileRequest{
				Url:   "https://example.com/file.txt",
				Scope: managertypes.SingleSeedPeerScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mj.PreheatSingleSeedPeer(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, nil).Times(1),
				)

				assert.NoError(svc.PreheatFile(context.Background(), req))
			},
		},
		{
			name: "preheat all_seed_peers",
			req: &schedulerv2.PreheatFileRequest{
				Url:   "https://example.com/file.txt",
				Scope: managertypes.AllSeedPeersScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mj.PreheatAllSeedPeers(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.PreheatResponse{SuccessTasks: make([]*internaljob.PreheatSuccessTask, 0), FailureTasks: make([]*internaljob.PreheatFailureTask, 0)}, nil).Times(1),
				)

				assert.NoError(svc.PreheatFile(context.Background(), req))
			},
		},
		{
			name: "preheat all_seed_peers failed",
			req: &schedulerv2.PreheatFileRequest{
				Url:   "https://example.com/file.txt",
				Scope: managertypes.AllSeedPeersScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mj.PreheatAllSeedPeers(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.PreheatResponse{SuccessTasks: make([]*internaljob.PreheatSuccessTask, 0), FailureTasks: make([]*internaljob.PreheatFailureTask, 0)}, nil).Times(1),
				)

				assert.NoError(svc.PreheatFile(context.Background(), req))
			},
		},
		{
			name: "preheat all_peers",
			req: &schedulerv2.PreheatFileRequest{
				Url:   "https://example.com/file.txt",
				Scope: managertypes.AllPeersScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mj.PreheatAllPeers(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.PreheatResponse{SuccessTasks: make([]*internaljob.PreheatSuccessTask, 0), FailureTasks: make([]*internaljob.PreheatFailureTask, 0)}, nil).Times(1),
				)

				assert.NoError(svc.PreheatFile(context.Background(), req))
			},
		},
		{
			name: "preheat all_peers failed",
			req: &schedulerv2.PreheatFileRequest{
				Url:   "https://example.com/file.txt",
				Scope: managertypes.AllPeersScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mj.PreheatAllPeers(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.PreheatResponse{SuccessTasks: make([]*internaljob.PreheatSuccessTask, 0), FailureTasks: make([]*internaljob.PreheatFailureTask, 0)}, nil).Times(1),
				)

				assert.NoError(svc.PreheatFile(context.Background(), req))
			},
		},

		{
			name: "list task entries failed",
			req: &schedulerv2.PreheatFileRequest{
				Url: "https://example.com/dir/",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				assert := assert.New(t)
				mj.ListTaskEntries(gomock.Any(), gomock.Cond(func(req *internaljob.ListTaskEntriesRequest) bool { return req.Url == "https://example.com/dir/" }), gomock.Any()).Return(nil, errors.New("foo")).Times(1)

				assert.ErrorIs(svc.PreheatFile(context.Background(), req), status.Errorf(codes.InvalidArgument, "failed to list task entries: foo"))
			},
		},
		{
			name: "directory has no entries",
			req: &schedulerv2.PreheatFileRequest{
				Url: "https://example.com/dir/",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				assert := assert.New(t)
				mj.ListTaskEntries(gomock.Any(), gomock.Any(), gomock.Any()).Return(&internaljob.ListTaskEntriesResponse{}, nil).Times(1)

				assert.ErrorIs(svc.PreheatFile(context.Background(), req), status.Errorf(codes.InvalidArgument, "preheat url is a directory, but with no entry: %s", req.Url))
			},
		},
		{
			name: "directory entries skip sub directories",
			req: &schedulerv2.PreheatFileRequest{
				Url:   "https://example.com/dir/",
				Scope: managertypes.AllPeersScope,
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.PreheatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				assert := assert.New(t)
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mj.ListTaskEntries(gomock.Any(), gomock.Any(), gomock.Any()).Return(&internaljob.ListTaskEntriesResponse{Entries: []*dfdaemonv2.Entry{
						{Url: "https://example.com/dir/file1.txt"},
						{Url: "https://example.com/dir/sub/", IsDir: true},
						{Url: "https://example.com/dir/file2.txt"},
					}}, nil).Times(1),
					mj.PreheatAllPeers(gomock.Any(), gomock.Cond(func(preheatRequest *internaljob.PreheatRequest) bool {
						return len(preheatRequest.URLs) == 2 && preheatRequest.URLs[0] == "https://example.com/dir/file1.txt" && preheatRequest.URLs[1] == "https://example.com/dir/file2.txt" && preheatRequest.Scope == managertypes.AllPeersScope
					}), gomock.Any()).Do(func(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.PreheatResponse{}, nil).Times(1),
				)

				assert.NoError(svc.PreheatFile(context.Background(), req))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)

			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig, Metrics: config.MetricsConfig{EnableHost: true}}, resource, persistentResource, persistentCacheResource, scheduling, job, nil, dynconfig)

			tc.run(t, svc, tc.req, job.EXPECT())
		})
	}
}

func TestServiceV2_StatFile(t *testing.T) {
	tests := []struct {
		name string
		req  *schedulerv2.StatFileRequest
		run  func(t *testing.T, svc *V2, req *schedulerv2.StatFileRequest, mj *jobmocks.MockJobMockRecorder)
	}{
		{
			name: "list task entries failed",
			req: &schedulerv2.StatFileRequest{
				Url: "https://example.com/dir/",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				mj.ListTaskEntries(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("list task entries failed")).Times(1)

				resp, err := svc.StatFile(context.Background(), req)
				assert := assert.New(t)
				assert.Nil(resp)
				assert.ErrorIs(err, status.Errorf(codes.InvalidArgument, "failed to list task entries: list task entries failed"))
			},
		},
		{
			name: "get task failed",
			req: &schedulerv2.StatFileRequest{
				Url: "https://example.com/file.txt",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, errors.New("get task failed")).Times(1),
				)

				resp, err := svc.StatFile(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Peers, 0)
			},
		},
		{
			name: "stat file by peer",
			req: &schedulerv2.StatFileRequest{
				Url: "https://example.com/file.txt",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1"}}}, nil).Times(1),
				)

				resp, err := svc.StatFile(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Peers, 1)
			},
		},
		{
			name: "stat multi files by different peer",
			req: &schedulerv2.StatFileRequest{
				Url: "https://example.com/dir/",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(2)
				defer wg.Wait()

				gomock.InOrder(
					mj.ListTaskEntries(gomock.Any(), gomock.Any(), gomock.Any()).Return(&internaljob.ListTaskEntriesResponse{Entries: []*dfdaemonv2.Entry{{Url: "https://example.com/dir/file1.txt"}, {Url: "https://example.com/dir/file2.txt"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-2"}}}, nil).Times(1),
				)

				resp, err := svc.StatFile(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Peers, 2)
			},
		},
		{
			name: "stat multi files by peers",
			req: &schedulerv2.StatFileRequest{
				Url: "https://example.com/dir/",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(2)
				defer wg.Wait()

				gomock.InOrder(
					mj.ListTaskEntries(gomock.Any(), gomock.Any(), gomock.Any()).Return(&internaljob.ListTaskEntriesResponse{Entries: []*dfdaemonv2.Entry{{Url: "https://example.com/dir/file1.txt"}, {Url: "https://example.com/dir/file2.txt"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1"}, {IP: "127.0.0.1", Hostname: "peer-2"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-2"}, {IP: "127.0.0.1", Hostname: "peer-1"}}}, nil).Times(1),
				)

				resp, err := svc.StatFile(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Peers, 2)
			},
		},
		{
			name: "stat multi files by peers, but one of the get task failed",
			req: &schedulerv2.StatFileRequest{
				Url: "https://example.com/dir/",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(2)
				defer wg.Wait()

				gomock.InOrder(
					mj.ListTaskEntries(gomock.Any(), gomock.Any(), gomock.Any()).Return(&internaljob.ListTaskEntriesResponse{Entries: []*dfdaemonv2.Entry{{Url: "https://example.com/dir/file1.txt"}, {Url: "https://example.com/dir/file2.txt"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1"}, {IP: "127.0.0.1", Hostname: "peer-2"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(nil, errors.New("get task failed")).Times(1),
				)

				resp, err := svc.StatFile(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Peers, 2)
			},
		},
		{
			name: "stat multi files by peers, but the get task failed",
			req: &schedulerv2.StatFileRequest{
				Url: "https://example.com/dir/",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Wait()

				gomock.InOrder(
					mj.ListTaskEntries(gomock.Any(), gomock.Any(), gomock.Any()).Return(&internaljob.ListTaskEntriesResponse{Entries: []*dfdaemonv2.Entry{{Url: "https://example.com/file.txt"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1"}}}, nil).Times(1),
				)

				resp, err := svc.StatFile(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Peers, 1)
			},
		},
		{
			name: "stat multi files by peers all finished",
			req: &schedulerv2.StatFileRequest{
				Url: "https://example.com/dir/",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(2)
				defer wg.Wait()

				gomock.InOrder(
					mj.ListTaskEntries(gomock.Any(), gomock.Any(), gomock.Any()).Return(&internaljob.ListTaskEntriesResponse{Entries: []*dfdaemonv2.Entry{{Url: "https://example.com/dir/file1.txt"}, {Url: "https://example.com/dir/file2.txt"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1", IsFinished: true}, {IP: "127.0.0.1", Hostname: "peer-2", IsFinished: true}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-2", IsFinished: true}, {IP: "127.0.0.1", Hostname: "peer-1", IsFinished: true}}}, nil).Times(1),
				)

				resp, err := svc.StatFile(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Peers, 2)
				assert.Len(resp.Peers[0].CachedFiles, 2)
				assert.Len(resp.Peers[1].CachedFiles, 2)
				assert.True(*resp.Peers[0].CachedFiles[0].IsFinished)
				assert.True(*resp.Peers[0].CachedFiles[1].IsFinished)
				assert.True(*resp.Peers[1].CachedFiles[0].IsFinished)
				assert.True(*resp.Peers[1].CachedFiles[1].IsFinished)
			},
		},
		{
			name: "stat multi files by peers no finished",
			req: &schedulerv2.StatFileRequest{
				Url: "https://example.com/dir/",
			},
			run: func(t *testing.T, svc *V2, req *schedulerv2.StatFileRequest, mj *jobmocks.MockJobMockRecorder) {
				var wg sync.WaitGroup
				wg.Add(2)
				defer wg.Wait()

				gomock.InOrder(
					mj.ListTaskEntries(gomock.Any(), gomock.Any(), gomock.Any()).Return(&internaljob.ListTaskEntriesResponse{Entries: []*dfdaemonv2.Entry{{Url: "https://example.com/dir/file1.txt"}, {Url: "https://example.com/dir/file2.txt"}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-1", IsFinished: false}, {IP: "127.0.0.1", Hostname: "peer-2", IsFinished: false}}}, nil).Times(1),
					mj.GetTask(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) { wg.Done() }).Return(&internaljob.GetTaskResponse{Peers: []*internaljob.Peer{{IP: "127.0.0.1", Hostname: "peer-2", IsFinished: false}, {IP: "127.0.0.1", Hostname: "peer-1", IsFinished: false}}}, nil).Times(1),
				)

				resp, err := svc.StatFile(context.Background(), req)
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(resp.Peers, 2)
				assert.Len(resp.Peers[0].CachedFiles, 2)
				assert.Len(resp.Peers[1].CachedFiles, 2)
				assert.False(*resp.Peers[0].CachedFiles[0].IsFinished)
				assert.False(*resp.Peers[0].CachedFiles[1].IsFinished)
				assert.False(*resp.Peers[1].CachedFiles[0].IsFinished)
				assert.False(*resp.Peers[1].CachedFiles[1].IsFinished)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := schedulingmocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)

			svc := NewV2(&config.Config{Scheduler: mockSchedulerConfig, Metrics: config.MetricsConfig{EnableHost: true}}, resource, persistentResource, persistentCacheResource, scheduling, job, nil, dynconfig)

			tc.run(t, svc, tc.req, job.EXPECT())
		})
	}
}
