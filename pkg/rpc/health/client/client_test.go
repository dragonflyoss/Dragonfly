/*
 *     Copyright 2022 The Dragonfly Authors
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

package client

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name   string
		target func(t *testing.T) string
		expect func(t *testing.T, elapsed time.Duration, err error)
	}{
		{
			name: "serving",
			target: func(t *testing.T) string {
				lis, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}

				svr := grpc.NewServer()
				healthpb.RegisterHealthServer(svr, health.NewServer())
				go func() {
					_ = svr.Serve(lis)
				}()
				t.Cleanup(svr.Stop)

				return lis.Addr().String()
			},
			expect: func(t *testing.T, elapsed time.Duration, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "not serving",
			target: func(t *testing.T) string {
				lis, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}

				svr := grpc.NewServer()
				healthServer := health.NewServer()
				healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
				healthpb.RegisterHealthServer(svr, healthServer)
				go func() {
					_ = svr.Serve(lis)
				}()
				t.Cleanup(svr.Stop)

				return lis.Addr().String()
			},
			expect: func(t *testing.T, elapsed time.Duration, err error) {
				assert := assert.New(t)
				assert.ErrorContains(err, "health failed")
			},
		},
		{
			name: "connection refused",
			target: func(t *testing.T) string {
				lis, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}

				addr := lis.Addr().String()
				if err := lis.Close(); err != nil {
					t.Fatal(err)
				}

				return addr
			},
			expect: func(t *testing.T, elapsed time.Duration, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "server accepts but never responds",
			target: func(t *testing.T) string {
				lis, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { lis.Close() })

				go func() {
					for {
						conn, err := lis.Accept()
						if err != nil {
							return
						}
						defer conn.Close()
					}
				}()

				return lis.Addr().String()
			},
			expect: func(t *testing.T, elapsed time.Duration, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.Less(elapsed, 2*contextTimeout)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target := tc.target(t)
			start := time.Now()
			err := Check(context.Background(), target, grpc.WithTransportCredentials(insecure.NewCredentials()))
			tc.expect(t, time.Since(start), err)
		})
	}
}
