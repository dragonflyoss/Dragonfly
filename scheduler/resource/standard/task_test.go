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

package standard

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/looplab/fsm"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"

	commonv1 "d7y.io/api/v2/pkg/apis/common/v1"
	commonv2 "d7y.io/api/v2/pkg/apis/common/v2"
	schedulerv1 "d7y.io/api/v2/pkg/apis/scheduler/v1"
	v1mocks "d7y.io/api/v2/pkg/apis/scheduler/v1/mocks"

	"d7y.io/dragonfly/v2/pkg/container/set"
	"d7y.io/dragonfly/v2/pkg/digest"
	"d7y.io/dragonfly/v2/pkg/idgen"
	"d7y.io/dragonfly/v2/pkg/types"
)

var (
	mockPiece = &Piece{
		Number:      1,
		ParentID:    idgen.PeerID(),
		Offset:      0,
		Length:      100,
		Digest:      mockPieceDigest,
		TrafficType: commonv2.TrafficType_REMOTE_PEER,
		Cost:        1 * time.Second,
		CreatedAt:   time.Now(),
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
	mockPieceDigest                    = digest.New(digest.AlgorithmMD5, "ad83a945518a4ef007d8b2db2ef165b3")
)

func TestTask_NewTask(t *testing.T) {
	tests := []struct {
		name    string
		options []TaskOption
		expect  func(t *testing.T, task *Task)
	}{
		{
			name:    "new task",
			options: []TaskOption{},
			expect: func(t *testing.T, task *Task) {
				assert := assert.New(t)
				assert.Equal(mockTaskID, task.ID)
				assert.Equal(commonv2.TaskType_STANDARD, task.Type)
				assert.Equal(mockTaskURL, task.URL)
				assert.Nil(task.Digest)
				assert.Equal(mockTaskTag, task.Tag)
				assert.Equal(mockTaskApplication, task.Application)
				assert.EqualValues(mockTaskFilteredQueryParams, task.FilteredQueryParams)
				assert.EqualValues(mockTaskHeader, task.Header)
				assert.Empty(task.DirectPiece)
				assert.Equal(int64(-1), task.ContentLength.Load())
				assert.Equal(uint64(0), task.PieceLength)
				assert.Equal(int32(0), task.TotalPieceCount.Load())
				assert.Equal(int32(200), task.BackToSourceLimit.Load())
				assert.Equal(uint(0), task.BackToSourcePeers.Len())
				assert.Equal(TaskStatePending, task.FSM.Current())
				assert.Empty(task.Pieces)
				assert.Equal(0, task.PeerCount())
				assert.Equal(int32(0), task.PeerFailedCount.Load())
				assert.NotEmpty(task.CreatedAt.Load())
				assert.NotEmpty(task.UpdatedAt.Load())
				assert.NotNil(task.Log)
			},
		},
		{
			name:    "new task with piece length",
			options: []TaskOption{WithPieceLength(mockTaskPieceLength)},
			expect: func(t *testing.T, task *Task) {
				assert := assert.New(t)
				assert.Equal(mockTaskID, task.ID)
				assert.Equal(commonv2.TaskType_STANDARD, task.Type)
				assert.Equal(mockTaskURL, task.URL)
				assert.Nil(task.Digest)
				assert.Equal(mockTaskTag, task.Tag)
				assert.Equal(mockTaskApplication, task.Application)
				assert.EqualValues(mockTaskFilteredQueryParams, task.FilteredQueryParams)
				assert.EqualValues(mockTaskHeader, task.Header)
				assert.Empty(task.DirectPiece)
				assert.Equal(int64(-1), task.ContentLength.Load())
				assert.Equal(mockTaskPieceLength, task.PieceLength)
				assert.Equal(int32(0), task.TotalPieceCount.Load())
				assert.Equal(int32(200), task.BackToSourceLimit.Load())
				assert.Equal(uint(0), task.BackToSourcePeers.Len())
				assert.Equal(TaskStatePending, task.FSM.Current())
				assert.Empty(task.Pieces)
				assert.Equal(0, task.PeerCount())
				assert.NotEmpty(task.CreatedAt.Load())
				assert.NotEmpty(task.UpdatedAt.Load())
				assert.NotNil(task.Log)
			},
		},
		{
			name:    "new task with digest",
			options: []TaskOption{WithDigest(mockTaskDigest)},
			expect: func(t *testing.T, task *Task) {
				assert := assert.New(t)
				assert.Equal(mockTaskID, task.ID)
				assert.Equal(commonv2.TaskType_STANDARD, task.Type)
				assert.Equal(mockTaskURL, task.URL)
				assert.EqualValues(mockTaskDigest, task.Digest)
				assert.Equal(mockTaskTag, task.Tag)
				assert.Equal(mockTaskApplication, task.Application)
				assert.EqualValues(mockTaskFilteredQueryParams, task.FilteredQueryParams)
				assert.EqualValues(mockTaskHeader, task.Header)
				assert.Empty(task.DirectPiece)
				assert.Equal(int64(-1), task.ContentLength.Load())
				assert.Equal(int32(0), task.TotalPieceCount.Load())
				assert.Equal(int32(200), task.BackToSourceLimit.Load())
				assert.Equal(uint(0), task.BackToSourcePeers.Len())
				assert.Equal(TaskStatePending, task.FSM.Current())
				assert.Empty(task.Pieces)
				assert.Equal(0, task.PeerCount())
				assert.NotEmpty(task.CreatedAt.Load())
				assert.NotEmpty(task.UpdatedAt.Load())
				assert.NotNil(task.Log)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.expect(t, NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, tc.options...))
		})
	}
}

func TestTask_FSMEvent(t *testing.T) {
	tests := []struct {
		name   string
		state  string
		event  string
		expect func(t *testing.T, task *Task, err error)
	}{
		{
			name:  "download from pending",
			state: TaskStatePending,
			event: TaskEventDownload,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(TaskStateRunning, task.FSM.Current())
			},
		},
		{
			name:  "download from succeeded",
			state: TaskStateSucceeded,
			event: TaskEventDownload,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(TaskStateRunning, task.FSM.Current())
			},
		},
		{
			name:  "download from failed",
			state: TaskStateFailed,
			event: TaskEventDownload,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(TaskStateRunning, task.FSM.Current())
			},
		},
		{
			name:  "download from leave",
			state: TaskStateLeave,
			event: TaskEventDownload,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(TaskStateRunning, task.FSM.Current())
			},
		},
		{
			name:  "download succeeded from running",
			state: TaskStateRunning,
			event: TaskEventDownloadSucceeded,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(TaskStateSucceeded, task.FSM.Current())
			},
		},
		{
			name:  "download succeeded from failed",
			state: TaskStateFailed,
			event: TaskEventDownloadSucceeded,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(TaskStateSucceeded, task.FSM.Current())
			},
		},
		{
			name:  "download failed from running",
			state: TaskStateRunning,
			event: TaskEventDownloadFailed,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(TaskStateFailed, task.FSM.Current())
			},
		},
		{
			name:  "leave from succeeded",
			state: TaskStateSucceeded,
			event: TaskEventLeave,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(TaskStateLeave, task.FSM.Current())
			},
		},
		{
			name:  "download from running is rejected",
			state: TaskStateRunning,
			event: TaskEventDownload,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.ErrorAs(err, new(fsm.InvalidEventError))
				assert.Equal(TaskStateRunning, task.FSM.Current())
			},
		},
		{
			name:  "download succeeded from pending is rejected",
			state: TaskStatePending,
			event: TaskEventDownloadSucceeded,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.ErrorAs(err, new(fsm.InvalidEventError))
				assert.Equal(TaskStatePending, task.FSM.Current())
			},
		},
		{
			name:  "download failed from pending is rejected",
			state: TaskStatePending,
			event: TaskEventDownloadFailed,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.ErrorAs(err, new(fsm.InvalidEventError))
				assert.Equal(TaskStatePending, task.FSM.Current())
			},
		},
		{
			name:  "download failed from succeeded is rejected",
			state: TaskStateSucceeded,
			event: TaskEventDownloadFailed,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.ErrorAs(err, new(fsm.InvalidEventError))
				assert.Equal(TaskStateSucceeded, task.FSM.Current())
			},
		},
		{
			name:  "leave from leave is rejected",
			state: TaskStateLeave,
			event: TaskEventLeave,
			expect: func(t *testing.T, task *Task, err error) {
				assert := assert.New(t)
				assert.ErrorAs(err, new(fsm.InvalidEventError))
				assert.Equal(TaskStateLeave, task.FSM.Current())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			task.FSM.SetState(tc.state)

			tc.expect(t, task, task.FSM.Event(context.Background(), tc.event))
		})
	}
}

func TestTask_LoadPeer(t *testing.T) {
	tests := []struct {
		name   string
		peerID string
		expect func(t *testing.T, peer *Peer, loaded bool)
	}{
		{
			name:   "load peer",
			peerID: mockPeerID,
			expect: func(t *testing.T, peer *Peer, loaded bool) {
				assert := assert.New(t)
				assert.True(loaded)
				assert.Equal(mockPeerID, peer.ID)
			},
		},
		{
			name:   "peer does not exist",
			peerID: idgen.PeerID(),
			expect: func(t *testing.T, peer *Peer, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
			},
		},
		{
			name:   "load key is empty",
			peerID: "",
			expect: func(t *testing.T, peer *Peer, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			mockPeer := NewPeer(mockPeerID, task, mockHost)

			task.StorePeer(mockPeer)
			peer, loaded := task.LoadPeer(tc.peerID)
			tc.expect(t, peer, loaded)
		})
	}
}

func TestTask_LoadRandomPeers(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, task *Task, host *Host)
	}{
		{
			name: "load random peers",
			expect: func(t *testing.T, task *Task, host *Host) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, host)
				mockPeerF := NewPeer(idgen.PeerID(), task, host)
				mockPeerG := NewPeer(idgen.PeerID(), task, host)
				mockPeerH := NewPeer(idgen.PeerID(), task, host)

				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				task.StorePeer(mockPeerH)

				assert.Empty(task.LoadRandomPeers(0))
				assert.Len(task.LoadRandomPeers(1), 1)
				assert.Len(task.LoadRandomPeers(2), 2)
				assert.Len(task.LoadRandomPeers(3), 3)
				assert.Len(task.LoadRandomPeers(4), 4)
				assert.Len(task.LoadRandomPeers(5), 4)
			},
		},
		{
			name: "load empty peers",
			expect: func(t *testing.T, task *Task, host *Host) {
				assert := assert.New(t)
				assert.Empty(task.LoadRandomPeers(0))
				assert.Empty(task.LoadRandomPeers(1))
				assert.Empty(task.LoadRandomPeers(2))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			tc.expect(t, task, host)
		})
	}
}

func TestTask_LoadPeers(t *testing.T) {
	tests := []struct {
		name      string
		peerCount int
		expect    func(t *testing.T, peerIDs []string, peers []*Peer)
	}{
		{
			name:      "task has no peers",
			peerCount: 0,
			expect: func(t *testing.T, peerIDs []string, peers []*Peer) {
				assert := assert.New(t)
				assert.Empty(peers)
			},
		},
		{
			name:      "task has peers",
			peerCount: 3,
			expect: func(t *testing.T, peerIDs []string, peers []*Peer) {
				assert := assert.New(t)
				assert.Len(peers, 3)
				var ids []string
				for _, peer := range peers {
					ids = append(ids, peer.ID)
				}

				assert.ElementsMatch(peerIDs, ids)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			var peerIDs []string
			for range tc.peerCount {
				peer := NewPeer(idgen.PeerID(), task, mockHost)
				task.StorePeer(peer)
				peerIDs = append(peerIDs, peer.ID)
			}

			tc.expect(t, peerIDs, task.LoadPeers())
		})
	}
}

func TestTask_LoadFinishedPeers(t *testing.T) {
	tests := []struct {
		name   string
		states []string
		expect func(t *testing.T, finishedPeers []*Peer)
	}{
		{
			name:   "task has no peers",
			states: []string{},
			expect: func(t *testing.T, finishedPeers []*Peer) {
				assert := assert.New(t)
				assert.Empty(finishedPeers)
			},
		},
		{
			name:   "task has no finished peers",
			states: []string{PeerStatePending, PeerStateReceivedNormal, PeerStateRunning, PeerStateBackToSource},
			expect: func(t *testing.T, finishedPeers []*Peer) {
				assert := assert.New(t)
				assert.Empty(finishedPeers)
			},
		},
		{
			name:   "only succeeded, failed and left peers are finished",
			states: []string{PeerStatePending, PeerStateRunning, PeerStateBackToSource, PeerStateSucceeded, PeerStateFailed, PeerStateLeave},
			expect: func(t *testing.T, finishedPeers []*Peer) {
				assert := assert.New(t)
				var states []string
				for _, peer := range finishedPeers {
					states = append(states, peer.FSM.Current())
				}

				assert.ElementsMatch([]string{PeerStateSucceeded, PeerStateFailed, PeerStateLeave}, states)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			for _, state := range tc.states {
				peer := NewPeer(idgen.PeerID(), task, mockHost)
				peer.FSM.SetState(state)
				task.StorePeer(peer)
			}

			tc.expect(t, task.LoadFinishedPeers())
		})
	}
}

func TestTask_StorePeer(t *testing.T) {
	tests := []struct {
		name   string
		peerID string
		expect func(t *testing.T, task *Task, mockPeer *Peer)
	}{
		{
			name:   "store peer",
			peerID: mockPeerID,
			expect: func(t *testing.T, task *Task, mockPeer *Peer) {
				assert := assert.New(t)
				peer, loaded := task.LoadPeer(mockPeerID)
				assert.True(loaded)
				assert.Equal(mockPeerID, peer.ID)
				assert.Equal(1, task.PeerCount())
			},
		},
		{
			name:   "store key is empty",
			peerID: "",
			expect: func(t *testing.T, task *Task, mockPeer *Peer) {
				assert := assert.New(t)
				peer, loaded := task.LoadPeer("")
				assert.True(loaded)
				assert.Equal("", peer.ID)
				assert.Equal(1, task.PeerCount())
			},
		},
		{
			name:   "store peer with duplicate id keeps the first peer",
			peerID: mockPeerID,
			expect: func(t *testing.T, task *Task, mockPeer *Peer) {
				assert := assert.New(t)
				task.StorePeer(NewPeer(mockPeerID, task, mockPeer.Host))
				peer, loaded := task.LoadPeer(mockPeerID)
				assert.True(loaded)
				assert.Same(mockPeer, peer)
				assert.Equal(1, task.PeerCount())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			mockPeer := NewPeer(tc.peerID, task, mockHost)

			task.StorePeer(mockPeer)
			tc.expect(t, task, mockPeer)
		})
	}
}

func TestTask_DeletePeer(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, task *Task, mockHost *Host, mockPeer *Peer)
	}{
		{
			name: "delete peer",
			expect: func(t *testing.T, task *Task, mockHost *Host, mockPeer *Peer) {
				assert := assert.New(t)
				task.StorePeer(mockPeer)
				task.DeletePeer(mockPeer.ID)
				_, loaded := task.LoadPeer(mockPeer.ID)
				assert.False(loaded)
				assert.Equal(0, task.PeerCount())
			},
		},
		{
			name: "delete key is empty",
			expect: func(t *testing.T, task *Task, mockHost *Host, mockPeer *Peer) {
				assert := assert.New(t)
				task.StorePeer(mockPeer)
				task.DeletePeer("")
				peer, loaded := task.LoadPeer(mockPeer.ID)
				assert.True(loaded)
				assert.Equal(mockPeer.ID, peer.ID)
				assert.Equal(1, task.PeerCount())
			},
		},
		{
			name: "delete peer releases the upload load of its parents and children",
			expect: func(t *testing.T, task *Task, mockHost *Host, mockPeer *Peer) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)
				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				if err := task.AddPeerEdge(mockPeerE, mockPeerF); err != nil {
					t.Fatal(err)
				}

				if err := task.AddPeerEdge(mockPeerF, mockPeerG); err != nil {
					t.Fatal(err)
				}

				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())

				task.DeletePeer(mockPeerF.ID)
				_, loaded := task.LoadPeer(mockPeerF.ID)
				assert.False(loaded)
				assert.Equal(2, task.PeerCount())
				assert.Empty(mockPeerE.Children())
				assert.Empty(mockPeerG.Parents())
				assert.Equal(int32(0), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			mockPeer := NewPeer(mockPeerID, task, mockHost)

			tc.expect(t, task, mockHost, mockPeer)
		})
	}
}

func TestTask_PeerCount(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, mockPeer *Peer, task *Task)
	}{
		{
			name: "task has no peers",
			expect: func(t *testing.T, mockPeer *Peer, task *Task) {
				assert := assert.New(t)
				assert.Equal(0, task.PeerCount())
			},
		},
		{
			name: "task has peers",
			expect: func(t *testing.T, mockPeer *Peer, task *Task) {
				assert := assert.New(t)
				task.StorePeer(mockPeer)
				assert.Equal(1, task.PeerCount())
				task.DeletePeer(mockPeer.ID)
				assert.Equal(0, task.PeerCount())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			mockPeer := NewPeer(mockPeerID, task, mockHost)

			tc.expect(t, mockPeer, task)
		})
	}
}

func TestTask_AddPeerEdge(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, mockHost *Host, task *Task)
	}{
		{
			name: "add peer edge failed",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)

				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				mockHost.StorePeer(mockPeerE)
				mockHost.StorePeer(mockPeerF)
				mockHost.StorePeer(mockPeerG)

				err := task.AddPeerEdge(mockPeerE, mockPeerF)
				assert.NoError(err)
				assert.Equal(mockPeerF.ID, mockPeerE.Children()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerF.Parents()[0].ID)
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(1), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerF, mockPeerG)
				assert.NoError(err)
				assert.Equal(mockPeerG.ID, mockPeerF.Children()[0].ID)
				assert.Equal(mockPeerF.ID, mockPeerG.Parents()[0].ID)
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerG, mockPeerE)
				assert.Error(err)
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())
			},
		},
		{
			name: "add peer edge",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)

				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				mockHost.StorePeer(mockPeerE)
				mockHost.StorePeer(mockPeerF)
				mockHost.StorePeer(mockPeerG)

				err := task.AddPeerEdge(mockPeerE, mockPeerF)
				assert.NoError(err)
				assert.Equal(mockPeerF.ID, mockPeerE.Children()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerF.Parents()[0].ID)
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(1), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerE, mockPeerG)
				assert.NoError(err)
				assert.Len(mockPeerE.Children(), 2)
				assert.Equal(mockPeerE.ID, mockPeerG.Parents()[0].ID)
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerG, mockPeerF)
				assert.NoError(err)
				assert.Len(mockPeerF.Parents(), 2)
				assert.Equal(mockPeerF.ID, mockPeerG.Children()[0].ID)
				assert.Equal(int32(3), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(3), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())
			},
		},
		{
			name: "add peer edge accumulates bandwidth, piece count and content length on the parent host",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				task.PieceLength = 1024
				task.ContentLength.Store(4096)
				mockParent := NewPeer(idgen.PeerID(), task, mockHost)
				mockChild := NewPeer(idgen.PeerID(), task, mockHost, WithConcurrentPieceCount(4))
				task.StorePeer(mockParent)
				task.StorePeer(mockChild)

				err := task.AddPeerEdge(mockParent, mockChild)
				assert.NoError(err)
				assert.Equal(uint64(1024*4*8), mockHost.TxBandwidth.Load())
				assert.Equal(uint64(4), mockHost.ConcurrentUploadPieceCount.Load())
				assert.Equal(uint64(4096), mockHost.UploadContentLength.Load())
			},
		},
		{
			name: "add peer edge with unknown content length does not count upload content length",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				task.PieceLength = 1024
				mockParent := NewPeer(idgen.PeerID(), task, mockHost)
				mockChild := NewPeer(idgen.PeerID(), task, mockHost)
				task.StorePeer(mockParent)
				task.StorePeer(mockChild)

				err := task.AddPeerEdge(mockParent, mockChild)
				assert.NoError(err)
				assert.Equal(uint64(1024*8*8), mockHost.TxBandwidth.Load())
				assert.Equal(uint64(8), mockHost.ConcurrentUploadPieceCount.Load())
				assert.Equal(uint64(0), mockHost.UploadContentLength.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			tc.expect(t, mockHost, task)
		})
	}
}

func TestTask_AddPeerEdges(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, mockHost *Host, task *Task)
	}{
		{
			name: "add edges from multiple parents",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				task.PieceLength = 1024
				task.ContentLength.Store(4096)
				mockParentE := NewPeer(idgen.PeerID(), task, mockHost)
				mockParentF := NewPeer(idgen.PeerID(), task, mockHost)
				mockChild := NewPeer(idgen.PeerID(), task, mockHost)
				task.StorePeer(mockParentE)
				task.StorePeer(mockParentF)
				task.StorePeer(mockChild)

				added := task.AddPeerEdges([]*Peer{mockParentE, mockParentF}, mockChild)
				var addedIDs []string
				for _, peer := range added {
					addedIDs = append(addedIDs, peer.ID)
				}

				assert.ElementsMatch([]string{mockParentE.ID, mockParentF.ID}, addedIDs)
				assert.Len(mockChild.Parents(), 2)
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
				assert.Equal(uint64(2*1024*8*8), mockHost.TxBandwidth.Load())
				assert.Equal(uint64(2*8), mockHost.ConcurrentUploadPieceCount.Load())
				assert.Equal(uint64(2*4096), mockHost.UploadContentLength.Load())
			},
		},
		{
			name: "skip parents with an existing edge, the child itself and parents that would create a cycle",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)
				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				if err := task.AddPeerEdge(mockPeerE, mockPeerF); err != nil {
					t.Fatal(err)
				}

				if err := task.AddPeerEdge(mockPeerF, mockPeerG); err != nil {
					t.Fatal(err)
				}

				added := task.AddPeerEdges([]*Peer{mockPeerE, mockPeerF, mockPeerG}, mockPeerF)
				assert.Empty(added)
				assert.Len(mockPeerF.Parents(), 1)
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
			},
		},
		{
			name: "skip parents that are not stored in task",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockParentE := NewPeer(idgen.PeerID(), task, mockHost)
				mockParentF := NewPeer(idgen.PeerID(), task, mockHost)
				mockChild := NewPeer(idgen.PeerID(), task, mockHost)
				task.StorePeer(mockParentE)
				task.StorePeer(mockChild)

				added := task.AddPeerEdges([]*Peer{mockParentE, mockParentF}, mockChild)
				assert.Len(added, 1)
				assert.Equal(mockParentE.ID, added[0].ID)
				assert.Len(mockChild.Parents(), 1)
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(1), mockHost.UploadCount.Load())
			},
		},
		{
			name: "child is not stored in task",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockParent := NewPeer(idgen.PeerID(), task, mockHost)
				mockChild := NewPeer(idgen.PeerID(), task, mockHost)
				task.StorePeer(mockParent)

				added := task.AddPeerEdges([]*Peer{mockParent}, mockChild)
				assert.Empty(added)
				assert.Equal(int32(0), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), mockHost.UploadCount.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			tc.expect(t, mockHost, task)
		})
	}
}

func TestTask_DeletePeerInEdges(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, mockHost *Host, task *Task)
	}{
		{
			name: "delete peer inedges failed",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				assert.Error(task.DeletePeerInEdges(mockPeerID))
			},
		},
		{
			name: "delete peer inedges",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)

				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				mockHost.StorePeer(mockPeerE)
				mockHost.StorePeer(mockPeerF)
				mockHost.StorePeer(mockPeerG)

				var (
					err    error
					degree int
				)
				err = task.AddPeerEdge(mockPeerE, mockPeerF)
				assert.NoError(err)
				assert.Equal(mockPeerF.ID, mockPeerE.Children()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerF.Parents()[0].ID)
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(1), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerE, mockPeerG)
				assert.NoError(err)
				assert.Len(mockPeerE.Children(), 2)
				assert.Equal(mockPeerE.ID, mockPeerG.Parents()[0].ID)
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerG, mockPeerF)
				assert.NoError(err)
				assert.Len(mockPeerF.Parents(), 2)
				assert.Equal(mockPeerF.ID, mockPeerG.Children()[0].ID)
				assert.Equal(int32(3), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(3), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.DeletePeerInEdges(mockPeerE.ID)
				assert.NoError(err)
				assert.Len(mockPeerE.Children(), 2)
				assert.Len(mockPeerF.Parents(), 2)
				assert.Equal(mockPeerE.ID, mockPeerG.Parents()[0].ID)
				assert.Equal(mockPeerF.ID, mockPeerG.Children()[0].ID)
				assert.Equal(int32(3), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(3), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.DeletePeerInEdges(mockPeerF.ID)
				assert.NoError(err)
				assert.Equal(mockPeerG.ID, mockPeerE.Children()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerG.Parents()[0].ID)
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(3), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.DeletePeerInEdges(mockPeerG.ID)
				assert.NoError(err)
				degree, err = task.PeerDegree(mockPeerE.ID)
				assert.NoError(err)
				assert.Equal(0, degree)
				degree, err = task.PeerDegree(mockPeerF.ID)
				assert.NoError(err)
				assert.Equal(0, degree)
				degree, err = task.PeerDegree(mockPeerG.ID)
				assert.NoError(err)
				assert.Equal(0, degree)
				assert.Equal(int32(0), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(3), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			tc.expect(t, mockHost, task)
		})
	}
}

func TestTask_DeletePeerOutEdges(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, mockHost *Host, task *Task)
	}{
		{
			name: "delete peer outedges failed",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				assert.Error(task.DeletePeerOutEdges(mockPeerID))
			},
		},
		{
			name: "delete peer outedges",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)

				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				mockHost.StorePeer(mockPeerE)
				mockHost.StorePeer(mockPeerF)
				mockHost.StorePeer(mockPeerG)

				var (
					err    error
					degree int
				)
				err = task.AddPeerEdge(mockPeerE, mockPeerF)
				assert.NoError(err)
				assert.Equal(mockPeerF.ID, mockPeerE.Children()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerF.Parents()[0].ID)
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(1), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerE, mockPeerG)
				assert.NoError(err)
				assert.Len(mockPeerE.Children(), 2)
				assert.Equal(mockPeerE.ID, mockPeerG.Parents()[0].ID)
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerG, mockPeerF)
				assert.NoError(err)
				assert.Len(mockPeerF.Parents(), 2)
				assert.Equal(mockPeerF.ID, mockPeerG.Children()[0].ID)
				assert.Equal(int32(3), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(3), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.DeletePeerOutEdges(mockPeerE.ID)
				assert.NoError(err)
				assert.Len(mockPeerF.Parents(), 1)
				assert.Empty(mockPeerG.Parents())
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(3), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.DeletePeerOutEdges(mockPeerF.ID)
				assert.NoError(err)
				assert.Equal(mockPeerG.ID, mockPeerF.Parents()[0].ID)
				assert.Equal(mockPeerF.ID, mockPeerG.Children()[0].ID)
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(3), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.DeletePeerOutEdges(mockPeerG.ID)
				assert.NoError(err)
				degree, err = task.PeerDegree(mockPeerE.ID)
				assert.NoError(err)
				assert.Equal(0, degree)
				degree, err = task.PeerDegree(mockPeerF.ID)
				assert.NoError(err)
				assert.Equal(0, degree)
				degree, err = task.PeerDegree(mockPeerG.ID)
				assert.NoError(err)
				assert.Equal(0, degree)
				assert.Equal(int32(0), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(3), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())
			},
		},
		{
			name: "delete peer outedges releases upload load of children with different concurrent piece counts",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				task.PieceLength = 1024
				task.ContentLength.Store(4096)

				mockParent := NewPeer(idgen.PeerID(), task, mockHost, WithConcurrentPieceCount(4))
				mockChildE := NewPeer(idgen.PeerID(), task, mockHost, WithConcurrentPieceCount(8))
				mockChildF := NewPeer(idgen.PeerID(), task, mockHost, WithConcurrentPieceCount(1))

				task.StorePeer(mockParent)
				task.StorePeer(mockChildE)
				task.StorePeer(mockChildF)
				mockHost.StorePeer(mockParent)
				mockHost.StorePeer(mockChildE)
				mockHost.StorePeer(mockChildF)

				var err error
				err = task.AddPeerEdge(mockParent, mockChildE)
				assert.NoError(err)
				err = task.AddPeerEdge(mockParent, mockChildF)
				assert.NoError(err)
				assert.Equal(uint64(1024*8*8+1024*1*8), mockHost.TxBandwidth.Load())
				assert.Equal(uint64(9), mockHost.ConcurrentUploadPieceCount.Load())
				assert.Equal(uint64(2*4096), mockHost.UploadContentLength.Load())
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())

				err = task.DeletePeerOutEdges(mockParent.ID)
				assert.NoError(err)
				assert.Equal(uint64(0), mockHost.TxBandwidth.Load())
				assert.Equal(uint64(0), mockHost.ConcurrentUploadPieceCount.Load())
				assert.Equal(uint64(0), mockHost.UploadContentLength.Load())
				assert.Equal(int32(0), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			tc.expect(t, mockHost, task)
		})
	}
}

func TestTask_CanAddPeerEdge(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, mockHost *Host, task *Task)
	}{
		{
			name: "peer can not add edge",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)

				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				mockHost.StorePeer(mockPeerE)
				mockHost.StorePeer(mockPeerF)
				mockHost.StorePeer(mockPeerG)

				err := task.AddPeerEdge(mockPeerE, mockPeerF)
				assert.NoError(err)
				assert.Equal(mockPeerF.ID, mockPeerE.Children()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerF.Parents()[0].ID)
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(1), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerF, mockPeerG)
				assert.NoError(err)
				assert.Equal(mockPeerG.ID, mockPeerF.Children()[0].ID)
				assert.Equal(mockPeerF.ID, mockPeerG.Parents()[0].ID)
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				assert.False(task.CanAddPeerEdge(mockPeerG.ID, mockPeerE.ID))
			},
		},
		{
			name: "peer can add edge",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)

				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				mockHost.StorePeer(mockPeerE)
				mockHost.StorePeer(mockPeerF)
				mockHost.StorePeer(mockPeerG)

				err := task.AddPeerEdge(mockPeerE, mockPeerF)
				assert.NoError(err)
				assert.Equal(mockPeerF.ID, mockPeerE.Children()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerF.Parents()[0].ID)
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(1), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerE, mockPeerG)
				assert.NoError(err)
				assert.Len(mockPeerE.Children(), 2)
				assert.Equal(mockPeerE.ID, mockPeerG.Parents()[0].ID)
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				assert.True(task.CanAddPeerEdge(mockPeerG.ID, mockPeerF.ID))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			tc.expect(t, mockHost, task)
		})
	}
}

func TestTask_CanAddPeerEdges(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, mockHost *Host, task *Task)
	}{
		{
			name: "all candidates can add edge",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)
				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)

				assert.Equal(map[string]struct{}{mockPeerE.ID: {}, mockPeerF.ID: {}}, task.CanAddPeerEdges([]string{mockPeerE.ID, mockPeerF.ID}, mockPeerG.ID))
			},
		},
		{
			name: "candidates with an existing edge, the child itself and unknown peers are filtered",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)
				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				if err := task.AddPeerEdge(mockPeerE, mockPeerF); err != nil {
					t.Fatal(err)
				}

				if err := task.AddPeerEdge(mockPeerF, mockPeerG); err != nil {
					t.Fatal(err)
				}

				assert.Equal(map[string]struct{}{mockPeerE.ID: {}}, task.CanAddPeerEdges([]string{mockPeerE.ID, mockPeerF.ID, mockPeerG.ID, idgen.PeerID()}, mockPeerG.ID))
			},
		},
		{
			name: "candidates that would create a cycle are filtered",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)
				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				if err := task.AddPeerEdge(mockPeerE, mockPeerF); err != nil {
					t.Fatal(err)
				}

				if err := task.AddPeerEdge(mockPeerF, mockPeerG); err != nil {
					t.Fatal(err)
				}

				assert.Empty(task.CanAddPeerEdges([]string{mockPeerF.ID, mockPeerG.ID}, mockPeerE.ID))
			},
		},
		{
			name: "child is not stored in task",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				task.StorePeer(mockPeerE)

				assert.Empty(task.CanAddPeerEdges([]string{mockPeerE.ID}, idgen.PeerID()))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			tc.expect(t, mockHost, task)
		})
	}
}

func TestTask_PeerDegree(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, mockHost *Host, task *Task)
	}{
		{
			name: "get peer degree failed",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				_, err := task.PeerDegree(mockPeerID)
				assert.Error(err)
			},
		},
		{
			name: "peer get degree",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)

				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				mockHost.StorePeer(mockPeerE)
				mockHost.StorePeer(mockPeerF)
				mockHost.StorePeer(mockPeerG)

				err := task.AddPeerEdge(mockPeerE, mockPeerF)
				assert.NoError(err)
				assert.Equal(mockPeerF.ID, mockPeerE.Children()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerF.Parents()[0].ID)
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(1), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerG, mockPeerE)
				assert.NoError(err)
				assert.Equal(mockPeerG.ID, mockPeerE.Parents()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerG.Children()[0].ID)
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				degree, err := task.PeerDegree(mockPeerE.ID)
				assert.NoError(err)
				assert.Equal(2, degree)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			tc.expect(t, mockHost, task)
		})
	}
}

func TestTask_PeerInDegree(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, mockHost *Host, task *Task)
	}{
		{
			name: "get peer indegree failed",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				_, err := task.PeerInDegree(mockPeerID)
				assert.Error(err)
			},
		},
		{
			name: "peer get indegree",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)

				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				mockHost.StorePeer(mockPeerE)
				mockHost.StorePeer(mockPeerF)
				mockHost.StorePeer(mockPeerG)

				err := task.AddPeerEdge(mockPeerE, mockPeerF)
				assert.NoError(err)
				assert.Equal(mockPeerF.ID, mockPeerE.Children()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerF.Parents()[0].ID)
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(1), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerG, mockPeerE)
				assert.NoError(err)
				assert.Equal(mockPeerG.ID, mockPeerE.Parents()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerG.Children()[0].ID)
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				degree, err := task.PeerInDegree(mockPeerE.ID)
				assert.NoError(err)
				assert.Equal(1, degree)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			tc.expect(t, mockHost, task)
		})
	}
}

func TestTask_PeerOutDegree(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, mockHost *Host, task *Task)
	}{
		{
			name: "get peer outdegree failed",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				_, err := task.PeerOutDegree(mockPeerID)
				assert.Error(err)
			},
		},
		{
			name: "peer get outdegree",
			expect: func(t *testing.T, mockHost *Host, task *Task) {
				assert := assert.New(t)
				mockPeerE := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerF := NewPeer(idgen.PeerID(), task, mockHost)
				mockPeerG := NewPeer(idgen.PeerID(), task, mockHost)

				task.StorePeer(mockPeerE)
				task.StorePeer(mockPeerF)
				task.StorePeer(mockPeerG)
				mockHost.StorePeer(mockPeerE)
				mockHost.StorePeer(mockPeerF)
				mockHost.StorePeer(mockPeerG)

				err := task.AddPeerEdge(mockPeerE, mockPeerF)
				assert.NoError(err)
				assert.Equal(mockPeerF.ID, mockPeerE.Children()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerF.Parents()[0].ID)
				assert.Equal(int32(1), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(1), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				err = task.AddPeerEdge(mockPeerG, mockPeerE)
				assert.NoError(err)
				assert.Equal(mockPeerG.ID, mockPeerE.Parents()[0].ID)
				assert.Equal(mockPeerE.ID, mockPeerG.Children()[0].ID)
				assert.Equal(int32(2), mockHost.ConcurrentUploadCount.Load())
				assert.Equal(int64(2), mockHost.UploadCount.Load())
				assert.Equal(int32(3), mockHost.PeerCount.Load())

				degree, err := task.PeerOutDegree(mockPeerE.ID)
				assert.NoError(err)
				assert.Equal(1, degree)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			tc.expect(t, mockHost, task)
		})
	}
}

func TestTask_HasAvailablePeer(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		hostID  string
		blocked bool
		expect  func(t *testing.T, hasAvailablePeer bool)
	}{
		{
			name:    "blocklist includes peer",
			state:   PeerStateSucceeded,
			hostID:  "",
			blocked: true,
			expect: func(t *testing.T, hasAvailablePeer bool) {
				assert := assert.New(t)
				assert.False(hasAvailablePeer)
			},
		},
		{
			name:   "host id is equal to the peer's host id",
			state:  PeerStateSucceeded,
			hostID: mockHostID,
			expect: func(t *testing.T, hasAvailablePeer bool) {
				assert := assert.New(t)
				assert.False(hasAvailablePeer)
			},
		},
		{
			name:   "peer state is PeerStateSucceeded",
			state:  PeerStateSucceeded,
			hostID: "",
			expect: func(t *testing.T, hasAvailablePeer bool) {
				assert := assert.New(t)
				assert.True(hasAvailablePeer)
			},
		},
		{
			name:   "peer state is PeerStateRunning",
			state:  PeerStateRunning,
			hostID: "",
			expect: func(t *testing.T, hasAvailablePeer bool) {
				assert := assert.New(t)
				assert.True(hasAvailablePeer)
			},
		},
		{
			name:   "peer state is PeerStateBackToSource",
			state:  PeerStateBackToSource,
			hostID: "",
			expect: func(t *testing.T, hasAvailablePeer bool) {
				assert := assert.New(t)
				assert.True(hasAvailablePeer)
			},
		},
		{
			name:   "peer state is PeerStatePending",
			state:  PeerStatePending,
			hostID: "",
			expect: func(t *testing.T, hasAvailablePeer bool) {
				assert := assert.New(t)
				assert.False(hasAvailablePeer)
			},
		},
		{
			name:   "peer state is PeerStateFailed",
			state:  PeerStateFailed,
			hostID: "",
			expect: func(t *testing.T, hasAvailablePeer bool) {
				assert := assert.New(t)
				assert.False(hasAvailablePeer)
			},
		},
		{
			name:   "peer state is PeerStateLeave",
			state:  PeerStateLeave,
			hostID: "",
			expect: func(t *testing.T, hasAvailablePeer bool) {
				assert := assert.New(t)
				assert.False(hasAvailablePeer)
			},
		},
		{
			name:   "peer does not exist",
			state:  "",
			hostID: "",
			expect: func(t *testing.T, hasAvailablePeer bool) {
				assert := assert.New(t)
				assert.False(hasAvailablePeer)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			mockPeer := NewPeer(mockPeerID, task, mockHost)
			if tc.state != "" {
				mockPeer.FSM.SetState(tc.state)
				task.StorePeer(mockPeer)
			}

			blocklist := set.NewSafeSet[string]()
			if tc.blocked {
				blocklist.Add(mockPeer.ID)
			}

			tc.expect(t, task.HasAvailablePeer(tc.hostID, blocklist))
		})
	}
}

func TestTask_LoadSeedPeer(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, task *Task, mockPeer *Peer, mockSeedPeer *Peer)
	}{
		{
			name: "load seed peer",
			expect: func(t *testing.T, task *Task, mockPeer *Peer, mockSeedPeer *Peer) {
				assert := assert.New(t)
				task.StorePeer(mockPeer)
				task.StorePeer(mockSeedPeer)
				peer, loaded := task.LoadSeedPeer()
				assert.True(loaded)
				assert.Equal(mockSeedPeer.ID, peer.ID)
			},
		},
		{
			name: "load latest seed peer",
			expect: func(t *testing.T, task *Task, mockPeer *Peer, mockSeedPeer *Peer) {
				assert := assert.New(t)
				mockPeer.Host.Type = types.HostTypeSuperSeed
				task.StorePeer(mockPeer)
				task.StorePeer(mockSeedPeer)

				mockPeer.UpdatedAt.Store(time.Now())
				mockSeedPeer.UpdatedAt.Store(time.Now().Add(1 * time.Second))

				peer, loaded := task.LoadSeedPeer()
				assert.True(loaded)
				assert.Equal(mockSeedPeer.ID, peer.ID)
			},
		},
		{
			name: "peers is empty",
			expect: func(t *testing.T, task *Task, mockPeer *Peer, mockSeedPeer *Peer) {
				assert := assert.New(t)
				_, loaded := task.LoadSeedPeer()
				assert.False(loaded)
			},
		},
		{
			name: "seed peers is empty",
			expect: func(t *testing.T, task *Task, mockPeer *Peer, mockSeedPeer *Peer) {
				assert := assert.New(t)
				task.StorePeer(mockPeer)
				_, loaded := task.LoadSeedPeer()
				assert.False(loaded)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockSeedHost := NewHost(
				mockRawSeedHost.ID, mockRawSeedHost.IP, mockRawSeedHost.Name, mockRawSeedHost.Hostname,
				mockRawSeedHost.Port, mockRawSeedHost.DownloadPort, mockRawSeedHost.ProxyPort, mockRawSeedHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			mockPeer := NewPeer(mockPeerID, task, mockHost)
			mockSeedPeer := NewPeer(mockSeedPeerID, task, mockSeedHost)

			tc.expect(t, task, mockPeer, mockSeedPeer)
		})
	}
}

func TestTask_IsSeedPeerFailed(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, task *Task, mockPeer *Peer, mockSeedPeer *Peer)
	}{
		{
			name: "seed peer state is PeerStateFailed",
			expect: func(t *testing.T, task *Task, mockPeer *Peer, mockSeedPeer *Peer) {
				assert := assert.New(t)
				task.StorePeer(mockPeer)
				task.StorePeer(mockSeedPeer)
				mockSeedPeer.FSM.SetState(PeerStateFailed)

				assert.True(task.IsSeedPeerFailed())
			},
		},
		{
			name: "can not find seed peer",
			expect: func(t *testing.T, task *Task, mockPeer *Peer, mockSeedPeer *Peer) {
				assert := assert.New(t)
				task.StorePeer(mockPeer)

				assert.False(task.IsSeedPeerFailed())
			},
		},
		{
			name: "seed peer state is PeerStateSucceeded",
			expect: func(t *testing.T, task *Task, mockPeer *Peer, mockSeedPeer *Peer) {
				assert := assert.New(t)
				task.StorePeer(mockPeer)
				task.StorePeer(mockSeedPeer)
				mockSeedPeer.FSM.SetState(PeerStateSucceeded)

				assert.False(task.IsSeedPeerFailed())
			},
		},
		{
			name: "seed peer failed timeout",
			expect: func(t *testing.T, task *Task, mockPeer *Peer, mockSeedPeer *Peer) {
				assert := assert.New(t)
				task.StorePeer(mockPeer)
				task.StorePeer(mockSeedPeer)
				mockSeedPeer.CreatedAt.Store(time.Now().Add(-SeedPeerFailedTimeout))
				mockSeedPeer.FSM.SetState(PeerStateFailed)

				assert.False(task.IsSeedPeerFailed())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockSeedHost := NewHost(
				mockRawSeedHost.ID, mockRawSeedHost.IP, mockRawSeedHost.Name, mockRawSeedHost.Hostname,
				mockRawSeedHost.Port, mockRawSeedHost.DownloadPort, mockRawSeedHost.ProxyPort, mockRawSeedHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			mockPeer := NewPeer(mockPeerID, task, mockHost)
			mockSeedPeer := NewPeer(mockSeedPeerID, task, mockSeedHost)

			tc.expect(t, task, mockPeer, mockSeedPeer)
		})
	}
}

func TestTask_LoadPiece(t *testing.T) {
	tests := []struct {
		name        string
		piece       *Piece
		pieceNumber int32
		expect      func(t *testing.T, piece *Piece, loaded bool)
	}{
		{
			name:        "load piece",
			piece:       mockPiece,
			pieceNumber: mockPiece.Number,
			expect: func(t *testing.T, piece *Piece, loaded bool) {
				assert := assert.New(t)
				assert.True(loaded)
				assert.Equal(mockPiece.Number, piece.Number)
				assert.Equal(mockPiece.ParentID, piece.ParentID)
				assert.Equal(mockPiece.Offset, piece.Offset)
				assert.Equal(mockPiece.Length, piece.Length)
				assert.EqualValues(mockPiece.Digest, piece.Digest)
				assert.Equal(mockPiece.TrafficType, piece.TrafficType)
				assert.Equal(mockPiece.Cost, piece.Cost)
				assert.Equal(mockPiece.CreatedAt, piece.CreatedAt)
			},
		},
		{
			name:        "piece does not exist",
			piece:       mockPiece,
			pieceNumber: 2,
			expect: func(t *testing.T, piece *Piece, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
			},
		},
		{
			name:        "load key is zero",
			piece:       mockPiece,
			pieceNumber: 0,
			expect: func(t *testing.T, piece *Piece, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			task.StorePiece(tc.piece)
			piece, loaded := task.LoadPiece(tc.pieceNumber)
			tc.expect(t, piece, loaded)
		})
	}
}

func TestTask_StorePiece(t *testing.T) {
	tests := []struct {
		name        string
		piece       *Piece
		pieceNumber int32
		expect      func(t *testing.T, piece *Piece, loaded bool)
	}{
		{
			name:        "store piece",
			piece:       mockPiece,
			pieceNumber: mockPiece.Number,
			expect: func(t *testing.T, piece *Piece, loaded bool) {
				assert := assert.New(t)
				assert.True(loaded)
				assert.Equal(mockPiece.Number, piece.Number)
				assert.Equal(mockPiece.ParentID, piece.ParentID)
				assert.Equal(mockPiece.Offset, piece.Offset)
				assert.Equal(mockPiece.Length, piece.Length)
				assert.EqualValues(mockPiece.Digest, piece.Digest)
				assert.Equal(mockPiece.TrafficType, piece.TrafficType)
				assert.Equal(mockPiece.Cost, piece.Cost)
				assert.Equal(mockPiece.CreatedAt, piece.CreatedAt)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			task.StorePiece(tc.piece)
			piece, loaded := task.LoadPiece(tc.pieceNumber)
			tc.expect(t, piece, loaded)
		})
	}
}

func TestTask_StoreBackToSourcePiece(t *testing.T) {
	digestA := digest.New(digest.AlgorithmSHA256, "0f2592159700536c81cd58094265e92089ebabb3e06f802c5f2d6e8e753cb233")
	digestB := digest.New(digest.AlgorithmSHA256, "a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e")

	type report struct {
		digest       *digest.Digest
		fromSeedPeer bool
		stored       bool
	}

	tests := []struct {
		name    string
		reports []report
		expect  *digest.Digest
	}{
		{
			name:    "store the first piece of a normal peer",
			reports: []report{{digestA, false, true}},
			expect:  digestA,
		},
		{
			name:    "keep the digest when a normal peer reports another digest",
			reports: []report{{digestA, false, true}, {digestB, false, false}},
			expect:  digestA,
		},
		{
			name:    "fill a piece without digest with the digest of a normal peer",
			reports: []report{{nil, false, true}, {digestB, false, true}},
			expect:  digestB,
		},
		{
			name:    "replace the digest of a normal peer with the digest of a seed peer",
			reports: []report{{digestA, false, true}, {digestB, true, true}},
			expect:  digestB,
		},
		{
			name:    "keep the digest of a seed peer when a normal peer reports another digest",
			reports: []report{{digestA, true, true}, {digestB, false, false}},
			expect:  digestA,
		},
		{
			name:    "replace the digest of a seed peer with the digest of a seed peer",
			reports: []report{{digestA, true, true}, {digestB, true, true}},
			expect:  digestB,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			for _, r := range tc.reports {
				stored := task.StoreBackToSourcePiece(&Piece{Number: 1, Length: 10, Digest: r.digest}, r.fromSeedPeer)
				assert.Equal(r.stored, stored)
			}

			piece, loaded := task.LoadPiece(1)
			assert.True(loaded)
			assert.Equal(tc.expect.String(), piece.Digest.String())
		})
	}
}

func TestTask_DeletePiece(t *testing.T) {
	tests := []struct {
		name        string
		piece       *Piece
		pieceNumber int32
		expect      func(t *testing.T, task *Task)
	}{
		{
			name:        "delete piece",
			piece:       mockPiece,
			pieceNumber: mockPiece.Number,
			expect: func(t *testing.T, task *Task) {
				assert := assert.New(t)
				_, loaded := task.LoadPiece(mockPiece.Number)
				assert.False(loaded)
			},
		},
		{
			name:        "delete key does not exist",
			piece:       mockPiece,
			pieceNumber: 0,
			expect: func(t *testing.T, task *Task) {
				assert := assert.New(t)
				piece, loaded := task.LoadPiece(mockPiece.Number)
				assert.True(loaded)
				assert.Equal(mockPiece.Number, piece.Number)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)

			task.StorePiece(tc.piece)
			task.DeletePiece(tc.pieceNumber)
			tc.expect(t, task)
		})
	}
}

func TestTask_SizeScope(t *testing.T) {
	tests := []struct {
		name            string
		contentLength   int64
		totalPieceCount int32
		expect          func(t *testing.T, sizeScope commonv2.SizeScope)
	}{
		{
			name:            "scope size is tiny",
			contentLength:   TinyFileSize,
			totalPieceCount: 1,
			expect: func(t *testing.T, sizeScope commonv2.SizeScope) {
				assert := assert.New(t)
				assert.Equal(commonv2.SizeScope_TINY, sizeScope)
			},
		},
		{
			name:            "scope size is empty",
			contentLength:   0,
			totalPieceCount: 0,
			expect: func(t *testing.T, sizeScope commonv2.SizeScope) {
				assert := assert.New(t)
				assert.Equal(commonv2.SizeScope_EMPTY, sizeScope)
			},
		},
		{
			name:            "scope size is small",
			contentLength:   TinyFileSize + 1,
			totalPieceCount: 1,
			expect: func(t *testing.T, sizeScope commonv2.SizeScope) {
				assert := assert.New(t)
				assert.Equal(commonv2.SizeScope_SMALL, sizeScope)
			},
		},
		{
			name:            "scope size is normal",
			contentLength:   TinyFileSize + 1,
			totalPieceCount: 2,
			expect: func(t *testing.T, sizeScope commonv2.SizeScope) {
				assert := assert.New(t)
				assert.Equal(commonv2.SizeScope_NORMAL, sizeScope)
			},
		},
		{
			name:            "invalid content length",
			contentLength:   -1,
			totalPieceCount: 2,
			expect: func(t *testing.T, sizeScope commonv2.SizeScope) {
				assert := assert.New(t)
				assert.Equal(commonv2.SizeScope_UNKNOW, sizeScope)
			},
		},
		{
			name:            "invalid total piece count",
			contentLength:   TinyFileSize + 1,
			totalPieceCount: -1,
			expect: func(t *testing.T, sizeScope commonv2.SizeScope) {
				assert := assert.New(t)
				assert.Equal(commonv2.SizeScope_UNKNOW, sizeScope)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			task.ContentLength.Store(tc.contentLength)
			task.TotalPieceCount.Store(tc.totalPieceCount)
			tc.expect(t, task.SizeScope())
		})
	}
}

func TestTask_CanBackToSource(t *testing.T) {
	tests := []struct {
		name              string
		typ               commonv2.TaskType
		backToSourceLimit int32
		backToSourcePeers int
		expect            func(t *testing.T, canBackToSource bool)
	}{
		{
			name:              "task can back-to-source",
			typ:               commonv2.TaskType_STANDARD,
			backToSourceLimit: 1,
			backToSourcePeers: 0,
			expect: func(t *testing.T, canBackToSource bool) {
				assert := assert.New(t)
				assert.True(canBackToSource)
			},
		},
		{
			name:              "task can not back-to-source",
			typ:               commonv2.TaskType_STANDARD,
			backToSourceLimit: -1,
			backToSourcePeers: 0,
			expect: func(t *testing.T, canBackToSource bool) {
				assert := assert.New(t)
				assert.False(canBackToSource)
			},
		},
		{
			name:              "back-to-source peers reach the limit",
			typ:               commonv2.TaskType_STANDARD,
			backToSourceLimit: 1,
			backToSourcePeers: 1,
			expect: func(t *testing.T, canBackToSource bool) {
				assert := assert.New(t)
				assert.True(canBackToSource)
			},
		},
		{
			name:              "back-to-source peers exceed the limit",
			typ:               commonv2.TaskType_STANDARD,
			backToSourceLimit: 1,
			backToSourcePeers: 2,
			expect: func(t *testing.T, canBackToSource bool) {
				assert := assert.New(t)
				assert.False(canBackToSource)
			},
		},
		{
			name:              "task can back-to-source and task type is PERSISTENT",
			typ:               commonv2.TaskType_PERSISTENT,
			backToSourceLimit: 1,
			backToSourcePeers: 0,
			expect: func(t *testing.T, canBackToSource bool) {
				assert := assert.New(t)
				assert.True(canBackToSource)
			},
		},
		{
			name:              "task type is PERSISTENT_CACHE",
			typ:               commonv2.TaskType_PERSISTENT_CACHE,
			backToSourceLimit: 1,
			backToSourcePeers: 0,
			expect: func(t *testing.T, canBackToSource bool) {
				assert := assert.New(t)
				assert.False(canBackToSource)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, tc.typ, mockTaskFilteredQueryParams, mockTaskHeader, tc.backToSourceLimit)
			for range tc.backToSourcePeers {
				task.BackToSourcePeers.Add(idgen.PeerID())
			}

			tc.expect(t, task.CanBackToSource())
		})
	}
}

func TestTask_CanReuseDirectPiece(t *testing.T) {
	tests := []struct {
		name          string
		directPiece   []byte
		contentLength int64
		expect        func(t *testing.T, canReuseDirectPiece bool)
	}{
		{
			name:          "task can reuse direct piece",
			directPiece:   []byte{1},
			contentLength: 1,
			expect: func(t *testing.T, canReuseDirectPiece bool) {
				assert := assert.New(t)
				assert.True(canReuseDirectPiece)
			},
		},
		{
			name:          "direct piece is empty",
			directPiece:   []byte{},
			contentLength: 1,
			expect: func(t *testing.T, canReuseDirectPiece bool) {
				assert := assert.New(t)
				assert.False(canReuseDirectPiece)
			},
		},
		{
			name:          "content length is error",
			directPiece:   []byte{1},
			contentLength: 2,
			expect: func(t *testing.T, canReuseDirectPiece bool) {
				assert := assert.New(t)
				assert.False(canReuseDirectPiece)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			task.DirectPiece = tc.directPiece
			task.ContentLength.Store(tc.contentLength)
			tc.expect(t, task.CanReuseDirectPiece())
		})
	}
}

func TestTask_ReportPieceResultToPeers(t *testing.T) {
	tests := []struct {
		name   string
		state  string
		event  string
		mock   func(ms *v1mocks.MockScheduler_ReportPieceResultServerMockRecorder)
		expect func(t *testing.T, mockPeer *Peer)
	}{
		{
			name:  "peer state is PeerStatePending",
			state: PeerStatePending,
			event: PeerEventDownloadFailed,
			expect: func(t *testing.T, mockPeer *Peer) {
				assert := assert.New(t)
				assert.True(mockPeer.FSM.Is(PeerStatePending))
			},
		},
		{
			name:  "peer state is PeerStateRunning and stream is empty",
			state: PeerStateRunning,
			event: PeerEventDownloadFailed,
			expect: func(t *testing.T, mockPeer *Peer) {
				assert := assert.New(t)
				assert.True(mockPeer.FSM.Is(PeerStateRunning))
			},
		},
		{
			name:  "peer state is PeerStateRunning and stream sending failed",
			state: PeerStateRunning,
			event: PeerEventDownloadFailed,
			mock: func(ms *v1mocks.MockScheduler_ReportPieceResultServerMockRecorder) {
				ms.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedTaskStatusError})).Return(errors.New("foo")).Times(1)
			},
			expect: func(t *testing.T, mockPeer *Peer) {
				assert := assert.New(t)
				assert.True(mockPeer.FSM.Is(PeerStateRunning))
			},
		},
		{
			name:  "peer state is PeerStateRunning and state changing failed",
			state: PeerStateRunning,
			event: PeerEventRegisterNormal,
			mock: func(ms *v1mocks.MockScheduler_ReportPieceResultServerMockRecorder) {
				ms.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedTaskStatusError})).Return(nil).Times(1)
			},
			expect: func(t *testing.T, mockPeer *Peer) {
				assert := assert.New(t)
				assert.True(mockPeer.FSM.Is(PeerStateRunning))
			},
		},
		{
			name:  "peer state is PeerStateRunning and report peer successfully",
			state: PeerStateRunning,
			event: PeerEventDownloadFailed,
			mock: func(ms *v1mocks.MockScheduler_ReportPieceResultServerMockRecorder) {
				ms.Send(gomock.Eq(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedTaskStatusError})).Return(nil).Times(1)
			},
			expect: func(t *testing.T, mockPeer *Peer) {
				assert := assert.New(t)
				assert.True(mockPeer.FSM.Is(PeerStateFailed))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			stream := v1mocks.NewMockScheduler_ReportPieceResultServer(ctl)

			mockHost := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			task := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit)
			mockPeer := NewPeer(mockPeerID, task, mockHost)
			task.StorePeer(mockPeer)
			mockPeer.FSM.SetState(tc.state)
			if tc.mock != nil {
				mockPeer.StoreReportPieceResultStream(stream)
				tc.mock(stream.EXPECT())
			}

			task.ReportPieceResultToPeers(&schedulerv1.PeerPacket{Code: commonv1.Code_SchedTaskStatusError}, tc.event)
			tc.expect(t, mockPeer)
		})
	}
}
