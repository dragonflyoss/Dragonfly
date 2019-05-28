package main

import (
	"fmt"

	"github.com/dragonflyoss/Dragonfly/test/command"
	"github.com/dragonflyoss/Dragonfly/test/request"

	"github.com/go-check/check"
)

// APIPingSuite is the test suite for info related API.
type APIPingSuite struct {
	starter *command.Starter
}

func init() {
	check.Suite(&APIPingSuite{})
}

// SetUpTest does common setup in the beginning of each test.
func (s *APIPingSuite) SetUpTest(c *check.C) {
	s.starter = command.NewStarter("SupernodeAPITestSuite")
	if _, err := s.starter.Supernode(0); err != nil {
		panic(fmt.Sprintf("start supernode failed:%v", err))
	}
}

func (s *APIPingSuite) TearDownSuite(c *check.C) {
	s.starter.Clean()
}

// TestPing tests /info API.
func (s *APIPingSuite) TestPing(c *check.C) {
	resp, err := request.Get("/_ping")
	c.Assert(err, check.IsNil)
	defer resp.Body.Close()

	CheckRespStatus(c, resp, 200)
}
