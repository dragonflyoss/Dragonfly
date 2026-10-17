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
	"google.golang.org/grpc"

	"d7y.io/dragonfly/v2/pkg/rpc/scheduler/server"
	"d7y.io/dragonfly/v2/scheduler/config"
	"d7y.io/dragonfly/v2/scheduler/job"
	"d7y.io/dragonfly/v2/scheduler/resource/persistent"
	"d7y.io/dragonfly/v2/scheduler/resource/persistentcache"
	"d7y.io/dragonfly/v2/scheduler/resource/standard"
	"d7y.io/dragonfly/v2/scheduler/scheduling"
)

// New returns a new scheduler server from the given options.
func New(
	cfg *config.Config,
	resource standard.Resource,
	persistentResource persistent.Resource,
	persistentCacheResource persistentcache.Resource,
	scheduling scheduling.Scheduling,
	job job.Job,
	dynconfig config.DynconfigInterface,
	peerDialOptions []grpc.DialOption,
	opts ...grpc.ServerOption,
) *grpc.Server {
	return server.New(
		newSchedulerServerV1(cfg, resource, scheduling, dynconfig),
		newSchedulerServerV2(cfg, resource, persistentResource, persistentCacheResource, scheduling, job, dynconfig, peerDialOptions),
		cfg.Server.RequestRateLimit,
		opts...)
}
