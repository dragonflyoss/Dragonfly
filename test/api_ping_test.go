package main

import (
	"fmt"

	"github.com/dragonflyoss/Dragonfly/test/request"

	"github.com/go-check/check"
)

// APIPingSuite is the test suite for info related API.
type APIPingSuite struct{}

func init() {
	check.Suite(&APIPingSuite{})
}

// SetUpTest does common setup in the beginning of each test.
func (suite *APIPingSuite) SetUpTest(c *check.C) {
}

// TestPing tests /info API.
func (suite *APIPingSuite) TestPing(c *check.C) {
	fmt.Println("start test Ping")
	resp, err := request.Get("/_ping")
	c.Assert(err, check.IsNil)
	defer resp.Body.Close()

	CheckRespStatus(c, resp, 200)
}
