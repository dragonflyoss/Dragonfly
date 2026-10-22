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

package job

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/dragonflyoss/machinery/v1"
	machineryv1config "github.com/dragonflyoss/machinery/v1/config"
	machineryv1log "github.com/dragonflyoss/machinery/v1/log"
	machineryv1tasks "github.com/dragonflyoss/machinery/v1/tasks"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"golang.org/x/sync/errgroup"

	logger "d7y.io/dragonfly/v2/internal/dflog"
)

// tracer is a global tracer for job.
var tracer = otel.Tracer("job")

type Config struct {
	Addrs            []string
	MasterName       string
	Username         string
	Password         string
	SentinelUsername string
	SentinelPassword string
	BrokerDB         int
	BackendDB        int
	TLSConfig        *tls.Config
}

type Job struct {
	Server *machinery.Server
	Worker *machinery.Worker
	Queue  Queue
}

func New(cfg *Config, queue Queue) (*Job, error) {
	// Set logger
	machineryv1log.Set(&MachineryLogger{})

	if err := ping(&redis.UniversalOptions{
		Addrs:            cfg.Addrs,
		MasterName:       cfg.MasterName,
		Username:         cfg.Username,
		Password:         cfg.Password,
		SentinelUsername: cfg.SentinelUsername,
		SentinelPassword: cfg.SentinelPassword,
		DB:               cfg.BackendDB,
		TLSConfig:        cfg.TLSConfig,
	}); err != nil {
		return nil, err
	}

	var broker string
	if cfg.Username != "" {
		broker = fmt.Sprintf("redis://%s:%s@%s/%d", url.QueryEscape(cfg.Username), url.QueryEscape(cfg.Password), strings.Join(cfg.Addrs, ","), cfg.BrokerDB)
	} else {
		broker = fmt.Sprintf("redis://%s@%s/%d", url.QueryEscape(cfg.Password), strings.Join(cfg.Addrs, ","), cfg.BrokerDB)
	}

	var backend string
	if cfg.Username != "" {
		backend = fmt.Sprintf("redis://%s:%s@%s/%d", url.QueryEscape(cfg.Username), url.QueryEscape(cfg.Password), strings.Join(cfg.Addrs, ","), cfg.BackendDB)
	} else {
		backend = fmt.Sprintf("redis://%s@%s/%d", url.QueryEscape(cfg.Password), strings.Join(cfg.Addrs, ","), cfg.BackendDB)
	}

	server, err := machinery.NewServer(&machineryv1config.Config{
		Broker:          broker,
		DefaultQueue:    queue.String(),
		ResultBackend:   backend,
		ResultsExpireIn: DefaultResultsExpireIn,
		TLSConfig:       cfg.TLSConfig,
		Redis: &machineryv1config.RedisConfig{
			MasterName:             cfg.MasterName,
			MaxIdle:                DefaultRedisMaxIdle,
			MaxActive:              DefaultRedisMaxActive,
			IdleTimeout:            DefaultRedisIdleTimeout,
			ReadTimeout:            DefaultRedisReadTimeout,
			WriteTimeout:           DefaultRedisWriteTimeout,
			ConnectTimeout:         DefaultRedisConnectTimeout,
			NormalTasksPollPeriod:  DefaultRedisNormalTasksPollPeriod,
			DelayedTasksPollPeriod: DefaultRedisDelayedTasksPollPeriod,
		},
	})
	if err != nil {
		return nil, err
	}

	return &Job{
		Server: server,
		Queue:  queue,
	}, nil
}

func ping(options *redis.UniversalOptions) error {
	return redis.NewUniversalClient(options).Ping(context.Background()).Err()
}

func (t *Job) RegisterJob(namedJobFuncs map[string]any) error {
	return t.Server.RegisterTasks(namedJobFuncs)
}

func (t *Job) LaunchWorker(consumerTag string, concurrency int) error {
	t.Worker = t.Server.NewWorker(consumerTag, concurrency)
	return t.Worker.Launch()
}

type GroupJobState struct {
	GroupUUID string     `json:"group_uuid"`
	State     string     `json:"state"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	JobStates []jobState `json:"job_states"`
}

type jobState struct {
	TaskUUID  string    `json:"task_uuid"`
	TaskName  string    `json:"task_name"`
	State     string    `json:"state"`
	Results   []any     `json:"results"`
	Error     string    `json:"error"`
	CreatedAt time.Time `json:"created_at"`
	TTL       int64     `json:"ttl"`
}

func (t *Job) GetGroupJobState(name string, groupUUID string) (*GroupJobState, error) {
	taskStates, err := t.Server.GetBackend().GroupTaskStates(groupUUID, 0)
	if err != nil {
		return nil, err
	}

	if len(taskStates) == 0 {
		return nil, errors.New("empty group")
	}

	var mu sync.Mutex
	jobStates := make([]jobState, 0, len(taskStates))
	eg, _ := errgroup.WithContext(context.Background())
	eg.SetLimit(GroupJobStateConcurrencyLimit)
	for _, taskState := range taskStates {
		eg.Go(func() error {
			var results []any
			for _, result := range taskState.Results {
				switch name {
				case PreheatJob:
					var resp PreheatResponse
					if err := UnmarshalTaskResult(result.Value, &resp); err != nil {
						return err
					}

					results = append(results, resp)
				case GetTaskJob:
					var resp GetTaskResponse
					if err := UnmarshalTaskResult(result.Value, &resp); err != nil {
						return err
					}

					results = append(results, resp)
				case DeleteTaskJob:
					var resp DeleteTaskResponse
					if err := UnmarshalTaskResult(result.Value, &resp); err != nil {
						return err
					}

					results = append(results, resp)
				default:
					return errors.New("unsupported unmarshal task result")
				}
			}

			mu.Lock()
			jobStates = append(jobStates, jobState{
				TaskUUID:  taskState.TaskUUID,
				TaskName:  taskState.TaskName,
				State:     taskState.State,
				Results:   results,
				Error:     taskState.Error,
				CreatedAt: taskState.CreatedAt,
				TTL:       taskState.TTL,
			})
			mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		logger.WithGroupUUID(groupUUID).Errorf("failed to get group job state: %v", err)
		return nil, err
	}

	for _, taskState := range taskStates {
		if taskState.IsFailure() {
			logger.WithGroupAndTaskUUID(groupUUID, taskState.TaskUUID).Errorf("task is failed: %#v", taskState)
			return constructGroupJobState(groupUUID, machineryv1tasks.StateFailure, taskState.CreatedAt, jobStates), nil
		}
	}

	for _, taskState := range taskStates {
		if !taskState.IsSuccess() {
			logger.WithGroupAndTaskUUID(groupUUID, taskState.TaskUUID).Infof("task is not succeeded: %#v", taskState)
			return constructGroupJobState(groupUUID, machineryv1tasks.StatePending, taskState.CreatedAt, jobStates), nil
		}
	}

	return constructGroupJobState(groupUUID, machineryv1tasks.StateSuccess, taskStates[0].CreatedAt, jobStates), nil
}

func constructGroupJobState(groupUUID, state string, createdAt time.Time, jobStates []jobState) *GroupJobState {
	return &GroupJobState{
		GroupUUID: groupUUID,
		State:     state,
		CreatedAt: createdAt,
		UpdatedAt: time.Now(),
		JobStates: jobStates,
	}
}

func MarshalResponse(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func MarshalRequest(v any) ([]machineryv1tasks.Arg, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return []machineryv1tasks.Arg{{
		Type:  "string",
		Value: string(b),
	}}, nil
}

func UnmarshalTaskResult(data any, v any) error {
	s, ok := data.(string)
	if !ok {
		return errors.New("invalid task result")
	}

	return json.Unmarshal([]byte(s), v)
}

func UnmarshalResponse(data []reflect.Value, v any) error {
	if len(data) == 0 {
		return errors.New("empty data is not specified")
	}

	if err := json.Unmarshal([]byte(data[0].String()), v); err != nil {
		return err
	}

	return nil
}

func UnmarshalRequest(data string, v any) error {
	return json.Unmarshal([]byte(data), v)
}
