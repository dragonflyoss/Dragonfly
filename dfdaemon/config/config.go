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
	"io/ioutil"
	"net/url"
	"regexp"

	"github.com/dragonflyoss/Dragonfly/common/util"

	"github.com/pkg/errors"
)

const officialRegistry = "https://index.docker.io"

// -----------------------------------------------------------------------------
// Properties

// NewProperties create a new properties with default values.
func NewProperties() *Properties {
	u, _ := NewURL(officialRegistry)
	return &Properties{
		RegistryMirror: &RegistryMirror{
			Remote: u,
		},
	}
}

// Properties holds all configurable properties of dfdaemon.
// The default path is '/etc/dragonfly/dfdaemon.yml'
// For examples:
//     registry_mirror:
//       # url for the registry mirror
//       remote: https://index.docker.io
//       # whether to ignore https certificate errors
//       insecure: false
//       # optional certificates if the remote server uses self-signed certificates
//       certs: []
//
//     proxies:
//     # proxy all http image layer download requests with dfget
//     - regx: blobs/sha256:.*
//     # change http requests to some-registry to https and proxy them with dfget
//     - regx: some-registry/
//       use_https: true
//     # proxy requests directly, without dfget
//     - regx: no-proxy-reg
//       direct: true
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
	RegistryMirror *RegistryMirror `yaml:"registry_mirror"`

	// Proxies is the list of rules for the transparent proxy. If no rules
	// are provided, all requests will be proxied directly. Request will be
	// proxied with the first matching rule.
	Proxies []*Proxy `yaml:"proxies"`

	// HijackHTTPS is the list of hosts whose https requests should be hijacked
	// by dfdaemon. Dfdaemon will be able to proxy requests from them with dfget
	// if the url matches the proxy rules. The first matched rule will be used.
	HijackHTTPS *HijackConfig `yaml:"hijack_https"`
}

// RegistryMirror configures the mirror of the official docker registry
type RegistryMirror struct {
	// Remote url for the registry mirror, default is https://index.docker.io
	Remote *URL `yaml:"remote"`

	// Optional certificates if the mirror uses self-signed certificates
	Certs *CertPool `yaml:"certs"`

	// Whether to ignore certificates errors for the registry
	Insecure bool `yaml:"insecure"`
}

// TLSConfig returns the tls.Config used to communicate with the mirror
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

// HijackConfig represents how dfdaemon hijacks http requests
type HijackConfig struct {
	Cert  string        `yaml:"cert"`
	Key   string        `yaml:"key"`
	Hosts []*HijackHost `yaml:"hosts"`
}

// HijackHost is a hijack rule for the hosts that matches Regx
type HijackHost struct {
	Regx     *Regexp   `yaml:"regx"`
	Insecure bool      `yaml:"insecure"`
	Certs    *CertPool `yaml:"certs"`
}

// URL is simple wrapper around url.URL to make it unmarshallable from a string
type URL struct {
	*url.URL
}

// NewURL parses url from the given string
func NewURL(s string) (*URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	return &URL{u}, nil
}

// UnmarshalYAML implements yaml.Unmarshaller
func (u *URL) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return u.unmarshal(unmarshal)
}

// UnmarshalJSON implements json.Unmarshaller
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

// MarshalJSON implements json.Marshaller to print the url
func (u *URL) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

// CertPool is a wrapper around x509.CertPool, which can be unmarshalled and
// constructed from a list of filenames
type CertPool struct {
	files []string
	*x509.CertPool
}

// UnmarshalYAML implements yaml.Unmarshaller
func (cp *CertPool) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return cp.unmarshal(unmarshal)
}

// UnmarshalJSON implements json.Unmarshaller
func (cp *CertPool) UnmarshalJSON(b []byte) error {
	return cp.unmarshal(func(v interface{}) error { return json.Unmarshal(b, v) })
}

func (cp *CertPool) unmarshal(unmarshal func(interface{}) error) error {
	if err := unmarshal(&cp.files); err != nil {
		return err
	}

	pool, err := certPoolFromFiles(cp.files...)
	if err != nil {
		return err
	}

	cp.CertPool = pool
	return nil
}

// Regexp is simple wrapper around regexp.Regexp to make it unmarshallable from a string
type Regexp struct {
	*regexp.Regexp
}

// NewRegexp returns new Regexp instance compiled from the given string
func NewRegexp(exp string) (*Regexp, error) {
	r, err := regexp.Compile(exp)
	if err != nil {
		return nil, err
	}
	return &Regexp{r}, nil
}

// UnmarshalYAML implements yaml.Unmarshaller
func (r *Regexp) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return r.unmarshal(unmarshal)
}

// UnmarshalJSON implements json.Unmarshaller
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

// MarshalJSON implements json.Marshaller to print the regexp
func (r *Regexp) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

// certPoolFromFiles returns an *x509.CertPool constructed from the given files.
// If no files are given, (nil, nil) will be returned.
func certPoolFromFiles(files ...string) (*x509.CertPool, error) {
	if len(files) == 0 {
		return nil, nil
	}

	roots := x509.NewCertPool()
	for _, f := range files {
		cert, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, errors.Wrapf(err, "read cert file %s", f)
		}
		if !roots.AppendCertsFromPEM(cert) {
			return nil, errors.Errorf("invalid cert: %s", f)
		}
	}
	return roots, nil
}

// Load loads properties from config file.
func (p *Properties) Load(path string) error {
	if err := util.LoadYaml(path, p); err != nil {
		return err
	}

	return nil
}

// Proxy describe a regular expression matching rule for how to proxy a request
type Proxy struct {
	Regx     *Regexp `yaml:"regx"`
	UseHTTPS bool    `yaml:"use_https"`
	Direct   bool    `yaml:"direct"`
}

// NewProxy returns a new proxy rule with given attributes
func NewProxy(regx string, useHTTPS bool, direct bool) (*Proxy, error) {
	exp, err := NewRegexp(regx)
	if err != nil {
		return nil, errors.Wrap(err, "invalid regexp")
	}

	return &Proxy{
		Regx:     exp,
		UseHTTPS: useHTTPS,
		Direct:   direct,
	}, nil
}

// Match checks if the given url matches the rule
func (r *Proxy) Match(url string) bool {
	return r.Regx != nil && r.Regx.MatchString(url)
}
