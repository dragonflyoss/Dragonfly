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

package httpclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	netUrl "net/url"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"

	strfmt "github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
)

type StatusCodeChecker func(int) bool

// OriginHTTPClient supply apis that interact with the source.
type OriginHTTPClient interface {
	RegisterTLSConfig(rawURL string, insecure bool, caBlock []strfmt.Base64)
	GetContentLength(url string, headers map[string]string) (int64, int, error)
	IsSupportRange(url string, headers map[string]string) (bool, error)
	IsExpired(url string, headers map[string]string, lastModified int64, eTag string) (bool, error)
	Download(url string, headers map[string]string, checkCode StatusCodeChecker) (*http.Response, error)
}

// OriginClient is an implementation of the interface of OriginHTTPClient.
type OriginClient struct {
	clientMap         *sync.Map
	defaultHTTPClient *http.Client
}

// NewOriginClient returns a new OriginClient.
func NewOriginClient() OriginHTTPClient {
	defaultTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	httputils.RegisterProtocolOnTransport(defaultTransport)
	return &OriginClient{
		clientMap: &sync.Map{},
		defaultHTTPClient: &http.Client{
			Transport: defaultTransport,
		},
	}
}

// RegisterTLSConfig saves tls config into map as http client.
// tlsMap:
// key->host value->*http.Client
func (client *OriginClient) RegisterTLSConfig(rawURL string, insecure bool, caBlock []strfmt.Base64) {
	url, err := netUrl.Parse(rawURL)
	if err != nil {
		return
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecure,
	}
	appendSuccess := false
	roots := x509.NewCertPool()
	for _, caBytes := range caBlock {
		appendSuccess = appendSuccess || roots.AppendCertsFromPEM(caBytes)
	}
	if appendSuccess {
		tlsConfig.RootCAs = roots
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       tlsConfig,
	}

	httputils.RegisterProtocolOnTransport(transport)

	client.clientMap.Store(url.Host, &http.Client{
		Transport: transport,
	})
}

// GetContentLength sends a head request to get file length.
func (client *OriginClient) GetContentLength(url string, headers map[string]string) (int64, int, error) {
	// send request
	resp, err := client.HTTPWithHeaders(http.MethodGet, url, headers, 4*time.Second)
	if err != nil {
		return 0, 0, err
	}
	resp.Body.Close()

	return resp.ContentLength, resp.StatusCode, nil
}

// IsSupportRange checks if the source url support partial requests.
func (client *OriginClient) IsSupportRange(url string, headers map[string]string) (bool, error) {
	// set headers: headers is a reference to map, should not change it
	copied := CopyHeader(nil, headers)
	copied["Range"] = "bytes=0-0"

	// send request
	resp, err := client.HTTPWithHeaders(http.MethodGet, url, copied, 4*time.Second)
	if err != nil {
		return false, err
	}
	_ = resp.Body.Close()

	if resp.StatusCode == http.StatusPartialContent {
		return true, nil
	}
	return false, nil
}

// IsExpired checks if a resource received or stored is the same.
func (client *OriginClient) IsExpired(url string, headers map[string]string, lastModified int64, eTag string) (bool, error) {
	if lastModified <= 0 && stringutils.IsEmptyStr(eTag) {
		return true, nil
	}

	// set headers: headers is a reference to map, should not change it
	copied := CopyHeader(nil, headers)
	if lastModified > 0 {
		lastModifiedStr, _ := netutils.ConvertTimeIntToString(lastModified)
		copied["If-Modified-Since"] = lastModifiedStr
	}
	if !stringutils.IsEmptyStr(eTag) {
		copied["If-None-Match"] = eTag
	}

	// send request
	resp, err := client.HTTPWithHeaders(http.MethodGet, url, copied, 4*time.Second)
	if err != nil {
		return false, err
	}
	resp.Body.Close()

	return resp.StatusCode != http.StatusNotModified, nil
}

// Download downloads the file from the original address
func (client *OriginClient) Download(url string, headers map[string]string, checkCode StatusCodeChecker) (*http.Response, error) {
	// TODO: add timeout
	resp, err := client.HTTPWithHeaders(http.MethodGet, url, headers, 0)
	if err != nil {
		return nil, err
	}

	if checkCode(resp.StatusCode) {
		return resp, nil
	}
	return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// HTTPWithHeaders uses host-matched client to request the origin resource.
func (client *OriginClient) HTTPWithHeaders(method, url string, headers map[string]string, timeout time.Duration) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		req = req.WithContext(ctx)
		defer cancel()
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	httpClientObject, existed := client.clientMap.Load(req.Host)
	if !existed {
		// use client.defaultHTTPClient to support custom protocols
		httpClientObject = client.defaultHTTPClient
	}

	httpClient, ok := httpClientObject.(*http.Client)
	if !ok {
		return nil, errors.Wrapf(errortypes.ErrInvalidValue, "http client type check error: %T", httpClientObject)
	}
	return httpClient.Do(req)
}

// CopyHeader copies the src to dst and return a non-nil dst map.
func CopyHeader(dst, src map[string]string) map[string]string {
	if dst == nil {
		dst = make(map[string]string)
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
