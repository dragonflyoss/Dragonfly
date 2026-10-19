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

//go:generate mockgen -destination mocks/client_mock.go -source client.go -package mocks

package client

import (
	"context"
	"fmt"
	"math"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	logger "d7y.io/dragonfly/v2/internal/dflog"
)

const (
	// contextTimeout is timeout of grpc invoke.
	contextTimeout = 2 * time.Second
)

// GetClient returns health client.
func GetClient(ctx context.Context, target string, opts ...grpc.DialOption) (Client, error) {
	conn, err := grpc.NewClient(
		target,
		append([]grpc.DialOption{
			grpc.WithIdleTimeout(0),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(math.MaxInt32),
				grpc.MaxCallSendMsgSize(math.MaxInt32),
			),
			grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
				grpc_prometheus.UnaryClientInterceptor,
				grpc_zap.UnaryClientInterceptor(logger.GrpcLogger.Desugar()),
			)),
			grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(
				grpc_prometheus.StreamClientInterceptor,
				grpc_zap.StreamClientInterceptor(logger.GrpcLogger.Desugar()),
			)),
		}, opts...)...,
	)
	if err != nil {
		return nil, err
	}

	return &client{
		HealthClient: healthpb.NewHealthClient(conn),
		ClientConn:   conn,
	}, nil
}

// Check checks health of grpc server.
func Check(ctx context.Context, target string, opts ...grpc.DialOption) error {
	ctx, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()

	healthClient, err := GetClient(ctx, target, opts...)
	if err != nil {
		return err
	}
	defer healthClient.Close()

	if err := healthClient.Check(ctx, &healthpb.HealthCheckRequest{}); err != nil {
		return err
	}

	return nil
}

// Client is the interface for grpc client.
type Client interface {
	// Check checks health of grpc server.
	Check(context.Context, *healthpb.HealthCheckRequest, ...grpc.CallOption) error

	// Close tears down the ClientConn and all underlying connections.
	Close() error
}

// client provides health grpc function.
type client struct {
	healthpb.HealthClient
	*grpc.ClientConn
}

// Check checks health of grpc server.
func (c *client) Check(ctx context.Context, req *healthpb.HealthCheckRequest, opts ...grpc.CallOption) error {
	ctx, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()

	resp, err := c.HealthClient.Check(ctx, req, opts...)
	if err != nil {
		return err
	}

	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		return fmt.Errorf("check %s health failed, because of status is %d", c.ClientConn.Target(), resp.Status)
	}

	return nil
}
