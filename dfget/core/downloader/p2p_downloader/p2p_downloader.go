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

package downloader

import (
	"bytes"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/dragonflyoss/Dragonfly/common/constants"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/downloader"
	backDown "github.com/dragonflyoss/Dragonfly/dfget/core/downloader/back_downloader"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/core/regist"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/dfget/util"

	"github.com/sirupsen/logrus"
)

const (
	reset = "reset"
	last  = "last"
)

var (
	uploaderAPI = api.NewUploaderAPI(cutil.DefaultTimeout)
)

// P2PDownloader is one implementation of Downloader that uses p2p pattern
// to download files.
type P2PDownloader struct {
	// API holds the instance of SupernodeAPI to interact with supernode.
	API api.SupernodeAPI
	// Register holds the instance of SupernodeRegister.
	Register regist.SupernodeRegister
	// RegisterResult indicates the result set of registering to supernode.
	RegisterResult *regist.RegisterResult

	cfg *config.Config

	// node indicates the IP address of the currently registered supernode.
	node string
	// taskID a string which represents a unique task.
	taskID string
	// targetFile indicates the full target path whose value is equal to the `Output`.
	targetFile string
	// taskFileName is a string composed of `the last element of RealTarget path + "-" + sign`.
	taskFileName string

	pieceSizeHistory [2]int32
	// queue maintains a queue of tasks that to be downloaded.
	// The downloader will get download tasks from supernode and put them into this queue.
	// And the downloader will poll values from this queue constantly and do the actual download actions.
	queue util.Queue
	// clientQueue maintains a queue of tasks that need to be written to disk.
	// The downloader will put the piece into this queue after it downloaded a piece successfully.
	// And clientWriter will poll values from this queue constantly and write to disk.
	clientQueue util.Queue

	// clientFilePath is the full path of the temp file.
	clientFilePath string
	// serviceFilePath is the full path of the temp service file which
	// always ends with ".service".
	serviceFilePath string

	// pieceSet range -> bool
	// true: if the range is processed successfully
	// false: if the range is in processing
	// not in: the range hasn't been processed
	pieceSet map[string]bool
	// total indicates the total length of the downloaded file.
	total int64

	// rateLimiter limit the download speed.
	rateLimiter *cutil.RateLimiter
	// pullRateTime the time when the pull rate API is called to
	// control the time interval between two calls to the API.
	pullRateTime time.Time
}

var _ downloader.Downloader = &P2PDownloader{}

// NewP2PDownloader create P2PDownloader
func NewP2PDownloader(cfg *config.Config,
	api api.SupernodeAPI,
	register regist.SupernodeRegister,
	result *regist.RegisterResult) *P2PDownloader {
	p2p := &P2PDownloader{
		cfg:            cfg,
		API:            api,
		Register:       register,
		RegisterResult: result,
	}
	p2p.init()
	return p2p
}

func (p2p *P2PDownloader) init() {
	p2p.node = p2p.RegisterResult.Node
	p2p.taskID = p2p.RegisterResult.TaskID
	p2p.targetFile = p2p.cfg.RV.RealTarget
	p2p.taskFileName = p2p.cfg.RV.TaskFileName

	p2p.pieceSizeHistory[0], p2p.pieceSizeHistory[1] =
		p2p.RegisterResult.PieceSize, p2p.RegisterResult.PieceSize

	p2p.queue = util.NewQueue(0)
	p2p.queue.Put(NewPieceSimple(p2p.taskID, p2p.node, constants.TaskStatusStart))

	p2p.clientQueue = util.NewQueue(p2p.cfg.ClientQueueSize)

	p2p.clientFilePath = helper.GetTaskFile(p2p.taskFileName, p2p.cfg.RV.DataDir)
	p2p.serviceFilePath = helper.GetServiceFile(p2p.taskFileName, p2p.cfg.RV.DataDir)

	p2p.pieceSet = make(map[string]bool)

	p2p.rateLimiter = cutil.NewRateLimiter(int32(p2p.cfg.LocalLimit), 2)
	p2p.pullRateTime = time.Now().Add(-3 * time.Second)
}

// Run starts to download the file.
func (p2p *P2PDownloader) Run() error {
	var (
		lastItem *Piece
		goNext   bool
	)

	// start ClientWriter
	clientWriter, err := NewClientWriter(p2p.clientFilePath, p2p.serviceFilePath, p2p.clientQueue, p2p.API, p2p.cfg)
	if err != nil {
		return err
	}
	go func() {
		clientWriter.Run()
	}()

	for {
		goNext, lastItem = p2p.getItem(lastItem)
		if !goNext {
			continue
		}
		logrus.Infof("downloading piece:%v", lastItem)

		curItem := *lastItem
		curItem.Content = &bytes.Buffer{}
		lastItem = nil

		response, err := p2p.pullPieceTask(&curItem)
		if err == nil {
			code := response.Code
			if code == constants.CodePeerContinue {
				p2p.processPiece(response, &curItem)
			} else if code == constants.CodePeerFinish {
				p2p.finishTask(response, clientWriter)
				return nil
			} else {
				logrus.Warnf("request piece result:%v", response)
				if code == constants.CodeSourceError {
					p2p.cfg.BackSourceReason = config.BackSourceReasonSourceError
				}
			}
		} else {
			logrus.Errorf("download piece fail: %v", err)
			if p2p.cfg.BackSourceReason == 0 {
				p2p.cfg.BackSourceReason = config.BackSourceReasonDownloadError
			}
		}

		// NOTE: Should we call it directly here？
		// Maybe we should return an error and let the caller decide whether to call it.
		if p2p.cfg.BackSourceReason != 0 {
			backDownloader := backDown.NewBackDownloader(p2p.cfg, p2p.RegisterResult)
			return backDownloader.Run()
		}
	}
}

// Cleanup clean all temporary resources generated by executing Run.
func (p2p *P2PDownloader) Cleanup() {
}

// GetNode returns supernode ip.
func (p2p *P2PDownloader) GetNode() string {
	return p2p.node
}

// GetTaskID returns downloading taskID.
func (p2p *P2PDownloader) GetTaskID() string {
	return p2p.taskID
}

func (p2p *P2PDownloader) pullPieceTask(item *Piece) (
	*types.PullPieceTaskResponse, error) {
	var (
		res *types.PullPieceTaskResponse
		err error
	)
	req := &types.PullPieceTaskRequest{
		SrcCid: p2p.cfg.RV.Cid,
		DstCid: item.DstCid,
		Range:  item.Range,
		Result: item.Result,
		Status: item.Status,
		TaskID: item.TaskID,
	}

	for {
		if res, err = p2p.API.PullPieceTask(item.SuperNode, req); err != nil {
			logrus.Errorf("pull piece task error: %v", err)
		} else if res.Code == constants.CodePeerWait {
			sleepTime := time.Duration(rand.Intn(1400)+600) * time.Millisecond
			logrus.Infof("pull piece task result:%s and sleep %.3fs",
				res, sleepTime.Seconds())
			time.Sleep(sleepTime)
			continue
		}
		break
	}

	if res == nil || (res.Code != constants.CodePeerContinue &&
		res.Code != constants.CodePeerFinish &&
		res.Code != constants.CodePeerLimited &&
		res.Code != constants.Success) {
		logrus.Errorf("pull piece task fail:%v and will migrate", res)

		var registerRes *regist.RegisterResult
		if registerRes, err = p2p.Register.Register(p2p.cfg.RV.PeerPort); !cutil.IsNil(err) {
			return nil, err
		}
		p2p.pieceSizeHistory[1] = registerRes.PieceSize
		item.Status = constants.TaskStatusStart
		item.SuperNode = registerRes.Node
		item.TaskID = registerRes.TaskID
		util.Printer.Println("migrated to node:" + item.SuperNode)
		return p2p.pullPieceTask(item)
	}

	return res, err
}

// getPullRate get download rate limit dynamically.
func (p2p *P2PDownloader) getPullRate(data *types.PullPieceTaskResponseContinueData) {
	if time.Since(p2p.pullRateTime).Seconds() < 3 {
		return
	}
	p2p.pullRateTime = time.Now()

	start := time.Now()
	var localRate int
	if p2p.cfg.LocalLimit > 0 {
		localRate = p2p.cfg.LocalLimit
	} else {
		localRate = data.DownLink * 1024
	}

	// Calculate the download speed limit
	// that the current download task can be assigned
	// by the uploader server.
	req := &api.ParseRateRequest{
		RateLimit:    localRate,
		TaskFileName: p2p.taskFileName,
	}
	resp, err := uploaderAPI.ParseRate(p2p.cfg.RV.LocalIP, p2p.cfg.RV.PeerPort, req)
	if err != nil {
		logrus.Errorf("failed to pullRate: %v", err)
		p2p.rateLimiter.SetRate(cutil.TransRate(localRate))
		return
	}

	reqRate, err := strconv.Atoi(resp)
	if err != nil {
		logrus.Errorf("failed to parse rate from resp %s: %v", resp, err)
		p2p.rateLimiter.SetRate(cutil.TransRate(localRate))
		return
	}
	logrus.Infof("pull rate result:%d cost:%v", reqRate, time.Since(start))
	p2p.rateLimiter.SetRate(cutil.TransRate(reqRate))
}

func (p2p *P2PDownloader) startTask(data *types.PullPieceTaskResponseContinueData) {
	powerClient := &PowerClient{
		taskID:      p2p.taskID,
		node:        p2p.node,
		pieceTask:   data,
		cfg:         p2p.cfg,
		queue:       p2p.queue,
		clientQueue: p2p.clientQueue,
		rateLimiter: p2p.rateLimiter,
		downloadAPI: api.NewDownloadAPI(),
	}
	if err := powerClient.Run(); err != nil && powerClient.ClientError() != nil {
		p2p.API.ReportClientError(p2p.node, powerClient.ClientError())
	}
}

func (p2p *P2PDownloader) getItem(latestItem *Piece) (bool, *Piece) {
	var (
		needMerge = true
	)
	if v, ok := p2p.queue.PollTimeout(2 * time.Second); ok {
		item := v.(*Piece)
		if item.PieceSize != 0 && item.PieceSize != p2p.pieceSizeHistory[1] {
			return false, latestItem
		}
		if item.SuperNode != p2p.node {
			item.DstCid = ""
			item.SuperNode = p2p.node
			item.TaskID = p2p.taskID
		}
		if item.Range != "" {
			v, ok := p2p.pieceSet[item.Range]
			if !ok {
				logrus.Warnf("pieceRange:%s is neither running nor success", item.Range)
				return false, latestItem
			}
			if !v && (item.Result == constants.ResultSemiSuc ||
				item.Result == constants.ResultSuc) {
				p2p.total += int64(item.Content.Len())
				p2p.pieceSet[item.Range] = true
			} else if !v {
				delete(p2p.pieceSet, item.Range)
			}
		}
		latestItem = item
	} else {
		logrus.Warnf("get item timeout(2s) from queue.")
		needMerge = false
	}
	if cutil.IsNil(latestItem) {
		return false, latestItem
	}
	if latestItem.Result == constants.ResultSuc ||
		latestItem.Result == constants.ResultFail ||
		latestItem.Result == constants.ResultInvalid {
		needMerge = false
	}
	runningCount := 0
	for _, v := range p2p.pieceSet {
		if !v {
			runningCount++
		}
	}
	if needMerge && (p2p.queue.Len() > 0 || runningCount > 2) {
		return false, latestItem
	}
	return true, latestItem
}

func (p2p *P2PDownloader) processPiece(response *types.PullPieceTaskResponse,
	item *Piece) {
	var (
		hasTask         = false
		alreadyDownload []string
	)
	p2p.refresh(item)

	data := response.ContinueData()
	logrus.Debugf("pieces to be processed:%v", data)
	for _, pieceTask := range data {
		pieceRange := pieceTask.Range
		v, ok := p2p.pieceSet[pieceRange]
		if ok && v {
			alreadyDownload = append(alreadyDownload, pieceRange)
			p2p.queue.Put(NewPiece(p2p.taskID,
				p2p.node,
				pieceTask.Cid,
				pieceRange,
				constants.ResultSemiSuc,
				constants.TaskStatusRunning))
			continue
		}
		if !ok {
			p2p.pieceSet[pieceRange] = false
			p2p.getPullRate(pieceTask)
			go p2p.startTask(pieceTask)
			hasTask = true
		}
	}
	if !hasTask {
		logrus.Warnf("has not available pieceTask, maybe resource lack")
	}
	if len(alreadyDownload) > 0 {
		logrus.Warnf("already downloaded pieces:%v", alreadyDownload)
	}
}

func (p2p *P2PDownloader) finishTask(response *types.PullPieceTaskResponse, clientWriter *ClientWriter) {
	// wait client writer finished
	logrus.Infof("remaining piece to be written count:%d", p2p.clientQueue.Len())
	p2p.clientQueue.Put(last)
	waitStart := time.Now()
	clientWriter.Wait()
	logrus.Infof("wait client writer finish cost:%.3f,main qu size:%d,client qu size:%d",
		time.Since(waitStart).Seconds(), p2p.queue.Len(), p2p.clientQueue.Len())

	if p2p.cfg.BackSourceReason > 0 {
		return
	}

	// get the temp path where the downloaded file exists.
	var src string
	if clientWriter.acrossWrite || !helper.IsP2P(p2p.cfg.Pattern) {
		src = p2p.cfg.RV.TempTarget
	} else {
		if _, err := os.Stat(p2p.clientFilePath); err != nil {
			logrus.Warnf("client file path:%s not found", p2p.clientFilePath)
			if e := cutil.Link(p2p.serviceFilePath, p2p.clientFilePath); e != nil {
				logrus.Warnln("hard link failed, instead of use copy")
				cutil.CopyFile(p2p.serviceFilePath, p2p.clientFilePath)
			}
		}
		src = p2p.clientFilePath
	}

	// move file to the target file path.
	if err := downloader.MoveFile(src, p2p.targetFile, p2p.cfg.Md5); err != nil {
		return
	}
	logrus.Infof("download successfully from dragonfly")
}

func (p2p *P2PDownloader) refresh(item *Piece) {
	needReset := false
	if p2p.pieceSizeHistory[0] != p2p.pieceSizeHistory[1] {
		p2p.pieceSizeHistory[0] = p2p.pieceSizeHistory[1]
		needReset = true
	}

	if needReset {
		p2p.clientQueue.Put(reset)
		for k := range p2p.pieceSet {
			delete(p2p.pieceSet, k)
			p2p.total = 0
			// console log reset
		}
	}
	if p2p.node != item.SuperNode {
		p2p.node = item.SuperNode
		p2p.taskID = item.TaskID
	}
}
