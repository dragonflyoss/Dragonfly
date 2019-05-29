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

package transport

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/downloader"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/downloader/dfget"
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
	Round          *http.Transport
	Round2         http.RoundTripper
	ShouldUseDfget func(req *http.Request) bool
	Downloader     downloader.Interface
}

// New return the default DFRoundTripper.
func New(cfg *tls.Config) *DFRoundTripper {
	if cfg == nil {
		cfg = &tls.Config{InsecureSkipVerify: true}
	}

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
			TLSClientConfig:       cfg,
		},
		Round2:         http.NewFileTransport(http.Dir("/")),
		ShouldUseDfget: needUseGetter,
		Downloader: dfget.NewDFGetter(
			global.CommandLine.DFRepo,
			global.CommandLine.CallSystem,
			global.CommandLine.Notbs,
			global.CommandLine.RateLimit,
			global.CommandLine.URLFilter,
			global.CommandLine.SupernodeList,
		),
	}
}

// RoundTrip only process first redirect at present
// fix resource release
func (roundTripper *DFRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if roundTripper.ShouldUseDfget(req) {
		// delete the Accept-Encoding header to avoid returning the same cached
		// result for different requests
		req.Header.Del("Accept-Encoding")
		logrus.Debugf("round trip with dfget: %s", req.URL.String())
		if res, err := roundTripper.download(req, req.URL.String()); err == nil || !exception.IsNotAuth(err) {
			return res, err
		}
	}
	logrus.Debugf("round trip directly: %s %s", req.Method, req.URL.String())
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
	logrus.Infof("start download url:%s to %s in repo", url, name)
	return roundTripper.Downloader.Download(url, header, name)
}

// needUseGetter is the default value for ShouldUseDfget, which downloads all
// images layers with dfget.
func needUseGetter(req *http.Request) bool {
	if req.Method != http.MethodGet {
		return false
	}

	if compiler.MatchString(req.URL.Path) {
		return true
	}
	if req.URL.String() != "" {
		return global.MatchDfPattern(req.URL.String())
	}

	return false
}
