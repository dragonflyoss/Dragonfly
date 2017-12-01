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

	log "github.com/Sirupsen/logrus"

	. "github.com/alibaba/Dragonfly/src/daemon/src/df-daemon/global"
	"github.com/alibaba/Dragonfly/src/daemon/src/df-daemon/util"
)

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
		if G_UseHttps {
			r.URL.Scheme = "https"
		} else {
			r.URL.Scheme = "http"
		}

	}
	log.Debugf("pre access:%s", r.URL.String())

	targetUrl := new(url.URL)
	*targetUrl = *r.URL
	targetUrl.Path = ""
	targetUrl.RawQuery = ""

	hostIp := util.ExtractHost(r.URL.Host)
	switch hostIp {
	case "127.0.0.1", "localhost":
		if len(G_CommandLine.Registry) > 0 {
			targetUrl.Host = G_RegDomain
			targetUrl.Scheme = G_RegProto
		} else {
			log.Warnf("registry not config but url host is %s", hostIp)
		}
	default:
		// non localhost access should be denied explictly, otherwise we
		// are falling into a dead loop: a reverse proxy for itself.
		// TODO: we do not need such check actually, anything that served
		// by df-daemon should only be accessed by localhost which should
		// be controlled by the listener addr.
		w.WriteHeader(http.StatusForbidden)
		return
	}

	log.Debugf("post access:%s", targetUrl.String())

	// TODO: do we really need to construct this every time?
	reverseProxy := httputil.NewSingleHostReverseProxy(targetUrl)

	reverseProxy.Transport = dfRoundTripper

	reverseProxy.ServeHTTP(w, r)
}
