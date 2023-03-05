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
	"errors"
	"io"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/downloader"
)

func init() {
	downloader.Register("p2p", func(patternConfig config.DownloadConfig, c config.Properties) downloader.Stream {
		return NewClient(c.DFGetConfig())
	})
}

type Client struct {
}

func (c *Client) DownloadContext(ctx context.Context, url string, header map[string][]string, name string) (string, error) {
	return "", errors.New("Not Implementation")
}

func (c *Client) DownloadStreamContext(ctx context.Context, url string, header map[string][]string, name string) (io.Reader, error) {
	return nil, errors.New("Not Implementation")
}

func NewClient(cfg config.DFGetConfig) *Client {
	return &Client{}
}
