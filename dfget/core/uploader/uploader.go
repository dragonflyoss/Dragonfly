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

// Package uploader implements an uploader server. It is the important role
// - peer - in P2P pattern that will wait for other P2PDownloader to download
// its downloaded files.
package uploader

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/gorilla/mux"
)

const (
	ctype = "application/octet-stream"
)

var (
	syncTaskMap sync.Map
)

// peerServer offer file-block to other clients
type peerServer struct {
	ctx *config.Context

	// server related fields
	host   string
	port   int
	server *http.Server
}

// taskConfig refers to some info about peer task.
type taskConfig struct {
	taskID    string
	cid       string
	dataDir   string
	superNode string
	finished  bool
}

// uploadParam refers to all params needed in the handler of upload.
type uploadParam struct {
	pieceLen int64
	start    int64
	readLen  int64
}

// LaunchPeerServer launch a server to send piece data
func LaunchPeerServer(ctx *config.Context) (int, error) {
	taskFileName := ctx.RV.TaskFileName

	// retrieve peer port from meta file: config.Ctx.RV.MetaPath
	if sevicePort, err := getPort(ctx.RV.MetaPath); err == nil {
		// check the peer port(config.Ctx.RV.PeerPort) whether is available
		url := fmt.Sprintf("http://%s:%d%s%s", ctx.RV.LocalIP, sevicePort, config.LocalHTTPPathCheck, taskFileName)
		result, err := checkPort(url, ctx.RV.TargetDir, util.DefaultTimeout)
		if err == nil {
			ctx.ServerLogger.Infof("local http result:%s for path:%s", result, config.LocalHTTPPathCheck)

			// reuse exist service port
			if result == taskFileName {
				ctx.ServerLogger.Infof("reuse exist service with port:%d", sevicePort)
				return sevicePort, nil
			}
			ctx.ServerLogger.Warnf("not found process on port:%d, version:%s", sevicePort, version.DFGetVersion)
		}
		ctx.ServerLogger.Warnf("request local http path:%s, error:%v", config.LocalHTTPPathCheck, err)
	}
	// TODO: start a goroutine to check alive and sever gc.

	sevicePort := generatePort()
	p2pServer, err := newPeerServer(ctx, sevicePort)
	if err != nil {
		return 0, err
	}

	// TODO: start a new server and roolback if any errors happened.

	// persist the new peer port into meta file
	// NOTE: we should truncates the meta file before service down.
	err = ioutil.WriteFile(ctx.RV.MetaPath, []byte(strconv.Itoa(sevicePort)), 0666)
	if err != nil {
		return 0, err
	}
	ctx.ServerLogger.Infof("server on host: %s, port: %d", p2pServer.host, p2pServer.port)

	return sevicePort, nil
}

// newPeerServer return a new P2PServer.
func newPeerServer(ctx *config.Context, port int) (*peerServer, error) {
	s := &peerServer{
		host: ctx.RV.LocalIP,
		port: port,
	}

	// init router
	r := mux.NewRouter()
	r.HandleFunc(config.PeerHTTPPathPrefix+"{taskFileName:.*}", s.uploadHandler).Methods("GET")
	r.HandleFunc(config.LocalHTTPPathRate+"{taskFileName:.*}", s.parseRateHandler).Methods("GET")
	r.HandleFunc(config.LocalHTTPPathCheck+"{taskFileName:.*}", s.checkHandler).Methods("GET")
	r.HandleFunc(config.LocalHTTPPathClient+"finish", s.oneFinishHandler).Methods("GET")

	s.server = &http.Server{
		Addr:    net.JoinHostPort(s.host, strconv.Itoa(port)),
		Handler: r,
	}

	return s, nil
}

// uploadHandler use to upload a task file when other peers download from it.
func (ps *peerServer) uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Step1: parse param
	taskFileName := mux.Vars(r)["taskFileName"]
	rangeStr := r.Header.Get("range")
	params, err := parseRange(rangeStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		ps.ctx.ServerLogger.Errorf("failed to parse param from request %v, %v", r, err)
	}

	// Step2: get task file
	f, err := getTaskFile(taskFileName)
	if f == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, err.Error())
		ps.ctx.ServerLogger.Errorf("failed to open TaskFile %s, %v", taskFileName, err)
	}
	defer f.Close()

	// Step3: write header
	w.Header().Set("Content-Length", strconv.FormatInt(params.pieceLen, 10))
	sendSuccess(w)

	// Step4: tans task file
	if err := transFile(f, w, params.start, params.readLen); err != nil {
		ps.ctx.ServerLogger.Errorf("send range:%s error: %v", rangeStr, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "read task file failed: %v", err)
	}
}

// TODO: implement it.
func (ps *peerServer) parseRateHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "not implemented yet")
}

// checkHandler use to check the server status.
func (ps *peerServer) checkHandler(w http.ResponseWriter, r *http.Request) {
	sendSuccess(w)

	// get parameters
	taskFileName := mux.Vars(r)["taskFileName"]
	dataDir := r.Header.Get("dataDir")

	param := &taskConfig{
		dataDir:  dataDir,
		finished: false,
	}
	syncTaskMap.LoadOrStore(taskFileName, param)
	fmt.Fprintf(w, "%s@%s", taskFileName, version.DFGetVersion)
}

// oneFinishHandler use to update the status of peer task.
func (ps *peerServer) oneFinishHandler(w http.ResponseWriter, r *http.Request) {
	taskFileName := mux.Vars(r)["taskFileName"]
	param := &taskConfig{
		taskID:    mux.Vars(r)["taskId"],
		cid:       mux.Vars(r)["cid"],
		superNode: mux.Vars(r)["superNode"],
		finished:  true,
	}
	syncTaskMap.LoadOrStore(taskFileName, param)
	fmt.Fprintf(w, "success")
}

func sendSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-type", ctype)
	w.WriteHeader(http.StatusOK)
}
