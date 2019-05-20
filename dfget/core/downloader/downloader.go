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
	"fmt"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/sirupsen/logrus"
)

// Downloader is the interface to download files
type Downloader interface {
	Run() error
	Cleanup()
}

// DoDownloadTimeout downloads the file and waits for response during
// the given timeout duration.
func DoDownloadTimeout(downloader Downloader, timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("download timeout(%.3fs)", timeout.Seconds())
	}

	var ch = make(chan error)
	go func() {
		ch <- downloader.Run()
	}()
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

// ConvertHeaders converts headers from array type to map type for http request.
func ConvertHeaders(headers []string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	hm := make(map[string]string)
	for _, header := range headers {
		kv := strings.SplitN(header, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		if v == "" {
			continue
		}
		if _, in := hm[k]; in {
			hm[k] = hm[k] + "," + v
		} else {
			hm[k] = v
		}
	}
	return hm
}

// MoveFile moves a file from src to dst and
// checks if the MD5 code is expected before that.
func MoveFile(src string, dst string, expectMd5 string) error {
	start := time.Now()
	if expectMd5 != "" {
		realMd5 := util.Md5Sum(src)
		logrus.Infof("compute raw md5:%s for file:%s cost:%.3fs", realMd5,
			src, time.Since(start).Seconds())
		if realMd5 != expectMd5 {
			return fmt.Errorf("Md5NotMatch, real:%s expect:%s", realMd5, expectMd5)
		}
	}
	err := util.MoveFile(src, dst)
	logrus.Infof("move src:%s to dst:%s result:%t cost:%.3f",
		src, dst, err == nil, time.Since(start).Seconds())
	return err
}
