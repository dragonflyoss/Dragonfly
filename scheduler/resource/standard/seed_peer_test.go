/*
 *     Copyright 2022 The Dragonfly Authors
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

package standard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"d7y.io/hashring"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	cdnsystemv1 "d7y.io/api/v2/pkg/apis/cdnsystem/v1"
	cdnsystemv1mocks "d7y.io/api/v2/pkg/apis/cdnsystem/v1/mocks"
	commonv1 "d7y.io/api/v2/pkg/apis/common/v1"
	commonv2 "d7y.io/api/v2/pkg/apis/common/v2"
	dfdaemonv2 "d7y.io/api/v2/pkg/apis/dfdaemon/v2"
	dfdaemonv2mocks "d7y.io/api/v2/pkg/apis/dfdaemon/v2/mocks"
	schedulerv1 "d7y.io/api/v2/pkg/apis/scheduler/v1"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	"d7y.io/dragonfly/v2/pkg/gc"
	nethttp "d7y.io/dragonfly/v2/pkg/net/http"
	"d7y.io/dragonfly/v2/pkg/rpc/common"
	dfdaemonclientmocks "d7y.io/dragonfly/v2/pkg/rpc/dfdaemon/client/mocks"
)

func mockSeedHost(port int32) *Host {
	return NewHost(
		mockRawSeedHost.ID, mockRawSeedHost.IP, mockRawSeedHost.Name, mockRawSeedHost.Hostname,
		port, mockRawSeedHost.DownloadPort, mockRawSeedHost.ProxyPort, mockRawSeedHost.Type)
}

func mockSeedHostServer(t *testing.T, register func(svr *grpc.Server)) *Host {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	svr := grpc.NewServer()
	register(svr)
	go func() {
		if err := svr.Serve(lis); err != nil {
			logger.Errorf("failed to serve the seed host: %v", err)
		}
	}()
	t.Cleanup(svr.Stop)

	return mockSeedHost(int32(lis.Addr().(*net.TCPAddr).Port))
}

func mockUnreachableSeedHost(t *testing.T) *Host {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	port := int32(lis.Addr().(*net.TCPAddr).Port)
	if err := lis.Close(); err != nil {
		t.Fatal(err)
	}

	return mockSeedHost(port)
}

func mockHangingSeedHost(t *testing.T) *Host {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lis.Close() })

	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			defer conn.Close()
		}
	}()

	return mockSeedHost(int32(lis.Addr().(*net.TCPAddr).Port))
}

func TestSeedPeer_newSeedPeer(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, s SeedPeer)
	}{
		{
			name: "new seed peer",
			expect: func(t *testing.T, s SeedPeer) {
				assert := assert.New(t)
				assert.Equal("seedPeer", reflect.TypeOf(s).Elem().Name())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			hostManager := NewMockHostManager(ctl)
			peerManager := NewMockPeerManager(ctl)
			clientPool := dfdaemonclientmocks.NewMockPool(ctl)

			tc.expect(t, newSeedPeer(peerManager, hostManager, clientPool))
		})
	}
}

func TestSeedPeer_refresh(t *testing.T) {
	tests := []struct {
		name   string
		hosts  func(t *testing.T) []*Host
		mock   func(m *MockHostManagerMockRecorder, hosts []*Host)
		expect func(t *testing.T, seedPeer *seedPeer, hosts []*Host)
	}{
		{
			name: "refresh healthy seed peer",
			hosts: func(t *testing.T) []*Host {
				return []*Host{mockSeedHostServer(t, func(svr *grpc.Server) {
					healthpb.RegisterHealthServer(svr, health.NewServer())
				})}
			},
			mock: func(m *MockHostManagerMockRecorder, hosts []*Host) {
				m.LoadAllSeeds().Return(hosts)
			},
			expect: func(t *testing.T, seedPeer *seedPeer, hosts []*Host) {
				assert := assert.New(t)
				seedPeer.refresh(context.Background())
				assert.True(seedPeer.HasAvailable())

				host, err := seedPeer.Select(context.Background(), mockTaskID)
				assert.NoError(err)
				assert.Equal(hosts[0].ID, host.ID)
			},
		},
		{
			name: "filter unhealthy seed peer",
			hosts: func(t *testing.T) []*Host {
				return []*Host{mockUnreachableSeedHost(t)}
			},
			mock: func(m *MockHostManagerMockRecorder, hosts []*Host) {
				m.LoadAllSeeds().Return(hosts)
			},
			expect: func(t *testing.T, seedPeer *seedPeer, hosts []*Host) {
				assert := assert.New(t)
				seedPeer.refresh(context.Background())
				assert.False(seedPeer.HasAvailable())

				_, err := seedPeer.Select(context.Background(), mockTaskID)
				assert.Error(err)
			},
		},
		{
			name: "keep the most recently announced host of the same seed peer",
			hosts: func(t *testing.T) []*Host {
				register := func(svr *grpc.Server) {
					healthpb.RegisterHealthServer(svr, health.NewServer())
				}

				stale := mockSeedHostServer(t, register)
				stale.UpdatedAt.Store(time.Now().Add(-time.Minute))
				return []*Host{stale, mockSeedHostServer(t, register)}
			},
			mock: func(m *MockHostManagerMockRecorder, hosts []*Host) {
				m.LoadAllSeeds().Return(hosts)
			},
			expect: func(t *testing.T, seedPeer *seedPeer, hosts []*Host) {
				assert := assert.New(t)
				seedPeer.refresh(context.Background())
				assert.Len(seedPeer.hosts, 1)
				assert.Equal(hashring.DefaultVNodesPerNode, seedPeer.hashring.Len())

				host, err := seedPeer.Select(context.Background(), mockTaskID)
				assert.NoError(err)
				assert.Same(hosts[1], host)
			},
		},
		{
			name: "reuse the hashring when the seed peers are unchanged",
			hosts: func(t *testing.T) []*Host {
				return []*Host{mockSeedHostServer(t, func(svr *grpc.Server) {
					healthpb.RegisterHealthServer(svr, health.NewServer())
				})}
			},
			mock: func(m *MockHostManagerMockRecorder, hosts []*Host) {
				m.LoadAllSeeds().Return(hosts).Times(2)
			},
			expect: func(t *testing.T, seedPeer *seedPeer, hosts []*Host) {
				assert := assert.New(t)
				seedPeer.refresh(context.Background())
				ring := seedPeer.hashring

				seedPeer.refresh(context.Background())
				assert.Same(ring, seedPeer.hashring)
				assert.Same(hosts[0], seedPeer.hosts[hosts[0].Name])
			},
		},
		{
			name: "refresh and select concurrently with changing seed peers",
			hosts: func(t *testing.T) []*Host {
				register := func(svr *grpc.Server) {
					healthpb.RegisterHealthServer(svr, health.NewServer())
				}

				var hosts []*Host
				for i := range 3 {
					host := mockSeedHostServer(t, register)
					host.Name = fmt.Sprintf("default-seed-client-%d", i)
					hosts = append(hosts, host)
				}

				return hosts
			},
			mock: func(m *MockHostManagerMockRecorder, hosts []*Host) {
				var calls atomic.Int64
				m.LoadAllSeeds().DoAndReturn(func() []*Host {
					switch calls.Add(1) % 3 {
					case 0:
						return hosts[:1]
					case 1:
						return hosts
					default:
						return []*Host{}
					}
				}).AnyTimes()
			},
			expect: func(t *testing.T, seedPeer *seedPeer, hosts []*Host) {
				var wg sync.WaitGroup
				wg.Add(3)
				go func() {
					defer wg.Done()
					for range 100 {
						seedPeer.refresh(context.Background())
					}
				}()

				go func() {
					defer wg.Done()
					for range 100 {
						seedPeer.refresh(context.Background())
					}
				}()

				go func() {
					assert := assert.New(t)
					defer wg.Done()
					for i := range 2000 {
						host, err := seedPeer.Select(context.Background(), fmt.Sprintf("%s-%d", mockTaskID, i))
						if err != nil {
							assert.EqualError(err, "no available seed peer")
							continue
						}

						assert.Contains(hosts, host)
					}
				}()

				wg.Wait()
			},
		},
		{
			name: "check the health of seed peers concurrently",
			hosts: func(t *testing.T) []*Host {
				hosts := []*Host{mockSeedHostServer(t, func(svr *grpc.Server) {
					healthpb.RegisterHealthServer(svr, health.NewServer())
				})}
				for i := range 8 {
					host := mockHangingSeedHost(t)
					host.Name = fmt.Sprintf("hanging-seed-client-%d", i)
					hosts = append(hosts, host)
				}

				return hosts
			},
			mock: func(m *MockHostManagerMockRecorder, hosts []*Host) {
				m.LoadAllSeeds().Return(hosts)
			},
			expect: func(t *testing.T, seedPeer *seedPeer, hosts []*Host) {
				assert := assert.New(t)
				start := time.Now()
				seedPeer.refresh(context.Background())
				assert.Less(time.Since(start), 8*time.Second)
				assert.Len(seedPeer.hosts, 1)

				host, err := seedPeer.Select(context.Background(), mockTaskID)
				assert.NoError(err)
				assert.Same(hosts[0], host)
			},
		},
		{
			name: "clear stale seed peers when host manager is empty",
			hosts: func(t *testing.T) []*Host {
				return []*Host{}
			},
			mock: func(m *MockHostManagerMockRecorder, hosts []*Host) {
				m.LoadAllSeeds().Return(hosts)
			},
			expect: func(t *testing.T, seedPeer *seedPeer, hosts []*Host) {
				assert := assert.New(t)
				mockHost := mockSeedHost(4000)
				seedPeer.hosts[mockHost.Name] = mockHost
				seedPeer.hashring.Add(mockHost.Name)

				seedPeer.refresh(context.Background())
				assert.Empty(seedPeer.hosts)
				assert.False(seedPeer.HasAvailable())

				_, err := seedPeer.Select(context.Background(), mockTaskID)
				assert.Error(err)
			},
		},
		{
			name: "refresh and select concurrently",
			hosts: func(t *testing.T) []*Host {
				return []*Host{}
			},
			mock: func(m *MockHostManagerMockRecorder, hosts []*Host) {
				m.LoadAllSeeds().Return(hosts).AnyTimes()
			},
			expect: func(t *testing.T, seedPeer *seedPeer, hosts []*Host) {
				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					defer wg.Done()
					for range 100 {
						seedPeer.refresh(context.Background())
					}
				}()

				go func() {
					assert := assert.New(t)
					defer wg.Done()
					for range 100 {
						seedPeer.HasAvailable()
						_, err := seedPeer.Select(context.Background(), mockTaskID)
						assert.Error(err)
					}
				}()

				wg.Wait()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			hostManager := NewMockHostManager(ctl)
			peerManager := NewMockPeerManager(ctl)
			clientPool := dfdaemonclientmocks.NewMockPool(ctl)

			hosts := tc.hosts(t)
			tc.mock(hostManager.EXPECT(), hosts)

			seedPeer := newSeedPeer(peerManager, hostManager, clientPool,
				grpc.WithTransportCredentials(insecure.NewCredentials())).(*seedPeer)
			tc.expect(t, seedPeer, hosts)
		})
	}
}

func TestSeedPeer_Select(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, seedPeer *seedPeer, mockHost *Host)
	}{
		{
			name: "no available seed peer",
			expect: func(t *testing.T, seedPeer *seedPeer, mockHost *Host) {
				assert := assert.New(t)
				_, err := seedPeer.Select(context.Background(), mockTaskID)
				assert.Error(err)
				assert.False(seedPeer.HasAvailable())
			},
		},
		{
			name: "select seed peer by task id",
			expect: func(t *testing.T, seedPeer *seedPeer, mockHost *Host) {
				assert := assert.New(t)
				seedPeer.hosts[mockHost.Name] = mockHost
				seedPeer.hashring.Add(mockHost.Name)
				host, err := seedPeer.Select(context.Background(), mockTaskID)
				assert.NoError(err)
				assert.Same(mockHost, host)
				assert.True(seedPeer.HasAvailable())
			},
		},
		{
			name: "hashring member has no host",
			expect: func(t *testing.T, seedPeer *seedPeer, mockHost *Host) {
				assert := assert.New(t)
				seedPeer.hashring.Add(mockHost.Name)
				_, err := seedPeer.Select(context.Background(), mockTaskID)
				assert.Error(err)
				assert.True(seedPeer.HasAvailable())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			hostManager := NewMockHostManager(ctl)
			peerManager := NewMockPeerManager(ctl)
			clientPool := dfdaemonclientmocks.NewMockPool(ctl)

			seedPeer := newSeedPeer(peerManager, hostManager, clientPool).(*seedPeer)
			tc.expect(t, seedPeer, mockSeedHost(mockRawSeedHost.Port))
		})
	}
}

func TestSeedPeer_TriggerDownloadTask(t *testing.T) {
	tests := []struct {
		name      string
		available bool
		mock      func(clientPool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadTaskClient, addr string)
		expect    func(t *testing.T, err error)
	}{
		{
			name:      "no available seed peer",
			available: false,
			mock: func(clientPool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadTaskClient, addr string) {
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name:      "get client from pool failed",
			available: true,
			mock: func(clientPool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadTaskClient, addr string) {
				clientPool.EXPECT().Get(addr).Return(nil, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name:      "download task failed",
			available: true,
			mock: func(clientPool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadTaskClient, addr string) {
				clientPool.EXPECT().Get(addr).Return(client, nil).Times(1)
				client.EXPECT().DownloadTask(gomock.Any(), mockTaskID, gomock.Any()).Return(nil, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name:      "receive from download task stream failed",
			available: true,
			mock: func(clientPool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadTaskClient, addr string) {
				clientPool.EXPECT().Get(addr).Return(client, nil).Times(1)
				client.EXPECT().DownloadTask(gomock.Any(), mockTaskID, gomock.Any()).Return(stream, nil).Times(1)
				stream.EXPECT().Recv().Return(nil, errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name:      "download task completes at end of stream",
			available: true,
			mock: func(clientPool *dfdaemonclientmocks.MockPool, client *dfdaemonclientmocks.MockClient, stream *dfdaemonv2mocks.MockDfdaemonUpload_DownloadTaskClient, addr string) {
				clientPool.EXPECT().Get(addr).Return(client, nil).Times(1)
				client.EXPECT().DownloadTask(gomock.Any(), mockTaskID, gomock.Any()).Return(stream, nil).Times(1)
				gomock.InOrder(
					stream.EXPECT().Recv().Return(&dfdaemonv2.DownloadTaskResponse{}, nil).Times(1),
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
			hostManager := NewMockHostManager(ctl)
			peerManager := NewMockPeerManager(ctl)
			clientPool := dfdaemonclientmocks.NewMockPool(ctl)
			client := dfdaemonclientmocks.NewMockClient(ctl)
			stream := dfdaemonv2mocks.NewMockDfdaemonUpload_DownloadTaskClient(ctl)

			seedPeer := newSeedPeer(peerManager, hostManager, clientPool).(*seedPeer)
			var addr string
			if tc.available {
				mockHost := mockSeedHost(mockRawSeedHost.Port)
				addr = net.JoinHostPort(mockHost.IP, strconv.Itoa(int(mockHost.Port)))
				seedPeer.hosts[mockHost.Name] = mockHost
				seedPeer.hashring.Add(mockHost.Name)
			}

			tc.mock(clientPool, client, stream, addr)

			tc.expect(t, seedPeer.TriggerDownloadTask(context.Background(), mockTaskID, &dfdaemonv2.DownloadTaskRequest{}))
		})
	}
}

func TestSeedPeer_TriggerTask(t *testing.T) {
	tests := []struct {
		name       string
		rg         *nethttp.Range
		available  bool
		hostStored bool
		peerStored bool
		mock       func(ms *cdnsystemv1mocks.MockSeederServerMockRecorder, mockHost *Host)
		expect     func(t *testing.T, peerManager PeerManager, mockTask *Task, peer *Peer, result *schedulerv1.PeerResult, err error)
	}{
		{
			name:      "start obtain seed stream failed",
			available: false,
			expect: func(t *testing.T, peerManager PeerManager, mockTask *Task, peer *Peer, result *schedulerv1.PeerResult, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.Nil(peer)
				assert.Nil(result)
			},
		},
		{
			name:       "obtain seeds failed",
			available:  true,
			hostStored: true,
			mock: func(ms *cdnsystemv1mocks.MockSeederServerMockRecorder, mockHost *Host) {
				ms.ObtainSeeds(gomock.Any(), gomock.Any()).Return(status.Error(codes.Internal, "foo")).Times(1)
			},
			expect: func(t *testing.T, peerManager PeerManager, mockTask *Task, peer *Peer, result *schedulerv1.PeerResult, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.Nil(peer)
				assert.Nil(result)
				_, loaded := peerManager.Load(mockSeedPeerID)
				assert.False(loaded)
			},
		},
		{
			name:       "seed host is not found in host manager",
			available:  true,
			hostStored: false,
			mock: func(ms *cdnsystemv1mocks.MockSeederServerMockRecorder, mockHost *Host) {
				ms.ObtainSeeds(gomock.Any(), gomock.Any()).DoAndReturn(func(_ *cdnsystemv1.SeedRequest, stream cdnsystemv1.Seeder_ObtainSeedsServer) error {
					return stream.Send(&cdnsystemv1.PieceSeed{
						HostId:    mockHost.ID,
						PeerId:    mockSeedPeerID,
						PieceInfo: &commonv1.PieceInfo{PieceNum: common.BeginOfPiece},
					})
				}).Times(1)
			},
			expect: func(t *testing.T, peerManager PeerManager, mockTask *Task, peer *Peer, result *schedulerv1.PeerResult, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.Nil(peer)
				assert.Nil(result)
				_, loaded := peerManager.Load(mockSeedPeerID)
				assert.False(loaded)
			},
		},
		{
			name:       "trigger task with range stores the seed peer and its pieces",
			rg:         &nethttp.Range{Start: 0, Length: 10},
			available:  true,
			hostStored: true,
			mock: func(ms *cdnsystemv1mocks.MockSeederServerMockRecorder, mockHost *Host) {
				ms.ObtainSeeds(gomock.Cond(func(req *cdnsystemv1.SeedRequest) bool {
					return req.TaskId == mockTaskID &&
						req.Url == mockTaskURL &&
						req.UrlMeta.Tag == mockTaskTag &&
						req.UrlMeta.Application == mockTaskApplication &&
						req.UrlMeta.Filter == "bar" &&
						req.UrlMeta.Digest == mockTaskDigest.String() &&
						req.UrlMeta.Range == "0-9" &&
						req.UrlMeta.Priority == commonv1.Priority_LEVEL0
				}), gomock.Any()).DoAndReturn(func(_ *cdnsystemv1.SeedRequest, stream cdnsystemv1.Seeder_ObtainSeedsServer) error {
					for _, pieceSeed := range []*cdnsystemv1.PieceSeed{
						{
							HostId:    mockHost.ID,
							PeerId:    mockSeedPeerID,
							PieceInfo: &commonv1.PieceInfo{PieceNum: common.BeginOfPiece},
						},
						{
							HostId: mockHost.ID,
							PeerId: mockSeedPeerID,
							PieceInfo: &commonv1.PieceInfo{
								PieceNum:     0,
								RangeStart:   0,
								RangeSize:    100,
								PieceMd5:     mockPieceDigest.Encoded,
								DownloadCost: 10,
							},
						},
						{
							HostId: mockHost.ID,
							PeerId: mockSeedPeerID,
							Reuse:  true,
							PieceInfo: &commonv1.PieceInfo{
								PieceNum:     1,
								RangeStart:   100,
								RangeSize:    50,
								DownloadCost: 20,
							},
						},
						{
							HostId:          mockHost.ID,
							PeerId:          mockSeedPeerID,
							Done:            true,
							TotalPieceCount: 2,
							ContentLength:   150,
						},
					} {
						if err := stream.Send(pieceSeed); err != nil {
							return err
						}
					}

					return nil
				}).Times(1)
			},
			expect: func(t *testing.T, peerManager PeerManager, mockTask *Task, peer *Peer, result *schedulerv1.PeerResult, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(int32(2), result.TotalPieceCount)
				assert.Equal(int64(150), result.ContentLength)
				assert.Equal(mockSeedPeerID, peer.ID)
				assert.Equal(mockSeedHostID, peer.Host.ID)
				assert.EqualValues(&nethttp.Range{Start: 0, Length: 10}, peer.Range)
				assert.Equal(PeerStateRunning, peer.FSM.Current())
				assert.True(peer.FinishedPieces.Test(0))
				assert.True(peer.FinishedPieces.Test(1))
				assert.Equal([]time.Duration{10 * time.Millisecond, 20 * time.Millisecond}, peer.PieceCosts())

				storedPeer, loaded := peerManager.Load(mockSeedPeerID)
				assert.True(loaded)
				assert.Same(peer, storedPeer)

				piece, loaded := mockTask.LoadPiece(0)
				assert.True(loaded)
				assert.Equal(uint64(0), piece.Offset)
				assert.Equal(uint64(100), piece.Length)
				assert.Equal(mockPieceDigest, piece.Digest)
				assert.Equal(commonv2.TrafficType_BACK_TO_SOURCE, piece.TrafficType)
				assert.Equal(10*time.Millisecond, piece.Cost)

				piece, loaded = mockTask.LoadPiece(1)
				assert.True(loaded)
				assert.Equal(uint64(100), piece.Offset)
				assert.Equal(uint64(50), piece.Length)
				assert.Nil(piece.Digest)
				assert.Equal(commonv2.TrafficType_BACK_TO_SOURCE, piece.TrafficType)
				assert.Equal(20*time.Millisecond, piece.Cost)
			},
		},
		{
			name:       "trigger task reuses the seed peer stored in peer manager",
			available:  true,
			hostStored: true,
			peerStored: true,
			mock: func(ms *cdnsystemv1mocks.MockSeederServerMockRecorder, mockHost *Host) {
				ms.ObtainSeeds(gomock.Cond(func(req *cdnsystemv1.SeedRequest) bool {
					return req.TaskId == mockTaskID && req.UrlMeta.Range == ""
				}), gomock.Any()).DoAndReturn(func(_ *cdnsystemv1.SeedRequest, stream cdnsystemv1.Seeder_ObtainSeedsServer) error {
					for _, pieceSeed := range []*cdnsystemv1.PieceSeed{
						{
							HostId:    mockHost.ID,
							PeerId:    mockSeedPeerID,
							PieceInfo: &commonv1.PieceInfo{PieceNum: common.BeginOfPiece},
						},
						{
							HostId:          mockHost.ID,
							PeerId:          mockSeedPeerID,
							Done:            true,
							TotalPieceCount: 1,
							ContentLength:   100,
						},
					} {
						if err := stream.Send(pieceSeed); err != nil {
							return err
						}
					}

					return nil
				}).Times(1)
			},
			expect: func(t *testing.T, peerManager PeerManager, mockTask *Task, peer *Peer, result *schedulerv1.PeerResult, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(int32(1), result.TotalPieceCount)
				assert.Equal(int64(100), result.ContentLength)
				assert.Nil(peer.Range)
				assert.Equal(PeerStateRunning, peer.FSM.Current())
				assert.Empty(peer.PieceCosts())

				storedPeer, loaded := peerManager.Load(mockSeedPeerID)
				assert.True(loaded)
				assert.Same(storedPeer, peer)
			},
		},
		{
			name:       "stream failed after seed peer is initialized",
			available:  true,
			hostStored: true,
			mock: func(ms *cdnsystemv1mocks.MockSeederServerMockRecorder, mockHost *Host) {
				ms.ObtainSeeds(gomock.Any(), gomock.Any()).DoAndReturn(func(_ *cdnsystemv1.SeedRequest, stream cdnsystemv1.Seeder_ObtainSeedsServer) error {
					if err := stream.Send(&cdnsystemv1.PieceSeed{
						HostId:    mockHost.ID,
						PeerId:    mockSeedPeerID,
						PieceInfo: &commonv1.PieceInfo{PieceNum: common.BeginOfPiece},
					}); err != nil {
						return err
					}

					return status.Error(codes.Internal, "foo")
				}).Times(1)
			},
			expect: func(t *testing.T, peerManager PeerManager, mockTask *Task, peer *Peer, result *schedulerv1.PeerResult, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.Nil(peer)
				assert.Nil(result)

				storedPeer, loaded := peerManager.Load(mockSeedPeerID)
				assert.True(loaded)
				assert.Equal(PeerStateFailed, storedPeer.FSM.Current())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			gc := gc.NewMockGC(ctl)
			gc.EXPECT().Add(gomock.Any()).Return(nil).Times(2)
			seeder := cdnsystemv1mocks.NewMockSeederServer(ctl)
			clientPool := dfdaemonclientmocks.NewMockPool(ctl)

			hostManager, err := newHostManager(mockHostGCConfig, gc)
			if err != nil {
				t.Fatal(err)
			}

			peerManager, err := newPeerManager(mockPeerGCConfig, gc)
			if err != nil {
				t.Fatal(err)
			}

			mockHost := mockSeedHostServer(t, func(svr *grpc.Server) {
				cdnsystemv1.RegisterSeederServer(svr, seeder)
			})
			mockTask := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, WithDigest(mockTaskDigest))
			if tc.hostStored {
				hostManager.Store(mockHost)
			}

			if tc.peerStored {
				mockSeedPeer := NewPeer(mockSeedPeerID, mockTask, mockHost)
				mockSeedPeer.FSM.SetState(PeerStateReceivedNormal)
				peerManager.Store(mockSeedPeer)
			}

			seedPeer := newSeedPeer(peerManager, hostManager, clientPool,
				grpc.WithTransportCredentials(insecure.NewCredentials())).(*seedPeer)
			if tc.available {
				seedPeer.hosts[mockHost.Name] = mockHost
				seedPeer.hashring.Add(mockHost.Name)
			}

			if tc.mock != nil {
				tc.mock(seeder.EXPECT(), mockHost)
			}

			peer, result, err := seedPeer.TriggerTask(context.Background(), tc.rg, mockTask)
			tc.expect(t, peerManager, mockTask, peer, result, err)
		})
	}
}
