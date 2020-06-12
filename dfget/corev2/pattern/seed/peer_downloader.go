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
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/basic"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/api"
	"github.com/dragonflyoss/Dragonfly/dfget/local/seed"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/limitreader"
	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// downloaderFactory is an implementation of github.com/dragonflyoss/Dragonfly/dfget/local/seed.DownloaderFactory.
type downloaderFactory struct {
	downloadAPI api.DownloadAPI
	sm          *supernodeManager
	localPeer   *types.PeerInfo
}

func newDownloaderFactory(sm *supernodeManager, localPeer *types.PeerInfo, downAPI api.DownloadAPI) *downloaderFactory {
	return &downloaderFactory{
		downloadAPI: downAPI,
		sm:          sm,
		localPeer:   localPeer,
	}
}

func (df *downloaderFactory) Create(opt seed.DownloaderFactoryCreateOpt) seed.Downloader {
	peers := df.sm.Schedule(context.Background(), &rangeRequest{url: opt.URL})
	if len(peers) == 0 {
		return nil
	}

	var targetPeer *basic.SchedulePeerInfo
	// exclude self peer
	for _, peer := range peers {
		if peer.IP == df.localPeer.IP && peer.Port == df.localPeer.Port {
			continue
		}

		targetPeer = peer
		break
	}

	if targetPeer == nil {
		return nil
	}

	return &peerDownloader{
		url:             opt.URL,
		rl:              opt.RateLimiter,
		openMemoryCache: opt.OpenMemoryCache,
		peer:            targetPeer,
		downAPI:         df.downloadAPI,
	}
}

// peerDownloader is an implementation of github.com/dragonflyoss/Dragonfly/dfget/local/seed.Downloader.
// it will download data from peer.
type peerDownloader struct {
	url             string
	rl              *ratelimiter.RateLimiter
	openMemoryCache bool
	peer            *basic.SchedulePeerInfo
	downAPI         api.DownloadAPI
}

func (ld *peerDownloader) DownloadToWriterAt(ctx context.Context, rangeStruct httputils.RangeStruct, timeout time.Duration, writeOff int64, writerAt io.WriterAt, rateLimit bool) (length int64, err error) {
	down := NewDownloader(ld.peer, timeout, ld.downAPI)

	var (
		written int64
		n       int
		rd      io.Reader
	)

	size := rangeStruct.EndIndex - rangeStruct.StartIndex + 1
	rc, err := down.Download(ctx, rangeStruct.StartIndex, size)
	if err != nil {
		return 0, err
	}

	expectedLen := rangeStruct.EndIndex - rangeStruct.StartIndex + 1

	defer rc.Close()
	rd = rc
	if rateLimit {
		rd = limitreader.NewLimitReaderWithLimiter(ld.rl, rc, false)
	}

	// in copyCache pattern, the bytes buffer will be transferred to io.WriterAt, and will be held by io.WriterAt.
	if ld.openMemoryCache {
		buf := bytes.NewBuffer(nil)
		buf.Grow(int(expectedLen))
		written, err = io.CopyN(buf, rd, expectedLen)
		if err != nil && err != io.EOF {
			logrus.Errorf("failed to read data [%d, %d] from resp.body: %v", rangeStruct.StartIndex, rangeStruct.EndIndex, err)
		}

		if written < expectedLen {
			return 0, errors.Wrap(io.ErrShortWrite, fmt.Sprintf("download from [%d,%d], expecte read %d, but got %d", rangeStruct.StartIndex, rangeStruct.EndIndex, expectedLen, written))
		}

		n, err = writerAt.WriteAt(buf.Bytes(), writeOff)
		written = int64(n)
	} else {
		written, err = seed.CopyBufferToWriterAt(writeOff, writerAt, rd)
	}

	if err == io.EOF {
		err = nil
	}

	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("failed to download from [%d,%d]", rangeStruct.StartIndex, rangeStruct.EndIndex))
	}

	if written < expectedLen {
		return 0, errors.Wrap(io.ErrShortWrite, fmt.Sprintf("download from [%d,%d], expecte read %d, but got %d", rangeStruct.StartIndex, rangeStruct.EndIndex, expectedLen, written))
	}

	return written, err
}
