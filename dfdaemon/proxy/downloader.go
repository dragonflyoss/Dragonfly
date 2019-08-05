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

package proxy

import (
	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/downloader"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/downloader/dfget"
)

// proxyDownloader consists of proxy rule and its downloader
type proxyDownloader struct {
	*config.Proxy
	downloader downloader.Interface
}

func newProxyDownloaders(p *config.Properties) []*proxyDownloader {
	pds := make([]*proxyDownloader, len(p.Proxies))
	for i, proxy := range p.Proxies {
		pds[i] = &proxyDownloader{
			Proxy:      proxy,
			downloader: dfget.NewGetter(p.DFGetConfig(proxy.DfgetFlags)),
		}
	}
	return pds
}

// Download is used to implement Download interface.
func (pd *proxyDownloader) Download(url string, header map[string][]string, name string) (string, error) {
	return pd.downloader.Download(url, header, name)
}
