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
	. "net/http"
	"os"
	"regexp"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pborman/uuid"

	"github.com/alibaba/Dragonfly/src/daemon/src/df-daemon/exception"
	"github.com/alibaba/Dragonfly/src/daemon/src/df-daemon/global"
)

type DFRoundTripper struct {
	Round  *Transport
	Round2 RoundTripper
}

var dfRoundTripper = &DFRoundTripper{
	Round: &Transport{
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
	Round2: NewFileTransport(Dir("/")),
}

var compiler = regexp.MustCompile("^.+/blobs/sha256.*$")

func needUseGetter(req *Request, location string) bool {
	var useGetter bool
	if req.Method == MethodGet {
		if compiler.MatchString(req.URL.Path) {
			return true
		}
		if location != "" {
			return global.MatchDfPattern(location)
		}
	}
	return useGetter
}

//only process first redirect at present
//fix resource release
func (roundTripper *DFRoundTripper) RoundTrip(req *Request) (*Response, error) {
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

func (roundTripper *DFRoundTripper) download(req *Request, urlString string) (*Response, error) {
	//use dfget to download
	if dstPath, err := DownloadByGetter(urlString, req.Header, uuid.New()); err == nil {
		defer os.Remove(dstPath)
		if fileReq, err := NewRequest("GET", "file:///"+dstPath, nil); err == nil {
			response, err := dfRoundTripper.Round2.RoundTrip(fileReq)
			if err == nil {
				response.Header.Set("Content-Disposition", "attachment; filename="+dstPath)
			} else {
				logrus.Errorf("read response from file:%s error:%v", dstPath, err)
			}
			return response, err
		} else {
			return nil, err
		}
	} else {
		logrus.Errorf("download fail:%v", err)
		return nil, err
	}
}
