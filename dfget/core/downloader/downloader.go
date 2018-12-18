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
	"net/http"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/regist"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
	"github.com/sirupsen/logrus"
)

// Downloader is the interface to download files
type Downloader interface {
	Run() error
	Cleanup()
}

// NewBackDownloader create BackDownloader
func NewBackDownloader(ctx *config.Context, result *regist.RegisterResult) Downloader {
	var (
		taskID string
		node   string
	)
	if result != nil {
		taskID = result.TaskID
		node = result.Node
	}
	return &BackDownloader{
		Ctx:     ctx,
		URL:     ctx.URL,
		Target:  ctx.RV.RealTarget,
		Md5:     ctx.Md5,
		TaskID:  taskID,
		Node:    node,
		Total:   0,
		Success: false,
	}
}

// NewP2PDownloader create P2PDownloader
func NewP2PDownloader(ctx *config.Context,
	api api.SupernodeAPI,
	register regist.SupernodeRegister,
	result *regist.RegisterResult) Downloader {
	p2p := &P2PDownloader{
		Ctx:            ctx,
		API:            api,
		Register:       register,
		RegisterResult: result,
	}
	p2p.init()
	return p2p
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

func convertHeaders(headers []string) map[string]string {
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

func moveFile(src string, dst string, expectMd5 string, log *logrus.Logger) error {
	start := time.Now()
	if expectMd5 != "" {
		realMd5 := util.Md5Sum(src)
		log.Infof("compute raw md5:%s for file:%s cost:%.3fs", realMd5,
			src, time.Since(start).Seconds())
		if realMd5 != expectMd5 {
			return fmt.Errorf("Md5NotMatch, real:%s expect:%s", realMd5, expectMd5)
		}
	}
	err := util.MoveFile(src, dst)

	log.Infof("move src:%s to dst:%s result:%t cost:%.3f",
		src, dst, err == nil, time.Since(start).Seconds())
	return err
}

func httpGetWithHeaders(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	return http.DefaultClient.Do(req)
}
