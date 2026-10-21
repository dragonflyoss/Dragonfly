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

//go:generate mockgen -destination task_manager_mock.go -source task_manager.go -package persistentcache

package persistentcache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	pkgredis "d7y.io/dragonfly/v2/pkg/redis"
	"d7y.io/dragonfly/v2/scheduler/config"
)

// TaskManager is the interface used for persistent cache task manager.
type TaskManager interface {
	// Load returns persistent cache task by a key.
	Load(context.Context, string) (*Task, bool)

	// LoadCurrentReplicaCount returns current replica count of the persistent cache task.
	LoadCurrentReplicaCount(context.Context, string) (uint64, error)

	// LoadCurrentPersistentReplicaCount returns current persistent replica count of the persistent cache task.
	LoadCurrentPersistentReplicaCount(context.Context, string) (uint64, error)

	// Store sets persistent cache task.
	Store(context.Context, *Task) error

	// Delete deletes persistent cache task by a key.
	Delete(context.Context, string) error

	// LoadAll returns all persistent cache tasks.
	LoadAll(context.Context) ([]*Task, error)
}

// taskManager contains content for persistent cache task manager.
type taskManager struct {
	// Config is scheduler config.
	config *config.Config

	// Redis universal client interface.
	rdb redis.UniversalClient
}

// New persistent cache task manager interface.
func newTaskManager(cfg *config.Config, rdb redis.UniversalClient) TaskManager {
	return &taskManager{config: cfg, rdb: rdb}
}

// Load returns persistent cache task by a key.
func (t *taskManager) Load(ctx context.Context, taskID string) (*Task, bool) {
	log := logger.WithTaskID(taskID)
	rawTask, err := t.rdb.HGetAll(ctx, pkgredis.MakePersistentCacheTaskKeyInScheduler(t.config.Manager.SchedulerClusterID, taskID)).Result()
	if err != nil {
		log.Errorf("getting task failed from redis: %v", err)
		return nil, false
	}

	if len(rawTask) == 0 {
		return nil, false
	}

	// Set integer fields from raw task.
	persistentReplicaCount, err := strconv.ParseUint(rawTask["persistent_replica_count"], 10, 64)
	if err != nil {
		log.Errorf("parsing persistent replica count failed: %v", err)
		return nil, false
	}

	pieceLength, err := strconv.ParseUint(rawTask["piece_length"], 10, 64)
	if err != nil {
		log.Errorf("parsing piece length failed: %v", err)
		return nil, false
	}

	contentLength, err := strconv.ParseUint(rawTask["content_length"], 10, 64)
	if err != nil {
		log.Errorf("parsing content length failed: %v", err)
		return nil, false
	}

	totalPieceCount, err := strconv.ParseUint(rawTask["total_piece_count"], 10, 32)
	if err != nil {
		log.Errorf("parsing total piece count failed: %v", err)
		return nil, false
	}

	// Set time fields from raw task.
	ttl, err := strconv.ParseInt(rawTask["ttl"], 10, 64)
	if err != nil {
		log.Errorf("parsing ttl failed: %v", err)
		return nil, false
	}

	createdAt, err := time.Parse(time.RFC3339, rawTask["created_at"])
	if err != nil {
		log.Errorf("parsing created at failed: %v", err)
		return nil, false
	}

	updatedAt, err := time.Parse(time.RFC3339, rawTask["updated_at"])
	if err != nil {
		log.Errorf("parsing updated at failed: %v", err)
		return nil, false
	}

	return NewTask(
		rawTask["id"],
		rawTask["tag"],
		rawTask["application"],
		rawTask["state"],
		persistentReplicaCount,
		pieceLength,
		contentLength,
		uint32(totalPieceCount),
		time.Duration(ttl),
		createdAt,
		updatedAt,
		logger.WithTaskID(rawTask["id"]),
	), true
}

// LoadCurrentReplicaCount returns current replica count of the persistent cache task.
func (t *taskManager) LoadCurrentReplicaCount(ctx context.Context, taskID string) (uint64, error) {
	count, err := t.rdb.SCard(ctx, pkgredis.MakePersistentCachePeersOfPersistentCacheTaskInScheduler(t.config.Manager.SchedulerClusterID, taskID)).Result()
	return uint64(count), err
}

// LoadCurrentPersistentReplicaCount returns current persistent replica count of the persistent cache task.
func (t *taskManager) LoadCurrentPersistentReplicaCount(ctx context.Context, taskID string) (uint64, error) {
	count, err := t.rdb.SCard(ctx, pkgredis.MakePersistentPeersOfPersistentCacheTaskInScheduler(t.config.Manager.SchedulerClusterID, taskID)).Result()
	return uint64(count), err
}

// Store sets persistent cache task.
func (t *taskManager) Store(ctx context.Context, task *Task) error {
	// Calculate remaining TTL in seconds.
	ttl := task.TTL - time.Since(task.CreatedAt)
	remainingTTLSeconds := int64(ttl.Seconds())

	// Define the Lua script as a string.
	const storeTaskScript = `
-- Extract keys
local task_key = KEYS[1]  -- Key for the task hash

-- Extract arguments
local task_id = ARGV[1]
local persistent_replica_count = ARGV[2]
local tag = ARGV[3]
local application = ARGV[4]
local piece_length = ARGV[5]
local content_length = ARGV[6]
local total_piece_count = ARGV[7]
local state = ARGV[8]
local created_at = ARGV[9]
local updated_at = ARGV[10]
local ttl = tonumber(ARGV[11])
local ttl_seconds = tonumber(ARGV[12])

-- Perform HSET operation to store task details
redis.call("HSET", task_key,
    "id", task_id,
    "persistent_replica_count", persistent_replica_count,
    "tag", tag,
    "application", application,
    "piece_length", piece_length,
    "content_length", content_length,
    "total_piece_count", total_piece_count,
    "state", state,
    "ttl", ttl,
    "created_at", created_at,
    "updated_at", updated_at)

-- Perform EXPIRE operation if TTL is still valid
redis.call("EXPIRE", task_key, ttl_seconds)

return true
`

	// Create a new Redis script.
	script := redis.NewScript(storeTaskScript)

	// Prepare keys.
	keys := []string{
		pkgredis.MakePersistentCacheTaskKeyInScheduler(t.config.Manager.SchedulerClusterID, task.ID),
	}

	// Prepare arguments.
	args := []any{
		task.ID,
		task.PersistentReplicaCount,
		task.Tag,
		task.Application,
		task.PieceLength,
		task.ContentLength,
		task.TotalPieceCount,
		task.FSM.Current(),
		task.CreatedAt.Format(time.RFC3339),
		task.UpdatedAt.Format(time.RFC3339),
		task.TTL.Nanoseconds(),
		remainingTTLSeconds,
	}

	// Execute the script.
	if err := script.Run(ctx, t.rdb, keys, args...).Err(); err != nil {
		task.Log.Errorf("store task failed: %v", err)
		return err
	}

	return nil
}

// Delete deletes persistent cache task by a key.
func (t *taskManager) Delete(ctx context.Context, taskID string) error {
	_, err := t.rdb.Del(ctx, pkgredis.MakePersistentCacheTaskKeyInScheduler(t.config.Manager.SchedulerClusterID, taskID)).Result()
	return err
}

// LoadAll returns all persistent cache tasks.
func (t *taskManager) LoadAll(ctx context.Context) ([]*Task, error) {
	var (
		tasks  []*Task
		cursor uint64
	)

	for {
		var (
			taskKeys []string
			err      error
		)

		// Example: If {prefix} is "scheduler:scheduler-clusters:1:persistent-cache-tasks:".
		//
		// Keys include:
		// - "{prefix}{taskID}:persistent-cache-peers"
		// - "{prefix}{taskID}:persistent-peers"
		// - "{prefix}{taskID}"
		//
		// Purpose: Scan all keys starting with {prefix}.
		prefix := fmt.Sprintf("%s:", pkgredis.MakePersistentCacheTasksInScheduler(t.config.Manager.SchedulerClusterID))
		taskKeys, cursor, err = pkgredis.ScanSchedulerKeys(ctx, t.rdb, cursor, fmt.Sprintf("%s*", prefix), 10)
		if err != nil {
			logger.Error("scan tasks failed")
			return nil, err
		}

		taskIDs := make(map[string]struct{})
		for _, taskKey := range taskKeys {
			// If context is done, return error.
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			// Remove prefix from task key.
			suffix := strings.TrimPrefix(taskKey, prefix)
			if suffix == "" {
				logger.Error("invalid task key")
				continue
			}

			// Suffix is a non-empty string.
			//
			// Keys include:
			// - "{taskID}:persistent-cache-peers"
			// - "{taskID}:persistent-peers"
			// - "{taskID}"
			//
			// Purpose: Extract taskID from suffix and ensure no duplicate taskIDs.
			taskID := strings.Split(suffix, ":")[0]
			if _, ok := taskIDs[taskID]; ok {
				continue
			}
			taskIDs[taskID] = struct{}{}

			task, loaded := t.Load(ctx, taskID)
			if !loaded {
				logger.WithTaskID(taskID).Error("load task failed")
				continue
			}

			tasks = append(tasks, task)
		}

		if cursor == 0 {
			break
		}
	}

	return tasks, nil
}
