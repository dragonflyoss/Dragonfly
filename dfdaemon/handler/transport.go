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

	"github.com/dragonflyoss/Dragonfly/dfdaemon/exception"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/global"
)

var (
	// compiler the regex to determine if it is an image download
	compiler = regexp.MustCompile("^.+/blobs/sha256.*$")
)

// DFRoundTripper implements RoundTripper for dfget.
// It uses http.fileTransport to serve requests that need to use dfget,
// and uses http.Transport to serve the other requests.
type DFRoundTripper struct {
	Round  *http.Transport
	Round2 http.RoundTripper
}

// NewDFRoundTripper return the default DFRoundTripper.
func NewDFRoundTripper() *DFRoundTripper {
	return &DFRoundTripper{
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
}

// RoundTrip only process first redirect at present
// fix resource release
func (roundTripper *DFRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	urlString := req.URL.String()

	if roundTripper.needUseGetter(req, urlString) {
		if res, err := roundTripper.download(req, urlString); err == nil || !exception.IsNotAuth(err) {
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
	dstPath, err := roundTripper.downloadByGetter(urlString, req.Header, uuid.New())
	if err != nil {
		logrus.Errorf("download fail: %v", err)
		return nil, err
	}
	defer os.Remove(dstPath)

	fileReq, err := http.NewRequest("GET", "file:///"+dstPath, nil)
	if err != nil {
		return nil, err
	}

	response, err := roundTripper.Round2.RoundTrip(fileReq)
	if err == nil {
		response.Header.Set("Content-Disposition", "attachment; filename="+dstPath)
	} else {
		logrus.Errorf("read response from file:%s error:%v", dstPath, err)
	}
	return response, err
}

// downloadByGetter is to download file by DFGetter
func (roundTripper *DFRoundTripper) downloadByGetter(url string, header map[string][]string, name string) (string, error) {
	var getter = NewDFGetter()
	logrus.Infof("start download url:%s to %s in repo", url, name)
	getter.once.Do(func() {
		getter.dstDir = global.CommandLine.DFRepo
		getter.callSystem = global.CommandLine.CallSystem
		getter.notbs = global.CommandLine.Notbs
		getter.rateLimit = global.CommandLine.RateLimit
		getter.urlFilter = global.CommandLine.URLFilter
	})
	return getter.Download(url, header, name)
}

// needUseGetter whether to download by DFGetter
func (roundTripper *DFRoundTripper) needUseGetter(req *http.Request, location string) bool {
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
