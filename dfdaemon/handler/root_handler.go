// Copyright 1999-2017 Alibaba Group.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	log "github.com/sirupsen/logrus"

	"github.com/alibaba/Dragonfly/dfdaemon/global"
	"github.com/alibaba/Dragonfly/dfdaemon/util"
)

// Process makes the dfdaemon as a reverse proxy to download image layers by dragonfly
func Process(w http.ResponseWriter, r *http.Request) {

	if r.URL.Host == "" {
		r.URL.Host = r.Host
		if r.URL.Host == "" {
			r.URL.Host = r.Header.Get("Host")
		}
		if r.URL.Host == "" {
			log.Errorf("url host is empty")
		}
	}
	r.Host = r.URL.Host
	r.Header.Set("Host", r.Host)
	if r.URL.Scheme == "" {
		if global.UseHTTPS {
			r.URL.Scheme = "https"
		} else {
			r.URL.Scheme = "http"
		}

	}
	log.Debugf("pre access:%s", r.URL.String())

	targetURL := new(url.URL)
	*targetURL = *r.URL
	targetURL.Path = ""
	targetURL.RawQuery = ""

	hostIP := util.ExtractHost(r.URL.Host)
	switch hostIP {
	case "127.0.0.1", "localhost", global.CommandLine.HostIP:
		if len(global.CommandLine.Registry) > 0 {
			targetURL.Host = global.RegDomain
			targetURL.Scheme = global.RegProto
		} else {
			log.Warnf("registry not config but url host is %s", hostIP)
		}
	default:
		// non localhost access should be denied explicitly, otherwise we
		// are falling into a dead loop: a reverse proxy for itself.
		// TODO: we do not need such check actually, anything that served
		// by dfdaemon should only be accessed by localhost which should
		// be controlled by the listener addr.
		w.WriteHeader(http.StatusForbidden)
		return
	}

	log.Debugf("post access:%s", targetURL.String())

	// TODO: do we really need to construct this every time?
	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)

	reverseProxy.Transport = dfRoundTripper

	reverseProxy.ServeHTTP(w, r)
}
