/*
 *     Copyright 2025 The Dragonfly Authors
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

//go:generate mockgen -destination peer_manager_mock.go -source peer_manager.go -package persistent

package persistent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bits-and-blooms/bitset"
	redis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc/credentials"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	pkgredis "d7y.io/dragonfly/v2/pkg/redis"
	"d7y.io/dragonfly/v2/scheduler/config"
)

// PeerManager is the interface used for peer manager.
type PeerManager interface {
	// Load returns peer by a key.
	Load(context.Context, string) (*Peer, bool)

	// Store sets peer.
	Store(context.Context, *Peer) error

	// Delete deletes peer by a key.
	Delete(context.Context, string) error

	// LoadAll returns all peers.
	LoadAll(context.Context) ([]*Peer, error)

	// LoadAllByTaskID returns all peers by task id.
	LoadAllByTaskID(context.Context, string) ([]*Peer, error)

	// LoadAllIDsByTaskID returns all peer ids by task id.
	LoadAllIDsByTaskID(context.Context, string) ([]string, error)

	// LoadPersistentAllByTaskID returns all persistent peers by task id.
	LoadPersistentAllByTaskID(context.Context, string) ([]*Peer, error)

	// DeleteAllByTaskID deletes all peers by task id.
	DeleteAllByTaskID(context.Context, string) error

	// LoadAllByHostID returns all peers by host id.
	LoadAllByHostID(context.Context, string) ([]*Peer, error)

	// LoadAllIDsByHostID returns all peer ids by host id.
	LoadAllIDsByHostID(context.Context, string) ([]string, error)

	// DeleteAllByHostID deletes all peers by host id.
	DeleteAllByHostID(context.Context, string) error
}

// peerManager contains content for peer manager.
type peerManager struct {
	// Config is scheduler config.
	config *config.Config

	// taskManager is the manager of task.
	taskManager TaskManager

	// hostManager is the manager of host.
	hostManager HostManager

	// Redis universal client interface.
	rdb redis.UniversalClient

	// transportCredentials is used to mTLS for peer grpc connection.
	transportCredentials credentials.TransportCredentials
}

// New peer manager interface.
func newPeerManager(cfg *config.Config, rdb redis.UniversalClient, taskManager TaskManager, hostManager HostManager, transportCredentials credentials.TransportCredentials) PeerManager {
	return &peerManager{config: cfg, rdb: rdb, taskManager: taskManager, hostManager: hostManager, transportCredentials: transportCredentials}
}

// Load returns persistent peer by a key.
func (p *peerManager) Load(ctx context.Context, peerID string) (*Peer, bool) {
	log := logger.WithPeerID(peerID)
	rawPeer, err := p.rdb.HGetAll(ctx, pkgredis.MakePersistentCachePeerKeyForPersistentTaskInScheduler(p.config.Manager.SchedulerClusterID, peerID)).Result()
	if err != nil {
		log.Errorf("getting peer failed from redis: %v", err)
		return nil, false
	}

	if len(rawPeer) == 0 {
		return nil, false
	}

	persistent, err := strconv.ParseBool(rawPeer["persistent"])
	if err != nil {
		log.Errorf("parsing persistent failed: %v", err)
		return nil, false
	}

	finishedPieces := &bitset.BitSet{}
	if err := finishedPieces.UnmarshalBinary([]byte(rawPeer["finished_pieces"])); err != nil {
		log.Errorf("unmarshal finished pieces failed: %v", err)
		return nil, false
	}

	blockParents := []string{}
	if err := json.Unmarshal([]byte(rawPeer["block_parents"]), &blockParents); err != nil {
		log.Errorf("unmarshal block parents failed: %v", err)
		return nil, false
	}

	// Set time fields from raw task.
	cost, err := strconv.ParseInt(rawPeer["cost"], 10, 64)
	if err != nil {
		log.Errorf("parsing cost failed: %v", err)
		return nil, false
	}

	createdAt, err := time.Parse(time.RFC3339, rawPeer["created_at"])
	if err != nil {
		log.Errorf("parsing created at failed: %v", err)
		return nil, false
	}

	updatedAt, err := time.Parse(time.RFC3339, rawPeer["updated_at"])
	if err != nil {
		log.Errorf("parsing updated at failed: %v", err)
		return nil, false
	}

	host, loaded := p.hostManager.Load(ctx, rawPeer["host_id"])
	if !loaded {
		log.Errorf("host not found")
		return nil, false
	}

	task, loaded := p.taskManager.Load(ctx, rawPeer["task_id"])
	if !loaded {
		log.Errorf("task not found")
		return nil, false
	}

	concurrentPieceCount, err := strconv.ParseUint(rawPeer["concurrent_piece_count"], 10, 32)
	if err != nil {
		log.Errorf("parsing concurrent piece count failed: %v", err)
		return nil, false
	}

	return NewPeer(
		rawPeer["id"],
		rawPeer["state"],
		persistent,
		finishedPieces,
		blockParents,
		task,
		host,
		time.Duration(cost),
		createdAt,
		updatedAt,
		logger.WithPeer(host.ID, task.ID, rawPeer["id"]),
		WithConcurrentPieceCount(uint32(concurrentPieceCount)),
	), true
}

// Store sets persistent peer.
func (p *peerManager) Store(ctx context.Context, peer *Peer) error {
	// Marshal finished pieces and block parents.
	finishedPieces, err := peer.FinishedPieces.MarshalBinary()
	if err != nil {
		peer.Log.Errorf("marshal finished pieces failed: %v", err)
		return err
	}

	blockParents, err := json.Marshal(peer.BlockParents)
	if err != nil {
		peer.Log.Errorf("marshal block parents failed: %v", err)
		return err
	}

	// Calculate remaining TTL in seconds.
	ttl := peer.Task.TTL - time.Since(peer.Task.CreatedAt)
	remainingTTLSeconds := int64(ttl.Seconds())

	// Define the Lua script as a string.
	const storePeerScript = `
-- Extract keys
local peer_key = KEYS[1]  -- Key for the peer hash
local task_peers_key = KEYS[2]  -- Key for the task joint-set
local persistent_task_peers_key = KEYS[3]  -- Key for the persistent task joint-set
local host_peers_key = KEYS[4]  -- Key for the host joint-set

-- Extract arguments
local peer_id = ARGV[1]
local persistent = tonumber(ARGV[2])
local finished_pieces = ARGV[3]
local state = ARGV[4]
local block_parents = ARGV[5]
local task_id = ARGV[6]
local host_id = ARGV[7]
local cost = ARGV[8]
local created_at = ARGV[9]
local updated_at = ARGV[10]
local concurrent_piece_count = ARGV[11]
local ttl_seconds = tonumber(ARGV[12])

-- Store peer information
redis.call("HSET", peer_key,
    "id", peer_id,
    "persistent", persistent,
    "finished_pieces", finished_pieces,
    "state", state,
    "block_parents", block_parents,
    "task_id", task_id,
    "host_id", host_id,
    "cost", cost,
    "created_at", created_at,
    "updated_at", updated_at,
		"concurrent_piece_count", concurrent_piece_count)

-- Set expiration for the peer key
redis.call("EXPIRE", peer_key, ttl_seconds)

-- Add peer ID to the task joint-set
redis.call("SADD", task_peers_key, peer_id)
redis.call("EXPIRE", task_peers_key, ttl_seconds)

-- Add peer ID to the persistent task joint-set if persistent
if persistent == 1 then
    redis.call("SADD", persistent_task_peers_key, peer_id)
		redis.call("EXPIRE", persistent_task_peers_key, ttl_seconds)
end

-- Add peer ID to the host joint-set
redis.call("SADD", host_peers_key, peer_id)
redis.call("EXPIRE", host_peers_key, ttl_seconds)

return true
`

	// Create a new Redis script.
	script := redis.NewScript(storePeerScript)

	// Prepare keys.
	keys := []string{
		pkgredis.MakePersistentCachePeerKeyForPersistentTaskInScheduler(p.config.Manager.SchedulerClusterID, peer.ID),
		pkgredis.MakePersistentCachePeersOfPersistentTaskInScheduler(p.config.Manager.SchedulerClusterID, peer.Task.ID),
		pkgredis.MakePersistentPeersOfPersistentTaskInScheduler(p.config.Manager.SchedulerClusterID, peer.Task.ID),
		pkgredis.MakePersistentCachePeersOfPersistentHostInScheduler(p.config.Manager.SchedulerClusterID, peer.Host.ID),
	}

	// Prepare arguments.
	args := []any{
		peer.ID,
		peer.Persistent,
		string(finishedPieces),
		peer.FSM.Current(),
		string(blockParents),
		peer.Task.ID,
		peer.Host.ID,
		peer.Cost.Nanoseconds(),
		peer.CreatedAt.Format(time.RFC3339),
		peer.UpdatedAt.Format(time.RFC3339),
		peer.ConcurrentPieceCount,
		remainingTTLSeconds,
	}

	// Execute the script.
	if err = script.Run(ctx, p.rdb, keys, args...).Err(); err != nil {
		peer.Log.Errorf("store peer failed: %v", err)
		return err
	}

	return nil
}

// Delete deletes persistent peer by a key, and it will delete the association with task and host at the same time.
func (p *peerManager) Delete(ctx context.Context, peerID string) error {
	// Define the Lua script as a string.
	const deletePeerScript = `
-- Extract keys
local peer_key = KEYS[1]  -- Key for the peer hash
local task_peers_key = KEYS[2]  -- Key for the task joint-set
local persistent_task_peers_key = KEYS[3]  -- Key for the persistent task joint-set
local host_peers_key = KEYS[4]  -- Key for the host joint-set

-- Extract arguments
local peer_id = ARGV[1]
local persistent = tonumber(ARGV[2])
local task_id = ARGV[3]
local host_id = ARGV[4]

-- Check if the peer exists
if redis.call("EXISTS", peer_key) == 0 then
    return {err = "peer not found"}
end

-- Delete the peer key
redis.call("DEL", peer_key)

-- Remove peer ID from the task joint-set
redis.call("SREM", task_peers_key, peer_id)

-- Remove peer ID from the persistent task joint-set if persistent is 1
if persistent == 1 then
    redis.call("SREM", persistent_task_peers_key, peer_id)
end

-- Remove peer ID from the host joint-set
redis.call("SREM", host_peers_key, peer_id)

return true
`

	log := logger.WithPeerID(peerID)

	// Create a new Redis script.
	script := redis.NewScript(deletePeerScript)

	// Load the peer to get its metadata.
	peer, found := p.Load(ctx, peerID)
	if !found {
		log.Errorf("getting peer failed from redis")
		return errors.New("getting peer failed from redis")
	}

	// Prepare keys.
	keys := []string{
		pkgredis.MakePersistentCachePeerKeyForPersistentTaskInScheduler(p.config.Manager.SchedulerClusterID, peerID),
		pkgredis.MakePersistentCachePeersOfPersistentTaskInScheduler(p.config.Manager.SchedulerClusterID, peer.Task.ID),
		pkgredis.MakePersistentPeersOfPersistentTaskInScheduler(p.config.Manager.SchedulerClusterID, peer.Task.ID),
		pkgredis.MakePersistentCachePeersOfPersistentHostInScheduler(p.config.Manager.SchedulerClusterID, peer.Host.ID),
	}

	// Prepare arguments.
	args := []any{
		peerID,
		peer.Persistent,
		peer.Task.ID,
		peer.Host.ID,
	}

	// Execute the script.
	if err := script.Run(ctx, p.rdb, keys, args...).Err(); err != nil {
		peer.Log.Errorf("delete peer failed: %v", err)
		return err
	}

	return nil
}

// LoadAll returns all persistent peers.
func (p *peerManager) LoadAll(ctx context.Context) ([]*Peer, error) {
	var (
		peers  []*Peer
		cursor uint64
	)

	for {
		var (
			peerKeys []string
			err      error
		)

		// For example, if {prefix} is "scheduler:scheduler-clusters:{1}:persistent-cache-peers-for-persistent-task:", keys could be:
		// "{prefix}{peerID1}", "{prefix}{peerID2}", "{prefix}{peerID3}", ...
		// Scan all keys with prefix.
		prefix := fmt.Sprintf("%s:", pkgredis.MakePersistentCachePeersForPersistentTaskInScheduler(p.config.Manager.SchedulerClusterID))
		peerKeys, cursor, err = pkgredis.ScanSchedulerKeys(ctx, p.rdb, cursor, fmt.Sprintf("%s*", prefix), 10)
		if err != nil {
			logger.Errorf("scan tasks failed: %v", err)
			return nil, err
		}

		for _, peerKey := range peerKeys {
			// If context is done, return error.
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			// Remove prefix from peer key, then the peerID is got.
			peerID := strings.TrimPrefix(peerKey, prefix)
			if peerID == "" {
				logger.Error("invalid peer key")
				continue
			}

			peer, loaded := p.Load(ctx, peerID)
			if !loaded {
				logger.WithPeerID(peerID).Error("load peer failed")
				continue
			}

			peers = append(peers, peer)
		}

		if cursor == 0 {
			break
		}
	}

	return peers, nil
}

// LoadAllByTaskID returns all persistent peers by task id.
func (p *peerManager) LoadAllByTaskID(ctx context.Context, taskID string) ([]*Peer, error) {
	log := logger.WithTaskID(taskID)
	peerIDs, err := p.rdb.SMembers(ctx, pkgredis.MakePersistentCachePeersOfPersistentTaskInScheduler(p.config.Manager.SchedulerClusterID, taskID)).Result()
	if err != nil {
		log.Errorf("get peer ids failed: %v", err)
		return nil, err
	}

	peers := make([]*Peer, 0, len(peerIDs))
	for _, peerID := range peerIDs {
		peer, loaded := p.Load(ctx, peerID)
		if !loaded {
			log.Errorf("load peer %s failed: %v", peerID, err)
			continue
		}

		peers = append(peers, peer)
	}

	return peers, nil
}

// LoadAllIDsByTaskID returns all peer ids by task id.
func (p *peerManager) LoadAllIDsByTaskID(ctx context.Context, taskID string) ([]string, error) {
	log := logger.WithTaskID(taskID)
	peerIDs, err := p.rdb.SMembers(ctx, pkgredis.MakePersistentCachePeersOfPersistentTaskInScheduler(p.config.Manager.SchedulerClusterID, taskID)).Result()
	if err != nil {
		log.Errorf("get peer ids failed: %v", err)
		return nil, err
	}

	return peerIDs, nil
}

// LoadPersistentAllByTaskID returns all persistent peers by task id.
func (p *peerManager) LoadPersistentAllByTaskID(ctx context.Context, taskID string) ([]*Peer, error) {
	log := logger.WithTaskID(taskID)
	peerIDs, err := p.rdb.SMembers(ctx, pkgredis.MakePersistentCachePeersOfPersistentTaskInScheduler(p.config.Manager.SchedulerClusterID, taskID)).Result()
	if err != nil {
		log.Errorf("get peer ids failed: %v", err)
		return nil, err
	}

	peers := make([]*Peer, 0, len(peerIDs))
	for _, peerID := range peerIDs {
		peer, loaded := p.Load(ctx, peerID)
		if !loaded {
			log.Errorf("load peer %s failed: %v", peerID, err)
			continue
		}

		peers = append(peers, peer)
	}

	return peers, nil
}

// DeleteAllByTaskID deletes all persistent peers by task id.
func (p *peerManager) DeleteAllByTaskID(ctx context.Context, taskID string) error {
	log := logger.WithTaskID(taskID)
	ids, err := p.LoadAllIDsByTaskID(ctx, taskID)
	if err != nil {
		log.Errorf("load peers failed: %v", err)
		return err
	}

	for _, id := range ids {
		if err := p.Delete(ctx, id); err != nil {
			log.Errorf("delete peer %s failed: %v", id, err)
			continue
		}
	}

	return nil
}

// LoadAllByHostID returns all persistent peers by host id.
func (p *peerManager) LoadAllByHostID(ctx context.Context, hostID string) ([]*Peer, error) {
	log := logger.WithHostID(hostID)
	peerIDs, err := p.rdb.SMembers(ctx, pkgredis.MakePersistentCachePeersOfPersistentHostInScheduler(p.config.Manager.SchedulerClusterID, hostID)).Result()
	if err != nil {
		log.Errorf("get peer ids failed: %v", err)
		return nil, err
	}

	peers := make([]*Peer, 0, len(peerIDs))
	for _, peerID := range peerIDs {
		peer, loaded := p.Load(ctx, peerID)
		if !loaded {
			log.Errorf("load peer %s failed: %v", peerID, err)
			continue
		}

		peers = append(peers, peer)
	}

	return peers, nil
}

// LoadAllIDsByHostID returns all persistent peers by host id.
func (p *peerManager) LoadAllIDsByHostID(ctx context.Context, hostID string) ([]string, error) {
	log := logger.WithHostID(hostID)
	peerIDs, err := p.rdb.SMembers(ctx, pkgredis.MakePersistentCachePeersOfPersistentHostInScheduler(p.config.Manager.SchedulerClusterID, hostID)).Result()
	if err != nil {
		log.Errorf("get peer ids failed: %v", err)
		return nil, err
	}

	return peerIDs, nil
}

// DeleteAllByHostID deletes all persistent peers by host id.
func (p *peerManager) DeleteAllByHostID(ctx context.Context, hostID string) error {
	log := logger.WithTaskID(hostID)
	ids, err := p.LoadAllIDsByHostID(ctx, hostID)
	if err != nil {
		log.Errorf("load peers failed: %v", err)
		return err
	}

	for _, id := range ids {
		if err := p.Delete(ctx, id); err != nil {
			log.Errorf("delete peer %s failed: %v", id, err)
			continue
		}
	}

	return nil
}
