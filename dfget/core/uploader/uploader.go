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
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/gorilla/mux"
)

const (
	ctype = "application/octet-stream"
)

var (
	p2p *peerServer

	syncTaskMap sync.Map
	aliveQueue  = util.NewQueue(0)
)

// StartPeerServerProcess starts an independent peer server process for uploading downloaded files
// if it doesn't exist.
// This function is invoked when dfget starts to download files in p2p pattern.
func StartPeerServerProcess(cfg *config.Config) (port int, err error) {
	if port = checkPeerServerExist(cfg, 0); port > 0 {
		return port, nil
	}

	cmd := exec.Command(os.Args[0], "server",
		"--ip", cfg.RV.LocalIP,
		"--meta", cfg.RV.MetaPath,
		"--data", cfg.RV.SystemDataDir,
		"--expiretime", cfg.RV.DataExpireTime.String(),
		"--alivetime", cfg.RV.ServerAliveTime.String())

	var stdout io.ReadCloser
	if stdout, err = cmd.StdoutPipe(); err != nil {
		return 0, err
	}
	if err = cmd.Start(); err == nil {
		port, err = readPort(stdout)
	}
	if err == nil && checkPeerServerExist(cfg, port) <= 0 {
		err = fmt.Errorf("invalid server on port:%d", port)
		port = 0
	}

	return
}

func readPort(r io.Reader) (port int, err error) {
	done := make(chan struct{})
	go func() {
		var n = 0
		buf := make([]byte, 256)
		n, err = r.Read(buf)
		if err == nil {
			content := strings.TrimSpace(string(buf[:n]))
			if port, err = strconv.Atoi(content); err != nil {
				err = fmt.Errorf("%s", content)
			}
		}
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		err = fmt.Errorf("get peer server's port timeout")
		close(done)
	}

	return
}

// checkPeerServerExist checks the peer server on port whether is available.
// if the parameter port <= 0, it will get port from meta file and checks.
func checkPeerServerExist(cfg *config.Config, port int) int {
	taskFileName := cfg.RV.TaskFileName
	if port <= 0 {
		port = getPortFromMeta(cfg.RV.MetaPath)
	}

	// check the peer server whether is available
	result, err := checkServer(cfg.RV.LocalIP, port, cfg.RV.TargetDir, taskFileName, 0)
	cfg.ServerLogger.Infof("local http result:%s err:%v, port:%d path:%s",
		result, err, port, config.LocalHTTPPathCheck)

	if err == nil {
		if result == taskFileName {
			cfg.ServerLogger.Infof("use peer server on port:%d", port)
			return port
		}
		cfg.ServerLogger.Warnf("not found process on port:%d, version:%s", port, version.DFGetVersion)
	}
	return 0
}

// ----------------------------------------------------------------------------
// dfget server functions

// WaitForShutdown wait for peer server shutdown
func WaitForShutdown() {
	if p2p != nil {
		<-p2p.finished
	}
}

// LaunchPeerServer launch a server to send piece data
func LaunchPeerServer(cfg *config.Config) (int, error) {
	var (
		servicePort = 0
		retryCount  = 10
		c           = make(chan error)
	)

	if cfg.RV.PeerPort > 0 {
		servicePort = cfg.RV.PeerPort
		retryCount = 1
	}

	cfg.ServerLogger.Infof("********************")
	cfg.ServerLogger.Infof("start peer server...")
	go func() {
		var err error
		shouldGeneratePort := servicePort <= 0
		for i := 0; i < retryCount; i++ {
			if shouldGeneratePort {
				servicePort = generatePort(i)
			}
			p2p = newPeerServer(cfg, servicePort)
			if err = p2p.ListenAndServe(); err != nil {
				if strings.Index(err.Error(), "address already in use") < 0 {
					// start failed
					c <- err
					return
				} else if pingServer(p2p.host, p2p.port) {
					// a peer server is already existing
					c <- nil
					close(p2p.finished)
					cfg.ServerLogger.Infof("reuse exist service with port:%d", servicePort)
					return
				}
				cfg.ServerLogger.Warnf("start error:%v, remain retry times:%d",
					err, retryCount-i)
			}
		}
		// send last error
		c <- err
	}()

	var err error
	if err = waitTimeout(c, 100*time.Millisecond); err == nil {
		updateServicePortInMeta(cfg, servicePort)
		cfg.ServerLogger.Infof("start peer server success, host:%s, port:%d",
			cfg.RV.LocalIP, servicePort)
		go monitorAlive(cfg, 15*time.Second)
		return servicePort, nil
	}
	cfg.ServerLogger.Errorf("start peer server error:%v, exit directly", err)
	return 0, err
}

func waitTimeout(c chan error, timeout time.Duration) error {
	select {
	case err := <-c:
		return err
	case <-time.After(timeout):
		return nil
	}
}

func updateServicePortInMeta(cfg *config.Config, port int) {
	meta := config.NewMetaData(cfg.RV.MetaPath)
	meta.Load()
	if meta.ServicePort != port {
		meta.ServicePort = port
		meta.Persist()
	}
}

func serverGC(cfg *config.Config, interval time.Duration) {
	cfg.ServerLogger.Info("start server gc, expireTime:", cfg.RV.DataExpireTime)

	supernode := api.NewSupernodeAPI()
	var walkFn filepath.WalkFunc = func(path string, info os.FileInfo, err error) error {
		if path == cfg.RV.SystemDataDir || info == nil || err != nil {
			return nil
		}
		if info.IsDir() {
			os.RemoveAll(path)
			return filepath.SkipDir
		}
		if deleteExpiredFile(supernode, path, info, cfg.RV.DataExpireTime) {
			cfg.ServerLogger.Info("server gc, delete file:", path)
		}
		return nil
	}

	for {
		if err := filepath.Walk(cfg.RV.SystemDataDir, walkFn); err != nil {
			cfg.ServerLogger.Warnf("server gc error:%v", err)
		}
		time.Sleep(interval)
	}
}

func deleteExpiredFile(api api.SupernodeAPI, path string, info os.FileInfo,
	expireTime time.Duration) bool {
	taskName := helper.GetTaskName(info.Name())
	if v, ok := syncTaskMap.Load(taskName); ok {
		task, ok := v.(*taskConfig)
		if ok && !task.finished {
			return false
		}
		if time.Now().Sub(info.ModTime()) > expireTime {
			if ok {
				api.ServiceDown(task.superNode, task.taskID, task.cid)
			}
			os.Remove(path)
			syncTaskMap.Delete(taskName)
			return true
		}
	} else {
		os.Remove(path)
		return true
	}
	return false
}

func monitorAlive(cfg *config.Config, interval time.Duration) {
	cfg.ServerLogger.Info("monitor peer server whether is alive, aliveTime:",
		cfg.RV.ServerAliveTime)
	go serverGC(cfg, interval)

	for {
		if _, ok := aliveQueue.PollTimeout(cfg.RV.ServerAliveTime); !ok {
			if aliveQueue.Len() > 0 {
				continue
			}
			cfg.ServerLogger.Info("no more task, peer server will stop...")
			if p2p != nil {
				c, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
				p2p.Shutdown(c)
				cancel()
				close(p2p.finished)
				updateServicePortInMeta(cfg, 0)
			}
			cfg.ServerLogger.Info("peer server is shutdown.")
			return
		}
	}
}

// ----------------------------------------------------------------------------
// peerServer structure

// newPeerServer return a new P2PServer.

func newPeerServer(cfg *config.Config, port int) *peerServer {
	s := &peerServer{
		cfg:      cfg,
		finished: make(chan struct{}),
		host:     cfg.RV.LocalIP,
		port:     port,
	}

	// init router
	r := mux.NewRouter()
	r.HandleFunc(config.PeerHTTPPathPrefix+"{taskFileName:.*}", s.uploadHandler).Methods("GET")
	r.HandleFunc(config.LocalHTTPPathRate+"{taskFileName:.*}", s.parseRateHandler).Methods("GET")
	r.HandleFunc(config.LocalHTTPPathCheck+"{taskFileName:.*}", s.checkHandler).Methods("GET")
	r.HandleFunc(config.LocalHTTPPathClient+"finish", s.oneFinishHandler).Methods("GET")
	r.HandleFunc(config.LocalHTTPPing, s.pingHandler).Methods("GET")

	s.Server = &http.Server{
		Addr:    net.JoinHostPort(s.host, strconv.Itoa(port)),
		Handler: r,
	}

	return s
}

// peerServer offer file-block to other clients
type peerServer struct {
	cfg      *config.Config
	finished chan struct{}

	// server related fields
	host string
	port int
	*http.Server
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

// uploadHandler use to upload a task file when other peers download from it.
func (ps *peerServer) uploadHandler(w http.ResponseWriter, r *http.Request) {
	aliveQueue.Put(true)
	// Step1: parse param
	taskFileName := mux.Vars(r)["taskFileName"]
	rangeStr := r.Header.Get("range")
	params, err := parseRange(rangeStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		ps.cfg.ServerLogger.Errorf("failed to parse param from request %v, %v", r, err)
	}

	// Step2: get task file
	f, err := getTaskFile(taskFileName)
	if f == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, err.Error())
		ps.cfg.ServerLogger.Errorf("failed to open TaskFile %s, %v", taskFileName, err)
	}
	defer f.Close()

	// Step3: write header
	w.Header().Set("Content-Length", strconv.FormatInt(params.pieceLen, 10))
	sendSuccess(w)

	// Step4: tans task file
	if err := transFile(f, w, params.start, params.readLen); err != nil {
		ps.cfg.ServerLogger.Errorf("send range:%s error: %v", rangeStr, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "read task file failed: %v", err)
	}
}

// TODO: implement it.
func (ps *peerServer) parseRateHandler(w http.ResponseWriter, r *http.Request) {
	aliveQueue.Put(true)
	fmt.Fprintf(w, "not implemented yet")
}

// checkHandler use to check the server status.
func (ps *peerServer) checkHandler(w http.ResponseWriter, r *http.Request) {
	aliveQueue.Put(true)
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
	if err := r.ParseForm(); err != nil {
		sendHeader(w, http.StatusBadRequest)
		fmt.Fprintf(w, err.Error())
		return
	}

	taskFileName := r.FormValue("taskFileName")
	taskID := r.FormValue("taskId")
	cid := r.FormValue("cid")
	superNode := r.FormValue("superNode")
	if v, ok := syncTaskMap.Load(taskFileName); ok {
		task := v.(*taskConfig)
		task.taskID = taskID
		task.cid = cid
		task.superNode = superNode
		task.finished = true
	}
	sendSuccess(w)
	fmt.Fprintf(w, "success")
}

func (ps *peerServer) pingHandler(w http.ResponseWriter, r *http.Request) {
	sendSuccess(w)
	fmt.Fprintf(w, "success")
}

// ----------------------------------------------------------------------------
// helper functions

func sendSuccess(w http.ResponseWriter) {
	sendHeader(w, http.StatusOK)
}

func sendHeader(w http.ResponseWriter, code int) {
	w.Header().Set("Content-type", ctype)
	w.WriteHeader(code)
}
