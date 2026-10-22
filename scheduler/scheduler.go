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

package scheduler

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	managertypes "d7y.io/dragonfly/v2/manager/types"
	"d7y.io/dragonfly/v2/pkg/dfpath"
	"d7y.io/dragonfly/v2/pkg/gc"
	pkgredis "d7y.io/dragonfly/v2/pkg/redis"
	"d7y.io/dragonfly/v2/pkg/rpc"
	managerclient "d7y.io/dragonfly/v2/pkg/rpc/manager/client"
	"d7y.io/dragonfly/v2/scheduler/announcer"
	"d7y.io/dragonfly/v2/scheduler/config"
	"d7y.io/dragonfly/v2/scheduler/job"
	"d7y.io/dragonfly/v2/scheduler/metrics"
	"d7y.io/dragonfly/v2/scheduler/resource/persistent"
	"d7y.io/dragonfly/v2/scheduler/resource/persistentcache"
	"d7y.io/dragonfly/v2/scheduler/resource/standard"
	"d7y.io/dragonfly/v2/scheduler/rpcserver"
	"d7y.io/dragonfly/v2/scheduler/scheduling"
)

const (
	// gracefulStopTimeout specifies a time limit for
	// grpc server to complete a graceful shutdown.
	gracefulStopTimeout = 10 * time.Minute
)

// Server is the scheduler server.
type Server struct {
	// Server configuration.
	config *config.Config

	// GRPC server.
	grpcServer *grpc.Server

	// Metrics server.
	metricsServer *http.Server

	// Manager client.
	managerClient managerclient.V2

	// Resource interface.
	resource standard.Resource

	// Persistent resource interface.
	persistentResource persistent.Resource

	// Persistent cache resource interface.
	persistentCacheResource persistentcache.Resource

	// Dynamic config.
	dynconfig config.DynconfigInterface

	// Async job.
	job job.Job

	// Announcer interface.
	announcer announcer.Announcer

	// GC service.
	gc gc.GC
}

// New creates a new scheduler server.
func New(ctx context.Context, cfg *config.Config, d dfpath.Dfpath, dynconfigPath string) (*Server, error) {
	s := &Server{config: cfg}

	// Initialize redis client if redis is enabled.
	rdb, err := newRedisClient(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize manager client and announcer when the manager addr is
	// configured. If it is nil, the scheduler runs without a manager and
	// the announcer is disabled.
	if cfg.Manager.Addr != nil {
		managerClient, err := newManagerClient(ctx, cfg)
		if err != nil {
			return nil, err
		}
		s.managerClient = managerClient

		// If job is enabled, add scheduler feature preheat.
		schedulerFeatures := []string{managertypes.SchedulerFeatureSchedule}
		if cfg.Job.Enable && rdb != nil {
			schedulerFeatures = append(schedulerFeatures, managertypes.SchedulerFeaturePreheat)
		}

		announcer, err := announcer.New(cfg, s.managerClient, schedulerFeatures)
		if err != nil {
			return nil, err
		}
		s.announcer = announcer
	}

	// Initialize GC.
	s.gc = gc.New(gc.WithLogger(logger.GCLogger))

	// Initialize dynconfig.
	dynconfig, err := config.NewDynconfig(s.managerClient, dynconfigPath, cfg)
	if err != nil {
		return nil, err
	}
	s.dynconfig = dynconfig

	// Initialize seed peer client transport credentials.
	seedPeerClientTransportCredentials, err := newClientTransportCredentials(cfg.SeedPeer.TLS)
	if err != nil {
		return nil, err
	}

	// Initialize resource.
	resource, err := standard.New(cfg, s.gc, seedPeerClientTransportCredentials)
	if err != nil {
		return nil, err
	}
	s.resource = resource

	// Initialize persistent resource and persistent cache resource if redis
	// is enabled.
	if rdb != nil {
		peerClientTransportCredentials, err := newClientTransportCredentials(cfg.Peer.TLS)
		if err != nil {
			return nil, err
		}

		s.persistentResource, err = persistent.New(cfg, s.gc, rdb, peerClientTransportCredentials)
		if err != nil {
			return nil, err
		}

		s.persistentCacheResource, err = persistentcache.New(cfg, s.gc, rdb, peerClientTransportCredentials)
		if err != nil {
			return nil, err
		}
	}

	// Initialize job service if job is enabled and redis is enabled.
	if cfg.Job.Enable && rdb != nil {
		s.job, err = job.New(cfg, resource,
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
			grpc.WithTransportCredentials(seedPeerClientTransportCredentials))
		if err != nil {
			return nil, err
		}
	}

	// Initialize scheduling.
	scheduling := scheduling.New(&cfg.Scheduler, s.persistentResource, s.persistentCacheResource, dynconfig, d.PluginDir())

	// Initialize server transport credentials of scheduler grpc server.
	serverTransportCredentials := rpc.NewInsecureCredentials()
	if cfg.Server.TLS != nil {
		serverTransportCredentials, err = rpc.NewServerCredentials(cfg.Server.TLS.CACert, cfg.Server.TLS.Cert, cfg.Server.TLS.Key)
		if err != nil {
			return nil, err
		}
	}
	s.grpcServer = rpcserver.New(cfg, resource, s.persistentResource, s.persistentCacheResource, scheduling, s.job, dynconfig, grpc.Creds(serverTransportCredentials))

	// Initialize metrics server if metrics is enabled.
	if cfg.Metrics.Enable {
		s.metricsServer = metrics.New(&cfg.Metrics, s.grpcServer)
	}

	return s, nil
}

// Serve starts the scheduler server.
func (s *Server) Serve() error {
	// Serve GC.
	s.gc.Start(context.Background())
	logger.Info("gc start successfully")

	// Serve Job.
	if s.job != nil {
		s.job.Serve()
		logger.Info("job start successfully")
	}

	// Started metrics server.
	if s.metricsServer != nil {
		go func() {
			logger.Infof("started metrics server at %s", s.metricsServer.Addr)
			if err := s.metricsServer.ListenAndServe(); err != nil {
				if err == http.ErrServerClosed {
					return
				}

				logger.Fatalf("metrics server closed unexpect: %s", err.Error())
			}
		}()
	}

	// Serve announcer.
	if s.announcer != nil {
		go func() {
			s.announcer.Serve()
			logger.Info("announcer start successfully")
		}()
	}

	// Serve resource.
	go func() {
		if err := s.resource.Serve(); err != nil {
			logger.Fatalf("resource start failed: %s", err.Error())
		}

		logger.Info("resource start successfully")
	}()

	listener, err := net.Listen("tcp", net.JoinHostPort(s.config.Server.ListenIP.String(), strconv.Itoa(s.config.Server.Port)))
	if err != nil {
		logger.Fatalf("net listener failed to start: %s", err.Error())
	}
	defer listener.Close()

	// Started GRPC server.
	logger.Infof("started grpc server at %s://%s", listener.Addr().Network(), listener.Addr().String())
	if err := s.grpcServer.Serve(listener); err != nil {
		logger.Errorf("stoped grpc server: %s", err.Error())
		return err
	}

	return nil
}

// Stop stops the scheduler server.
func (s *Server) Stop() {
	// Stop resource.
	s.resource.Stop()
	logger.Info("stop resource closed")

	// Stop GC.
	s.gc.Stop()
	logger.Info("gc closed")

	// Stop metrics server.
	if s.metricsServer != nil {
		if err := s.metricsServer.Shutdown(context.Background()); err != nil {
			logger.Errorf("metrics server failed to stop: %s", err.Error())
		} else {
			logger.Info("metrics server closed under request")
		}
	}

	// Stop announcer.
	if s.announcer != nil {
		s.announcer.Stop()
		logger.Info("stop announcer closed")
	}

	// Stop manager client.
	if s.managerClient != nil {
		if err := s.managerClient.Close(); err != nil {
			logger.Errorf("manager client failed to stop: %s", err.Error())
		} else {
			logger.Info("manager client closed")
		}
	}

	// Stop GRPC server.
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		logger.Info("grpc server closed under request")
		close(stopped)
	}()

	t := time.NewTimer(gracefulStopTimeout)
	select {
	case <-t.C:
		s.grpcServer.Stop()
	case <-stopped:
		t.Stop()
	}
}

// newManagerClient returns a new manager client.
func newManagerClient(ctx context.Context, cfg *config.Config) (managerclient.V2, error) {
	clientTransportCredentials, err := newClientTransportCredentials(cfg.Manager.TLS)
	if err != nil {
		return nil, err
	}

	return managerclient.GetV2ByAddr(ctx, *cfg.Manager.Addr,
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithTransportCredentials(clientTransportCredentials))
}

// newRedisClient returns a new redis client. If redis is not enabled, it
// returns nil.
func newRedisClient(cfg *config.Config) (redis.UniversalClient, error) {
	if !pkgredis.IsEnabled(cfg.Database.Redis.Addrs) {
		return nil, nil
	}

	redisOpts := &redis.UniversalOptions{
		Addrs:            cfg.Database.Redis.Addrs,
		MasterName:       cfg.Database.Redis.MasterName,
		Username:         cfg.Database.Redis.Username,
		Password:         cfg.Database.Redis.Password,
		SentinelUsername: cfg.Database.Redis.SentinelUsername,
		SentinelPassword: cfg.Database.Redis.SentinelPassword,
		PoolSize:         cfg.Database.Redis.PoolSize,
		PoolTimeout:      cfg.Database.Redis.PoolTimeout,
	}

	if redisTLS := cfg.Database.Redis.TLS; redisTLS != nil {
		tlsCfg, err := pkgredis.NewTLSClientConfig(redisTLS.CACert, redisTLS.Cert, redisTLS.Key, redisTLS.InsecureSkipVerify)
		if err != nil {
			return nil, err
		}

		redisOpts.TLSConfig = tlsCfg
	}

	return pkgredis.NewRedis(redisOpts)
}

// newClientTransportCredentials returns the client transport credentials for
// the given TLS configuration. If it is nil, it returns insecure transport
// credentials.
func newClientTransportCredentials(tlsConfig *config.GRPCTLSClientConfig) (credentials.TransportCredentials, error) {
	if tlsConfig == nil {
		return rpc.NewInsecureCredentials(), nil
	}

	return rpc.NewClientCredentials(tlsConfig.CACert, tlsConfig.Cert, tlsConfig.Key)
}
