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

// Package downloader contains 2 types of downloader: P2PDownloader,
// DirectDownloader.
// P2PDownloader uses P2P pattern to download files from peers.
// DirectDownloader downloads files from file source directly. It's
// used when P2PDownloader download files failed.
package downloader

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"

	"github.com/sirupsen/logrus"
)

// Downloader is the interface to download files
type Downloader interface {
	Run(ctx context.Context) error
	// RunStream return a io.Reader instead of writing a file without any disk io.
	RunStream(ctx context.Context) (io.Reader, error)
	Cleanup()
}

// DoDownloadTimeout downloads the file and waits for response during
// the given timeout duration.
func DoDownloadTimeout(downloader Downloader, timeout time.Duration) error {
	if timeout <= 0 {
		logrus.Warnf("invalid download timeout(%.3fs), use default:(%.3fs)",
			timeout.Seconds(), config.DefaultDownloadTimeout.Seconds())
		timeout = config.DefaultDownloadTimeout
	}
	ctx, cancel := context.WithCancel(context.Background())

	var ch = make(chan error)
	go func() {
		ch <- downloader.Run(ctx)
	}()
	defer cancel()

	var err error
	select {
	case err = <-ch:
		return err
	case <-time.After(timeout):
		err = fmt.Errorf("download timeout(%.3fs)", timeout.Seconds())
		downloader.Cleanup()
	}
	return err
}

// MoveFile moves a file from src to dst and
// checks if the MD5 code is expected before that.
func MoveFile(src string, dst string, expectMd5 string) error {
	start := time.Now()
	if expectMd5 != "" {
		realMd5 := fileutils.Md5Sum(src)
		logrus.Infof("compute raw md5:%s for file:%s cost:%.3fs", realMd5,
			src, time.Since(start).Seconds())
		if realMd5 != expectMd5 {
			return fmt.Errorf("Md5NotMatch, real:%s expect:%s", realMd5, expectMd5)
		}
	}
	err := fileutils.MoveFile(src, dst)
	logrus.Infof("move src:%s to dst:%s result:%t cost:%.3f",
		src, dst, err == nil, time.Since(start).Seconds())
	return err
}
