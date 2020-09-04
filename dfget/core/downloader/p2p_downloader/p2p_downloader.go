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
	"context"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"time"

	apiTypes "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/downloader"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/core/regist"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/pkg/constants"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/printer"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"
	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"

	"github.com/sirupsen/logrus"
)

const (
	reset = "reset"
	last  = "last"
)

var (
	uploaderAPI = api.NewUploaderAPI(httputils.DefaultTimeout)
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
	// headers is the extra HTTP headers when downloading the task.
	headers []string

	pieceSizeHistory [2]int32
	// queue maintains a queue of tasks that to be downloaded.
	// The downloader will get download tasks from supernode and put them into this queue.
	// And the downloader will poll values from this queue constantly and do the actual download actions.
	queue queue.Queue
	// clientQueue maintains a queue of tasks that need to be written to disk.
	// The downloader will put the piece into this queue after it downloaded a piece successfully.
	// And clientWriter will poll values from this queue constantly and write to disk.
	clientQueue queue.Queue

	// notifyQueue maintains a queue for notifying p2p downloader to pull next download tasks.
	notifyQueue queue.Queue

	// clientFilePath is the full path of the temp file.
	clientFilePath string
	// serviceFilePath is the full path of the temp service file which
	// always ends with ".service".
	serviceFilePath string

	// streamMode indicates send piece data into a pipe
	// this is useful for use dfget as a library
	streamMode bool

	// pieceSet range -> bool
	// true: if the range is processed successfully
	// false: if the range is in processing
	// not in: the range hasn't been processed
	pieceSet map[string]bool
	// total indicates the total length of the downloaded file.
	total int64

	// rateLimiter limits the download speed.
	rateLimiter *ratelimiter.RateLimiter
	// pullRateTime the time when the pull rate API is called to
	// control the time interval between two calls to the API.
	pullRateTime time.Time

	// dfget will sleep some time which between minTimeout and maxTimeout
	// unit: Millisecond
	minTimeout int
	maxTimeout int
}

var _ downloader.Downloader = &P2PDownloader{}

// NewP2PDownloader creates a P2PDownloader.
func NewP2PDownloader(cfg *config.Config,
	api api.SupernodeAPI,
	register regist.SupernodeRegister,
	result *regist.RegisterResult) *P2PDownloader {
	p2p := &P2PDownloader{
		cfg:            cfg,
		API:            api,
		Register:       register,
		RegisterResult: result,
		minTimeout:     50,
		maxTimeout:     100,
	}
	p2p.init()
	return p2p
}

func (p2p *P2PDownloader) init() {
	p2p.node = p2p.RegisterResult.Node
	p2p.taskID = p2p.RegisterResult.TaskID
	p2p.targetFile = p2p.cfg.RV.RealTarget
	p2p.streamMode = p2p.cfg.RV.StreamMode
	p2p.taskFileName = p2p.cfg.RV.TaskFileName
	if p2p.RegisterResult.CDNSource == apiTypes.CdnSourceSource {
		p2p.headers = p2p.cfg.Header
	}

	p2p.pieceSizeHistory[0], p2p.pieceSizeHistory[1] =
		p2p.RegisterResult.PieceSize, p2p.RegisterResult.PieceSize

	p2p.queue = queue.NewQueue(0)
	p2p.queue.Put(NewPieceSimple(p2p.taskID, p2p.node, constants.TaskStatusStart, p2p.RegisterResult.CDNSource))

	p2p.clientQueue = queue.NewQueue(p2p.cfg.ClientQueueSize)
	p2p.notifyQueue = queue.NewQueue(p2p.cfg.ClientQueueSize)

	p2p.clientFilePath = helper.GetTaskFile(p2p.taskFileName, p2p.cfg.RV.DataDir)
	p2p.serviceFilePath = helper.GetServiceFile(p2p.taskFileName, p2p.cfg.RV.DataDir)

	p2p.pieceSet = make(map[string]bool)

	p2p.rateLimiter = ratelimiter.NewRateLimiter(int64(p2p.cfg.LocalLimit), 2)
	p2p.pullRateTime = time.Now().Add(-3 * time.Second)
}

// Run starts to download the file.
func (p2p *P2PDownloader) Run(ctx context.Context) error {
	if p2p.streamMode {
		return fmt.Errorf("streamMode enabled, should be disable")
	}
	clientWriter := NewClientWriter(p2p.clientFilePath, p2p.serviceFilePath,
		p2p.clientQueue, p2p.notifyQueue,
		p2p.API, p2p.cfg, p2p.RegisterResult.CDNSource)
	return p2p.run(ctx, clientWriter)
}

// RunStream starts to download the file, but return a io.Reader instead of writing a file to local disk.
func (p2p *P2PDownloader) RunStream(ctx context.Context) (io.Reader, error) {
	if !p2p.streamMode {
		return nil, fmt.Errorf("streamMode disable, should be enabled")
	}
	clientStreamWriter := NewClientStreamWriter(p2p.clientQueue, p2p.notifyQueue, p2p.API, p2p.cfg)
	go func() {
		err := p2p.run(ctx, clientStreamWriter)
		if err != nil {
			logrus.Warnf("P2PDownloader run error: %s", err)
		}
	}()
	return clientStreamWriter, nil
}

func (p2p *P2PDownloader) run(ctx context.Context, pieceWriter PieceWriter) error {
	var (
		lastItem *Piece
		goNext   bool
	)

	// start PieceWriter
	if err := pieceWriter.PreRun(ctx); err != nil {
		return err
	}
	go func() {
		pieceWriter.Run(ctx)
	}()

	for {
		goNext, lastItem = p2p.getItem(lastItem)
		if !goNext {
			continue
		}
		logrus.Infof("downloading piece:%v", lastItem)

		curItem := *lastItem
		curItem.Content = nil
		lastItem = nil

		response, err := p2p.pullPieceTask(&curItem)
		if err != nil {
			logrus.Errorf("failed to download piece: %v", err)
			if p2p.cfg.BackSourceReason == 0 {
				p2p.cfg.BackSourceReason = config.BackSourceReasonDownloadError
			}
		} else {
			code := response.Code
			if code == constants.CodePeerContinue {
				p2p.processPiece(response, &curItem)
			} else if code == constants.CodePeerFinish {
				if p2p.cfg.Md5 == "" {
					p2p.cfg.Md5 = response.FinishData().Md5
				}
				p2p.finishTask(ctx, pieceWriter)
				return nil
			} else {
				logrus.Warnf("request piece result:%v", response)
				if code == constants.CodePeerWait {
					continue
				}
				if code == constants.CodeSourceError {
					p2p.cfg.BackSourceReason = config.BackSourceReasonSourceError
				}
			}
		}

		if p2p.cfg.BackSourceReason != 0 {
			return fmt.Errorf("failed to download with %s pattern, reason: %d", p2p.cfg.Pattern, p2p.cfg.BackSourceReason)
		}
	}
}

// Cleanup cleans all temporary resources generated by executing Run.
func (p2p *P2PDownloader) Cleanup() {}

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
		res, err = p2p.API.PullPieceTask(item.SuperNode, req)
		if err != nil {
			logrus.Errorf("failed to pull piece task(%+v): %v", item, err)
			break
		}
		if res.Code != constants.CodePeerWait {
			break
		}
		if p2p.queue.Len() > 0 {
			break
		}

		actual, expected := p2p.sleepInterval()
		if expected > actual || logrus.IsLevelEnabled(logrus.DebugLevel) {
			logrus.Infof("pull piece task(%+v) result:%s and sleep actual:%.3fs expected:%.3fs",
				item, res, actual.Seconds(), expected.Seconds())
		}
	}

	// FIXME: try to abstract the judgement to make it more readable.
	if res != nil && !(res.Code != constants.CodePeerContinue &&
		res.Code != constants.CodePeerFinish &&
		res.Code != constants.CodePeerLimited &&
		res.Code != constants.Success &&
		res.Code != constants.CodePeerWait) {
		return res, err
	}

	logrus.Errorf("pull piece task fail:%v and will migrate", res)
	var registerRes *regist.RegisterResult
	registerRes, registerErr := p2p.Register.Register(p2p.cfg.RV.PeerPort)
	if registerErr != nil {
		return nil, registerErr
	}
	p2p.pieceSizeHistory[1] = registerRes.PieceSize
	item.Status = constants.TaskStatusStart
	item.SuperNode = registerRes.Node
	item.TaskID = registerRes.TaskID
	printer.Println("migrated to node:" + item.SuperNode)
	return p2p.pullPieceTask(item)
}

// sleepInterval sleep for a while to wait for next pulling piece task until
// receiving a notification which indicating that all the previous works have
// been completed.
func (p2p *P2PDownloader) sleepInterval() (actual, expected time.Duration) {
	expected = time.Duration(rand.Intn(p2p.maxTimeout-p2p.minTimeout)+p2p.minTimeout) * time.Millisecond
	start := time.Now()
	p2p.notifyQueue.PollTimeout(expected)
	actual = time.Now().Sub(start)

	// gradually increase the sleep time, up to [800-1600]
	if p2p.minTimeout < 800 {
		p2p.minTimeout *= 2
		p2p.maxTimeout *= 2
	}
	return actual, expected
}

// getPullRate gets download rate limit dynamically.
func (p2p *P2PDownloader) getPullRate(data *types.PullPieceTaskResponseContinueData) {
	if time.Since(p2p.pullRateTime).Seconds() < 3 {
		return
	}
	p2p.pullRateTime = time.Now()

	start := time.Now()

	localRate := data.DownLink * 1024
	if p2p.cfg.LocalLimit > 0 {
		localRate = int(p2p.cfg.LocalLimit)
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
		logrus.Errorf("failed to parse rate in pull rate: %v", err)
		p2p.rateLimiter.SetRate(ratelimiter.TransRate(int64(localRate)))
		return
	}

	reqRate, err := strconv.Atoi(resp)
	if err != nil {
		logrus.Errorf("failed to parse rate from resp %s: %v", resp, err)
		p2p.rateLimiter.SetRate(ratelimiter.TransRate(int64(localRate)))
		return
	}
	logrus.Infof("pull rate result:%d cost:%v", reqRate, time.Since(start))
	p2p.rateLimiter.SetRate(ratelimiter.TransRate(int64(reqRate)))
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
		headers:     p2p.headers,
		cdnSource:   p2p.RegisterResult.CDNSource,
		fileLength:  p2p.RegisterResult.FileLength,
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
				p2p.total += item.ContentLength()
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
	if latestItem == nil {
		return false, nil
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
				constants.TaskStatusRunning,
				p2p.RegisterResult.CDNSource))
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

func (p2p *P2PDownloader) finishTask(ctx context.Context, pieceWriter PieceWriter) {
	// wait client writer finished
	logrus.Infof("remaining piece to be written count:%d", p2p.clientQueue.Len())
	p2p.clientQueue.Put(last)
	waitStart := time.Now()
	pieceWriter.Wait()
	logrus.Infof("wait client writer finish cost:%.3f,main qu size:%d,client qu size:%d",
		time.Since(waitStart).Seconds(), p2p.queue.Len(), p2p.clientQueue.Len())

	if p2p.cfg.BackSourceReason > 0 {
		return
	}

	err := pieceWriter.PostRun(ctx)
	if err != nil {
		logrus.Warnf("post run error: %s", err)
	}

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

// wipeOutOfRange will update the end index of range when it's greater than the maxLength.
func wipeOutOfRange(pieceRange string, maxLength int64) string {
	if maxLength < 0 {
		return pieceRange
	}
	indexes := strings.Split(pieceRange, config.RangeSeparator)
	if len(indexes) != 2 {
		return pieceRange
	}

	endIndex, err := strconv.Atoi(indexes[1])
	if err != nil {
		return pieceRange
	}

	if int64(endIndex) < maxLength {
		return pieceRange
	}
	return fmt.Sprintf("%s%s%s", indexes[0], config.RangeSeparator, strconv.FormatInt(maxLength-1, 10))
}
