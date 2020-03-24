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
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	apiTypes "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/limitreader"
	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// newPeerServer returns a new P2PServer.
func newPeerServer(cfg *config.Config, port int) *peerServer {
	s := &peerServer{
		cfg:      cfg,
		finished: make(chan struct{}),
		host:     cfg.RV.LocalIP,
		port:     port,
		api:      api.NewSupernodeAPI(),
	}

	r := s.initRouter()
	s.Server = &http.Server{
		Addr:    net.JoinHostPort(s.host, strconv.Itoa(port)),
		Handler: r,
	}

	return s
}

// ----------------------------------------------------------------------------
// peerServer structure

// peerServer offers file-block to other clients.
type peerServer struct {
	cfg *config.Config

	// finished indicates whether the peer server is shutdown
	finished chan struct{}

	// server related fields
	host string
	port int
	*http.Server

	api         api.SupernodeAPI
	rateLimiter *ratelimiter.RateLimiter

	// totalLimitRate is the total network bandwidth shared by tasks on the same host
	totalLimitRate int

	// syncTaskMap stores the meta name of tasks on the host
	syncTaskMap sync.Map
}

// taskConfig refers to some name about peer task.
type taskConfig struct {
	taskID     string
	rateLimit  int
	cid        string
	dataDir    string
	superNode  string
	finished   bool
	accessTime time.Time
}

// uploadParam refers to all params needed in the handler of upload.
type uploadParam struct {
	padSize int64
	start   int64
	length  int64

	pieceSize int64
	pieceNum  int64
}

// ----------------------------------------------------------------------------
// init method of peerServer

func (ps *peerServer) initRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc(config.PeerHTTPPathPrefix+"{commonFile:.*}", ps.uploadHandler).Methods("GET")
	r.HandleFunc(config.LocalHTTPPathRate+"{commonFile:.*}", ps.parseRateHandler).Methods("GET")
	r.HandleFunc(config.LocalHTTPPathCheck+"{commonFile:.*}", ps.checkHandler).Methods("GET")
	r.HandleFunc(config.LocalHTTPPathClient+"finish", ps.oneFinishHandler).Methods("GET")
	r.HandleFunc(config.LocalHTTPPing, ps.pingHandler).Methods("GET")

	return r
}

// ----------------------------------------------------------------------------
// peerServer handlers

// uploadHandler uses to upload a task file when other peers download from it.
func (ps *peerServer) uploadHandler(w http.ResponseWriter, r *http.Request) {
	sendAlive(ps.cfg)

	var (
		up   *uploadParam
		f    *os.File
		size int64
		err  error
	)

	taskFileName := mux.Vars(r)["commonFile"]
	rangeStr := r.Header.Get(config.StrRange)
	cdnSource := r.Header.Get(config.StrCDNSource)

	logrus.Debugf("upload file:%s to %s, req:%v", taskFileName, r.RemoteAddr, jsonStr(r.Header))

	// Step1: parse param
	if up, err = parseParams(rangeStr, r.Header.Get(config.StrPieceNum),
		r.Header.Get(config.StrPieceSize)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logrus.Warnf("invalid param file:%s req:%v, %v", taskFileName, r.Header, err)
		return
	}

	// Step2: get task file
	if f, size, err = ps.getTaskFile(taskFileName); err != nil {
		rangeErrorResponse(w, err)
		logrus.Errorf("failed to open file:%s, %v", taskFileName, err)
		return
	}
	defer f.Close()

	// Step3: amend range with piece meta data
	if err = amendRange(size, cdnSource != string(apiTypes.CdnSourceSource), up); err != nil {
		rangeErrorResponse(w, err)
		logrus.Errorf("failed to amend range of file %s: %v", taskFileName, err)
		return
	}

	// Step4: send piece wrapped by meta data
	if err := ps.uploadPiece(f, w, up); err != nil {
		logrus.Errorf("failed to send range(%s) of file(%s): %v", rangeStr, taskFileName, err)
	}
}

func (ps *peerServer) parseRateHandler(w http.ResponseWriter, r *http.Request) {
	sendAlive(ps.cfg)

	// get params from request
	taskFileName := mux.Vars(r)["commonFile"]
	rateLimit := r.Header.Get(config.StrRateLimit)
	clientRate, err := strconv.Atoi(rateLimit)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		logrus.Errorf("failed to convert rateLimit %v, %v", rateLimit, err)
		return
	}
	sendSuccess(w)

	// update the rateLimit of commonFile
	if v, ok := ps.syncTaskMap.Load(taskFileName); ok {
		param := v.(*taskConfig)
		param.rateLimit = clientRate
	}

	// no need to calculate rate when totalLimitRate less than or equals zero.
	if ps.totalLimitRate <= 0 {
		fmt.Fprint(w, rateLimit)
		return
	}

	clientRate = ps.calculateRateLimit(clientRate)

	fmt.Fprint(w, strconv.Itoa(clientRate))
}

// checkHandler is used to check the server status.
// TODO: Disassemble this function for too many things done.
func (ps *peerServer) checkHandler(w http.ResponseWriter, r *http.Request) {
	sendAlive(ps.cfg)
	sendSuccess(w)

	// handle totalLimit
	totalLimit, err := strconv.Atoi(r.Header.Get(config.StrTotalLimit))
	if err == nil && totalLimit > 0 {
		if ps.rateLimiter == nil {
			ps.rateLimiter = ratelimiter.NewRateLimiter(int64(totalLimit), 2)
		} else {
			ps.rateLimiter.SetRate(ratelimiter.TransRate(int64(totalLimit)))
		}
		ps.totalLimitRate = totalLimit
		logrus.Infof("update total limit to %d", totalLimit)
	}

	// get parameters
	taskFileName := mux.Vars(r)["commonFile"]
	dataDir := r.Header.Get(config.StrDataDir)

	param := &taskConfig{
		dataDir: dataDir,
	}
	ps.syncTaskMap.Store(taskFileName, param)
	fmt.Fprintf(w, "%s@%s", taskFileName, version.DFGetVersion)
}

// oneFinishHandler is used to update the status of peer task.
func (ps *peerServer) oneFinishHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		sendHeader(w, http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

	taskFileName := r.FormValue(config.StrTaskFileName)
	taskID := r.FormValue(config.StrTaskID)
	cid := r.FormValue(config.StrClientID)
	superNode := r.FormValue(config.StrSuperNode)
	if taskFileName == "" || taskID == "" || cid == "" {
		sendHeader(w, http.StatusBadRequest)
		fmt.Fprintf(w, "invalid params")
		return
	}

	if v, ok := ps.syncTaskMap.Load(taskFileName); ok {
		task := v.(*taskConfig)
		task.taskID = taskID
		task.rateLimit = 0
		task.cid = cid
		task.superNode = superNode
		task.finished = true
		task.accessTime = time.Now()
	} else {
		ps.syncTaskMap.Store(taskFileName, &taskConfig{
			taskID:     taskID,
			cid:        cid,
			dataDir:    ps.cfg.RV.SystemDataDir,
			superNode:  superNode,
			finished:   true,
			accessTime: time.Now(),
		})
	}
	sendSuccess(w)
	fmt.Fprintf(w, "success")
}

func (ps *peerServer) pingHandler(w http.ResponseWriter, r *http.Request) {
	sendSuccess(w)
	fmt.Fprintf(w, "success")
}

// ----------------------------------------------------------------------------
// handler process

// getTaskFile finds the file and returns the File object.
func (ps *peerServer) getTaskFile(taskFileName string) (*os.File, int64, error) {
	errSize := int64(-1)

	v, ok := ps.syncTaskMap.Load(taskFileName)
	if !ok {
		return nil, errSize, fmt.Errorf("failed to get taskPath: %s", taskFileName)
	}
	tc, ok := v.(*taskConfig)
	if !ok {
		return nil, errSize, fmt.Errorf("failed to assert: %s", taskFileName)
	}

	// update the accessTime of taskFileName
	tc.accessTime = time.Now()

	taskPath := helper.GetServiceFile(taskFileName, tc.dataDir)

	fileInfo, err := os.Stat(taskPath)
	if err != nil {
		return nil, errSize, err
	}

	taskFile, err := os.Open(taskPath)
	if err != nil {
		return nil, errSize, err
	}
	return taskFile, fileInfo.Size(), nil
}

func amendRange(size int64, needPad bool, up *uploadParam) error {
	up.padSize = 0
	if needPad {
		up.padSize = config.PieceMetaSize
		up.start -= up.pieceNum * up.padSize
	}

	// we must send an whole piece with both piece head and tail
	if up.length < up.padSize || up.start < 0 {
		return errortypes.ErrRangeNotSatisfiable
	}

	if up.start >= size && !needPad {
		return errortypes.ErrRangeNotSatisfiable
	}

	if up.start+up.length-up.padSize > size {
		up.length = size - up.start + up.padSize
		if size == 0 {
			up.length = up.padSize
		}
	}

	return nil
}

// parseParams validates the parameter range and parses it.
func parseParams(rangeVal, pieceNumStr, pieceSizeStr string) (*uploadParam, error) {
	var (
		err error
		up  = &uploadParam{}
	)

	if up.pieceNum, err = strconv.ParseInt(pieceNumStr, 10, 64); err != nil {
		return nil, err
	}

	if up.pieceSize, err = strconv.ParseInt(pieceSizeStr, 10, 64); err != nil {
		return nil, err
	}

	if strings.Count(rangeVal, "=") != 1 {
		return nil, fmt.Errorf("invalid range: %s", rangeVal)
	}
	rangeStr := strings.Split(rangeVal, "=")[1]

	if strings.Count(rangeStr, "-") != 1 {
		return nil, fmt.Errorf("invalid range: %s", rangeStr)
	}
	rangeArr := strings.Split(rangeStr, "-")
	if up.start, err = strconv.ParseInt(rangeArr[0], 10, 64); err != nil {
		return nil, err
	}

	var end int64
	if end, err = strconv.ParseInt(rangeArr[1], 10, 64); err != nil {
		return nil, err
	}

	if end <= up.start {
		return nil, fmt.Errorf("invalid range: %s", rangeStr)
	}
	up.length = end - up.start + 1
	return up, nil
}

// uploadPiece sends a piece of the file to the remote peer.
func (ps *peerServer) uploadPiece(f *os.File, w http.ResponseWriter, up *uploadParam) (e error) {
	w.Header().Set(config.StrContentLength, strconv.FormatInt(up.length, 10))
	sendHeader(w, http.StatusPartialContent)

	readLen := up.length - up.padSize
	buf := make([]byte, 256*1024)

	if up.padSize > 0 {
		binary.BigEndian.PutUint32(buf, uint32((readLen)|(up.pieceSize)<<4))
		w.Write(buf[:config.PieceHeadSize])
		defer w.Write([]byte{config.PieceTailChar})
	}

	f.Seek(up.start, 0)
	r := io.LimitReader(f, readLen)
	if ps.rateLimiter != nil {
		lr := limitreader.NewLimitReaderWithLimiter(ps.rateLimiter, r, false)
		_, e = io.CopyBuffer(w, lr, buf)
	} else {
		_, e = io.CopyBuffer(w, r, buf)
	}

	return
}

func (ps *peerServer) calculateRateLimit(clientRate int) int {
	total := 0

	// define a function that Range will call it sequentially
	// for each key and value present in the map
	f := func(key, value interface{}) bool {
		if task, ok := value.(*taskConfig); ok {
			if !task.finished {
				total += task.rateLimit
			}
		}
		return true
	}
	ps.syncTaskMap.Range(f)

	// calculate the rate limit again according to totalLimit
	if total > ps.totalLimitRate {
		return (clientRate*ps.totalLimitRate + total - 1) / total
	}
	return clientRate
}

// ----------------------------------------------------------------------------
// methods of peerServer

func (ps *peerServer) isFinished() bool {
	if ps.finished == nil {
		return true
	}

	select {
	case _, notClose := <-ps.finished:
		return !notClose
	default:
		return false
	}
}

func (ps *peerServer) setFinished() {
	if !ps.isFinished() {
		close(ps.finished)
	}
}

func (ps *peerServer) waitForShutdown() {
	if ps.finished == nil {
		return
	}
	for {
		select {
		case _, notClose := <-ps.finished:
			if !notClose {
				return
			}
		}
	}
}

func (ps *peerServer) shutdown() {
	// tell supernode this peer node is down and delete related files.
	ps.syncTaskMap.Range(func(key, value interface{}) bool {
		task, ok := value.(*taskConfig)
		if ok {
			ps.api.ServiceDown(task.superNode, task.taskID, task.cid)
			serviceFile := helper.GetServiceFile(key.(string), task.dataDir)
			os.Remove(serviceFile)
			logrus.Infof("shutdown, remove task id:%s file:%s",
				task.taskID, serviceFile)
		}
		return true
	})

	c, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
	ps.Shutdown(c)
	cancel()
	updateServicePortInMeta(ps.cfg.RV.MetaPath, 0)
	logrus.Info("peer server is shutdown.")
	ps.setFinished()
}

func (ps *peerServer) deleteExpiredFile(path string, info os.FileInfo,
	expireTime time.Duration) bool {
	taskName := helper.GetTaskName(info.Name())
	if v, ok := ps.syncTaskMap.Load(taskName); ok {
		task, ok := v.(*taskConfig)
		if ok && !task.finished {
			return false
		}

		var lastAccessTime = task.accessTime
		// use the bigger of access time and modify time to
		// check whether the task is expired
		if task.accessTime.Sub(info.ModTime()) < 0 {
			lastAccessTime = info.ModTime()
		}
		// if the last access time is expireTime ago
		if time.Since(lastAccessTime) > expireTime {
			if ok {
				ps.api.ServiceDown(task.superNode, task.taskID, task.cid)
			}
			os.Remove(path)
			ps.syncTaskMap.Delete(taskName)
			return true
		}
	} else {
		os.Remove(path)
		return true
	}
	return false
}

// ----------------------------------------------------------------------------
// helper functions

func sendSuccess(w http.ResponseWriter) {
	sendHeader(w, http.StatusOK)
}

func sendHeader(w http.ResponseWriter, code int) {
	w.Header().Set(config.StrContentType, ctype)
	w.WriteHeader(code)
}

func rangeErrorResponse(w http.ResponseWriter, err error) {
	if errortypes.IsRangeNotSatisfiable(err) {
		http.Error(w, config.RangeNotSatisfiableDesc, http.StatusRequestedRangeNotSatisfiable)
	} else if os.IsPermission(err) {
		http.Error(w, err.Error(), http.StatusForbidden)
	} else if os.IsNotExist(err) {
		http.Error(w, err.Error(), http.StatusNotFound)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func jsonStr(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
