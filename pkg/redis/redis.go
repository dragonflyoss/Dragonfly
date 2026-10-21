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
	"strings"

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
	if cfg.MasterName == "" && len(cfg.Addrs) > 1 && cfg.DB != 0 {
		return nil, fmt.Errorf("redis cluster requires database 0, got %d", cfg.DB)
	}
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
		_ = client.Close()
		logger.Errorf("failed to ping redis: %v", err)
		return nil, err
	}

	return client, nil
}

// IsEnabled check redis is enabled.
func IsEnabled(addrs []string) bool {
	return len(addrs) != 0
}

// ScanSchedulerKeys scans the master owning the scheduler cluster hash tag.
// ClusterClient.Scan otherwise selects an arbitrary node and can miss records.
func ScanSchedulerKeys(ctx context.Context, client redis.UniversalClient, cursor uint64, pattern string, count int64) ([]string, uint64, error) {
	if cluster, ok := client.(*redis.ClusterClient); ok {
		master, err := cluster.MasterForKey(ctx, pattern)
		if err != nil {
			return nil, 0, err
		}
		return master.Scan(ctx, cursor, pattern, count).Result()
	}
	return client.Scan(ctx, cursor, pattern, count).Result()
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
	if namespace == SchedulerClustersNamespace {
		clusterID, suffix, found := strings.Cut(id, ":")
		if found {
			// Host, peer and task Lua scripts touch several related keys. Pin all
			// keys for one scheduler cluster to the same Redis Cluster hash slot.
			return fmt.Sprintf("%s:{%s}:%s", MakeNamespaceKeyInScheduler(namespace), clusterID, suffix)
		}
	}
	return fmt.Sprintf("%s:%s", MakeNamespaceKeyInScheduler(namespace), id)
}

// MakePersistentTaskKeyInScheduler make persistent task key in scheduler.
func MakePersistentTaskKeyInScheduler(schedulerClusterID uint, taskID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s:%s", schedulerClusterID, PersistentTasksNamespace, taskID))
}

// MakePersistentTasksInScheduler make persistent tasks in scheduler.
func MakePersistentTasksInScheduler(schedulerClusterID uint) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s", schedulerClusterID, PersistentTasksNamespace))
}

// MakePersistentCachePeersOfPersistentTaskInScheduler make persistent cache peers of persistent task in scheduler.
func MakePersistentCachePeersOfPersistentTaskInScheduler(schedulerClusterID uint, taskID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s:%s:%s", schedulerClusterID, PersistentTasksNamespace, taskID, PersistentCachePeersForPersistentTaskNamespace))
}

// MakePersistentPeersOfPersistentTaskInScheduler make persistent peers of persistent task in scheduler.
func MakePersistentPeersOfPersistentTaskInScheduler(schedulerClusterID uint, taskID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s:%s:%s", schedulerClusterID, PersistentTasksNamespace, taskID, PersistentPeersForPersistentTaskNamespace))
}

// MakePersistentCachePeerKeyForPersistentTaskInScheduler make persistent cache peer key in scheduler for persistent task.
func MakePersistentCachePeerKeyForPersistentTaskInScheduler(schedulerClusterID uint, peerID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s:%s", schedulerClusterID, PersistentCachePeersForPersistentTaskNamespace, peerID))
}

// MakePersistentCachePeersForPersistentTaskInScheduler make persistent cache peers for persistent task in scheduler.
func MakePersistentCachePeersForPersistentTaskInScheduler(schedulerClusterID uint) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s", schedulerClusterID, PersistentCachePeersForPersistentTaskNamespace))
}

// MakePersistentHostKeyInScheduler make persistent host key in scheduler.
func MakePersistentHostKeyInScheduler(schedulerClusterID uint, hostID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s:%s", schedulerClusterID, PersistentHostsNamespace, hostID))
}

// MakePersistentHostsInScheduler make persistent hosts in scheduler.
func MakePersistentHostsInScheduler(schedulerClusterID uint) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s", schedulerClusterID, PersistentHostsNamespace))
}

// MakePersistentCachePeersOfPersistentHostInScheduler make persistent cache peers of persistent host in scheduler.
func MakePersistentCachePeersOfPersistentHostInScheduler(schedulerClusterID uint, hostID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s:%s:%s", schedulerClusterID, PersistentHostsNamespace, hostID, PersistentCachePeersForPersistentTaskNamespace))
}

// MakePersistentCacheTaskKeyInScheduler make persistent cache task key in scheduler.
func MakePersistentCacheTaskKeyInScheduler(schedulerClusterID uint, taskID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s:%s", schedulerClusterID, PersistentCacheTasksNamespace, taskID))
}

// MakePersistentCacheTasksInScheduler make persistent cache tasks in scheduler.
func MakePersistentCacheTasksInScheduler(schedulerClusterID uint) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s", schedulerClusterID, PersistentCacheTasksNamespace))
}

// MakePersistentCachePeersOfPersistentCacheTaskInScheduler make persistent cache peers of persistent cache task in scheduler.
func MakePersistentCachePeersOfPersistentCacheTaskInScheduler(schedulerClusterID uint, taskID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s:%s:%s", schedulerClusterID, PersistentCacheTasksNamespace, taskID, PersistentCachePeersForPersistentCacheTaskNamespace))
}

// MakePersistentPeersOfPersistentCacheTaskInScheduler make persistent peers of persistent cache task in scheduler.
func MakePersistentPeersOfPersistentCacheTaskInScheduler(schedulerClusterID uint, taskID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s:%s:%s", schedulerClusterID, PersistentCacheTasksNamespace, taskID, PersistentPeersForPersistentCacheTaskNamespace))
}

// MakePersistentCachePeerKeyForPersistentCacheTaskInScheduler make persistent cache peer key in scheduler for persistent cache task.
func MakePersistentCachePeerKeyForPersistentCacheTaskInScheduler(schedulerClusterID uint, peerID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s:%s", schedulerClusterID, PersistentCachePeersForPersistentCacheTaskNamespace, peerID))
}

// MakePersistentCachePeersForPersistentCacheTaskInScheduler make persistent cache peers for persistent cache task in scheduler.
func MakePersistentCachePeersForPersistentCacheTaskInScheduler(schedulerClusterID uint) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s", schedulerClusterID, PersistentCachePeersForPersistentCacheTaskNamespace))
}

// MakePersistentCacheHostKeyInScheduler make persistent cache host key in scheduler.
func MakePersistentCacheHostKeyInScheduler(schedulerClusterID uint, hostID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s:%s", schedulerClusterID, PersistentCacheHostsNamespace, hostID))
}

// MakePersistentCacheHostsInScheduler make persistent cache hosts in scheduler.
func MakePersistentCacheHostsInScheduler(schedulerClusterID uint) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s", schedulerClusterID, PersistentCacheHostsNamespace))
}

// MakePersistentCachePeersOfPersistentCacheHostInScheduler make persistent cache peers of persistent cache host in scheduler.
func MakePersistentCachePeersOfPersistentCacheHostInScheduler(schedulerClusterID uint, hostID string) string {
	return MakeKeyInScheduler(SchedulerClustersNamespace, fmt.Sprintf("%d:%s:%s:%s", schedulerClusterID, PersistentCacheHostsNamespace, hostID, PersistentCachePeersForPersistentCacheTaskNamespace))
}
