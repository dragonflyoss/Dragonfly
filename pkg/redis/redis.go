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

package redis

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/redis/go-redis/v9"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	"d7y.io/dragonfly/v2/pkg/types"
)

const (
	// KeySeparator is the separator of redis key.
	KeySeparator = ":"
)

const (
	// SeedPeerNamespace prefix of seed peers namespace cache key.
	SeedPeersNamespace = "seed-peers"

	// PeersNamespace prefix of peers namespace cache key.
	PeersNamespace = "peers"

	// SchedulersNamespace prefix of schedulers namespace cache key.
	SchedulersNamespace = "schedulers"

	// SchedulerClustersNamespace prefix of scheduler clusters namespace cache key.
	SchedulerClustersNamespace = "scheduler-clusters"

	// PersistentTasksNamespace prefix of tasks namespace cache key.
	PersistentTasksNamespace = "persistent-tasks"

	// PersistentHostsNamespace prefix of persistent cache hosts namespace cache key.
	PersistentHostsNamespace = "persistent-hosts"

	// PersistentCachePeersForPersistentTaskNamespace prefix of persistent cache peers namespace cache key for persistent task.
	PersistentCachePeersForPersistentTaskNamespace = "persistent-cache-peers-for-persistent-task"

	// PersistentPeersForPersistentTaskNamespace prefix of persistent peers namespace cache key for persistent task.
	PersistentPeersForPersistentTaskNamespace = "persistent-peers-for-persistent-task"

	// PersistentCacheTasksNamespace prefix of tasks namespace cache key.
	PersistentCacheTasksNamespace = "persistent-cache-tasks"

	// PersistentCacheHostsNamespace prefix of persistent cache hosts namespace cache key.
	PersistentCacheHostsNamespace = "persistent-cache-hosts"

	// PersistentCachePeersForPersistentCacheTaskNamespace prefix of persistent cache peers namespace cache key for persistent cache task.
	PersistentCachePeersForPersistentCacheTaskNamespace = "persistent-cache-peers-for-persistent-cache-task"

	// PersistentPeersForPersistentCacheTaskNamespace prefix of persistent peers namespace cache key for persistent cache task.
	PersistentPeersForPersistentCacheTaskNamespace = "persistent-peers-for-persistent-cache-task"

	// ApplicationsNamespace prefix of applications namespace cache key.
	ApplicationsNamespace = "applications"
)

// NewRedis returns a new redis client.
func NewRedis(cfg *redis.UniversalOptions) (redis.UniversalClient, error) {
	redis.SetLogger(&redisLogger{})
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:            cfg.Addrs,
		MasterName:       cfg.MasterName,
		DB:               cfg.DB,
		Username:         cfg.Username,
		Password:         cfg.Password,
		SentinelUsername: cfg.SentinelUsername,
		SentinelPassword: cfg.SentinelPassword,
		PoolSize:         cfg.PoolSize,
		PoolTimeout:      cfg.PoolTimeout,
		TLSConfig:        cfg.TLSConfig,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		logger.Errorf("failed to ping redis: %v", err)
		return nil, err
	}

	return client, nil
}

// IsEnabled check redis is enabled.
func IsEnabled(addrs []string) bool {
	return len(addrs) != 0
}

// clusterModeEnabled records whether the scheduler's Redis client is a
// Redis Cluster client. It defaults to false (legacy, non-hash-tagged key
// format), and must be set once via SetClusterModeEnabled during process
// startup, before any scheduler key below is constructed. It is a
// package-level flag (rather than a parameter threaded through every key
// builder below) so that call sites and tests that only care about key
// names are unaffected by Redis topology.
var clusterModeEnabled atomic.Bool

// SetClusterModeEnabled records whether the scheduler's Redis client is a
// Redis Cluster client. Call this once during scheduler startup, right
// after the redis.UniversalClient is created (e.g. by checking whether it
// is a *redis.ClusterClient), and before constructing any resource manager
// that builds scheduler keys. When enabled, the scheduler key builders
// below embed the scheduler cluster ID in a Redis Cluster hash tag so that
// multi-key Lua scripts stay within a single slot. When disabled (the
// default), keys keep their original, non-hash-tagged format so that
// existing standalone/Sentinel deployments are unaffected.
func SetClusterModeEnabled(enabled bool) {
	clusterModeEnabled.Store(enabled)
}

// IsClusterModeEnabled reports whether scheduler keys are currently being
// built using the Redis Cluster hash-tag format.
func IsClusterModeEnabled() bool {
	return clusterModeEnabled.Load()
}

// IsClusterClient reports whether rdb is a Redis Cluster client. It is a
// thin wrapper around a type assertion so callers (e.g. scheduler startup)
// don't need to import go-redis's concrete types directly.
func IsClusterClient(rdb redis.UniversalClient) bool {
	_, ok := rdb.(*redis.ClusterClient)
	return ok
}

// schedulerClusterIDSegment renders the scheduler cluster ID segment used
// inside scheduler keys below. In Redis Cluster mode it is wrapped in a
// hash tag (e.g. "{1}"); otherwise it is rendered as a plain integer (e.g.
// "1"), exactly as it always has been, to avoid changing key names for
// standalone/Sentinel Redis.
func schedulerClusterIDSegment(schedulerClusterID uint) string {
	if clusterModeEnabled.Load() {
		return fmt.Sprintf("{%d}", schedulerClusterID)
	}

	return strconv.FormatUint(uint64(schedulerClusterID), 10)
}

// ScanSchedulerKeys is a Redis Cluster-aware wrapper around SCAN for keys
// produced by the scheduler key builders below. Redis Cluster's SCAN
// command is node-local: issuing SCAN against a redis.UniversalClient
// backed by a Redis Cluster only enumerates keys stored on whichever node
// happens to receive that keyless command, silently missing keys on other
// nodes. However, every key sharing the same "{schedulerClusterID}" hash
// tag always lives on a single Redis Cluster slot (and therefore a single
// node). So, when rdb is a Redis Cluster client, ScanSchedulerKeys first
// resolves the one primary node that owns pattern's hash tag (via
// MasterForKey) and scans only that node, keeping the returned cursor
// valid across repeated calls for the same pattern. When rdb is not a
// Redis Cluster client, it behaves exactly like rdb.Scan.
func ScanSchedulerKeys(ctx context.Context, rdb redis.UniversalClient, cursor uint64, pattern string, count int64) ([]string, uint64, error) {
	cluster, ok := rdb.(*redis.ClusterClient)
	if !ok {
		return rdb.Scan(ctx, cursor, pattern, count).Result()
	}

	master, err := cluster.MasterForKey(ctx, pattern)
	if err != nil {
		return nil, 0, err
	}

	return master.Scan(ctx, cursor, pattern, count).Result()
}

// MakeNamespaceKeyInManager make namespace key in manager.
func MakeNamespaceKeyInManager(namespace string) string {
	return fmt.Sprintf("%s:%s", types.ManagerName, namespace)
}

// MakeKeyInManager make key in manager.
func MakeKeyInManager(namespace, id string) string {
	return fmt.Sprintf("%s:%s", MakeNamespaceKeyInManager(namespace), id)
}

// MakeSeedPeerKeyInManager make seed peer key in manager.
func MakeSeedPeerKeyInManager(clusterID uint, hostname, ip string) string {
	return MakeKeyInManager(SeedPeersNamespace, fmt.Sprintf("%d-%s-%s", clusterID, hostname, ip))
}

// MakeSchedulerKeyInManager make scheduler key in manager.
func MakeSchedulerKeyInManager(clusterID uint, hostname, ip string) string {
	return MakeKeyInManager(SchedulersNamespace, fmt.Sprintf("%d-%s-%s", clusterID, hostname, ip))
}

// MakeSeedPeersKeyForPeerInManager make seed peers key for peer in manager.
func MakeSeedPeersKeyForPeerInManager(hostname, ip string) string {
	return MakeKeyInManager(PeersNamespace, fmt.Sprintf("%s-%s:seed-peers", hostname, ip))
}

// MakeSchedulersKeyForPeerInManager make schedulers key for peer in manager.
func MakeSchedulersKeyForPeerInManager(hostname, ip, version string) string {
	return MakeKeyInManager(PeersNamespace, fmt.Sprintf("%s-%s-%s:schedulers", hostname, ip, version))
}

// MakeSchedulersByClusterIDKeyForPeerInManager make schedulers by cluster ID key for peer in manager.
func MakeSchedulersByClusterIDKeyForPeerInManager(schedulerClusterID uint) string {
	return MakeKeyInManager(SchedulerClustersNamespace, fmt.Sprintf("%d:schedulers", schedulerClusterID))
}

// MakeApplicationsKeyInManager make applications key in manager.
func MakeApplicationsKeyInManager() string {
	return MakeNamespaceKeyInManager(ApplicationsNamespace)
}

// MakeNamespaceKeyInScheduler make namespace key in scheduler.
func MakeNamespaceKeyInScheduler(namespace string) string {
	return fmt.Sprintf("%s:%s", types.SchedulerName, namespace)
}

// MakeKeyInScheduler make key in scheduler.
func MakeKeyInScheduler(namespace, id string) string {
	return fmt.Sprintf("%s:%s", MakeNamespaceKeyInScheduler(namespace), id)
}

// All scheduler keys below embed the scheduler cluster ID via
// schedulerClusterIDSegment. When SetClusterModeEnabled(true) has been
// called (because the scheduler's Redis client is a Redis Cluster client),
// that segment is wrapped in curly braces, e.g.
// "{1}:persistent-cache-hosts:example-host". The curly braces are a Redis
// Cluster hash tag: when present, Redis Cluster hashes only the substring
// inside the braces to compute the key's slot, instead of the whole key.
// Several of these keys are read and written together inside a single Lua
// script (see host_manager.go and peer_manager.go in
// scheduler/resource/persistent and scheduler/resource/persistentcache),
// which Redis Cluster only allows if every key in the script hashes to the
// same slot. Since every key for a given scheduler cluster shares the same
// "{schedulerClusterID}" hash tag, those scripts always stay within a
// single slot, avoiding CROSSSLOT errors when the scheduler is configured
// against a Redis Cluster.
//
// When the scheduler's Redis client is not a Redis Cluster client (the
// default), schedulerClusterIDSegment renders a plain integer instead, so
// keys keep their original, pre-existing format and standalone/Sentinel
// deployments are unaffected by Redis Cluster support.

// MakePersistentTaskKeyInScheduler make persistent task key in scheduler.
func MakePersistentTaskKeyInScheduler(schedulerClusterID uint, taskID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentTasksNamespace, taskID))
}

// MakePersistentTasksInScheduler make persistent tasks in scheduler.
func MakePersistentTasksInScheduler(schedulerClusterID uint) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentTasksNamespace))
}

// MakePersistentCachePeersOfPersistentTaskInScheduler make persistent cache peers of persistent task in scheduler.
func MakePersistentCachePeersOfPersistentTaskInScheduler(schedulerClusterID uint, taskID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s:%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentTasksNamespace, taskID, PersistentCachePeersForPersistentTaskNamespace))
}

// MakePersistentPeersOfPersistentTaskInScheduler make persistent peers of persistent task in scheduler.
func MakePersistentPeersOfPersistentTaskInScheduler(schedulerClusterID uint, taskID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s:%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentTasksNamespace, taskID, PersistentPeersForPersistentTaskNamespace))
}

// MakePersistentCachePeerKeyForPersistentTaskInScheduler make persistent cache peer key in scheduler for persistent task.
func MakePersistentCachePeerKeyForPersistentTaskInScheduler(schedulerClusterID uint, peerID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentCachePeersForPersistentTaskNamespace, peerID))
}

// MakePersistentCachePeersForPersistentTaskInScheduler make persistent cache peers for persistent task in scheduler.
func MakePersistentCachePeersForPersistentTaskInScheduler(schedulerClusterID uint) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentCachePeersForPersistentTaskNamespace))
}

// MakePersistentHostKeyInScheduler make persistent host key in scheduler.
func MakePersistentHostKeyInScheduler(schedulerClusterID uint, hostID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentHostsNamespace, hostID))
}

// MakePersistentHostsInScheduler make persistent hosts in scheduler.
func MakePersistentHostsInScheduler(schedulerClusterID uint) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentHostsNamespace))
}

// MakePersistentCachePeersOfPersistentHostInScheduler make persistent cache peers of persistent host in scheduler.
func MakePersistentCachePeersOfPersistentHostInScheduler(schedulerClusterID uint, hostID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s:%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentHostsNamespace, hostID, PersistentCachePeersForPersistentTaskNamespace))
}

// MakePersistentCacheTaskKeyInScheduler make persistent cache task key in scheduler.
func MakePersistentCacheTaskKeyInScheduler(schedulerClusterID uint, taskID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentCacheTasksNamespace, taskID))
}

// MakePersistentCacheTasksInScheduler make persistent cache tasks in scheduler.
func MakePersistentCacheTasksInScheduler(schedulerClusterID uint) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentCacheTasksNamespace))
}

// MakePersistentCachePeersOfPersistentCacheTaskInScheduler make persistent cache peers of persistent cache task in scheduler.
func MakePersistentCachePeersOfPersistentCacheTaskInScheduler(schedulerClusterID uint, taskID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s:%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentCacheTasksNamespace, taskID, PersistentCachePeersForPersistentCacheTaskNamespace))
}

// MakePersistentPeersOfPersistentCacheTaskInScheduler make persistent peers of persistent cache task in scheduler.
func MakePersistentPeersOfPersistentCacheTaskInScheduler(schedulerClusterID uint, taskID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s:%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentCacheTasksNamespace, taskID, PersistentPeersForPersistentCacheTaskNamespace))
}

// MakePersistentCachePeerKeyForPersistentCacheTaskInScheduler make persistent cache peer key in scheduler for persistent cache task.
func MakePersistentCachePeerKeyForPersistentCacheTaskInScheduler(schedulerClusterID uint, peerID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentCachePeersForPersistentCacheTaskNamespace, peerID))
}

// MakePersistentCachePeersForPersistentCacheTaskInScheduler make persistent cache peers for persistent cache task in scheduler.
func MakePersistentCachePeersForPersistentCacheTaskInScheduler(schedulerClusterID uint) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentCachePeersForPersistentCacheTaskNamespace))
}

// MakePersistentCacheHostKeyInScheduler make persistent cache host key in scheduler.
func MakePersistentCacheHostKeyInScheduler(schedulerClusterID uint, hostID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentCacheHostsNamespace, hostID))
}

// MakePersistentCacheHostsInScheduler make persistent cache hosts in scheduler.
func MakePersistentCacheHostsInScheduler(schedulerClusterID uint) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentCacheHostsNamespace))
}

// MakePersistentCachePeersOfPersistentCacheHostInScheduler make persistent cache peers of persistent cache host in scheduler.
func MakePersistentCachePeersOfPersistentCacheHostInScheduler(schedulerClusterID uint, hostID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%s:%s:%s:%s", schedulerClusterIDSegment(schedulerClusterID), PersistentCacheHostsNamespace, hostID, PersistentCachePeersForPersistentCacheTaskNamespace))
}
