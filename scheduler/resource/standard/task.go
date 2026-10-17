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
	"sync"
	"sync/atomic"
	"time"

	"github.com/looplab/fsm"

	commonv2 "d7y.io/api/v2/pkg/apis/common/v2"
	schedulerv1 "d7y.io/api/v2/pkg/apis/scheduler/v1"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	pkgatomic "d7y.io/dragonfly/v2/pkg/atomic"
	"d7y.io/dragonfly/v2/pkg/container/set"
	"d7y.io/dragonfly/v2/pkg/digest"
	"d7y.io/dragonfly/v2/pkg/graph/dag"
	pkgmath "d7y.io/dragonfly/v2/pkg/math"
	"d7y.io/dragonfly/v2/pkg/types"
)

const (
	// Tiny file size is 128 bytes.
	TinyFileSize = 128

	// Empty file size is 0 bytes.
	EmptyFileSize = 0
)

const (
	// Peer failure limit in task.
	FailedPeerCountLimit = 200

	// Peer count limit for task.
	PeerCountLimitForTask = 300
)

const (
	// Task has been created but did not start running.
	TaskStatePending = "Pending"

	// Task is downloading resources from seed peer or back-to-source.
	TaskStateRunning = "Running"

	// Task has been downloaded successfully.
	TaskStateSucceeded = "Succeeded"

	// Task has been downloaded failed.
	TaskStateFailed = "Failed"

	// Task has no peers.
	TaskStateLeave = "Leave"
)

const (
	// Task is downloading.
	TaskEventDownload = "Download"

	// Task downloaded successfully.
	TaskEventDownloadSucceeded = "DownloadSucceeded"

	// Task downloaded failed.
	TaskEventDownloadFailed = "DownloadFailed"

	// Task leaves.
	TaskEventLeave = "Leave"
)

// TaskOption is a functional option for task.
type TaskOption func(task *Task)

// WithDigest set Digest for task.
func WithDigest(d *digest.Digest) TaskOption {
	return func(t *Task) {
		t.Digest = d
	}
}

// WithPieceLength set PieceLength for task.
func WithPieceLength(pieceLength uint64) TaskOption {
	return func(t *Task) {
		t.PieceLength = pieceLength
	}
}

// Task contains content for task.
type Task struct {
	// ID is task id.
	ID string

	// Type is task type.
	Type commonv2.TaskType

	// URL is task download url.
	URL string

	// Digest of the task content, for example md5:xxx or sha256:yyy.
	Digest *digest.Digest

	// URL tag identifies different task for same url.
	Tag string

	// Application identifies different task for same url.
	Application string

	// FilteredQueryParams is filtered query params.
	FilteredQueryParams []string

	// Task request headers.
	Header map[string]string

	// DirectPiece is tiny piece data.
	DirectPiece []byte

	// ContentLength is task total content length.
	ContentLength *atomic.Int64

	// PieceLength is piece length.
	PieceLength uint64

	// TotalPieceCount is total piece count.
	TotalPieceCount *atomic.Int32

	// BackToSourceLimit is back-to-source limit.
	BackToSourceLimit *atomic.Int32

	// BackToSourcePeers is back-to-source sync map.
	BackToSourcePeers set.SafeSet[string]

	// Task state machine.
	FSM *fsm.FSM

	// Piece sync map.
	Pieces *sync.Map

	// DAG is directed acyclic graph of peers.
	DAG dag.DAG[*Peer]

	// PeerFailedCount is peer failed count,
	// if one peer succeeds, the value is reset to zero.
	PeerFailedCount *atomic.Int32

	// CreatedAt is task create time.
	CreatedAt *pkgatomic.Time

	// UpdatedAt is task update time.
	UpdatedAt *pkgatomic.Time

	// Task log.
	Log *logger.SugaredLoggerOnWith
}

// New task instance.
func NewTask(id, url, tag, application string, typ commonv2.TaskType, filteredQueryParams []string,
	header map[string]string, backToSourceLimit int32, options ...TaskOption) *Task {
	t := &Task{
		ID:                  id,
		Type:                typ,
		URL:                 url,
		Tag:                 tag,
		Application:         application,
		FilteredQueryParams: filteredQueryParams,
		Header:              header,
		DirectPiece:         []byte{},
		ContentLength:       new(atomic.Int64),
		TotalPieceCount:     new(atomic.Int32),
		BackToSourceLimit:   new(atomic.Int32),
		BackToSourcePeers:   set.NewSafeSet[string](),
		Pieces:              &sync.Map{},
		DAG:                 dag.NewDAG[*Peer](),
		PeerFailedCount:     new(atomic.Int32),
		CreatedAt:           pkgatomic.NewTime(time.Now()),
		UpdatedAt:           pkgatomic.NewTime(time.Now()),
		Log:                 logger.WithTask(id, url),
	}

	// ContentLength is unknown until it is reported by the peer.
	t.ContentLength.Store(-1)
	t.BackToSourceLimit.Store(backToSourceLimit)

	// Initialize state machine.
	t.FSM = fsm.NewFSM(
		TaskStatePending,
		fsm.Events{
			fsm.EventDesc{Name: TaskEventDownload, Src: []string{TaskStatePending, TaskStateSucceeded, TaskStateFailed, TaskStateLeave}, Dst: TaskStateRunning},
			fsm.EventDesc{Name: TaskEventDownloadSucceeded, Src: []string{TaskStateLeave, TaskStateRunning, TaskStateFailed}, Dst: TaskStateSucceeded},
			fsm.EventDesc{Name: TaskEventDownloadFailed, Src: []string{TaskStateRunning}, Dst: TaskStateFailed},
			fsm.EventDesc{Name: TaskEventLeave, Src: []string{TaskStatePending, TaskStateRunning, TaskStateSucceeded, TaskStateFailed}, Dst: TaskStateLeave},
		},
		fsm.Callbacks{
			TaskEventDownload: func(ctx context.Context, e *fsm.Event) {
				t.UpdatedAt.Store(time.Now())
				t.Log.Infof("task state is %s", e.FSM.Current())
			},
			TaskEventDownloadSucceeded: func(ctx context.Context, e *fsm.Event) {
				t.UpdatedAt.Store(time.Now())
				t.Log.Infof("task state is %s", e.FSM.Current())
			},
			TaskEventDownloadFailed: func(ctx context.Context, e *fsm.Event) {
				t.UpdatedAt.Store(time.Now())
				t.Log.Infof("task state is %s", e.FSM.Current())
			},
			TaskEventLeave: func(ctx context.Context, e *fsm.Event) {
				t.UpdatedAt.Store(time.Now())
				t.Log.Infof("task state is %s", e.FSM.Current())
			},
		},
	)

	for _, opt := range options {
		opt(t)
	}

	return t
}

// LoadPeer return peer for a key.
func (t *Task) LoadPeer(key string) (*Peer, bool) {
	vertex, err := t.DAG.GetVertex(key)
	if err != nil {
		t.Log.Error(err)
		return nil, false
	}

	return vertex.Value, true
}

// LoadRandomPeers return random peers.
func (t *Task) LoadRandomPeers(n uint) []*Peer {
	var peers []*Peer
	for _, vertex := range t.DAG.GetRandomVertices(n) {
		peers = append(peers, vertex.Value)
	}

	return peers
}

// LoadPeers return all peers.
func (t *Task) LoadPeers() []*Peer {
	var peers []*Peer
	for _, vertex := range t.DAG.GetVertices() {
		peers = append(peers, vertex.Value)
	}

	return peers
}

// LoadFinishedPeers return finished peers.
func (t *Task) LoadFinishedPeers() []*Peer {
	var finishedPeers []*Peer
	t.DAG.Range(func(_ string, vertex *dag.Vertex[*Peer]) bool {
		peer := vertex.Value
		switch peer.FSM.Current() {
		case PeerStateSucceeded, PeerStateFailed, PeerStateLeave:
			finishedPeers = append(finishedPeers, peer)
		}

		return true
	})

	return finishedPeers
}

// StorePeer set peer.
func (t *Task) StorePeer(peer *Peer) {
	if err := t.DAG.AddVertex(peer.ID, peer); err != nil {
		t.Log.Error(err)
	}
}

// DeletePeer deletes peer for a key.
func (t *Task) DeletePeer(key string) {
	if err := t.DeletePeerInEdges(key); err != nil {
		t.Log.Error(err)
	}

	if err := t.DeletePeerOutEdges(key); err != nil {
		t.Log.Error(err)
	}

	t.DAG.DeleteVertex(key)
}

// PeerCount returns count of peer.
func (t *Task) PeerCount() int {
	return int(t.DAG.VertexCount())
}

// AddPeerEdge adds inedges between two peers.
func (t *Task) AddPeerEdge(fromPeer *Peer, toPeer *Peer) error {
	if err := t.DAG.AddEdge(fromPeer.ID, toPeer.ID); err != nil {
		return err
	}

	fromPeer.Host.UploadCount.Add(1)
	fromPeer.Host.ConcurrentUploadCount.Add(1)
	fromPeer.Host.TxBandwidth.Add(toPeer.PeakBandwidthUsage(t.PieceLength))
	fromPeer.Host.ConcurrentUploadPieceCount.Add(uint64(toPeer.ConcurrentPieceCount))
	var contentLength uint64
	if cl := t.ContentLength.Load(); cl > 0 { // Handle -1 (unknown length).
		contentLength = uint64(cl)
	}

	fromPeer.Host.UploadContentLength.Add(contentLength)
	t.Log.Debugf("increment host %s metrics on adding edge %s -> %s: "+
		"UploadCount=%d(+1), ConcurrentUploadCount=%d(+1), "+
		"TxBandwidth=%d(+%d), ConcurrentUploadPieceCount=%d(+%d), "+
		"UploadContentLength=%d(+%d)",
		fromPeer.Host.ID,
		fromPeer.ID, toPeer.ID,
		fromPeer.Host.UploadCount.Load(),
		fromPeer.Host.ConcurrentUploadCount.Load(),
		fromPeer.Host.TxBandwidth.Load(), toPeer.PeakBandwidthUsage(t.PieceLength),
		fromPeer.Host.ConcurrentUploadPieceCount.Load(), toPeer.ConcurrentPieceCount,
		fromPeer.Host.UploadContentLength.Load(), contentLength,
	)
	return nil
}

// AddPeerEdges adds edges from the given peers to the peer in a single batch.
func (t *Task) AddPeerEdges(fromPeers []*Peer, toPeer *Peer) []*Peer {
	fromPeerIDs := make([]string, 0, len(fromPeers))
	for _, fromPeer := range fromPeers {
		fromPeerIDs = append(fromPeerIDs, fromPeer.ID)
	}

	added := t.DAG.AddEdges(fromPeerIDs, toPeer.ID)
	addedPeers := make([]*Peer, 0, len(added))
	for _, fromPeer := range fromPeers {
		if _, ok := added[fromPeer.ID]; !ok {
			continue
		}

		delete(added, fromPeer.ID)
		fromPeer.Host.UploadCount.Add(1)
		fromPeer.Host.ConcurrentUploadCount.Add(1)
		fromPeer.Host.TxBandwidth.Add(toPeer.PeakBandwidthUsage(t.PieceLength))
		fromPeer.Host.ConcurrentUploadPieceCount.Add(uint64(toPeer.ConcurrentPieceCount))
		var contentLength uint64
		if cl := t.ContentLength.Load(); cl > 0 { // Handle -1 (unknown length).
			contentLength = uint64(cl)
		}

		fromPeer.Host.UploadContentLength.Add(contentLength)
		t.Log.Debugf("increment host %s metrics on adding edge %s -> %s: "+
			"UploadCount=%d(+1), ConcurrentUploadCount=%d(+1), "+
			"TxBandwidth=%d(+%d), ConcurrentUploadPieceCount=%d(+%d), "+
			"UploadContentLength=%d(+%d)",
			fromPeer.Host.ID,
			fromPeer.ID, toPeer.ID,
			fromPeer.Host.UploadCount.Load(),
			fromPeer.Host.ConcurrentUploadCount.Load(),
			fromPeer.Host.TxBandwidth.Load(), toPeer.PeakBandwidthUsage(t.PieceLength),
			fromPeer.Host.ConcurrentUploadPieceCount.Load(), toPeer.ConcurrentPieceCount,
			fromPeer.Host.UploadContentLength.Load(), contentLength,
		)

		addedPeers = append(addedPeers, fromPeer)
	}

	return addedPeers
}

// DeletePeerInEdges deletes inedges of peer.
func (t *Task) DeletePeerInEdges(key string) error {
	vertex, err := t.DAG.GetVertex(key)
	if err != nil {
		return err
	}

	var contentLength uint64
	if cl := t.ContentLength.Load(); cl > 0 { // Handle -1 (unknown length).
		contentLength = uint64(cl)
	}

	for _, parent := range vertex.Parents.Values() {
		if parent.Value == nil {
			continue
		}

		parent.Value.Host.ConcurrentUploadCount.Add(-1)
		pkgmath.SafeSubAtomicUint64(parent.Value.Host.TxBandwidth, vertex.Value.PeakBandwidthUsage(t.PieceLength))
		pkgmath.SafeSubAtomicUint64(parent.Value.Host.ConcurrentUploadPieceCount, uint64(vertex.Value.ConcurrentPieceCount))
		pkgmath.SafeSubAtomicUint64(parent.Value.Host.UploadContentLength, contentLength)
		t.Log.Debugf("decrement host %s metrics on deleting edge %s -> %s: "+
			"ConcurrentUploadCount=%d(-1), TxBandwidth=%d(-%d),"+
			"ConcurrentUploadPieceCount=%d(-%d), UploadContentLength=%d(-%d)",
			parent.Value.Host.ID,
			parent.Value.ID, key,
			parent.Value.Host.ConcurrentUploadCount.Load(),
			parent.Value.Host.TxBandwidth.Load(), vertex.Value.PeakBandwidthUsage(t.PieceLength),
			parent.Value.Host.ConcurrentUploadPieceCount.Load(), vertex.Value.ConcurrentPieceCount,
			parent.Value.Host.UploadContentLength.Load(), contentLength,
		)
	}

	if err := t.DAG.DeleteVertexInEdges(key); err != nil {
		return err
	}

	return nil
}

// DeletePeerOutEdges deletes outedges of peer.
func (t *Task) DeletePeerOutEdges(key string) error {
	vertex, err := t.DAG.GetVertex(key)
	if err != nil {
		return err
	}

	peer := vertex.Value
	if peer == nil {
		return errors.New("vertex value is nil")
	}

	var (
		totalTxBandwidth           uint64
		totalConcurrentUploadPiece uint64
		totalUploadContentLength   uint64
	)
	var contentLength uint64
	if cl := t.ContentLength.Load(); cl > 0 { // Handle -1 (unknown length).
		contentLength = uint64(cl)
	}

	for _, child := range vertex.Children.Values() {
		if child.Value == nil {
			continue
		}

		totalTxBandwidth += child.Value.PeakBandwidthUsage(t.PieceLength)
		totalConcurrentUploadPiece += uint64(child.Value.ConcurrentPieceCount)
		totalUploadContentLength += contentLength
	}

	peer.Host.ConcurrentUploadCount.Add(-int32(vertex.Children.Len()))
	pkgmath.SafeSubAtomicUint64(peer.Host.TxBandwidth, totalTxBandwidth)
	pkgmath.SafeSubAtomicUint64(peer.Host.ConcurrentUploadPieceCount, totalConcurrentUploadPiece)
	pkgmath.SafeSubAtomicUint64(peer.Host.UploadContentLength, totalUploadContentLength)
	t.Log.Debugf("decrement host %s metrics on deleting out-edges of peer %s: "+
		"ConcurrentUploadCount=%d(-%d), TxBandwidth=%d(-%d),"+
		"ConcurrentUploadPieceCount=%d(-%d), UploadContentLength=%d(-%d)",
		peer.Host.ID,
		key,
		peer.Host.ConcurrentUploadCount.Load(), vertex.Children.Len(),
		peer.Host.TxBandwidth.Load(), totalTxBandwidth,
		peer.Host.ConcurrentUploadPieceCount.Load(), totalConcurrentUploadPiece,
		peer.Host.UploadContentLength.Load(), totalUploadContentLength,
	)

	if err := t.DAG.DeleteVertexOutEdges(key); err != nil {
		return err
	}

	return nil
}

// CanAddPeerEdge finds whether there are peer circles through depth-first search.
func (t *Task) CanAddPeerEdge(fromPeerKey, toPeerKey string) bool {
	return t.DAG.CanAddEdge(fromPeerKey, toPeerKey)
}

// CanAddPeerEdges reports which of the given peers can currently have an edge
// added to the peer, running the cycle-detection search once for all
// candidates instead of once per candidate.
func (t *Task) CanAddPeerEdges(fromPeerKeys []string, toPeerKey string) map[string]struct{} {
	return t.DAG.CanAddEdges(fromPeerKeys, toPeerKey)
}

// PeerDegree returns the degree of peer.
func (t *Task) PeerDegree(key string) (int, error) {
	vertex, err := t.DAG.GetVertex(key)
	if err != nil {
		return 0, err
	}

	return vertex.Degree(), nil
}

// PeerInDegree returns the indegree of peer.
func (t *Task) PeerInDegree(key string) (int, error) {
	vertex, err := t.DAG.GetVertex(key)
	if err != nil {
		return 0, err
	}

	return vertex.InDegree(), nil
}

// PeerOutDegree returns the outdegree of peer.
func (t *Task) PeerOutDegree(key string) (int, error) {
	vertex, err := t.DAG.GetVertex(key)
	if err != nil {
		return 0, err
	}

	return vertex.OutDegree(), nil
}

// HasAvailablePeer returns whether there is an available peer.
func (t *Task) HasAvailablePeer(hostID string, blocklist set.SafeSet[string]) bool {
	var hasAvailablePeer bool
	t.DAG.Range(func(_ string, vertex *dag.Vertex[*Peer]) bool {
		peer := vertex.Value
		if peer == nil {
			return true
		}

		if peer.Host.ID == hostID {
			return true
		}

		if blocklist.Contains(peer.ID) {
			return true
		}

		if peer.FSM.Is(PeerStateRunning) ||
			peer.FSM.Is(PeerStateSucceeded) ||
			peer.FSM.Is(PeerStateBackToSource) {
			t.Log.Debugf("found available peer %s state is %s in task %s for host %s", peer.ID, peer.FSM.Current(), t.ID, hostID)
			hasAvailablePeer = true
			return false
		}

		return true
	})

	return hasAvailablePeer
}

// LoadSeedPeer return latest seed peer in peers sync map.
func (t *Task) LoadSeedPeer() (*Peer, bool) {
	var seedPeer *Peer
	t.DAG.Range(func(_ string, vertex *dag.Vertex[*Peer]) bool {
		peer := vertex.Value
		if peer == nil {
			return true
		}

		if peer.Host.Type != types.HostTypeNormal {
			if seedPeer == nil || peer.UpdatedAt.Load().After(seedPeer.UpdatedAt.Load()) {
				seedPeer = peer
			}
		}

		return true
	})

	if seedPeer != nil {
		return seedPeer, true
	}

	return nil, false
}

// IsSeedPeerFailed returns whether the seed peer in the task failed.
func (t *Task) IsSeedPeerFailed() bool {
	seedPeer, loaded := t.LoadSeedPeer()
	return loaded && seedPeer.FSM.Is(PeerStateFailed) && time.Since(seedPeer.CreatedAt.Load()) < SeedPeerFailedTimeout
}

// LoadPiece return piece for a key.
func (t *Task) LoadPiece(key int32) (*Piece, bool) {
	rawPiece, loaded := t.Pieces.Load(key)
	if !loaded {
		return nil, false
	}

	return rawPiece.(*Piece), loaded
}

// StorePiece set piece.
func (t *Task) StorePiece(piece *Piece) {
	t.Pieces.Store(piece.Number, piece)
}

// StoreBackToSourcePiece stores the piece downloaded from the source by a peer,
// and returns whether it is stored. The digests of the stored pieces are
// distributed to the peers downloading the task as trusted digests, so the
// piece of a normal peer only fills a piece without digest, and only the piece
// of a seed peer replaces a stored digest.
func (t *Task) StoreBackToSourcePiece(piece *Piece, fromSeedPeer bool) bool {
	if fromSeedPeer {
		if stored, loaded := t.LoadPiece(piece.Number); loaded {
			t.warnPieceDigestConflict(stored, piece, true)
		}

		t.StorePiece(piece)
		return true
	}

	for {
		rawStored, loaded := t.Pieces.LoadOrStore(piece.Number, piece)
		if !loaded {
			return true
		}

		stored := rawStored.(*Piece)
		if stored.Digest != nil {
			t.warnPieceDigestConflict(stored, piece, false)
			return false
		}

		if t.Pieces.CompareAndSwap(piece.Number, stored, piece) {
			return true
		}
	}
}

// warnPieceDigestConflict logs a piece reported with a digest different from
// the stored one.
func (t *Task) warnPieceDigestConflict(stored, piece *Piece, replaced bool) {
	if stored.Digest == nil || piece.Digest == nil || stored.Digest.String() == piece.Digest.String() {
		return
	}

	t.Log.Warnf("piece %d reported with digest %s, stored digest is %s, replaced: %t",
		piece.Number, piece.Digest.String(), stored.Digest.String(), replaced)
}

// DeletePiece deletes piece for a key.
func (t *Task) DeletePiece(key int32) {
	t.Pieces.Delete(key)
}

// SizeScope return task size scope type.
func (t *Task) SizeScope() commonv2.SizeScope {
	if t.ContentLength.Load() < 0 {
		return commonv2.SizeScope_UNKNOW
	}

	if t.TotalPieceCount.Load() < 0 {
		return commonv2.SizeScope_UNKNOW
	}

	if t.ContentLength.Load() == EmptyFileSize {
		return commonv2.SizeScope_EMPTY
	}

	if t.ContentLength.Load() <= TinyFileSize {
		return commonv2.SizeScope_TINY
	}

	if t.TotalPieceCount.Load() == 1 {
		return commonv2.SizeScope_SMALL
	}

	return commonv2.SizeScope_NORMAL
}

// CanBackToSource represents whether task can back-to-source.
func (t *Task) CanBackToSource() bool {
	return int32(t.BackToSourcePeers.Len()) <= t.BackToSourceLimit.Load() && (t.Type != commonv2.TaskType_PERSISTENT_CACHE)
}

// CanReuseDirectPiece represents whether task can reuse data of direct piece.
func (t *Task) CanReuseDirectPiece() bool {
	return len(t.DirectPiece) > 0 && int64(len(t.DirectPiece)) == t.ContentLength.Load()
}

// ReportPieceResultToPeers reports all peers in the task with the state code.
// Used only in v1 version of the grpc.
func (t *Task) ReportPieceResultToPeers(peerPacket *schedulerv1.PeerPacket, event string) {
	for _, vertex := range t.DAG.GetVertices() {
		peer := vertex.Value
		if peer == nil {
			continue
		}

		if peer.FSM.Is(PeerStateRunning) {
			stream, loaded := peer.LoadReportPieceResultStream()
			if !loaded {
				continue
			}

			if err := stream.Send(peerPacket); err != nil {
				t.Log.Errorf("send packet to peer %s failed: %s", peer.ID, err.Error())
				continue
			}
			t.Log.Infof("task reports peer %s code %s", peer.ID, peerPacket.Code)

			if err := peer.FSM.Event(context.Background(), event); err != nil {
				peer.Log.Errorf("peer fsm event failed: %s", err.Error())
				continue
			}
		}
	}
}
