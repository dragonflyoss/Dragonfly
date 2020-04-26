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

package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/url"
	"path/filepath"
	"regexp"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/constant"
	"github.com/dragonflyoss/Dragonfly/pkg/dflog"
	dferr "github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/rate"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

var fs = afero.NewOsFs()

// -----------------------------------------------------------------------------
// Properties

// Properties holds all configurable properties of dfdaemon.
// The default path is '/etc/dragonfly/dfdaemon.yml'
// For examples:
//     dfget_flags: ["--node","192.168.33.21","--verbose","--ip","192.168.33.23",
//                   "--port","15001","--expiretime","3m0s","--alivetime","5m0s",
//                   "-f","filterParam1&filterParam2"]
//
//     registry_mirror:
//       # url for the registry mirror
//       remote: https://index.docker.io
//       # whether to ignore https certificate errors
//       insecure: false
//       # optional certificates if the remote server uses self-signed certificates
//       certs: []
//       # whether to request the remote registry directly
//       direct: false
//
//     proxies:
//     # proxy all http image layer download requests with dfget
//     - regx: blobs/sha256.*
//     # change http requests to some-registry to https and proxy them with dfget
//     - regx: some-registry/
//       use_https: true
//     # proxy requests directly, without dfget
//     - regx: no-proxy-reg
//       direct: true
//     # proxy requests with redirect
//     - regx: some-registry
//       redirect: another-registry
//
//     hijack_https:
//       # key pair used to hijack https requests
//       cert: df.crt
//       key: df.key
//       hosts:
//       - regx: mirror.aliyuncs.com:443  # regexp to match request hosts
//         # whether to ignore https certificate errors
//         insecure: false
//         # optional certificates if the host uses self-signed certificates
//         certs: []
type Properties struct {
	// Registry mirror settings
	RegistryMirror *RegistryMirror `yaml:"registry_mirror" json:"registry_mirror"`

	// Proxies is the list of rules for the transparent proxy. If no rules
	// are provided, all requests will be proxied directly. Request will be
	// proxied with the first matching rule.
	Proxies []*Proxy `yaml:"proxies" json:"proxies"`

	// HijackHTTPS is the list of hosts whose https requests should be hijacked
	// by dfdaemon. Dfdaemon will be able to proxy requests from them with dfget
	// if the url matches the proxy rules. The first matched rule will be used.
	HijackHTTPS *HijackConfig `yaml:"hijack_https" json:"hijack_https"`

	// https options
	Port    uint   `yaml:"port" json:"port"`
	HostIP  string `yaml:"hostIp" json:"hostIp"`
	CertPem string `yaml:"certpem" json:"certpem"`
	KeyPem  string `yaml:"keypem" json:"keypem"`

	Verbose bool `yaml:"verbose" json:"verbose"`

	MaxProcs int `yaml:"maxprocs" json:"maxprocs"`

	// dfget config
	DfgetFlags []string  `yaml:"dfget_flags" json:"dfget_flags"`
	SuperNodes []string  `yaml:"supernodes" json:"supernodes"`
	RateLimit  rate.Rate `yaml:"ratelimit" json:"ratelimit"`
	WorkHome   string    `yaml:"workHome" json:"workHome,omitempty"`
	DFRepo     string    `yaml:"localrepo" json:"localrepo"`
	DFPath     string    `yaml:"dfpath" json:"dfpath"`

	LogConfig  dflog.LogConfig `yaml:"logConfig" json:"logConfig"`
	LocalIP    string          `yaml:"localIP" json:"localIP"`
	PeerPort   int             `yaml:"peerPort" json:"peerPort"`
	StreamMode bool            `yaml:"streamMode" json:"streamMode"`
}

// Validate validates the config
func (p *Properties) Validate() error {
	if p.Port <= 0 || p.Port > 65535 {
		return dferr.Newf(
			constant.CodeExitPortInvalid,
			"invalid port %d", p.Port,
		)
	}

	if !filepath.IsAbs(p.DFRepo) {
		return dferr.Newf(
			constant.CodeExitPathNotAbs,
			"local repo %s is not absolute", p.DFRepo,
		)
	}

	return nil
}

// DFGetConfig returns config for dfget downloader.
func (p *Properties) DFGetConfig() DFGetConfig {
	// init DfgetFlags
	var dfgetFlags []string
	dfgetFlags = append(dfgetFlags, p.DfgetFlags...)
	dfgetFlags = append(dfgetFlags, "--dfdaemon")
	if p.Verbose {
		dfgetFlags = append(dfgetFlags, "--verbose")
	}

	dfgetConfig := DFGetConfig{
		DfgetFlags: dfgetFlags,
		SuperNodes: p.SuperNodes,
		RateLimit:  p.RateLimit.String(),
		DFRepo:     p.DFRepo,
		DFPath:     p.DFPath,
		LocalIP:    p.LocalIP,
		PeerPort:   p.PeerPort,
	}
	if p.HijackHTTPS != nil {
		dfgetConfig.HostsConfig = p.HijackHTTPS.Hosts
	}
	if p.RegistryMirror != nil {
		exp, err := NewRegexp(p.RegistryMirror.Remote.Host)
		if err == nil {
			dfgetConfig.HostsConfig = append(dfgetConfig.HostsConfig, &HijackHost{
				Regx:     exp,
				Insecure: p.RegistryMirror.Insecure,
				Certs:    p.RegistryMirror.Certs,
			})
		}
	}
	return dfgetConfig
}

// DFGetConfig configures how dfdaemon calls dfget.
type DFGetConfig struct {
	DfgetFlags  []string      `yaml:"dfget_flags"`
	SuperNodes  []string      `yaml:"supernodes"`
	RateLimit   string        `yaml:"ratelimit"`
	DFRepo      string        `yaml:"localrepo"`
	DFPath      string        `yaml:"dfpath"`
	HostsConfig []*HijackHost `yaml:"hosts" json:"hosts"`
	PeerPort    int           `yaml:"peerPort"`
	LocalIP     string        `yaml:"localIP"`
}

// RegistryMirror configures the mirror of the official docker registry
type RegistryMirror struct {
	// Remote url for the registry mirror, default is https://index.docker.io
	Remote *URL `yaml:"remote" json:"remote"`

	// Optional certificates if the mirror uses self-signed certificates
	Certs *CertPool `yaml:"certs" json:"certs"`

	// Whether to ignore certificates errors for the registry
	Insecure bool `yaml:"insecure" json:"insecure"`

	// Request the remote registry directly.
	Direct bool `yaml:"direct" json:"direct"`
}

// TLSConfig returns the tls.Config used to communicate with the mirror.
func (r *RegistryMirror) TLSConfig() *tls.Config {
	if r == nil {
		return nil
	}

	cfg := &tls.Config{
		InsecureSkipVerify: r.Insecure,
	}

	if r.Certs != nil {
		cfg.RootCAs = r.Certs.CertPool
	}

	return cfg
}

// HijackConfig represents how dfdaemon hijacks http requests.
type HijackConfig struct {
	Cert  string        `yaml:"cert" json:"cert"`
	Key   string        `yaml:"key" json:"key"`
	Hosts []*HijackHost `yaml:"hosts" json:"hosts"`
}

// HijackHost is a hijack rule for the hosts that matches Regx.
type HijackHost struct {
	Regx     *Regexp   `yaml:"regx" json:"regx"`
	Insecure bool      `yaml:"insecure" json:"insecure"`
	Certs    *CertPool `yaml:"certs" json:"certs"`
}

// URL is simple wrapper around url.URL to make it unmarshallable from a string.
type URL struct {
	*url.URL
}

// NewURL parses url from the given string.
func NewURL(s string) (*URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	return &URL{u}, nil
}

// UnmarshalYAML implements yaml.Unmarshaller.
func (u *URL) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return u.unmarshal(unmarshal)
}

// UnmarshalJSON implements json.Unmarshaller.
func (u *URL) UnmarshalJSON(b []byte) error {
	return u.unmarshal(func(v interface{}) error { return json.Unmarshal(b, v) })
}

func (u *URL) unmarshal(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	parsed, err := url.Parse(s)
	if err != nil {
		return err
	}

	u.URL = parsed
	return nil
}

// MarshalJSON implements json.Marshaller to print the url.
func (u *URL) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

// MarshalYAML implements yaml.Marshaller to print the url.
func (u *URL) MarshalYAML() (interface{}, error) {
	return u.String(), nil
}

// CertPool is a wrapper around x509.CertPool, which can be unmarshalled and
// constructed from a list of filenames.
type CertPool struct {
	Files []string
	*x509.CertPool
}

// UnmarshalYAML implements yaml.Unmarshaller.
func (cp *CertPool) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return cp.unmarshal(unmarshal)
}

// UnmarshalJSON implements json.Unmarshaller.
func (cp *CertPool) UnmarshalJSON(b []byte) error {
	return cp.unmarshal(func(v interface{}) error { return json.Unmarshal(b, v) })
}

func (cp *CertPool) unmarshal(unmarshal func(interface{}) error) error {
	if err := unmarshal(&cp.Files); err != nil {
		return err
	}

	pool, err := certPoolFromFiles(cp.Files...)
	if err != nil {
		return err
	}

	cp.CertPool = pool
	return nil
}

// MarshalJSON implements json.Marshaller to print the cert pool.
func (cp *CertPool) MarshalJSON() ([]byte, error) {
	return json.Marshal(cp.Files)
}

// MarshalYAML implements yaml.Marshaller to print the cert pool.
func (cp *CertPool) MarshalYAML() (interface{}, error) {
	return cp.Files, nil
}

// Regexp is a simple wrapper around regexp. Regexp to make it unmarshallable from a string.
type Regexp struct {
	*regexp.Regexp
}

// NewRegexp returns a new Regexp instance compiled from the given string.
func NewRegexp(exp string) (*Regexp, error) {
	r, err := regexp.Compile(exp)
	if err != nil {
		return nil, err
	}
	return &Regexp{r}, nil
}

// UnmarshalYAML implements yaml.Unmarshaller.
func (r *Regexp) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return r.unmarshal(unmarshal)
}

// UnmarshalJSON implements json.Unmarshaller.
func (r *Regexp) UnmarshalJSON(b []byte) error {
	return r.unmarshal(func(v interface{}) error { return json.Unmarshal(b, v) })
}

func (r *Regexp) unmarshal(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	exp, err := regexp.Compile(s)
	if err == nil {
		r.Regexp = exp
	}
	return err
}

// MarshalJSON implements json.Marshaller to print the regexp.
func (r *Regexp) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

// MarshalYAML implements yaml.Marshaller to print the regexp.
func (r *Regexp) MarshalYAML() (interface{}, error) {
	return r.String(), nil
}

// certPoolFromFiles returns an *x509.CertPool constructed from the given files.
// If no files are given, (nil, nil) will be returned.
func certPoolFromFiles(files ...string) (*x509.CertPool, error) {
	if len(files) == 0 {
		return nil, nil
	}

	roots := x509.NewCertPool()
	for _, f := range files {
		cert, err := afero.ReadFile(fs, f)
		if err != nil {
			return nil, errors.Wrapf(err, "read cert file %s", f)
		}
		if !roots.AppendCertsFromPEM(cert) {
			return nil, errors.Errorf("invalid cert: %s", f)
		}
	}
	return roots, nil
}

// Proxy describes a regular expression matching rule for how to proxy a request.
type Proxy struct {
	Regx     *Regexp `yaml:"regx" json:"regx"`
	UseHTTPS bool    `yaml:"use_https" json:"use_https"`
	Direct   bool    `yaml:"direct" json:"direct"`
	// Redirect is the host to redirect to, if not empty
	Redirect string `yaml:"redirect" json:"redirect"`
}

// NewProxy returns a new proxy rule with given attributes.
func NewProxy(regx string, useHTTPS bool, direct bool, redirect string) (*Proxy, error) {
	exp, err := NewRegexp(regx)
	if err != nil {
		return nil, errors.Wrap(err, "invalid regexp")
	}

	return &Proxy{
		Regx:     exp,
		UseHTTPS: useHTTPS,
		Direct:   direct,
		Redirect: redirect,
	}, nil
}

// Match checks if the given url matches the rule.
func (r *Proxy) Match(url string) bool {
	return r.Regx != nil && r.Regx.MatchString(url)
}
