package server

import (
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	rand.Seed(time.Now().Unix())
	check.Suite(&RouterTestSuite{})
}

type RouterTestSuite struct {
	addr     string
	listener net.Listener
}

func (rs *RouterTestSuite) SetUpSuite(c *check.C) {
	port := rand.Intn(1000) + 63000
	rs.addr = "127.0.0.1:" + strconv.Itoa(port)
	tmpDir, err := ioutil.TempDir("/tmp", "supernode-routerTestSuite-")
	c.Check(err, check.IsNil)

	testConf := &config.Config{
		BaseProperties: &config.BaseProperties{
			ListenPort: port,
			Debug:      true,
			HomeDir:    tmpDir,
		},
		Plugins:  nil,
		Storages: nil,
	}
	s, err := New(testConf)
	c.Check(err, check.IsNil)
	router := initRoute(s)
	rs.listener, err = net.Listen("tcp", rs.addr)
	c.Check(err, check.IsNil)
	go http.Serve(rs.listener, router)
}

func (rs *RouterTestSuite) TearDownSuite(c *check.C) {
	rs.listener.Close()
}

func (rs *RouterTestSuite) TestDebugHandler(c *check.C) {
	for _, tc := range []struct {
		url  string
		code int
	}{
		{"/debug/pprof/allocs", 200},
		{"/debug/pprof/block", 200},
		{"/debug/pprof/goroutine", 200},
		{"/debug/pprof/heap", 200},
		{"/debug/pprof/mutex", 200},
		{"/debug/pprof/threadcreate", 200},

		// these paths exist but will get 404 because of unknown profile
		{"/debug/pprof/cmdline", 404},
		{"/debug/pprof/profile", 404},
		{"/debug/pprof/trace", 404},

		// path not exists
		{"/debug/pprof/foo", 404},
	} {
		code, _, err := cutil.Get("http://"+rs.addr+tc.url, 0)
		c.Check(err, check.IsNil)
		c.Assert(code, check.Equals, tc.code)
	}
}
