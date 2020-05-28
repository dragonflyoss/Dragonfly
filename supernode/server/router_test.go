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

package server

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	_ "github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/cdn"
	_ "github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/sourcecdn"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/go-check/check"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	prom_testutil "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
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
			CDNPattern: config.CDNPatternLocal,
		},
		Plugins:  nil,
		Storages: nil,
	}
	s, err := New(testConf, logrus.StandardLogger(), prometheus.NewRegistry())
	c.Check(err, check.IsNil)
	version.DFVersion = &types.DragonflyVersion{
		Version:   "test",
		Revision:  "test",
		Arch:      runtime.GOARCH,
		OS:        runtime.GOOS,
		GoVersion: runtime.Version(),
	}

	rs.router = createRouter(s)
	rs.listener, err = net.Listen("tcp", rs.addr)
	c.Check(err, check.IsNil)
	go http.Serve(rs.listener, rs.router)
}

func (rs *RouterTestSuite) TearDownSuite(c *check.C) {
	err := rs.listener.Close()
	c.Check(err, check.IsNil)
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
		{"/debug/pprof/cmdline", 200},
		{"/debug/pprof/trace", 200},

		// path not exists
		{"/debug/pprof/foo", 404},
	} {
		code, _, err := httputils.Get("http://"+rs.addr+tc.url, 0)
		c.Check(err, check.IsNil)
		c.Assert(code, check.Equals, tc.code)
	}
}

func (rs *RouterTestSuite) TestVersionHandler(c *check.C) {
	code, res, err := httputils.Get("http://"+rs.addr+"/version", 0)
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
	code, _, err := httputils.Get("http://"+rs.addr+"/metrics", 0)
	c.Check(err, check.IsNil)
	c.Assert(code, check.Equals, 200)

	counter := m.requestCounter
	c.Assert(1, check.Equals,
		int(prom_testutil.ToFloat64(counter.WithLabelValues(strconv.Itoa(http.StatusOK), "/metrics"))))

	for i := 0; i < 5; i++ {
		code, _, err := httputils.Get("http://"+rs.addr+"/_ping", 0)
		c.Check(err, check.IsNil)
		c.Assert(code, check.Equals, 200)
		c.Assert(i+1, check.Equals,
			int(prom_testutil.ToFloat64(counter.WithLabelValues(strconv.Itoa(http.StatusOK), "/_ping"))))
	}
}
