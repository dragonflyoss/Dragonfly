package server

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/go-check/check"
	prom_testutil "github.com/prometheus/client_golang/prometheus/testutil"
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
	router   *mux.Router
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
	version.DFVersion = &types.DragonflyVersion{
		Version:   "test",
		Revision:  "test",
		Arch:      runtime.GOARCH,
		OS:        runtime.GOOS,
		GoVersion: runtime.Version(),
	}

	rs.router = initRoute(s)
	rs.listener, err = net.Listen("tcp", rs.addr)
	c.Check(err, check.IsNil)
	go http.Serve(rs.listener, rs.router)
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

func (rs *RouterTestSuite) TestVersionHandler(c *check.C) {
	code, res, err := cutil.Get("http://"+rs.addr+"/version", 0)
	c.Check(err, check.IsNil)
	c.Assert(code, check.Equals, 200)

	expectDFVersion, err := json.Marshal(&types.DragonflyVersion{
		Version:   "test",
		Revision:  "test",
		Arch:      runtime.GOARCH,
		OS:        runtime.GOOS,
		GoVersion: runtime.Version(),
	})

	c.Check(err, check.IsNil)
	c.Check(string(expectDFVersion), check.Equals, string(res))
}

func (rs *RouterTestSuite) TestHTTPMetrics(c *check.C) {
	// ensure /metrics is accessible
	code, _, err := cutil.Get("http://"+rs.addr+"/metrics", 0)
	c.Check(err, check.IsNil)
	c.Assert(code, check.Equals, 200)

	counter := m.requestCounter
	c.Assert(1, check.Equals,
		int(prom_testutil.ToFloat64(counter.WithLabelValues("/metrics", strconv.Itoa(http.StatusOK)))))

	for i := 0; i < 5; i++ {
		code, _, err := cutil.Get("http://"+rs.addr+"/_ping", 0)
		c.Check(err, check.IsNil)
		c.Assert(code, check.Equals, 200)
		c.Assert(i+1, check.Equals,
			int(prom_testutil.ToFloat64(counter.WithLabelValues("/_ping", strconv.Itoa(http.StatusOK)))))
	}
}
