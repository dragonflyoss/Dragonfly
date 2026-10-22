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
	"time"

	"go.opentelemetry.io/otel"
	"gorm.io/gorm"

	internaljob "d7y.io/dragonfly/v2/internal/job"
	"d7y.io/dragonfly/v2/manager/config"
	"d7y.io/dragonfly/v2/manager/models"
	nettls "d7y.io/dragonfly/v2/pkg/net/tls"
	pkgredis "d7y.io/dragonfly/v2/pkg/redis"
)

// DefaultTaskPollingInterval is the default interval for polling task.
const DefaultTaskPollingInterval = 10 * time.Second

// tracer is a global tracer for job.
var tracer = otel.Tracer("manager")

// Job is an implementation of job.
type Job struct {
	*internaljob.Job
	Preheat
	SyncPeers
	Task
}

// New returns a new Job.
func New(cfg *config.Config, gdb *gorm.DB) (*Job, error) {
	redisConfig := &internaljob.Config{
		Addrs:            cfg.Database.Redis.Addrs,
		MasterName:       cfg.Database.Redis.MasterName,
		Username:         cfg.Database.Redis.Username,
		Password:         cfg.Database.Redis.Password,
		SentinelUsername: cfg.Database.Redis.SentinelUsername,
		SentinelPassword: cfg.Database.Redis.SentinelPassword,
		BrokerDB:         cfg.Database.Redis.BrokerDB,
		BackendDB:        cfg.Database.Redis.BackendDB,
	}

	if redisTLS := cfg.Database.Redis.TLS; redisTLS != nil {
		tlsCfg, err := pkgredis.NewTLSClientConfig(redisTLS.CACert, redisTLS.Cert, redisTLS.Key, redisTLS.InsecureSkipVerify)
		if err != nil {
			return nil, err
		}

		redisConfig.TLSConfig = tlsCfg
	}

	j, err := internaljob.New(redisConfig, internaljob.GlobalQueue)
	if err != nil {
		return nil, err
	}

	certPool, err := nettls.PEMToCertPool(cfg.Job.Preheat.TLS.CACert.ToBytes())
	if err != nil {
		return nil, err
	}
	preheat := newPreheat(j, internaljob.NewImage(), certPool, cfg.Job.Preheat.TLS.InsecureSkipVerify)

	syncPeers, err := newSyncPeers(cfg, j, gdb)
	if err != nil {
		return nil, err
	}

	return &Job{
		Job:       j,
		Preheat:   preheat,
		SyncPeers: syncPeers,
		Task:      newTask(j),
	}, nil
}

// Serve starts the job server.
func (j *Job) Serve() {
	go j.SyncPeers.Serve()
}

// Stop stops the job server.
func (j *Job) Stop() {
	j.SyncPeers.Stop()
}

// getSchedulerQueues gets scheduler queues.
func getSchedulerQueues(schedulers []models.Scheduler) ([]internaljob.Queue, error) {
	var queues []internaljob.Queue
	for _, scheduler := range schedulers {
		queue, err := internaljob.GetSchedulerQueue(scheduler.SchedulerClusterID, scheduler.Hostname, scheduler.IP)
		if err != nil {
			return nil, err
		}

		queues = append(queues, queue)
	}

	return queues, nil
}

// getSchedulerQueue gets scheduler queue.
func getSchedulerQueue(scheduler models.Scheduler) (internaljob.Queue, error) {
	return internaljob.GetSchedulerQueue(scheduler.SchedulerClusterID, scheduler.Hostname, scheduler.IP)
}
