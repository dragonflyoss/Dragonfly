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

//go:generate mockgen -destination mocks/job_mock.go -source job.go -package mocks

package job

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"sort"
	"strconv"
	"sync"

	"github.com/dragonflyoss/machinery/v1"
	"github.com/go-playground/validator/v10"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	cdnsystemv1 "d7y.io/api/v2/pkg/apis/cdnsystem/v1"
	commonv1 "d7y.io/api/v2/pkg/apis/common/v1"
	commonv2 "d7y.io/api/v2/pkg/apis/common/v2"
	dfdaemonv2 "d7y.io/api/v2/pkg/apis/dfdaemon/v2"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	internaljob "d7y.io/dragonfly/v2/internal/job"
	managertypes "d7y.io/dragonfly/v2/manager/types"
	"d7y.io/dragonfly/v2/pkg/dfnet"
	"d7y.io/dragonfly/v2/pkg/idgen"
	cndsystemclient "d7y.io/dragonfly/v2/pkg/rpc/cdnsystem/client"
	"d7y.io/dragonfly/v2/scheduler/config"
	resource "d7y.io/dragonfly/v2/scheduler/resource/standard"
)

// Job is an interface for job.
type Job interface {
	// Serve starts the job.
	Serve()

	// GetTask retrieves task information from all hosts in the cluster.
	GetTask(context.Context, *internaljob.GetTaskRequest, *logger.SugaredLoggerOnWith) (*internaljob.GetTaskResponse, error)

	// ListTaskEntries lists all task entries.
	ListTaskEntries(context.Context, *internaljob.ListTaskEntriesRequest, *logger.SugaredLoggerOnWith) (*internaljob.ListTaskEntriesResponse, error)

	// PreheatSinglePeer preheats job by single seed peer, scheduler will trigger seed peer to download task.
	PreheatSingleSeedPeer(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) (*internaljob.PreheatResponse, error)

	// PreheatAllSeedPeers preheats job by all peer seed peers, only supported by v2 protocol. Scheduler will trigger all seed peers to download task.
	// If all the seed peers download task failed, return error. If some of the seed peers download task failed, return success tasks and failure tasks.
	// Notify the client that the preheat is successful.
	PreheatAllSeedPeers(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) (*internaljob.PreheatResponse, error)

	// PreheatAllPeers preheats job by all peers, only supported by v2 protocol. Scheduler will trigger all peers to download task.
	// If all the peers download task failed, return error. If some of the peers download task failed, return success tasks and
	// failure tasks. Notify the client that the preheat is successful.
	PreheatAllPeers(context.Context, *internaljob.PreheatRequest, *logger.SugaredLoggerOnWith) (*internaljob.PreheatResponse, error)
}

// job is an implementation of Job.
type job struct {
	globalJob    *internaljob.Job
	schedulerJob *internaljob.Job
	localJob     *internaljob.Job
	resource     resource.Resource
	config       *config.Config
	dialOptions  []grpc.DialOption
}

// New creates a new Job.
func New(cfg *config.Config, resource resource.Resource, dialOptions ...grpc.DialOption) (Job, error) {
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

	globalJob, err := internaljob.New(redisConfig, internaljob.GlobalQueue)
	if err != nil {
		logger.Errorf("create global job queue error: %s", err.Error())
		return nil, err
	}
	logger.Infof("create global job queue: %v", globalJob)

	schedulerJob, err := internaljob.New(redisConfig, internaljob.SchedulersQueue)
	if err != nil {
		logger.Errorf("create scheduler job queue error: %s", err.Error())
		return nil, err
	}
	logger.Infof("create scheduler job queue: %v", schedulerJob)

	localQueue, err := internaljob.GetSchedulerQueue(cfg.Manager.SchedulerClusterID, cfg.Server.Host, cfg.Server.AdvertiseIP.String())
	if err != nil {
		logger.Errorf("get local job queue name error: %s", err.Error())
		return nil, err
	}

	localJob, err := internaljob.New(redisConfig, localQueue)
	if err != nil {
		logger.Errorf("create local job queue error: %s", err.Error())
		return nil, err
	}
	logger.Infof("create local job queue: %v", localQueue)

	t := &job{
		globalJob:    globalJob,
		schedulerJob: schedulerJob,
		localJob:     localJob,
		resource:     resource,
		config:       cfg,
		dialOptions:  dialOptions,
	}

	namedJobFuncs := map[string]any{
		internaljob.PreheatJob:    t.preheat,
		internaljob.SyncPeersJob:  t.syncPeers,
		internaljob.GetTaskJob:    t.getTask,
		internaljob.DeleteTaskJob: t.deleteTask,
	}

	if err := localJob.RegisterJob(namedJobFuncs); err != nil {
		logger.Errorf("register preheat job to local queue error: %s", err.Error())
		return nil, err
	}

	return t, nil
}

// Serve starts the job.
func (j *job) Serve() {
	go func() {
		logger.Infof("ready to launch %d worker(s) on global queue", j.config.Job.GlobalWorkerNum)
		if err := j.globalJob.LaunchWorker("global_worker", int(j.config.Job.GlobalWorkerNum)); err != nil {
			if !errors.Is(err, machinery.ErrWorkerQuitGracefully) {
				logger.Fatalf("global queue worker error: %s", err.Error())
			}
		}
	}()

	go func() {
		logger.Infof("ready to launch %d worker(s) on scheduler queue", j.config.Job.SchedulerWorkerNum)
		if err := j.schedulerJob.LaunchWorker("scheduler_worker", int(j.config.Job.SchedulerWorkerNum)); err != nil {
			if !errors.Is(err, machinery.ErrWorkerQuitGracefully) {
				logger.Fatalf("scheduler queue worker error: %s", err.Error())
			}
		}
	}()

	go func() {
		logger.Infof("ready to launch %d worker(s) on local queue", j.config.Job.LocalWorkerNum)
		if err := j.localJob.LaunchWorker("local_worker", int(j.config.Job.LocalWorkerNum)); err != nil {
			if !errors.Is(err, machinery.ErrWorkerQuitGracefully) {
				logger.Fatalf("scheduler queue worker error: %s", err.Error())
			}
		}
	}()
}

// preheat is a job to preheat, it is not supported to preheat
// with range requests.
func (j *job) preheat(ctx context.Context, data string) (string, error) {
	req := &internaljob.PreheatRequest{}
	if err := internaljob.UnmarshalRequest(data, req); err != nil {
		logger.Errorf("[preheat]: unmarshal request err: %s, request body: %s", err.Error(), data)
		return "", err
	}

	if err := validator.New().Struct(req); err != nil {
		logger.Errorf("[preheat]: preheat %s validate failed: %s", req.URLs, err.Error())
		return "", err
	}

	log := logger.WithPreheatJob(req.GroupUUID, req.TaskUUID, req.URLs)
	log.Infof("[preheat]: preheat %s %d request: %#v", req.URLs, req.PieceLength, req)

	ctx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	switch req.Scope {
	case managertypes.SingleSeedPeerScope:
		log.Info("[preheat]: preheat single seed peer")
		resp, err := j.PreheatSingleSeedPeer(ctx, req, log)
		if err != nil {
			log.Errorf("[preheat]: preheat single seed peer failed: %s", err.Error())
			return "", err
		}

		return internaljob.MarshalResponse(resp)
	case managertypes.AllSeedPeersScope:
		log.Info("[preheat]: preheat all seed peers")
		resp, err := j.PreheatAllSeedPeers(ctx, req, log)
		if err != nil {
			log.Errorf("[preheat]: preheat all seed peers failed: %s", err.Error())
			return "", err
		}

		return internaljob.MarshalResponse(resp)
	case managertypes.AllPeersScope:
		log.Info("[preheat]: preheat all peers")
		resp, err := j.PreheatAllPeers(ctx, req, log)
		if err != nil {
			log.Errorf("[preheat]: preheat all peers failed: %s", err.Error())
			return "", err
		}

		return internaljob.MarshalResponse(resp)
	default:
		log.Warnf("[preheat]: scope is invalid %s, preheat single peer", req.Scope)
		resp, err := j.PreheatSingleSeedPeer(ctx, req, log)
		if err != nil {
			log.Errorf("[preheat]: preheat single seed peer failed: %s", err.Error())
			return "", err
		}

		return internaljob.MarshalResponse(resp)
	}
}

// PreheatSinglePeer preheats job by single seed peer, scheduler will trigger seed peer to download task.
func (j *job) PreheatSingleSeedPeer(ctx context.Context, req *internaljob.PreheatRequest, log *logger.SugaredLoggerOnWith) (*internaljob.PreheatResponse, error) {
	// If scheduler has no available seed peer, return error.
	if !j.resource.SeedPeer().HasAvailable() {
		return nil, fmt.Errorf("cluster %d scheduler %s no available seed peer", j.config.Manager.SchedulerClusterID, j.config.Server.AdvertiseIP)
	}

	// Preheat by v2 grpc protocol. If seed peer does not support
	// v2 protocol, preheat by v1 grpc protocol.
	resp, err := j.preheatV2SingleSeedPeer(ctx, req, log)
	if err != nil {
		log.Errorf("[preheat]: preheat failed: %s", err.Error())
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.Unimplemented {
				return j.preheatV1SingleSeedPeer(ctx, req, log)
			}
		}

		return nil, err
	}

	resp.SchedulerClusterID = j.config.Manager.SchedulerClusterID
	return resp, nil
}

// preheatV1SingleSeedPeer preheats job by v1 grpc protocol.
func (j *job) preheatV1SingleSeedPeer(ctx context.Context, req *internaljob.PreheatRequest, log *logger.SugaredLoggerOnWith) (*internaljob.PreheatResponse, error) {
	taskID, err := idgen.TaskIDV2(req.URL, req.PieceLength, req.Tag, req.Application, idgen.ParseFilteredQueryParams(req.FilteredQueryParams), "", req.EnableTaskIDBasedBlobDigest)
	if err != nil {
		return nil, err
	}

	urlMeta := &commonv1.UrlMeta{
		Tag:         req.Tag,
		Filter:      req.FilteredQueryParams,
		Header:      req.Headers,
		Application: req.Application,
		Priority:    commonv1.Priority(req.Priority),
	}

	selected, err := j.resource.SeedPeer().Select(ctx, taskID)
	if err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(selected.IP, strconv.Itoa(int(selected.Port)))
	log.Infof("[preheat]: selected seed peer %s", addr)

	// TODO(chlins): reuse the client if we encounter the performance issue in future.
	client, err := cndsystemclient.GetClientByAddr(ctx, dfnet.NetAddr{Type: dfnet.TCP, Addr: addr}, j.dialOptions...)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// Trigger seed peer download seeds.
	stream, err := client.ObtainSeeds(ctx, &cdnsystemv1.SeedRequest{
		TaskId:  taskID,
		Url:     req.URL,
		UrlMeta: urlMeta,
	})
	if err != nil {
		log.Errorf("[preheat]: preheat failed: %s", err.Error())
		return nil, err
	}

	for {
		piece, err := stream.Recv()
		if err != nil {
			log.Errorf("[preheat]: receive piece failed: %s", err.Error())
			return nil, err
		}

		if piece.Done {
			log.Info("[preheat]: preheat succeeded")
			if host, ok := j.resource.HostManager().Load(piece.HostId); ok {
				return &internaljob.PreheatResponse{
					SuccessTasks: []*internaljob.PreheatSuccessTask{{URL: req.URL, Hostname: host.Hostname, IP: host.IP}},
				}, nil
			}

			log.Warnf("[preheat]: host %s not found", piece.HostId)
			return &internaljob.PreheatResponse{
				SuccessTasks: []*internaljob.PreheatSuccessTask{{URL: req.URL, Hostname: "unknown", IP: "unknown"}},
			}, nil
		}
	}
}

// preheatV2SingleSeedPeer preheats job by v2 grpc protocol for single seed peer by multiple URLs.
func (j *job) preheatV2SingleSeedPeer(ctx context.Context, req *internaljob.PreheatRequest, log *logger.SugaredLoggerOnWith) (*internaljob.PreheatResponse, error) {
	var mu sync.Mutex
	preheatResp := &internaljob.PreheatResponse{}
	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(int(req.ConcurrentTaskCount))
	for _, url := range req.URLs {
		eg.Go(func() error {
			resp, err := j.preheatV2SingleSeedPeerByURL(ctx, url, req, log)
			if err != nil {
				log.Errorf("[preheat]: preheat failed for %s: %s", url, err.Error())
				return err
			}

			mu.Lock()
			preheatResp.SuccessTasks = append(preheatResp.SuccessTasks, resp.SuccessTasks...)
			preheatResp.FailureTasks = append(preheatResp.FailureTasks, resp.FailureTasks...)
			mu.Unlock()
			log.Infof("[preheat]: preheat succeeded for %s", url)
			return nil
		})
	}

	// Wait for all tasks to complete and print the errors.
	if err := eg.Wait(); err != nil {
		log.Errorf("[preheat]: preheat failed: %s", err.Error())
	}

	return preheatResp, nil
}

// preheatV2SingleSeedPeerByURL preheats job by v2 grpc protocol for single seed peer by URL.
func (j *job) preheatV2SingleSeedPeerByURL(ctx context.Context, url string, req *internaljob.PreheatRequest, log *logger.SugaredLoggerOnWith) (*internaljob.PreheatResponse, error) {
	filteredQueryParams := idgen.ParseFilteredQueryParams(req.FilteredQueryParams)
	taskID, err := idgen.TaskIDV2(url, req.PieceLength, req.Tag, req.Application, filteredQueryParams, "", req.EnableTaskIDBasedBlobDigest)
	if err != nil {
		return nil, err
	}

	advertiseIP := j.config.Server.AdvertiseIP.String()

	selected, err := j.resource.SeedPeer().Select(ctx, taskID)
	if err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(selected.IP, strconv.Itoa(int(selected.Port)))
	log.Infof("[preheat]: selected seed peer %s", addr)

	client, err := j.resource.PeerClientPool().Get(addr, j.dialOptions...)
	if err != nil {
		return nil, err
	}

	stream, err := client.DownloadTask(ctx, taskID, &dfdaemonv2.DownloadTaskRequest{
		Download: &commonv2.Download{
			Url:                         url,
			PieceLength:                 req.PieceLength,
			Type:                        commonv2.TaskType_STANDARD,
			Tag:                         &req.Tag,
			Application:                 &req.Application,
			Priority:                    commonv2.Priority(req.Priority),
			FilteredQueryParams:         filteredQueryParams,
			RequestHeader:               req.Headers,
			CertificateChain:            req.CertificateChain,
			RemoteIp:                    &advertiseIP,
			Timeout:                     durationpb.New(req.Timeout),
			ObjectStorage:               req.ObjectStorage,
			Hdfs:                        req.Hdfs,
			OutputPath:                  req.OutputPath,
			EnableTaskIdBasedBlobDigest: req.EnableTaskIDBasedBlobDigest,
		}})
	if err != nil {
		log.Errorf("[preheat]: preheat failed: %s", err.Error())
		return nil, err
	}

	// Wait for the download task to complete.
	var hostID string
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Info("[preheat]: preheat succeeded")
				if host, ok := j.resource.HostManager().Load(hostID); ok {
					return &internaljob.PreheatResponse{
						SuccessTasks: []*internaljob.PreheatSuccessTask{{URL: url, Hostname: host.Hostname, IP: host.IP}},
					}, nil
				}

				log.Warnf("[preheat]: host %s not found", hostID)
				return &internaljob.PreheatResponse{
					SuccessTasks: []*internaljob.PreheatSuccessTask{{URL: url, Hostname: "unknown", IP: "unknown"}},
				}, nil
			}

			log.Errorf("[preheat]: receive piece failed: %s", err.Error())
			return nil, err
		}

		hostID = resp.HostId
	}
}

// PreheatAllSeedPeers preheats job by all peer seed peers, only supported by v2 protocol. Scheduler will trigger all seed peers to download task.
// If all the seed peers download task failed, return error. If some of the seed peers download task failed, return success tasks and failure tasks.
// Notify the client that the preheat is successful.
func (j *job) PreheatAllSeedPeers(ctx context.Context, req *internaljob.PreheatRequest, log *logger.SugaredLoggerOnWith) (*internaljob.PreheatResponse, error) {
	// If scheduler has no available seed peer, return error.
	seedPeers, err := j.selectSeedPeers(req.IPs, req.Count, req.Percentage, log)
	if err != nil {
		return nil, err
	}

	var (
		successTasks = sync.Map{}
		failureTasks = sync.Map{}
	)
	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(int(req.ConcurrentTaskCount))
	for _, url := range req.URLs {
		eg.Go(func() error {
			peg, _ := errgroup.WithContext(ctx)
			peg.SetLimit(int(req.ConcurrentPeerCount))
			for _, seedPeer := range seedPeers {
				var (
					hostname = seedPeer.Hostname
					ip       = seedPeer.IP
					port     = seedPeer.Port
				)

				addr := net.JoinHostPort(ip, strconv.Itoa(int(port)))
				peg.Go(func() error {
					filteredQueryParams := idgen.ParseFilteredQueryParams(req.FilteredQueryParams)
					taskID, err := idgen.TaskIDV2(url, req.PieceLength, req.Tag, req.Application, filteredQueryParams, "", req.EnableTaskIDBasedBlobDigest)
					if err != nil {
						return err
					}

					hostID := idgen.HostID(ip, hostname, true)
					compositeID := fmt.Sprintf("%s-%s", taskID, hostID)
					log := logger.WithPreheatJobAndHost(req.GroupUUID, req.TaskUUID, taskID, url, idgen.HostID(ip, hostname, true), hostname, ip)
					log.Info("[preheat]: preheat started")

					dfdaemonClient, err := j.resource.PeerClientPool().Get(addr, j.dialOptions...)
					if err != nil {
						log.Errorf("[preheat]: preheat failed: %s", err.Error())
						failureTasks.Store(compositeID, &internaljob.PreheatFailureTask{
							URL:         url,
							Hostname:    hostname,
							IP:          ip,
							Description: fmt.Sprintf("group uuid %s failed: %s", req.GroupUUID, err.Error()),
						})

						return err
					}

					advertiseIP := j.config.Server.AdvertiseIP.String()
					stream, err := dfdaemonClient.DownloadTask(
						ctx,
						taskID,
						&dfdaemonv2.DownloadTaskRequest{Download: &commonv2.Download{
							Url:                         url,
							PieceLength:                 req.PieceLength,
							Type:                        commonv2.TaskType_STANDARD,
							Tag:                         &req.Tag,
							Application:                 &req.Application,
							Priority:                    commonv2.Priority(req.Priority),
							FilteredQueryParams:         filteredQueryParams,
							RequestHeader:               req.Headers,
							Timeout:                     durationpb.New(req.Timeout),
							CertificateChain:            req.CertificateChain,
							RemoteIp:                    &advertiseIP,
							ObjectStorage:               req.ObjectStorage,
							Hdfs:                        req.Hdfs,
							OutputPath:                  req.OutputPath,
							EnableTaskIdBasedBlobDigest: req.EnableTaskIDBasedBlobDigest,
						}})
					if err != nil {
						log.Errorf("[preheat]: preheat failed: %s", err.Error())
						failureTasks.Store(compositeID, &internaljob.PreheatFailureTask{
							URL:         url,
							Hostname:    hostname,
							IP:          ip,
							Description: fmt.Sprintf("task %s failed: %s", taskID, err.Error()),
						})

						return err
					}

					// Wait for the download task to complete.
					for {
						_, err := stream.Recv()
						if err != nil {
							if err == io.EOF {
								log.Info("[preheat]: preheat succeeded")
								successTasks.Store(compositeID, &internaljob.PreheatSuccessTask{
									URL:      url,
									Hostname: hostname,
									IP:       ip,
								})

								return nil
							}

							log.Errorf("[preheat]: preheat failed: %s", err.Error())
							failureTasks.Store(compositeID, &internaljob.PreheatFailureTask{
								URL:         url,
								Hostname:    hostname,
								IP:          ip,
								Description: fmt.Sprintf("task %s failed: %s", taskID, err.Error()),
							})

							return err
						}
					}
				})
			}

			// Wait for all seed peers to download single task and print the errors.
			if err := peg.Wait(); err != nil {
				log.Errorf("[preheat]: preheat failed: %s", err.Error())
			}

			return nil
		})
	}

	// Wait for all tasks to complete and print the errors.
	if err := eg.Wait(); err != nil {
		log.Errorf("[preheat]: preheat failed: %s", err.Error())
	}

	// If successTasks is not empty, return success tasks and failure tasks.
	// Notify the client that the preheat is successful.
	preheatResponse := internaljob.PreheatResponse{SchedulerClusterID: j.config.Manager.SchedulerClusterID, SuccessTasks: make([]*internaljob.PreheatSuccessTask, 0), FailureTasks: make([]*internaljob.PreheatFailureTask, 0)}
	failureTasks.Range(func(_, value any) bool {
		if failureTask, ok := value.(*internaljob.PreheatFailureTask); ok {
			preheatResponse.FailureTasks = append(preheatResponse.FailureTasks, failureTask)
		}

		return true
	})

	successTasks.Range(func(_, value any) bool {
		if successTask, ok := value.(*internaljob.PreheatSuccessTask); ok {
			for _, failureTask := range preheatResponse.FailureTasks {
				if failureTask.IP == successTask.IP {
					return true
				}
			}

			preheatResponse.SuccessTasks = append(preheatResponse.SuccessTasks, successTask)
		}

		return true
	})

	if len(preheatResponse.SuccessTasks) > 0 {
		return &preheatResponse, nil
	}

	msg := "no error message"
	if len(preheatResponse.FailureTasks) > 0 {
		msg = fmt.Sprintf("%s %s %s %s %s", req.GroupUUID, req.TaskUUID, preheatResponse.FailureTasks[0].IP, preheatResponse.FailureTasks[0].Hostname,
			preheatResponse.FailureTasks[0].Description)
	}

	return nil, fmt.Errorf("all peers preheat failed: %s", msg)
}

// selectSeedPeers selects seed peers based on provided IPs, count, or percentage, with the following priority:
// 1. IPs: If specific IP addresses are provided, only seed peers matching those IPs are selected.
// 2. Count: If count is provided, selects up to the specified number of seed peers. If count exceeds the number of available seed peers, all seed peers are selected.
// 3. Percentage: If percentage is provided, selects a proportional number of seed peers (rounded down). Ensures at least one seed peer is selected if percentage > 0.
// Priority: IPs > Count > Percentage
func (j *job) selectSeedPeers(ips []string, count *uint32, percentage *uint32, log *logger.SugaredLoggerOnWith) ([]*resource.Host, error) {
	// If scheduler has no available seed peer, return error.
	if !j.resource.SeedPeer().HasAvailable() {
		return nil, fmt.Errorf("cluster %d scheduler %s no available seed peer", j.config.Manager.SchedulerClusterID, j.config.Server.AdvertiseIP)
	}

	seedPeers := j.resource.HostManager().LoadAllSeeds()
	if len(seedPeers) == 0 {
		return nil, fmt.Errorf("cluster %d scheduler %s has no available seed peer", j.config.Manager.SchedulerClusterID, j.config.Server.AdvertiseIP)
	}

	if len(ips) > 0 {
		selectedSeedPeers := make([]*resource.Host, 0, len(ips))
		for _, seedPeer := range seedPeers {
			if slices.Contains(ips, seedPeer.IP) {
				selectedSeedPeers = append(selectedSeedPeers, seedPeer)
				continue
			}
		}

		if len(selectedSeedPeers) == 0 {
			return nil, fmt.Errorf("no seed peer found for ips %v", ips)
		}

		log.Infof("[preheat]: select %d seed peers from %d seed peers by ips", len(selectedSeedPeers), len(seedPeers))
		return selectedSeedPeers, nil
	}

	if count != nil {
		if *count > uint32(len(seedPeers)) {
			log.Infof("[preheat]: count is %d, but seed peers count is %d. use seed peers count", *count, len(seedPeers))
			*count = uint32(len(seedPeers))
		}

		log.Infof("[preheat]: select %d seed peers from %d seed peers", *count, len(seedPeers))
		return seedPeers[:*count], nil
	}

	if percentage != nil {
		seedPeerCount := (len(seedPeers) * int(*percentage)) / 100

		// Clamp to the number of available seed peers, a percentage over 100
		// would otherwise make the slice bound exceed len(seedPeers) and panic.
		seedPeerCount = min(seedPeerCount, len(seedPeers))

		// Ensure at least one peer is selected if percentage > 0.
		if seedPeerCount == 0 && *percentage > 0 {
			seedPeerCount = 1
		}

		log.Infof("[preheat]: select %d seed peers from %d seed peers, percentage is %d", seedPeerCount, len(seedPeers), *percentage)
		return seedPeers[:seedPeerCount], nil
	}

	log.Infof("[preheat]: count and percentage are both nil, select all seed peers, count is %d", len(seedPeers))
	return seedPeers, nil
}

// PreheatAllPeers preheats job by all peers, only supported by v2 protocol. Scheduler will trigger all peers to download task.
// If all the peers download task failed, return error. If some of the peers download task failed, return success tasks and
// failure tasks. Notify the client that the preheat is successful.
func (j *job) PreheatAllPeers(ctx context.Context, req *internaljob.PreheatRequest, log *logger.SugaredLoggerOnWith) (*internaljob.PreheatResponse, error) {
	peers, err := j.selectPeers(req.IPs, req.Count, req.Percentage, log)
	if err != nil {
		return nil, err
	}

	var (
		successTasks = sync.Map{}
		failureTasks = sync.Map{}
	)
	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(int(req.ConcurrentTaskCount))
	for _, url := range req.URLs {
		eg.Go(func() error {
			peg, _ := errgroup.WithContext(ctx)
			peg.SetLimit(int(req.ConcurrentPeerCount))
			for _, peer := range peers {
				var (
					hostname = peer.Hostname
					ip       = peer.IP
					port     = peer.Port
				)

				addr := net.JoinHostPort(ip, strconv.Itoa(int(port)))
				peg.Go(func() error {
					filteredQueryParams := idgen.ParseFilteredQueryParams(req.FilteredQueryParams)
					taskID, err := idgen.TaskIDV2(url, req.PieceLength, req.Tag, req.Application, filteredQueryParams, "", req.EnableTaskIDBasedBlobDigest)
					if err != nil {
						return err
					}

					hostID := idgen.HostID(ip, hostname, false)
					compositeID := fmt.Sprintf("%s-%s", taskID, hostID)
					log := logger.WithPreheatJobAndHost(req.GroupUUID, req.TaskUUID, taskID, url, idgen.HostID(ip, hostname, true), hostname, ip)
					log.Info("[preheat]: preheat started")

					dfdaemonClient, err := j.resource.PeerClientPool().Get(addr, j.dialOptions...)
					if err != nil {
						log.Errorf("[preheat]: preheat failed: %s", err.Error())
						failureTasks.Store(compositeID, &internaljob.PreheatFailureTask{
							URL:         url,
							Hostname:    hostname,
							IP:          ip,
							Description: fmt.Sprintf("task %s failed: %s", taskID, err.Error()),
						})

						return err
					}

					advertiseIP := j.config.Server.AdvertiseIP.String()
					stream, err := dfdaemonClient.DownloadTask(
						ctx,
						taskID,
						&dfdaemonv2.DownloadTaskRequest{Download: &commonv2.Download{
							Url:                         url,
							PieceLength:                 req.PieceLength,
							Type:                        commonv2.TaskType_STANDARD,
							Tag:                         &req.Tag,
							Application:                 &req.Application,
							Priority:                    commonv2.Priority(req.Priority),
							FilteredQueryParams:         filteredQueryParams,
							RequestHeader:               req.Headers,
							Timeout:                     durationpb.New(req.Timeout),
							CertificateChain:            req.CertificateChain,
							RemoteIp:                    &advertiseIP,
							ObjectStorage:               req.ObjectStorage,
							Hdfs:                        req.Hdfs,
							OutputPath:                  req.OutputPath,
							EnableTaskIdBasedBlobDigest: req.EnableTaskIDBasedBlobDigest,
						}})
					if err != nil {
						log.Errorf("[preheat]: preheat failed: %s", err.Error())
						failureTasks.Store(compositeID, &internaljob.PreheatFailureTask{
							URL:         url,
							Hostname:    hostname,
							IP:          ip,
							Description: fmt.Sprintf("task %s failed: %s", taskID, err.Error()),
						})

						return err
					}

					// Wait for the download task to complete.
					for {
						_, err := stream.Recv()
						if err != nil {
							if err == io.EOF {
								log.Info("[preheat]: preheat succeeded")
								successTasks.Store(compositeID, &internaljob.PreheatSuccessTask{
									URL:      url,
									Hostname: hostname,
									IP:       ip,
								})

								return nil
							}

							log.Errorf("[preheat]: preheat failed: %s", err.Error())
							failureTasks.Store(compositeID, &internaljob.PreheatFailureTask{
								URL:         url,
								Hostname:    hostname,
								IP:          ip,
								Description: fmt.Sprintf("task %s failed: %s", taskID, err.Error()),
							})

							return err
						}
					}
				})
			}

			// Wait for all peers to download single task and print the errors.
			if err := peg.Wait(); err != nil {
				log.Errorf("[preheat]: preheat failed: %s", err.Error())
			}

			return nil
		})
	}

	// Wait for all tasks to complete and print the errors.
	if err := eg.Wait(); err != nil {
		log.Errorf("[preheat]: preheat failed: %s", err.Error())
	}

	// If successTasks is not empty, return success tasks and failure tasks.
	// Notify the client that the preheat is successful.
	preheatResponse := internaljob.PreheatResponse{SchedulerClusterID: j.config.Manager.SchedulerClusterID, SuccessTasks: make([]*internaljob.PreheatSuccessTask, 0), FailureTasks: make([]*internaljob.PreheatFailureTask, 0)}
	failureTasks.Range(func(_, value any) bool {
		if failureTask, ok := value.(*internaljob.PreheatFailureTask); ok {
			preheatResponse.FailureTasks = append(preheatResponse.FailureTasks, failureTask)
		}

		return true
	})

	successTasks.Range(func(_, value any) bool {
		if successTask, ok := value.(*internaljob.PreheatSuccessTask); ok {
			for _, failureTask := range preheatResponse.FailureTasks {
				if failureTask.IP == successTask.IP {
					return true
				}
			}

			preheatResponse.SuccessTasks = append(preheatResponse.SuccessTasks, successTask)
		}

		return true
	})

	if len(preheatResponse.SuccessTasks) > 0 {
		return &preheatResponse, nil
	}

	msg := "no error message"
	if len(preheatResponse.FailureTasks) > 0 {
		msg = fmt.Sprintf("%s %s %s %s %s", req.GroupUUID, req.TaskUUID, preheatResponse.FailureTasks[0].IP, preheatResponse.FailureTasks[0].Hostname,
			preheatResponse.FailureTasks[0].Description)
	}

	return nil, fmt.Errorf("all peers preheat failed: %s", msg)
}

// selectPeers selects peers based on provided IPs, count, or percentage, with the following priority:
// 1. IPs: If specific IP addresses are provided, only peers matching those IPs are selected.
// 2. Count: If count is provided, selects up to the specified number of peers. If count exceeds the number of available peers, all peers are selected.
// 3. Percentage: If percentage is provided, selects a proportional number of peers (rounded down). Ensures at least one peer is selected if percentage > 0.
// Priority: IPs > Count > Percentage
func (j *job) selectPeers(ips []string, count *uint32, percentage *uint32, log *logger.SugaredLoggerOnWith) ([]*resource.Host, error) {
	peers := j.resource.HostManager().LoadAllNormals()
	if len(peers) == 0 {
		return nil, fmt.Errorf("[preheat]: cluster %d scheduler %s has no available peer", j.config.Manager.SchedulerClusterID, j.config.Server.AdvertiseIP)
	}

	if len(ips) > 0 {
		selectedPeers := make([]*resource.Host, 0, len(ips))
		for _, peer := range peers {
			if slices.Contains(ips, peer.IP) {
				selectedPeers = append(selectedPeers, peer)
				continue
			}
		}

		if len(selectedPeers) == 0 {
			return nil, fmt.Errorf("no peer found for ips %v", ips)
		}

		log.Infof("[preheat]: select %d peers from %d peers by ips", len(selectedPeers), len(peers))
		return selectedPeers, nil
	}

	if count != nil {
		if *count > uint32(len(peers)) {
			log.Infof("[preheat]: count is %d, but peers count is %d. use peers count", *count, len(peers))
			*count = uint32(len(peers))
		}

		log.Infof("[preheat]: select %d peers from %d seed peers", *count, len(peers))
		return peers[:*count], nil
	}

	if percentage != nil {
		peerCount := (len(peers) * int(*percentage)) / 100

		// Clamp to the number of available peers, a percentage over 100 would
		// otherwise make the slice bound exceed len(peers) and panic.
		peerCount = min(peerCount, len(peers))

		// Ensure at least one peer is selected if percentage > 0.
		if peerCount == 0 && *percentage > 0 {
			peerCount = 1
		}

		log.Infof("[preheat]: select %d peers from %d peers, percentage is %d", peerCount, len(peers), *percentage)
		return peers[:peerCount], nil
	}

	log.Infof("[preheat]: count and percentage are both nil, select all peers, count is %d", len(peers))
	return peers, nil
}

// syncPeers is a job to sync peers.
func (j *job) syncPeers(_ context.Context, data string) (string, error) {
	req := &internaljob.SyncPeerRequest{}
	if err := internaljob.UnmarshalRequest(data, req); err != nil {
		logger.Errorf("[sync-peers] unmarshal request err: %s, request body: %s", err.Error(), data)
		return "", err
	}

	if err := validator.New().Struct(req); err != nil {
		logger.Errorf("[sync-peers] sync peer task %s validate failed: %s", req.GroupUUID, err.Error())
		return "", err
	}

	hosts := make([]*resource.Host, 0, j.resource.HostManager().Len())
	j.resource.HostManager().Range(func(key, value any) bool {
		host, ok := value.(*resource.Host)
		if !ok {
			logger.Errorf("[sync-peers] invalid host %v: %v", key, value)
			return true
		}

		hosts = append(hosts, host)
		return true
	})

	// Sort hosts by ID.
	sort.Slice(hosts, func(i, j int) bool {
		return hosts[i].ID < hosts[j].ID
	})

	// We ignore changes in hosts data across multiple task requests within the same group.
	// TODO: (Could this cause problems? Perhaps we should create a snapshot of the hosts for same group to ensure that the host information is completely accurate.)
	// Split hosts by task count.
	total := len(hosts)
	groupSize := total / req.TotalGroupTask
	remainder := total % req.TotalGroupTask

	var start int
	if req.GroupTaskCount < remainder {
		start = req.GroupTaskCount * (groupSize + 1)
	} else {
		start = remainder*(groupSize+1) + (req.GroupTaskCount-remainder)*groupSize
	}
	end := start + groupSize
	if req.GroupTaskCount < remainder {
		end++
	}

	taskHosts := hosts[start:end]

	return internaljob.MarshalResponse(taskHosts)
}

// getTask is a job to get task.
func (j *job) getTask(ctx context.Context, data string) (string, error) {
	req := &internaljob.GetTaskRequest{}
	if err := internaljob.UnmarshalRequest(data, req); err != nil {
		logger.Errorf("[get-task] unmarshal request err: %s, request body: %s", err.Error(), data)
		return "", err
	}

	if err := validator.New().Struct(req); err != nil {
		logger.Errorf("[get-task] get task %s validate failed: %s", req.TaskID, err.Error())
		return "", err
	}

	log := logger.WithGetTaskJob(req.GroupUUID, req.TaskUUID, req.TaskID)
	log.Infof("[get-task] get task request: %#v", req)

	ctx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	resp, err := j.GetTask(ctx, req, log)
	if err != nil {
		log.Errorf("[get-task] get task failed: %s", err.Error())
		return "", err
	}
	log.Infof("[get-task] get length of peers: %d", len(resp.Peers))

	return internaljob.MarshalResponse(resp)
}

// GetTask retrieves task information from peers in the cluster. If the scope is
// all_seed_peers, only seed peers are queried, otherwise all peers are queried.
func (j *job) GetTask(ctx context.Context, req *internaljob.GetTaskRequest, log *logger.SugaredLoggerOnWith) (*internaljob.GetTaskResponse, error) {
	var peers []*resource.Host
	switch req.Scope {
	case managertypes.AllSeedPeersScope:
		peers = j.resource.HostManager().LoadAllSeeds()
	default:
		peers = j.resource.HostManager().LoadAll()
	}

	if len(peers) == 0 {
		log.Warn("[get-task] no peers found")
		return &internaljob.GetTaskResponse{
			SchedulerClusterID: j.config.Manager.SchedulerClusterID,
		}, nil
	}

	var mu sync.Mutex
	resp := &internaljob.GetTaskResponse{
		SchedulerClusterID: j.config.Manager.SchedulerClusterID,
		Peers:              make([]*internaljob.Peer, 0, len(peers)),
	}
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(int(req.ConcurrentPeerCount))
	for _, peer := range peers {
		var (
			id        = peer.ID
			hostname  = peer.Hostname
			ip        = peer.IP
			port      = peer.Port
			hostType  = peer.Type.Name()
			createdAt = peer.CreatedAt.Load()
			updatedAt = peer.UpdatedAt.Load()
		)

		addr := net.JoinHostPort(ip, strconv.Itoa(int(port)))
		eg.Go(func() error {
			dfdaemonClient, err := j.resource.PeerClientPool().Get(addr, j.dialOptions...)
			if err != nil {
				log.Warnf("[get-task] get client from %s failed: %s", addr, err.Error())
				return nil
			}

			advertiseIP := j.config.Server.AdvertiseIP.String()
			localTask, err := dfdaemonClient.StatLocalTask(ctx, &dfdaemonv2.StatLocalTaskRequest{
				TaskId:   req.TaskID,
				RemoteIp: &advertiseIP,
			})
			if err != nil {
				log.Errorf("[get-task] stat task failed: %s", err.Error())
				return nil
			}

			mu.Lock()
			resp.Peers = append(resp.Peers, &internaljob.Peer{
				ID:         id,
				Hostname:   hostname,
				IP:         ip,
				HostType:   hostType,
				CreatedAt:  createdAt,
				UpdatedAt:  updatedAt,
				IsFinished: localTask.GetFinishedAt() != nil,
			})
			mu.Unlock()
			return nil
		})
	}

	// If any of the goroutines return an error, ignore it and continue processing.
	if err := eg.Wait(); err != nil {
		logger.Errorf("[get-task] failed to get task: %w", err)
	}

	return resp, nil
}

// deleteTask is a job to delete task.
func (j *job) deleteTask(ctx context.Context, data string) (string, error) {
	req := &internaljob.DeleteTaskRequest{}
	if err := internaljob.UnmarshalRequest(data, req); err != nil {
		logger.Errorf("[delete-task] unmarshal request err: %s, request body: %s", err.Error(), data)
		return "", err
	}

	if err := validator.New().Struct(req); err != nil {
		logger.Errorf("[delete-task] delete task %s validate failed: %s", req.TaskID, err.Error())
		return "", err
	}

	log := logger.WithDeleteTaskJob(req.GroupUUID, req.TaskUUID, req.TaskID)
	log.Infof("[delete-task] delete task request: %#v", req)

	ctx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	task, ok := j.resource.TaskManager().Load(req.TaskID)
	if !ok {
		// Do not return error if task not found, just retunr empty response.
		log.Warn("[delete-task] task not found")
		return internaljob.MarshalResponse(&internaljob.DeleteTaskResponse{
			SchedulerClusterID: j.config.Manager.SchedulerClusterID,
		})
	}

	finishedPeers := task.LoadFinishedPeers()
	successTasks := make([]*internaljob.DeleteSuccessTask, 0, len(finishedPeers))
	failureTasks := make([]*internaljob.DeleteFailureTask, 0, len(finishedPeers))
	for _, finishedPeer := range finishedPeers {
		log := logger.WithDeleteTaskJobAndPeer(req.GroupUUID, req.TaskUUID, finishedPeer.Host.ID, finishedPeer.Task.ID, finishedPeer.ID)

		addr := net.JoinHostPort(finishedPeer.Host.IP, strconv.Itoa(int(finishedPeer.Host.Port)))
		dfdaemonClient, err := j.resource.PeerClientPool().Get(addr, j.dialOptions...)
		if err != nil {
			log.Errorf("[delete-task] get client from %s failed: %s", addr, err.Error())
			failureTasks = append(failureTasks, &internaljob.DeleteFailureTask{
				Hostname:    finishedPeer.Host.Hostname,
				IP:          finishedPeer.Host.IP,
				HostType:    finishedPeer.Host.Type.Name(),
				Description: fmt.Sprintf("task %s failed: %s", req.TaskID, err.Error()),
			})

			continue
		}

		advertiseIP := j.config.Server.AdvertiseIP.String()
		if err = dfdaemonClient.DeleteTask(ctx, &dfdaemonv2.DeleteTaskRequest{
			TaskId:   req.TaskID,
			RemoteIp: &advertiseIP,
		}); err != nil {
			log.Errorf("[delete-task] delete task failed: %s", err.Error())
			failureTasks = append(failureTasks, &internaljob.DeleteFailureTask{
				Hostname:    finishedPeer.Host.Hostname,
				IP:          finishedPeer.Host.IP,
				HostType:    finishedPeer.Host.Type.Name(),
				Description: fmt.Sprintf("task %s failed: %s", req.TaskID, err.Error()),
			})

			continue
		}

		task.DeletePeer(finishedPeer.ID)
		successTasks = append(successTasks, &internaljob.DeleteSuccessTask{
			Hostname: finishedPeer.Host.Hostname,
			IP:       finishedPeer.Host.IP,
			HostType: finishedPeer.Host.Type.Name(),
		})
	}

	return internaljob.MarshalResponse(&internaljob.DeleteTaskResponse{
		SuccessTasks:       successTasks,
		FailureTasks:       failureTasks,
		SchedulerClusterID: j.config.Manager.SchedulerClusterID,
	})
}

func (j *job) ListTaskEntries(ctx context.Context, req *internaljob.ListTaskEntriesRequest, log *logger.SugaredLoggerOnWith) (*internaljob.ListTaskEntriesResponse, error) {
	advertiseIP := j.config.Server.AdvertiseIP.String()

	// select a dfdaemon from peers or seed peers
	var selected *resource.Host
	if peers, err := j.selectPeers([]string{}, nil, nil, log); err != nil {
		log.Warnf("[list-task-entries] select peers failed: %s", err)
		seedPeer, err := j.resource.SeedPeer().Select(ctx, req.TaskID)
		if err != nil {
			return nil, err
		}

		selected = seedPeer
	} else {
		selected = peers[0]
	}

	addr := net.JoinHostPort(selected.IP, strconv.Itoa(int(selected.Port)))
	log.Infof("[list-task-entries] selected seed peer %s for task %s", addr, req.TaskID)

	dfdaemonClient, err := j.resource.PeerClientPool().Get(addr, j.dialOptions...)
	if err != nil {
		log.Errorf("[list-task-entries] get dfdaemon client failed: %s", err)
		return nil, err
	}

	res, err := dfdaemonClient.ListTaskEntries(ctx, &dfdaemonv2.ListTaskEntriesRequest{
		TaskId:           req.TaskID,
		Url:              req.Url,
		RequestHeader:    req.Header,
		Timeout:          req.Timeout,
		CertificateChain: req.CertificateChain,
		ObjectStorage:    req.ObjectStorage,
		Hdfs:             req.Hdfs,
		RemoteIp:         &advertiseIP,
	})
	if err != nil {
		log.Errorf("[list-task-entries]list task entries failed: %s", err)
		return nil, err
	}

	var recursive bool
	if len(res.Entries) > 1 {
		recursive = true
	}

	return &internaljob.ListTaskEntriesResponse{
		Recursive:   recursive,
		Entries:     res.Entries,
		SchedulerID: j.config.Manager.SchedulerClusterID,
	}, nil
}
