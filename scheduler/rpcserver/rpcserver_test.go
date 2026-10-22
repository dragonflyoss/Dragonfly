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

package rpcserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	"d7y.io/dragonfly/v2/scheduler/config"
	configmocks "d7y.io/dragonfly/v2/scheduler/config/mocks"
	jobmocks "d7y.io/dragonfly/v2/scheduler/job/mocks"
	"d7y.io/dragonfly/v2/scheduler/resource/persistent"
	"d7y.io/dragonfly/v2/scheduler/resource/persistentcache"
	"d7y.io/dragonfly/v2/scheduler/resource/standard"
	"d7y.io/dragonfly/v2/scheduler/scheduling/mocks"
)

var (
	mockSchedulerConfig = config.SchedulerConfig{
		RetryLimit:             10,
		RetryBackToSourceLimit: 3,
		RetryInterval:          10 * time.Millisecond,
		BackToSourceCount:      200,
	}
)

func TestRPCServer_New(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, svr *grpc.Server)
	}{
		{
			name: "v1, v2 and health services are registered",
			expect: func(t *testing.T, svr *grpc.Server) {
				assert := assert.New(t)
				services := svr.GetServiceInfo()
				assert.Contains(services, "scheduler.Scheduler")
				assert.Contains(services, "scheduler.v2.Scheduler")
				assert.Contains(services, "grpc.health.v1.Health")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			scheduling := mocks.NewMockScheduling(ctl)
			resource := standard.NewMockResource(ctl)
			persistentResource := persistent.NewMockResource(ctl)
			persistentCacheResource := persistentcache.NewMockResource(ctl)
			dynconfig := configmocks.NewMockDynconfigInterface(ctl)
			job := jobmocks.NewMockJob(ctl)

			svr := New(&config.Config{Scheduler: mockSchedulerConfig, Server: config.ServerConfig{RequestRateLimit: 100}}, resource, persistentResource, persistentCacheResource, scheduling, job, dynconfig, nil)
			tc.expect(t, svr)
		})
	}
}
