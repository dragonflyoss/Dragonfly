package main

import (
	"io/ioutil"
	"net/http"

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
