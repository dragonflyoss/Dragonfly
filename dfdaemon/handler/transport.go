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
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"

	"github.com/alibaba/Dragonfly/dfdaemon/exception"
	"github.com/alibaba/Dragonfly/dfdaemon/global"
)

// DFRoundTripper implements RoundTripper for dfget.
// It uses http.fileTransport to serve requests that need to use dfget,
// and uses http.Transport to serve the other requests.
type DFRoundTripper struct {
	Round  *http.Transport
	Round2 http.RoundTripper
}

var dfRoundTripper = &DFRoundTripper{
	Round: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	},
	Round2: http.NewFileTransport(http.Dir("/")),
}

var compiler = regexp.MustCompile("^.+/blobs/sha256.*$")

func needUseGetter(req *http.Request, location string) bool {
	if req.Method != http.MethodGet {
		return false
	}

	if compiler.MatchString(req.URL.Path) {
		return true
	}
	if location != "" {
		return global.MatchDfPattern(location)
	}

	return false
}

// RoundTrip only process first redirect at present
// fix resource release
func (roundTripper *DFRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	urlString := req.URL.String()

	if needUseGetter(req, urlString) {
		if res, err := dfRoundTripper.download(req, urlString); err == nil || !exception.IsNotAuth(err) {
			return res, err
		}
	}
	req.Host = req.URL.Host
	req.Header.Set("Host", req.Host)
	res, err := roundTripper.Round.RoundTrip(req)
	return res, err
}

// download uses dfget to download
func (roundTripper *DFRoundTripper) download(req *http.Request, urlString string) (*http.Response, error) {
	dstPath, err := DownloadByGetter(urlString, req.Header, uuid.New())
	if err != nil {
		logrus.Errorf("download fail: %v", err)
		return nil, err
	}
	defer os.Remove(dstPath)

	fileReq, err := http.NewRequest("GET", "file:///"+dstPath, nil)
	if err != nil {
		return nil, err
	}

	response, err := dfRoundTripper.Round2.RoundTrip(fileReq)
	if err == nil {
		response.Header.Set("Content-Disposition", "attachment; filename="+dstPath)
	} else {
		logrus.Errorf("read response from file:%s error:%v", dstPath, err)
	}
	return response, err
}
