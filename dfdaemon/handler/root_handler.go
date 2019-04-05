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

package handler

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/sirupsen/logrus"

	"github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/global"
)

// Proxy makes the dfdaemon as a reverse proxy to download image layers by dragonfly
func Proxy(w http.ResponseWriter, r *http.Request) {
	var (
		target *url.URL
		reg    *config.Registry
		err    error
	)

	if err = amendRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		logrus.Errorf("%v", err)
		return
	}

	logrus.Debugf("pre access:%v", r)

	hostIP := util.ExtractHost(r.URL.Host)
	if reg, err = matchRegistry(hostIP, global.Properties.Registries); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		logrus.Warnf("%v", err)
		return
	}
	target = proxyURL(r.URL, reg)

	logrus.Debugf("post access:%s", target)

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	reverseProxy.Transport = NewDFRoundTripper(reg.TLSConfig())
	reverseProxy.ServeHTTP(w, r)
}

func matchRegistry(host string, reg []*config.Registry) (*config.Registry, error) {
	for i := 0; i < len(reg); i++ {
		if reg[i].Match(host) {
			return reg[i], nil
		}
	}
	return nil, fmt.Errorf("no matched registry for %s", host)
}

func proxyURL(origin *url.URL, reg *config.Registry) (proxy *url.URL) {
	if origin == nil || reg == nil {
		return nil
	}

	proxy = new(url.URL)
	*proxy = *origin
	proxy.Path = ""
	proxy.RawQuery = ""

	if reg.Schema != "" {
		proxy.Scheme = reg.Schema
	}
	if reg.Host != "" {
		proxy.Host = reg.Host
	}
	return proxy
}

func amendRequest(r *http.Request) error {
	if r.URL.Host != "" {
		return nil
	}
	r.URL.Host = r.Host
	if r.URL.Host == "" {
		r.URL.Host = r.Header.Get("Host")
	}
	if r.URL.Host == "" {
		// if host is still empty, we need skip to forward it.
		return fmt.Errorf("url host is empty")
	}
	r.Host = r.URL.Host
	r.Header.Set("Host", r.Host)

	if r.URL.Scheme != "" {
		return nil
	}
	if global.UseHTTPS {
		r.URL.Scheme = "https"
	} else {
		r.URL.Scheme = "http"
	}
	return nil
}
