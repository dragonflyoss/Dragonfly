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
	"context"
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/downloader"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/exception"
)

var (
	// layerReg the regex to determine if it is an image download
	layerReg = regexp.MustCompile("^.+/blobs/sha256.*$")
)

// DFRoundTripper implements RoundTripper for dfget.
// It uses http.fileTransport to serve requests that need to use dfget,
// and uses http.Transport to serve the other requests.
type DFRoundTripper struct {
	Round            *http.Transport
	Round2           http.RoundTripper
	ShouldUseDfget   func(req *http.Request) bool
	Downloader       downloader.Interface
	StreamDownloader downloader.Stream
	streamMode       bool
}

// New returns the default DFRoundTripper.
func New(opts ...Option) (*DFRoundTripper, error) {
	rt := &DFRoundTripper{
		Round:          defaultHTTPTransport(nil),
		Round2:         http.NewFileTransport(http.Dir("/")),
		ShouldUseDfget: NeedUseGetter,
	}

	for _, opt := range opts {
		if err := opt(rt); err != nil {
			return nil, errors.Wrap(err, "apply options")
		}
	}

	if rt.StreamDownloader == nil {
		return nil, errors.Errorf("nil downloader")
	}

	return rt, nil
}

// Option is functional config for DFRoundTripper.
type Option func(rt *DFRoundTripper) error

// WithTLS configures TLS config used for http transport.
func WithTLS(cfg *tls.Config) Option {
	return func(rt *DFRoundTripper) error {
		rt.Round = defaultHTTPTransport(cfg)
		return nil
	}
}

func defaultHTTPTransport(cfg *tls.Config) *http.Transport {
	if cfg == nil {
		cfg = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       cfg,
	}
}

// WithDownloader sets the downloader for the roundTripper.
func WithDownloader(d downloader.Interface) Option {
	return func(rt *DFRoundTripper) error {
		rt.Downloader = d
		return nil
	}
}

func WithStreamDownloader(d downloader.Stream) Option {
	return func(rt *DFRoundTripper) error {
		rt.StreamDownloader = d
		return nil
	}
}

func WithStreamMode(streamMode bool) Option {
	return func(rt *DFRoundTripper) error {
		rt.streamMode = streamMode
		return nil
	}
}

// WithCondition configures how to decide whether to use dfget or not.
func WithCondition(c func(r *http.Request) bool) Option {
	return func(rt *DFRoundTripper) error {
		rt.ShouldUseDfget = c
		return nil
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
		if res, err := roundTripper.download(req, req.URL.String()); err == nil || exception.IsAuthError(err) {
			return res, err
		}
	}
	logrus.Debugf("round trip directly: %s %s", req.Method, req.URL.String())
	req.Host = req.URL.Host
	req.Header.Set("Host", req.Host)
	res, err := roundTripper.Round.RoundTrip(req)
	return res, err
}

// download uses dfget to download.
func (roundTripper *DFRoundTripper) download(req *http.Request, urlString string) (*http.Response, error) {
	if roundTripper.streamMode {
		return roundTripper.downloadByStream(req.Context(), urlString, req.Header, uuid.New())
	}

	dstPath, err := roundTripper.downloadByGetter(req.Context(), urlString, req.Header, uuid.New())
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

// downloadByGetter is used to download file by DFGetter.
func (roundTripper *DFRoundTripper) downloadByGetter(ctx context.Context, url string, header map[string][]string, name string) (string, error) {
	logrus.Infof("start download url:%s to %s in repo", url, name)
	return roundTripper.Downloader.DownloadContext(ctx, url, header, name)
}

func (roundTripper *DFRoundTripper) downloadByStream(ctx context.Context, url string, header map[string][]string, name string) (*http.Response, error) {
	logrus.Infof("start download url:%s to %s in repo", url, name)
	reader, err := roundTripper.StreamDownloader.DownloadStreamContext(ctx, url, header, name)
	if err != nil {
		logrus.Errorf("download fail: %v", err)
		return nil, err
	}

	resp := &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(reader),
	}
	return resp, nil
}

// needUseGetter is the default value for ShouldUseDfget, which downloads all
// images layers with dfget.
func NeedUseGetter(req *http.Request) bool {
	return req.Method == http.MethodGet && layerReg.MatchString(req.URL.Path)
}
