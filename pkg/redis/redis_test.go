/*
 *     Copyright 2024 The Dragonfly Authors
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
	"regexp"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// hashTagPattern matches a Redis Cluster hash tag, i.e. the substring
// wrapped in curly braces that Redis Cluster uses to compute a key's slot
// instead of hashing the whole key.
var hashTagPattern = regexp.MustCompile(`\{([^{}]*)\}`)

// hashTag returns the Redis Cluster hash tag embedded in key, or an empty
// string if key has no hash tag.
func hashTag(key string) string {
	match := hashTagPattern.FindStringSubmatch(key)
	if match == nil {
		return ""
	}

	return match[1]
}

func Test_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		addrs  []string
		expect func(t *testing.T, ok bool)
	}{
		{
			name:  "check redis is enabled",
			addrs: []string{"172.0.0.1"},
			expect: func(t *testing.T, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
			},
		},
		{
			name:  "addrs is empty",
			addrs: []string{},
			expect: func(t *testing.T, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.expect(t, IsEnabled(tc.addrs))
		})
	}
}

func Test_MakeNamespaceKeyInManager(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		expect    func(t *testing.T, s string)
	}{
		{
			name:      "make namespace key in manager",
			namespace: "namespace",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:namespace", s)
			},
		},
		{
			name:      "namespace is empty",
			namespace: "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:", s)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.expect(t, MakeNamespaceKeyInManager(tc.namespace))
		})
	}
}

func Test_MakeKeyInManager(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		id        string
		expect    func(t *testing.T, s string)
	}{
		{
			name:      "make key in manager",
			namespace: "namespace",
			id:        "foo",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:namespace:foo", s)
			},
		},
		{
			name:      "namespace is empty",
			namespace: "",
			id:        "foo",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager::foo", s)
			},
		},
		{
			name:      "key is empty",
			namespace: "namespace",
			id:        "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:namespace:", s)
			},
		},
		{
			name:      "namespace and key are empty",
			namespace: "",
			id:        "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager::", s)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.expect(t, MakeKeyInManager(tc.namespace, tc.id))
		})
	}
}

func Test_MakeSeedPeerKeyInManager(t *testing.T) {
	tests := []struct {
		name      string
		clusterID uint
		hostname  string
		ip        string
		expect    func(t *testing.T, s string)
	}{
		{
			name:      "make seed peer key in manager",
			clusterID: 1,
			hostname:  "foo",
			ip:        "127.0.0.1",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:seed-peers:1-foo-127.0.0.1", s)
			},
		},
		{
			name:      "hostname is empty",
			clusterID: 1,
			hostname:  "",
			ip:        "127.0.0.1",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:seed-peers:1--127.0.0.1", s)
			},
		},
		{
			name:      "ip is empty",
			clusterID: 1,
			hostname:  "bar",
			ip:        "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:seed-peers:1-bar-", s)
			},
		},
		{
			name:      "hostname and ip are empty",
			clusterID: 1,
			hostname:  "",
			ip:        "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:seed-peers:1--", s)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.expect(t, MakeSeedPeerKeyInManager(tc.clusterID, tc.hostname, tc.ip))
		})
	}
}

func Test_MakeSchedulerKeyInManager(t *testing.T) {
	tests := []struct {
		name      string
		clusterID uint
		hostname  string
		ip        string
		expect    func(t *testing.T, s string)
	}{
		{
			name:      "make scheduler key in manager",
			clusterID: 1,
			hostname:  "bar",
			ip:        "127.0.0.1",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:schedulers:1-bar-127.0.0.1", s)
			},
		},
		{
			name:      "hostname is empty",
			clusterID: 1,
			hostname:  "",
			ip:        "127.0.0.1",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:schedulers:1--127.0.0.1", s)
			},
		},
		{
			name:      "ip is empty",
			clusterID: 1,
			hostname:  "bar",
			ip:        "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:schedulers:1-bar-", s)
			},
		},
		{
			name:      "hostname and ip are empty",
			clusterID: 1,
			hostname:  "",
			ip:        "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:schedulers:1--", s)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.expect(t, MakeSchedulerKeyInManager(tc.clusterID, tc.hostname, tc.ip))
		})
	}
}

func Test_MakeSeedPeersKeyForPeerInManager(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		ip       string
		expect   func(t *testing.T, s string)
	}{
		{
			name:     "make seed peer key for peer in manager",
			hostname: "bar",
			ip:       "127.0.0.1",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:peers:bar-127.0.0.1:seed-peers", s)
			},
		},
		{
			name:     "hostname is empty",
			hostname: "",
			ip:       "127.0.0.1",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:peers:-127.0.0.1:seed-peers", s)
			},
		},
		{
			name:     "ip is empty",
			hostname: "bar",
			ip:       "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:peers:bar-:seed-peers", s)
			},
		},
		{
			name:     "hostname and ip are empty",
			hostname: "",
			ip:       "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:peers:-:seed-peers", s)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.expect(t, MakeSeedPeersKeyForPeerInManager(tc.hostname, tc.ip))
		})
	}
}

func Test_MakeSchedulersKeyForPeerInManager(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		ip       string
		version  string
		expect   func(t *testing.T, s string)
	}{
		{
			name:     "make scheduler key for peer in manager",
			hostname: "bar",
			ip:       "127.0.0.1",
			version:  "0.1.0",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:peers:bar-127.0.0.1-0.1.0:schedulers", s)
			},
		},
		{
			name:     "hostname is empty",
			hostname: "",
			ip:       "127.0.0.1",
			version:  "0.1.0",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:peers:-127.0.0.1-0.1.0:schedulers", s)
			},
		},
		{
			name:     "ip is empty",
			hostname: "bar",
			ip:       "",
			version:  "0.1.0",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:peers:bar--0.1.0:schedulers", s)
			},
		},
		{
			name:     "version is empty",
			hostname: "bar",
			ip:       "127.0.0.1",
			version:  "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:peers:bar-127.0.0.1-:schedulers", s)
			},
		},
		{
			name:     "hostname, ip and version are empty",
			hostname: "",
			ip:       "",
			version:  "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("manager:peers:--:schedulers", s)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.expect(t, MakeSchedulersKeyForPeerInManager(tc.hostname, tc.ip, tc.version))
		})
	}
}

func Test_MakeApplicationsKeyInManager(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("manager:applications", MakeApplicationsKeyInManager())
}

func Test_MakeNamespaceKeyInScheduler(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		expect    func(t *testing.T, s string)
	}{
		{
			name:      "make namespace key in scheduler",
			namespace: "baz",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("scheduler:baz", s)
			},
		},
		{
			name:      "namespace is empty",
			namespace: "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("scheduler:", s)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.expect(t, MakeNamespaceKeyInScheduler(tc.namespace))
		})
	}
}

func Test_MakeKeyInScheduler(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		id        string
		expect    func(t *testing.T, s string)
	}{
		{
			name:      "make key in scheduler",
			namespace: "bas",
			id:        "id",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("scheduler:bas:id", s)
			},
		},
		{
			name:      "namespace is empty",
			namespace: "",
			id:        "id",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("scheduler::id", s)
			},
		},
		{
			name:      "id is empty",
			namespace: "bas",
			id:        "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("scheduler:bas:", s)
			},
		},
		{
			name:      "namespace and id are empty",
			namespace: "",
			id:        "",
			expect: func(t *testing.T, s string) {
				assert := assert.New(t)
				assert.Equal("scheduler::", s)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.expect(t, MakeKeyInScheduler(tc.namespace, tc.id))
		})
	}
}

// withClusterModeEnabled enables the Redis Cluster hash-tag key format for
// the duration of a single test, restoring the previous value afterwards so
// other tests in this package are unaffected.
func withClusterModeEnabled(t *testing.T, enabled bool) {
	t.Helper()

	previous := IsClusterModeEnabled()
	SetClusterModeEnabled(enabled)
	t.Cleanup(func() {
		SetClusterModeEnabled(previous)
	})
}

func Test_MakePersistentHostKeyInScheduler(t *testing.T) {
	withClusterModeEnabled(t, true)
	assert := assert.New(t)
	assert.Equal("scheduler:scheduler-clusters:{1}:persistent-hosts:host-1", MakePersistentHostKeyInScheduler(1, "host-1"))
}

func Test_MakePersistentHostsInScheduler(t *testing.T) {
	withClusterModeEnabled(t, true)
	assert := assert.New(t)
	assert.Equal("scheduler:scheduler-clusters:{1}:persistent-hosts", MakePersistentHostsInScheduler(1))
}

func Test_MakePersistentCacheHostKeyInScheduler(t *testing.T) {
	withClusterModeEnabled(t, true)
	assert := assert.New(t)
	assert.Equal("scheduler:scheduler-clusters:{1}:persistent-cache-hosts:host-1", MakePersistentCacheHostKeyInScheduler(1, "host-1"))
}

func Test_MakePersistentCacheHostsInScheduler(t *testing.T) {
	withClusterModeEnabled(t, true)
	assert := assert.New(t)
	assert.Equal("scheduler:scheduler-clusters:{1}:persistent-cache-hosts", MakePersistentCacheHostsInScheduler(1))
}

// Test_SchedulerKeysClusterModeDisabledByDefault verifies that, without
// calling SetClusterModeEnabled, scheduler key builders keep rendering the
// original, non-hash-tagged key format. This locks in backward
// compatibility for standalone/Sentinel Redis deployments, whose existing
// keys must not change format on upgrade (see dragonflyoss/dragonfly#5050
// review discussion).
func Test_SchedulerKeysClusterModeDisabledByDefault(t *testing.T) {
	assert := assert.New(t)
	assert.False(IsClusterModeEnabled(), "cluster mode must default to disabled")
	assert.Equal("scheduler:scheduler-clusters:1:persistent-hosts:host-1", MakePersistentHostKeyInScheduler(1, "host-1"))
	assert.Equal("scheduler:scheduler-clusters:1:persistent-cache-hosts:host-1", MakePersistentCacheHostKeyInScheduler(1, "host-1"))
	assert.Empty(hashTag(MakePersistentHostKeyInScheduler(1, "host-1")), "legacy key format must not contain a hash tag")
}

func Test_SetClusterModeEnabled(t *testing.T) {
	previous := IsClusterModeEnabled()
	t.Cleanup(func() {
		SetClusterModeEnabled(previous)
	})

	assert := assert.New(t)

	SetClusterModeEnabled(true)
	assert.True(IsClusterModeEnabled())

	SetClusterModeEnabled(false)
	assert.False(IsClusterModeEnabled())
}

func Test_IsClusterClient(t *testing.T) {
	assert := assert.New(t)

	standalone := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer standalone.Close()
	assert.False(IsClusterClient(standalone))

	cluster := redis.NewClusterClient(&redis.ClusterOptions{Addrs: []string{"127.0.0.1:7000"}})
	defer cluster.Close()
	assert.True(IsClusterClient(cluster))
}

// Test_SchedulerKeysShareHashTagForClusterSafety verifies that every group
// of scheduler Redis keys which is read and written together inside a
// single Lua script (see host_manager.go and peer_manager.go in
// scheduler/resource/persistent and scheduler/resource/persistentcache)
// shares the same Redis Cluster hash tag. If the keys in a group did not
// share a hash tag, Redis Cluster would reject the script with a
// CROSSSLOT error whenever the keys land on different nodes, even though
// the same script works fine against standalone/Sentinel Redis (see
// dragonflyoss/dragonfly#5050).
func Test_SchedulerKeysShareHashTagForClusterSafety(t *testing.T) {
	const schedulerClusterID = uint(1)

	tests := []struct {
		name string
		keys func() []string
	}{
		{
			name: "persistent host store/delete script keys",
			keys: func() []string {
				return []string{
					MakePersistentHostKeyInScheduler(schedulerClusterID, "host-1"),
					MakePersistentHostsInScheduler(schedulerClusterID),
				}
			},
		},
		{
			name: "persistent cache host store/delete script keys",
			keys: func() []string {
				return []string{
					MakePersistentCacheHostKeyInScheduler(schedulerClusterID, "host-1"),
					MakePersistentCacheHostsInScheduler(schedulerClusterID),
				}
			},
		},
		{
			name: "persistent peer store/delete script keys",
			keys: func() []string {
				return []string{
					MakePersistentCachePeerKeyForPersistentTaskInScheduler(schedulerClusterID, "peer-1"),
					MakePersistentCachePeersOfPersistentTaskInScheduler(schedulerClusterID, "task-1"),
					MakePersistentPeersOfPersistentTaskInScheduler(schedulerClusterID, "task-1"),
					MakePersistentCachePeersOfPersistentHostInScheduler(schedulerClusterID, "host-1"),
				}
			},
		},
		{
			name: "persistent cache peer store/delete script keys",
			keys: func() []string {
				return []string{
					MakePersistentCachePeerKeyForPersistentCacheTaskInScheduler(schedulerClusterID, "peer-1"),
					MakePersistentCachePeersOfPersistentCacheTaskInScheduler(schedulerClusterID, "task-1"),
					MakePersistentPeersOfPersistentCacheTaskInScheduler(schedulerClusterID, "task-1"),
					MakePersistentCachePeersOfPersistentCacheHostInScheduler(schedulerClusterID, "host-1"),
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withClusterModeEnabled(t, true)
			assert := assert.New(t)

			keys := tc.keys()
			wantTag := fmt.Sprintf("%d", schedulerClusterID)
			for _, key := range keys {
				tag := hashTag(key)
				assert.NotEmptyf(tag, "key %q is missing a Redis Cluster hash tag", key)
				assert.Equalf(wantTag, tag, "key %q does not share the scheduler cluster hash tag %q with the rest of the group %v", key, wantTag, keys)
			}
		})
	}
}

// Test_ScanSchedulerKeys_NonCluster verifies that, for a non-cluster Redis
// client, ScanSchedulerKeys behaves exactly like a plain SCAN call.
func Test_ScanSchedulerKeys_NonCluster(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	assert := assert.New(t)

	mock.ExpectScan(0, "scheduler:scheduler-clusters:1:persistent-tasks:*", 10).SetVal([]string{"task-1", "task-2"}, 0)

	keys, cursor, err := ScanSchedulerKeys(context.Background(), rdb, 0, "scheduler:scheduler-clusters:1:persistent-tasks:*", 10)
	assert.NoError(err)
	assert.Equal(uint64(0), cursor)
	assert.Equal([]string{"task-1", "task-2"}, keys)
	assert.NoError(mock.ExpectationsWereMet())
}

// Test_ScanSchedulerKeys_Cluster verifies that, for a Redis Cluster client,
// ScanSchedulerKeys resolves the master node owning pattern's hash tag
// (via MasterForKey) instead of calling SCAN directly against the cluster
// client, so that SCAN is pinned to a single node. A real cluster isn't
// reachable in this unit test, so MasterForKey is expected to fail against
// the unconfigured client; this still proves ScanSchedulerKeys takes the
// cluster-aware code path (rather than silently falling back to a
// node-local cluster-wide SCAN) for *redis.ClusterClient.
func Test_ScanSchedulerKeys_Cluster(t *testing.T) {
	assert := assert.New(t)

	cluster := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{"127.0.0.1:0"},
	})
	defer cluster.Close()

	_, _, err := ScanSchedulerKeys(context.Background(), cluster, 0, "scheduler:scheduler-clusters:{1}:persistent-tasks:*", 10)
	assert.Error(err, "expected MasterForKey resolution to be attempted (and fail without a reachable cluster)")
}
