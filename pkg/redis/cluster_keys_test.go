/*
 *     Copyright 2026 The Dragonfly Authors
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
	"strings"
	"testing"

	goredis "github.com/redis/go-redis/v9"
)

func TestSchedulerClusterHashTags(t *testing.T) {
	keys := []string{
		MakePersistentHostKeyInScheduler(1, "host"), MakePersistentHostsInScheduler(1),
		MakePersistentTaskKeyInScheduler(1, "task"), MakePersistentTasksInScheduler(1),
		MakePersistentCachePeerKeyForPersistentTaskInScheduler(1, "peer"),
		MakePersistentCachePeersOfPersistentTaskInScheduler(1, "task"),
		MakePersistentPeersOfPersistentTaskInScheduler(1, "task"),
		MakePersistentCachePeersOfPersistentHostInScheduler(1, "host"),
		MakePersistentCachePeersForPersistentTaskInScheduler(1),
		MakePersistentCacheHostKeyInScheduler(1, "host"), MakePersistentCacheHostsInScheduler(1),
		MakePersistentCacheTaskKeyInScheduler(1, "task"), MakePersistentCacheTasksInScheduler(1),
		MakePersistentCachePeerKeyForPersistentCacheTaskInScheduler(1, "peer"),
		MakePersistentCachePeersOfPersistentCacheTaskInScheduler(1, "task"),
		MakePersistentPeersOfPersistentCacheTaskInScheduler(1, "task"),
		MakePersistentCachePeersOfPersistentCacheHostInScheduler(1, "host"),
		MakePersistentCachePeersForPersistentCacheTaskInScheduler(1),
	}
	for _, key := range keys {
		if !strings.HasPrefix(key, "scheduler:scheduler-clusters:{1}:") {
			t.Fatalf("Key does not share the cluster hash tag: %s", key)
		}
	}
	if !strings.Contains(MakePersistentHostKeyInScheduler(2, "host"), "{2}") {
		t.Fatal("Different scheduler clusters must have different tags")
	}
	if MakeKeyInScheduler("other", "id") != "scheduler:other:id" {
		t.Fatal("Unrelated namespaces must remain unchanged")
	}
}

func TestRejectClusterDatabase(t *testing.T) {
	if _, err := NewRedis(&goredis.UniversalOptions{Addrs: []string{"host1:6379", "host2:6379"}, DB: 10}); err == nil {
		t.Fatal("Nonzero Cluster database accepted")
	}
}
