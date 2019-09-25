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

package client

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	defaultHost    string
	defaultTimeout = time.Second * 10
	// defaultVersion is the version of the current stable API
	defaultVersion = "v1.24"
)

// APIClient is an API client that performs all operations.
// against a server
type APIClient struct {
	proto   string // socket type
	addr    string
	baseURL string
	HTTPCli *http.Client
	// version of the server talks to
	version string
}

// TLSConfig contains information of TLS which users can specify.
type TLSConfig struct {
	CA               string `json:"tlscacert,omitempty"`
	Cert             string `json:"tlscert,omitempty"`
	Key              string `json:"tlskey,omitempty"`
	VerifyRemote     bool   `json:"tlsverify,omitempty"`
	ManagerWhiteList string `json:"manager-whitelist,omitempty"`
}

// NewAPIClient initializes a new API client for the given host.
func NewAPIClient(host string, tls TLSConfig) (CommonAPIClient, error) {
	if host == "" {
		host = defaultHost
	}

	newURL, _, addr, err := ParseHost(host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse host %s: %v", host, err)
	}

	tlsConfig := generateTLSConfig(host, tls)

	httpCli := NewHTTPClient(newURL, tlsConfig, defaultTimeout)

	basePath := generateBaseURL(newURL, tls)

	version := os.Getenv("DRAGONFLY_API_VERSION")
	if version == "" {
		version = defaultVersion
	}

	return &APIClient{
		proto:   newURL.Scheme,
		addr:    addr,
		baseURL: basePath,
		HTTPCli: httpCli,
		version: version,
	}, nil
}

// generateTLSConfig configures TLS for API Client.
func generateTLSConfig(host string, tls TLSConfig) *tls.Config {
	// init tls config
	if tls.Key != "" && tls.Cert != "" && !strings.HasPrefix(host, "unix://") {
		tlsCfg, err := GenTLSConfig(tls.Key, tls.Cert, tls.CA)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fail to parse tls config %v", err)
			os.Exit(1)
		}
		tlsCfg.InsecureSkipVerify = !tls.VerifyRemote

		return tlsCfg
	}
	return nil
}

func generateBaseURL(u *url.URL, tls TLSConfig) string {
	if tls.Key != "" && tls.Cert != "" && u.Scheme != "unix" {
		return "https://" + u.Host
	}

	if u.Scheme == "unix" {
		return "http://d"
	}
	return "http://" + u.Host
}

// BaseURL returns the base URL of APIClient.
func (client *APIClient) BaseURL() string {
	return client.baseURL
}

// GetAPIPath returns the versioned request path to call the api.
// It appends the query parameters to the path if they are not empty.
func (client *APIClient) GetAPIPath(path string, query url.Values) string {
	var apiPath string
	if client.version != "" {
		v := strings.TrimPrefix(client.version, "v")
		apiPath = fmt.Sprintf("/v%s%s", v, path)
	} else {
		apiPath = path
	}

	u := url.URL{
		Path: apiPath,
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	return u.String()
}

// UpdateClientVersion sets client version new value.
func (client *APIClient) UpdateClientVersion(v string) {
	client.version = v
}
