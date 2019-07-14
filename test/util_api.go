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

// GetMetricValue get the specific metrics from /metrics endpoint,
// if cannot find this metrics, return not found.
func GetMetricValue(c *check.C, key string) (float64, bool) {
	var val float64
	resp, err := request.Get("/metrics")
	c.Assert(err, check.IsNil)
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, check.IsNil)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.Contains(line, key) {
			val, err = strconv.ParseFloat(strings.Split(line, " ")[1], 64)
			c.Assert(err, check.IsNil)
			return val, true
		}
	}

	return val, false
}
