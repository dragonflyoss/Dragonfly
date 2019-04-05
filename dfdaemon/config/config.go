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
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/dragonflyoss/Dragonfly/common/util"
)

// -----------------------------------------------------------------------------
// Properties

// NewProperties create a new properties with default values.
func NewProperties() *Properties {
	return &Properties{}
}

// Properties holds all configurable properties of dfdaemon.
// The default path is '/etc/dragonfly/dfdaemon.yml'
// For examples:
//     registries:
//         - regx: (^localhost$)|(^127.0.0.1$)
//           schema: http
//           host: a.com
//         - regx: ^reg.com:1001$
//           schema: http
//           host: reg.com
//         - regx: ^reg.com:1002$
//           schema: https
//           host: reg.com
//           certs: ['/etc/ssl/reg.com/server.crt']
//     proxies:
//     # proxy all http image layer download requests with dfget
//     - regx: blob/sha256/.*
//     # change http requests to some-registry to https and proxy them with dfget
//     - regx: some-registry/
//       use_https: true
//     # proxy requests directly, without dfget
//     - regx: no-proxy-reg
//       direct: true
type Properties struct {
	// Registries the more front the position, the higher priority.
	// You could add an empty Registry at the end to proxy all other requests
	// with those origin schema and host.
	Registries []*Registry `yaml:"registries"`
	// Proxies is the list of rules for the transparent proxy. If no rules
	// are provided, all requests will be proxied directly. Request will be
	// proxied with the first matching rule.
	Proxies []*Proxy `yaml:"proxies"`
}

// Load loads properties from config file.
func (p *Properties) Load(path string) error {
	if err := util.LoadYaml(path, p); err != nil {
		return err
	}

	var tmp []*Registry
	for _, v := range p.Registries {
		if v != nil {
			if err := v.init(); err != nil {
				return err
			}
			tmp = append(tmp, v)
		}
	}
	p.Registries = tmp

	var tmpProxies []*Proxy
	for _, v := range p.Proxies {
		if v != nil {
			if err := v.init(); err != nil {
				return err
			}
			tmpProxies = append(tmpProxies, v)
		}
	}
	p.Proxies = tmpProxies

	return nil
}

// -----------------------------------------------------------------------------
// Registry

// NewRegistry create and init registry proxy with the giving values.
func NewRegistry(schema, host, regx string, certs []string) (*Registry, error) {
	reg := &Registry{
		Schema: schema,
		Host:   host,
		Certs:  certs,
		Regx:   regx,
	}
	if err := reg.init(); err != nil {
		return nil, err
	}
	return reg, nil
}

// Registry is the proxied registry base information.
type Registry struct {
	// Schema can be 'http', 'https' or empty. It will use dfdaemon's schema if
	// it's empty.
	Schema string `yaml:"schema"`

	// Host is the host of proxied registry, including ip and port.
	Host string `yaml:"host"`

	// Certs is the path of server-side certification. It should be provided when
	// the 'Schema' is 'https' and the dfdaemon is worked on proxy pattern and
	// the proxied registry is self-certificated.
	// The server-side certification could be get by using this command:
	// openssl x509 -in <(openssl s_client -showcerts -servername xxx -connect xxx:443 -prexit 2>/dev/null)
	Certs []string `yaml:"certs"`

	// Regx is a regular expression, dfdaemon use this registry to process the
	// requests whose host is matched.
	Regx string `yaml:"regx"`

	compiler  *regexp.Regexp
	tlsConfig *tls.Config
}

// Match reports whether the Registry matches the string s.
func (r *Registry) Match(s string) bool {
	return r.compiler != nil && r.compiler.MatchString(s)
}

// TLSConfig returns a initialized tls.Config instance.
func (r *Registry) TLSConfig() *tls.Config {
	return r.tlsConfig
}

func (r *Registry) init() error {
	if err := r.validate(); err != nil {
		return err
	}

	c, err := regexp.Compile(r.Regx)
	if err != nil {
		return err
	}
	r.compiler = c

	return r.initTLSConfig()
}

func (r *Registry) validate() error {
	r.Schema = strings.ToLower(r.Schema)
	if r.Schema != "http" && r.Schema != "https" && r.Schema != "" {
		return fmt.Errorf("invalid schema:%s", r.Schema)
	}
	return nil
}

func (r *Registry) initTLSConfig() error {
	size := len(r.Certs)
	if size <= 0 {
		r.tlsConfig = &tls.Config{InsecureSkipVerify: true}
		return nil
	}

	roots := x509.NewCertPool()
	for i := 0; i < size; i++ {
		cert, err := ioutil.ReadFile(r.Certs[i])
		if err != nil {
			return err
		}
		if !roots.AppendCertsFromPEM(cert) {
			return fmt.Errorf("invalid cert:%s", r.Certs[i])
		}
	}
	r.tlsConfig = &tls.Config{RootCAs: roots}
	return nil
}

// Proxy describe a regular expression matching rule for how to proxy a request
type Proxy struct {
	Regx     string `yaml:"regx"`
	UseHTTPS bool   `yaml:"use_https"`
	Direct   bool   `yaml:"direct"`

	match *regexp.Regexp
}

// NewProxy returns a new proxy rule with given attributes
func NewProxy(regx string, useHTTPS bool, direct bool) (*Proxy, error) {
	p := &Proxy{
		Regx:     regx,
		UseHTTPS: useHTTPS,
		Direct:   direct,
	}
	if err := p.init(); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *Proxy) init() (err error) {
	r.match, err = regexp.Compile(r.Regx)
	return err
}

// Match checks if the given url matches the rule
func (r *Proxy) Match(url string) bool {
	return r.match != nil && r.match.MatchString(url)
}
