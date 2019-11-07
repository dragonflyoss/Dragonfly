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

package uploader

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/go-check/check"
	"github.com/gorilla/mux"
)

func init() {
	check.Suite(&UploaderUtilTestSuite{})
}

type UploaderUtilTestSuite struct {
	workHome string
	ip       string
	port     int
	server   *http.Server
}

func (s *UploaderUtilTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "dfget-UploaderUtilTestSuite-")
	s.startTestServer()
}

func (s *UploaderUtilTestSuite) TearDownSuite(c *check.C) {
	stopTestServer(s.server)
	if s.workHome != "" {
		if err := os.RemoveAll(s.workHome); err != nil {
			fmt.Printf("remove path:%s error", s.workHome)
		}
	}
}

// -----------------------------------------------------------------------------
// tests

func (s *UploaderUtilTestSuite) TestFinishTask(c *check.C) {
	e := FinishTask(s.ip, s.port, "a", "", "", "")
	c.Assert(e, check.IsNil)

	e = FinishTask(s.ip, s.port, "b", "", "", "")
	c.Assert(e, check.NotNil)
	c.Assert(e.Error(), check.Equals, "400:bad request")
}

func (s *UploaderUtilTestSuite) TestCheckServer(c *check.C) {
	// normal test
	result, err := checkServer(s.ip, s.port, s.workHome, commonFile, 0)
	c.Check(err, check.IsNil)
	c.Check(result, check.Equals, commonFile)

	// error url test
	result, err = checkServer(s.ip+"1", s.port, s.workHome, commonFile, 0)
	c.Check(err, check.NotNil)
	c.Check(result, check.Equals, "")
}

func (s *UploaderUtilTestSuite) TestGeneratePort(c *check.C) {
	port := generatePort(0)
	c.Assert(port >= config.ServerPortLowerLimit, check.Equals, true)
	c.Assert(port <= config.ServerPortUpperLimit, check.Equals, true)
}

func (s *UploaderUtilTestSuite) TestGetPort(c *check.C) {
	metaPath := filepath.Join(s.workHome, "meta")
	port := getPortFromMeta(metaPath)
	c.Assert(port, check.Equals, 0)

	servicePort := 8080
	meta := config.NewMetaData(metaPath)
	meta.ServicePort = servicePort
	err := meta.Persist()
	c.Check(err, check.IsNil)

	port = getPortFromMeta(metaPath)
	c.Assert(port, check.Equals, servicePort)
}

func (s *UploaderUtilTestSuite) TestUpdateServicePortInMeta(c *check.C) {
	expectedPort := 80
	metaPath := filepath.Join(s.workHome, "meta")
	updateServicePortInMeta(metaPath, expectedPort)
	port := getPortFromMeta(metaPath)
	c.Assert(port, check.Equals, expectedPort)
}

// -----------------------------------------------------------------------------
// helper functions

func (s *UploaderUtilTestSuite) startTestServer() {
	checkHandler := func(w http.ResponseWriter, r *http.Request) {
		fileName := mux.Vars(r)["commonFile"]
		fmt.Fprintf(w, "%s@%s", fileName, version.DFGetVersion)
	}

	finishHandler := func(w http.ResponseWriter, r *http.Request) {
		if e := r.ParseForm(); e == nil {
			fileName := r.FormValue(config.StrTaskFileName)
			if fileName == "a" {
				sendSuccess(w)
				return
			}
		}
		sendHeader(w, http.StatusBadRequest)
		fmt.Fprint(w, "bad request")
	}

	r := mux.NewRouter()
	r.HandleFunc(config.LocalHTTPPathCheck+"{commonFile:.*}", checkHandler).Methods("GET")
	r.HandleFunc(config.LocalHTTPPathClient+"finish", finishHandler).Methods("GET")

	s.ip, s.port, s.server = startTestServer(r)
}
