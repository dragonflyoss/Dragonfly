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

package dfget

import (
	"context"
	"fmt"
	netUrl "net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/constant"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/exception"

	log "github.com/sirupsen/logrus"
)

// DFGetter implements Downloader to download file by dragonfly.
type DFGetter struct {
	config config.DFGetConfig
}

// NewGetter returns a dfget downloader from the given config.
func NewGetter(cfg config.DFGetConfig) *DFGetter {
	return &DFGetter{config: cfg}
}

// DownloadContext downloads the resources as specified in url.
func (dfGetter *DFGetter) DownloadContext(ctx context.Context, url string, header map[string][]string, name string) (string, error) {
	startTime := time.Now()
	dstPath := filepath.Join(dfGetter.config.DFRepo, name)
	cmd := dfGetter.getCommand(ctx, url, header, dstPath)
	err := cmd.Run()
	if cmd.ProcessState.Success() {
		log.Infof("dfget url:%s [SUCCESS] cost:%.3fs", url, time.Since(startTime).Seconds())
		return dstPath, nil
	}
	if value, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
		if value.ExitStatus() == constant.CodeReqAuth {
			return "", &exception.AuthError{}
		}
	}
	return "", fmt.Errorf("dfget fail(%s):%v", cmd.ProcessState.String(), err)
}

// getCommand returns the command to download the given resource.
func (dfGetter *DFGetter) getCommand(
	ctx context.Context, url string, header map[string][]string, output string,
) (cmd *exec.Cmd) {
	args := []string{
		"-u", url,
		"-o", output,
	}
	args = append(args, dfGetter.config.DfgetFlags...)

	add := func(key, value string) {
		if v := strings.TrimSpace(value); v != "" {
			args = append(args, key, v)
		}
	}
	add("-s", dfGetter.config.RateLimit)
	add("--totallimit", dfGetter.config.RateLimit)
	if len(dfGetter.config.SuperNodes) > 0 {
		add("--node", strings.Join(dfGetter.config.SuperNodes, ","))
	}

	for key, value := range header {
		// discard HTTP host header for backing to source successfully
		if strings.EqualFold(key, "host") {
			continue
		}
		if len(value) > 0 {
			for _, v := range value {
				add("--header", fmt.Sprintf("%s:%s", key, v))
			}
		} else {
			add("--header", fmt.Sprintf("%s:%s", key, ""))
		}
	}

	urlInfo, _ := netUrl.Parse(url)
	for _, h := range dfGetter.config.HostsConfig {
		if urlInfo != nil && h.Regx.MatchString(urlInfo.Host) {
			if h.Insecure {
				args = append(args, "--insecure")
			}
			if h.Certs != nil && len(h.Certs.Files) != 0 {
				add("--cacerts", strings.Join(h.Certs.Files, ","))
			}
		}
	}

	return exec.CommandContext(ctx, dfGetter.config.DFPath, args...)
}
