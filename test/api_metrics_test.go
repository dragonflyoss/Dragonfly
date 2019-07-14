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
	requestCounter := `dragonfly_supernode_http_requests_total{code="%d",handler="%s",method="%s"}`
	responseSizeSum := `dragonfly_supernode_http_response_size_bytes_sum{code="%d",handler="%s",method="%s"}`
	responseSizeCount := `dragonfly_supernode_http_response_size_bytes_count{code="%d",handler="%s",method="%s"}`
	requestSizeCount := `dragonfly_supernode_http_request_size_bytes_count{code="%d",handler="%s",method="%s"}`

	resp, err := request.Get("/_ping")
	c.Assert(err, check.IsNil)
	CheckRespStatus(c, resp, 200)

	// Get httpRequest counter value equals 1.
	pingTimes, found := GetMetricValue(c, fmt.Sprintf(requestCounter, 200, "/_ping", "get"))
	c.Assert(found, check.Equals, true)
	c.Assert(pingTimes, check.Equals, float64(1))

	// Get httpResponse size sum value equals 2.
	responseBytes, found := GetMetricValue(c, fmt.Sprintf(responseSizeSum, 200, "/_ping", "get"))
	c.Assert(found, check.Equals, true)
	c.Assert(responseBytes, check.Equals, float64(2))

	// Get httpResponse size count value equals 1.
	responseCount, found := GetMetricValue(c, fmt.Sprintf(responseSizeCount, 200, "/_ping", "get"))
	c.Assert(found, check.Equals, true)
	c.Assert(responseCount, check.Equals, float64(1))

	// Get httpRequest size count value equals 1.
	requestCount, found := GetMetricValue(c, fmt.Sprintf(requestSizeCount, 200, "/_ping", "get"))
	c.Assert(found, check.Equals, true)
	c.Assert(requestCount, check.Equals, float64(1))
}
