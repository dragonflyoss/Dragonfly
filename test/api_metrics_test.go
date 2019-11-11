/*
 * Copyright The Dragonfly Authors.
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

package main

import (
	"fmt"

	"github.com/dragonflyoss/Dragonfly/test/command"
	"github.com/dragonflyoss/Dragonfly/test/request"

	"github.com/go-check/check"
)

// APIMetricsSuite is the test suite for Prometheus metrics.
type APIMetricsSuite struct {
	starter *command.Starter
}

func init() {
	check.Suite(&APIMetricsSuite{})
}

// SetUpSuite does common setup in the beginning of each test.
func (s *APIMetricsSuite) SetUpSuite(c *check.C) {
	s.starter = command.NewStarter("SupernodeMetricsTestSuite")
	if _, err := s.starter.Supernode(0); err != nil {
		panic(fmt.Sprintf("start supernode failed:%v", err))
	}
}

func (s *APIMetricsSuite) TearDownSuite(c *check.C) {
	s.starter.Clean()
}

// TestMetrics tests /metrics API.
func (s *APIMetricsSuite) TestMetrics(c *check.C) {
	resp, err := request.Get("/metrics")
	c.Assert(err, check.IsNil)
	defer resp.Body.Close()

	CheckRespStatus(c, resp, 200)
}

// TestMetricsRequestTotal tests http-related metrics.
func (s *APIMetricsSuite) TestHttpMetrics(c *check.C) {
	requestCounter := `dragonfly_supernode_http_requests_total{code="%d",handler="%s"}`
	responseSizeSum := `dragonfly_supernode_http_response_size_bytes_sum{handler="%s"}`
	responseSizeCount := `dragonfly_supernode_http_response_size_bytes_count{handler="%s"}`
	requestSizeCount := `dragonfly_supernode_http_request_size_bytes_count{handler="%s"}`

	resp, err := request.Get("/_ping")
	c.Assert(err, check.IsNil)
	CheckRespStatus(c, resp, 200)

	// Get httpRequest counter value equals 1.
	CheckMetric(c, fmt.Sprintf(requestCounter, 200, "/_ping"), 1)

	// Get httpResponse size sum value equals 2.
	CheckMetric(c, fmt.Sprintf(responseSizeSum, "/_ping"), 2)

	// Get httpResponse size count value equals 1.
	CheckMetric(c, fmt.Sprintf(responseSizeCount, "/_ping"), 1)

	// Get httpRequest size count value equals 1.
	CheckMetric(c, fmt.Sprintf(requestSizeCount, "/_ping"), 1)
}

// TestBuildInfoMetrics tests build info metric.
func (s *APIMetricsSuite) TestBuildInfoMetrics(c *check.C) {
	supernodeBuildInfo := `dragonfly_supernode_build_info{`
	// Ensure build_info metric exists.
	CheckMetric(c, supernodeBuildInfo, 1)
}
