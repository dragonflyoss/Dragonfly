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

//go:generate mockgen -destination peer_manager_mock.go -source peer_manager.go -package standard

package standard

import (
	"context"
	"sync"
	"time"

	"k8s.io/utils/keymutex"

	pkggc "d7y.io/dragonfly/v2/pkg/gc"
	"d7y.io/dragonfly/v2/scheduler/config"
)

const (
	// GC peer id.
	GCPeerID = "peer"
)

// peerMutexShardCount is the number of mutex shards for peer mutations.
const peerMutexShardCount = 256

// PeerManager is the interface used for peer manager.
type PeerManager interface {
	// Load returns peer for a key.
	Load(string) (*Peer, bool)

	// Store sets peer.
	Store(*Peer)

	// LoadOrStore returns peer the key if present.
	// Otherwise, it stores and returns the given peer.
	// The loaded result is true if the peer was loaded, false if stored.
	LoadOrStore(*Peer) (*Peer, bool)

	// Delete deletes peer for a key.
	Delete(string)

	// DeleteAllByHostID deletes all peers by host id.
	DeleteAllByHostID(string)

	// Range calls f sequentially for each key and value present in the map.
	// If f returns false, range stops the iteration.
	Range(f func(any, any) bool)

	// Try to reclaim peer.
	RunGC(context.Context) error
}

// peerManager contains content for peer manager.
type peerManager struct {
	// Peer sync map.
	*sync.Map

	// peerTTL is time to live of peer.
	peerTTL time.Duration

	// hostTTL is time to live of host.
	hostTTL time.Duration

	// pieceDownloadTimeout is timeout of downloading piece.
	pieceDownloadTimeout time.Duration

	// mu is a mutex for peer mutations.
	mu keymutex.KeyMutex
}

// New peer manager interface.
func newPeerManager(cfg *config.GCConfig, gc pkggc.GC) (PeerManager, error) {
	p := &peerManager{
		Map:                  &sync.Map{},
		peerTTL:              cfg.PeerTTL,
		hostTTL:              cfg.HostTTL,
		pieceDownloadTimeout: cfg.PieceDownloadTimeout,
		mu:                   keymutex.NewHashed(peerMutexShardCount),
	}

	if err := gc.Add(pkggc.Task{
		ID:       GCPeerID,
		Interval: cfg.PeerGCInterval,
		Timeout:  cfg.PeerGCInterval,
		Runner:   p,
	}); err != nil {
		return nil, err
	}

	return p, nil
}

// Load returns peer for a key.
func (p *peerManager) Load(key string) (*Peer, bool) {
	rawPeer, loaded := p.Map.Load(key)
	if !loaded {
		return nil, false
	}

	return rawPeer.(*Peer), loaded
}

// Store sets peer.
func (p *peerManager) Store(peer *Peer) {
	p.mu.LockKey(peer.ID)
	defer p.mu.UnlockKey(peer.ID) // nolint: errcheck

	p.Map.Store(peer.ID, peer)
	peer.Task.StorePeer(peer)
	peer.Host.StorePeer(peer)
}

// LoadOrStore returns peer the key if present.
// Otherwise, it stores and returns the given peer.
// The loaded result is true if the peer was loaded, false if stored.
func (p *peerManager) LoadOrStore(peer *Peer) (*Peer, bool) {
	p.mu.LockKey(peer.ID)
	defer p.mu.UnlockKey(peer.ID) // nolint: errcheck

	rawPeer, loaded := p.Map.LoadOrStore(peer.ID, peer)
	if !loaded {
		peer.Host.StorePeer(peer)
		peer.Task.StorePeer(peer)
	}

	return rawPeer.(*Peer), loaded
}

// Delete deletes peer for a key.
func (p *peerManager) Delete(key string) {
	p.mu.LockKey(key)
	defer p.mu.UnlockKey(key) // nolint: errcheck

	if peer, loaded := p.Load(key); loaded {
		p.Map.Delete(key)
		peer.Task.DeletePeer(key)
		peer.Host.DeletePeer(key)
	}
}

// DeleteAllByHostID deletes all peers by host id.
func (p *peerManager) DeleteAllByHostID(hostID string) {
	p.Map.Range(func(_, value any) bool {
		peer, ok := value.(*Peer)
		if !ok {
			return true
		}

		if peer.Host.ID == hostID {
			p.Delete(peer.ID)
		}

		return true
	})
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (p *peerManager) Range(f func(key, value any) bool) {
	p.Map.Range(f)
}

// Try to reclaim peer.
func (p *peerManager) RunGC(ctx context.Context) error {
	p.Map.Range(func(_, value any) bool {
		peer, ok := value.(*Peer)
		if !ok {
			return true
		}

		// If the peer state is PeerStateLeave,
		// peer will be reclaimed.
		if peer.FSM.Is(PeerStateLeave) {
			p.Delete(peer.ID)
			peer.Log.Info("peer has been reclaimed")
			return true
		}

		// If host is disabled shared, then set the peer state to PeerStateLeave and then delete peer.
		// Avoid the disabled shared host to be scheduled, and store the unused peer in the peer manager.
		if peer.Host.DisableShared {
			peer.Log.Info("peer host is disabled shared, causing the peer to leave")
			if err := peer.FSM.Event(ctx, PeerEventLeave); err != nil {
				peer.Log.Errorf("peer fsm event failed: %s", err.Error())
				return true
			}

			return true
		}

		// If the peer's elapsed of downloading piece exceeds the pieceDownloadTimeout,
		// then sets the peer state to PeerStateLeave and then delete peer.
		if peer.FSM.Is(PeerStateRunning) || peer.FSM.Is(PeerStateBackToSource) {
			elapsed := time.Since(peer.PieceUpdatedAt.Load())
			if elapsed > p.pieceDownloadTimeout {
				peer.Log.Info("peer elapsed exceeds the timeout of downloading piece, causing the peer to leave")
				if err := peer.FSM.Event(ctx, PeerEventLeave); err != nil {
					peer.Log.Errorf("peer fsm event failed: %s", err.Error())
					return true
				}

				return true
			}
		}

		// If the peer's elapsed exceeds the peer ttl,
		// then set the peer state to PeerStateLeave and then delete peer.
		elapsed := time.Since(peer.UpdatedAt.Load())
		if elapsed > p.peerTTL {
			peer.Log.Info("peer elapsed exceeds the peer ttl, causing the peer to leave")
			if err := peer.FSM.Event(ctx, PeerEventLeave); err != nil {
				peer.Log.Errorf("peer fsm event failed: %s", err.Error())
				return true
			}

			return true
		}

		// If the host's elapsed exceeds the host ttl,
		// then set the peer state to PeerStateLeave and then delete peer.
		elapsed = time.Since(peer.Host.UpdatedAt.Load())
		if elapsed > p.hostTTL {
			peer.Log.Info("peer elapsed exceeds the host ttl, causing the peer to leave")
			if err := peer.FSM.Event(ctx, PeerEventLeave); err != nil {
				peer.Log.Errorf("peer fsm event failed: %s", err.Error())
				return true
			}

			return true
		}

		// If the peer's state is PeerStateFailed,
		// then set the peer state to PeerStateLeave and then delete peer.
		if peer.FSM.Is(PeerStateFailed) {
			peer.Log.Info("peer state is PeerStateFailed, causing the peer to leave")
			if err := peer.FSM.Event(ctx, PeerEventLeave); err != nil {
				peer.Log.Errorf("peer fsm event failed: %s", err.Error())
				return true
			}
		}

		// If no peer exists in the dag of the task,
		// delete the peer.
		degree, err := peer.Task.PeerDegree(peer.ID)
		if err != nil {
			p.Delete(peer.ID)
			peer.Log.Info("peer has been reclaimed")
			return true
		}

		// If the task dag size exceeds the limit,
		// then set the peer state to PeerStateLeave which state is
		// PeerStateSucceeded, and degree is zero.
		if peer.Task.PeerCount() > PeerCountLimitForTask &&
			peer.FSM.Is(PeerStateSucceeded) && degree == 0 {
			peer.Log.Info("task dag size exceeds the limit, causing the peer to leave")
			if err := peer.FSM.Event(ctx, PeerEventLeave); err != nil {
				peer.Log.Errorf("peer fsm event failed: %s", err.Error())
				return true
			}

			p.Delete(peer.ID)
			peer.Log.Info("peer has been reclaimed")
			return true
		}

		return true
	})

	return nil
}
