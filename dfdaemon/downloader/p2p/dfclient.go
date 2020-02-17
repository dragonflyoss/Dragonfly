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

package p2p

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	dfgetcfg "github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/regist"
)

type DFClient struct {
	config       config.DFGetConfig
	dfGetConfig  *dfgetcfg.Config
	supernodeAPI api.SupernodeAPI
	register     regist.SupernodeRegister
	dfClient     core.DFGet
}

func (c *DFClient) DownloadContext(ctx context.Context, url string, header map[string][]string, name string) (string, error) {
	//	startTime := time.Now()
	dstPath := filepath.Join(c.config.DFRepo, name)
	// r, err := c.doDownload(ctx, url, header, dstPath)
	// if err != nil {
	// 	return "", fmt.Errorf("dfget fail %v", err)
	// }
	// log.Infof("dfget url:%s [SUCCESS] cost:%.3fs", url, time.Since(startTime).Seconds())
	return dstPath, nil
}

func (c *DFClient) DownloadStreamContext(ctx context.Context, url string, header map[string][]string, name string) (io.Reader, error) {
	dstPath := filepath.Join(c.config.DFRepo, name)
	r, err := c.doDownload(ctx, url, header, dstPath)
	if err != nil {
		return nil, fmt.Errorf("dfget fail %v", err)
	}
	return r, nil
}

func convertToDFGetConfig(cfg config.DFGetConfig) *dfgetcfg.Config {
	return &dfgetcfg.Config{
		Nodes:    cfg.SuperNodes,
		DFDaemon: true,
		Pattern:  dfgetcfg.PatternCDN,
		Sign: fmt.Sprintf("%d-%.3f",
			os.Getpid(), float64(time.Now().UnixNano())/float64(time.Second)),
		RV: dfgetcfg.RuntimeVariable{
			LocalIP:  cfg.LocalIP,
			PeerPort: cfg.PeerPort,
		},
	}
}

func NewClient(cfg config.DFGetConfig) *DFClient {
	supernodeAPI := api.NewSupernodeAPI()
	dfGetConfig := convertToDFGetConfig(cfg)
	register := regist.NewSupernodeRegister(dfGetConfig, supernodeAPI)

	client := &DFClient{
		config:       cfg,
		dfGetConfig:  dfGetConfig,
		supernodeAPI: supernodeAPI,
		register:     register,
		dfClient:     core.NewDFGet(),
	}
	client.init()
	return client
}

func (c *DFClient) init() {
	c.dfGetConfig.RV.Cid = getCid(c.dfGetConfig.RV.LocalIP, c.dfGetConfig.Sign)
}

func (c *DFClient) doDownload(ctx context.Context, url string, header map[string][]string, destPath string) (io.Reader, error) {
	runtimeConfig := *c.dfGetConfig
	runtimeConfig.URL = url
	runtimeConfig.RV.TaskURL = url
	runtimeConfig.RV.TaskFileName = getTaskFileName(destPath, c.dfGetConfig.Sign)
	runtimeConfig.Header = flattenHeader(header)
	runtimeConfig.Output = destPath
	runtimeConfig.RV.RealTarget = destPath
	runtimeConfig.RV.TargetDir = filepath.Dir(destPath)

	return c.dfClient.GetReader(ctx, &runtimeConfig)
}

func getTaskFileName(realTarget string, sign string) string {
	return filepath.Base(realTarget) + "-" + sign
}

func getCid(localIP string, sign string) string {
	return localIP + "-" + sign
}

func flattenHeader(header map[string][]string) []string {
	var res []string
	for key, value := range header {
		// discard HTTP host header for backing to source successfully
		if strings.EqualFold(key, "host") {
			continue
		}
		if len(value) > 0 {
			for _, v := range value {
				res = append(res, fmt.Sprintf("%s:%s", key, v))
			}
		} else {
			res = append(res, fmt.Sprintf("%s:%s", key, ""))
		}
	}
	return res
}
