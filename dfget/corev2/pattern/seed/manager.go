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

package seed

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"

	dfgetcfg "github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/basic"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/api"
	"github.com/dragonflyoss/Dragonfly/dfget/locator"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/protocol"

	"github.com/sirupsen/logrus"
)

type rangeRequest struct {
	url    string
	off    int64
	size   int64
	header map[string]string
}

func (rr rangeRequest) URL() string {
	return rr.url
}

func (rr rangeRequest) Offset() int64 {
	return rr.off
}

func (rr rangeRequest) Size() int64 {
	return rr.size
}

func (rr rangeRequest) Header() map[string]string {
	return rr.header
}

func (rr rangeRequest) Extra() interface{} {
	return nil
}

type superNodeWrapper struct {
	superNode string

	// version of supernode, if changed, it indicates supernode has been restarted.
	version string

	// scheduler which the task is belong to the supernode.
	sm *scheduleManager
}

func (s *superNodeWrapper) versionChanged(version string) bool {
	return s.version != "" && version != "" && s.version != version
}

const (
	maxTry = 10
)

// Manager provides an implementation of downloader.Stream.
//
type Manager struct {
	client        protocol.Client
	downloaderAPI api.DownloadAPI

	// supernode locator which could select supernode by key.
	locator locator.SupernodeLocator

	superNodeMap map[string]*superNodeWrapper

	ctx    context.Context
	cancel func()
}

func NewManager() *Manager {
	//todo: init Manager by input config.
	return &Manager{}
}

//todo: add local seed manager and p2pNetwork updater.

// DownloadStreamContext implementation of downloader.Stream.
func (m *Manager) DownloadStreamContext(ctx context.Context, url string, header map[string][]string, name string) (io.ReadCloser, error) {
	reqRange, err := m.getRangeFromHeader(header)
	if err != nil {
		return nil, err
	}

	logrus.Debugf("start to download stream in seed pattern, url: %s, header: %v, range: [%d, %d]", url,
		header, reqRange.StartIndex, reqRange.EndIndex)

	hd := make(map[string]string)
	for k, v := range header {
		hd[k] = v[0]
	}

	rr := &rangeRequest{
		url:    url,
		off:    reqRange.StartIndex,
		size:   reqRange.EndIndex - reqRange.StartIndex + 1,
		header: hd,
	}

	for i := 0; i < maxTry; i++ {
		// try to get the peer by internal schedule
		dwInfos := m.schedule(ctx, rr)
		if len(dwInfos) == 0 {
			// try to apply to be the seed node
			m.tryToApplyForSeedNode(m.ctx, url, header)
			continue
		}

		return m.runStream(ctx, rr, dwInfos)
	}

	return nil, errortypes.NewHTTPError(http.StatusInternalServerError, "failed to select a peer to download")
}

func (m *Manager) runStream(ctx context.Context, rr basic.RangeRequest, peers []*basic.SchedulePeerInfo) (io.ReadCloser, error) {
	var (
		rc  io.ReadCloser
		err error
	)

	for _, peer := range peers {
		rc, err = m.tryToDownload(ctx, peer, rr)
		if err != nil {
			continue
		}
	}

	if rc == nil {
		return nil, fmt.Errorf("failed to select a peer to download")
	}

	var (
		pr, pw = io.Pipe()
	)

	// todo: divide request data into pieces in consideration of peer disconnect.
	go func(sourceRc io.ReadCloser) {
		cw := NewClientWriter()
		notify, err := cw.Run(ctx, pw)
		if err != nil {
			pw.CloseWithError(err)
			sourceRc.Close()
			return
		}

		defer func() {
			<-notify.Done()
			sourceRc.Close()
		}()

		data, _ := NewSeedData(sourceRc, rr.Size(), true)
		cw.PutData(data)
		cw.PutData(protocol.NewEoFDistributionData())
	}(rc)

	return pr, nil
}

func (m *Manager) tryToDownload(ctx context.Context, peer *basic.SchedulePeerInfo, rr basic.RangeRequest) (io.ReadCloser, error) {
	down := NewDownloader(peer, 0, m.downloaderAPI)
	rc, err := down.Download(ctx, rr.Offset(), rr.Size())
	if err != nil {
		return nil, err
	}

	return rc, nil
}

func (m *Manager) schedule(ctx context.Context, rr basic.RangeRequest) []*basic.SchedulePeerInfo {
	dwInfos := []*basic.SchedulePeerInfo{}

	for _, sw := range m.superNodeMap {
		result, err := sw.sm.Schedule(ctx, rr, nil)
		if err != nil {
			continue
		}

		dr := result.Result()
		for _, r := range dr {
			dwInfos = append(dwInfos, r.PeerInfos...)
		}
	}

	return dwInfos
}

func (m *Manager) getRangeFromHeader(header map[string][]string) (*httputils.RangeStruct, error) {
	hr := http.Header(header)
	if headerStr := hr.Get(dfgetcfg.StrRange); headerStr != "" {
		ds, err := httputils.GetRangeSE(headerStr, math.MaxInt64)
		if err != nil {
			return nil, err
		}

		// todo: support the multi range
		if len(ds) != 1 {
			return nil, fmt.Errorf("not support multi range")
		}

		// if EndIndex is max int64, set EndIndex to (StartIndex - 1),
		// so that means the end index is tail of file length.
		if ds[0].EndIndex == math.MaxInt64-1 {
			ds[0].EndIndex = ds[0].StartIndex - 1
		}

		return ds[0], nil
	}

	return &httputils.RangeStruct{
		StartIndex: 0,
		EndIndex:   -1,
	}, nil
}

func (m *Manager) tryToApplyForSeedNode(ctx context.Context, url string, header map[string][]string) {
	//todo: apply for seed node.
}
