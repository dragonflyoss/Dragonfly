package main

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/dragonflyoss/Dragonfly/test/request"

	"github.com/go-check/check"
)

// CheckRespStatus checks the http.Response.Status is equal to status.
func CheckRespStatus(c *check.C, resp *http.Response, status int) {
	if resp.StatusCode != status {
		body, err := ioutil.ReadAll(resp.Body)
		c.Assert(err, check.IsNil)
		c.Assert(resp.StatusCode, check.Equals, status, check.Commentf("Response Body: %v", string(body)))
	}
}

// CheckMetric find the specific metric from /metrics endpoint and it will compare the metric
// value with expected value.
func CheckMetric(c *check.C, metric string, value float64) {
	var val float64
	resp, err := request.Get("/metrics")
	c.Assert(err, check.IsNil)
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, check.IsNil)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.Contains(line, metric) {
			vals := strings.Split(line, " ")
			if len(vals) != 2 {
				c.Errorf("bad metrics format")
			}
			val, err = strconv.ParseFloat(vals[1], 64)
			c.Assert(err, check.IsNil)
			c.Assert(val, check.Equals, value)
			return
		}
	}

	// Cannot find expected metric and fail the test.
	c.FailNow()
}
