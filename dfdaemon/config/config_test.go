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
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/constant"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/rate"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

var testCrt = `-----BEGIN CERTIFICATE-----
MIICKzCCAZQCCQDZrCsm2rX81DANBgkqhkiG9w0BAQUFADBaMQswCQYDVQQGEwJD
TjERMA8GA1UECAwIWmhlamlhbmcxETAPBgNVBAcMCEhhbmd6aG91MQ4wDAYDVQQK
DAVsb3d6ajEVMBMGA1UEAwwMZGZkYWVtb24uY29tMB4XDTE5MDIyNTAyNDYwN1oX
DTE5MDMyNzAyNDYwN1owWjELMAkGA1UEBhMCQ04xETAPBgNVBAgMCFpoZWppYW5n
MREwDwYDVQQHDAhIYW5nemhvdTEOMAwGA1UECgwFbG93emoxFTATBgNVBAMMDGRm
ZGFlbW9uLmNvbTCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAtX1VzZRg1tgF
D0AFkUW2FpakkrhRzFuukWepoN0LfFSS/rNf8v1823de1SkpXBHsm2pMf94BIdmY
NDWH1tk27i4V5xydjNqxbdjjNjGHedBAM2tRQWWQuJAEo12sWUVYwDyN7RbL6wnz
7Egeac023FA9JhfMxaDvJHqJHVuKW3kCAwEAATANBgkqhkiG9w0BAQUFAAOBgQCT
VrDbo4m3QkcUT8ohuAUD8OHjTwJAuoxqVdHm+SpgjBYMLQgqXAPwaTGsIvx+32h2
J88xU3xXABE5QsNNbqLcMgQoXeMmqk1WuUhxXzTXT5h5gdW53faxV5M5Cb3zI8My
PPpBF5Cw+khgkJcY/ezKjHIvyABJwdzW8aAqwDBFAQ==
-----END CERTIFICATE-----`

type configTestSuite struct {
	suite.Suite
}

func (ts *configTestSuite) TestValidatePort() {
	c := defaultConfig()
	r := ts.Require()

	for _, p := range []uint{0, 65536} {
		c.Port = p
		err := c.Validate()
		r.NotNil(err)
		de, ok := err.(*errortypes.DfError)
		r.True(ok)
		r.Equal(constant.CodeExitPortInvalid, de.Code)
	}

	for _, p := range []uint{80, 2001, 65001, 65535} {
		c.Port = p
		r.Nil(c.Validate())
	}
}

func (ts *configTestSuite) TestValidateDFRepo() {
	c := defaultConfig()
	r := ts.Require()

	c.DFRepo = "/tmp"
	r.Nil(c.Validate())

	c.DFRepo = "tmp"
	r.Equal(constant.CodeExitPathNotAbs, getCode(c.Validate()))
}

func (ts *configTestSuite) TestURLNew() {
	r := ts.Require()

	validURL := "http://xxx/aaa"
	u, err := NewURL(validURL)
	r.Nil(err)
	r.Equal(validURL, u.String())

	invalidURL := ":"
	u, err = NewURL(invalidURL)
	r.NotNil(err)
	r.Nil(u)
}

func (ts *configTestSuite) TestURLUnmarshal() {
	r := ts.Require()

	var wrapper struct {
		URL *URL `yaml:"url"`
	}

	exampleURL := "http://xxx"

	r.Nil(
		yaml.Unmarshal(
			[]byte(fmt.Sprintf("url: %s", exampleURL)),
			&wrapper,
		),
	)
	r.NotNil(wrapper.URL)
	r.Equal(wrapper.URL.String(), exampleURL)
}

func (ts *configTestSuite) TestRegexpNew() {
	r := ts.Require()

	invalidRegexp := `\K`
	_, err := NewRegexp(invalidRegexp)
	r.NotNil(err)

	validRegexp := ".*"
	_, err = NewRegexp(validRegexp)
	r.Nil(err)
}

func (ts *configTestSuite) TestRegexpUnmarshal() {
	r := ts.Require()

	var wrapper struct {
		Regexp *Regexp `yaml:"regx"`
	}

	exampleRegexp := "http://xxx"

	content := fmt.Sprintf("regx: %s", exampleRegexp)
	r.Nil(yaml.Unmarshal([]byte(content), &wrapper))
	r.NotNil(wrapper.Regexp)
	r.Equal(wrapper.Regexp.String(), exampleRegexp)
	marshaled, err := yaml.Marshal(wrapper)
	r.Nil(err)
	r.True(len(marshaled) > 0)
}

func (ts *configTestSuite) TestCertPoolUnmarshal() {
	r := ts.Require()

	currentFs := fs
	defer func() { fs = currentFs }()
	fs = afero.NewMemMapFs()

	type certWrapper struct {
		Cert *CertPool `yaml:"cert"`
	}

	{
		w := certWrapper{}
		r.Nil(yaml.Unmarshal([]byte("cert: []"), &w))
		r.Nil(w.Cert.CertPool)
	}

	{
		w := certWrapper{}
		certFile := "test.crt"
		r.Nil(afero.WriteFile(fs, certFile, []byte(testCrt), os.ModePerm))
		r.Nil(
			yaml.Unmarshal(
				[]byte(fmt.Sprintf("cert: [%s]", certFile)),
				&w,
			),
		)
		r.NotNil(w.Cert)
		r.NotNil(w.Cert.CertPool)
	}

	{
		w := certWrapper{}
		err := yaml.Unmarshal(
			[]byte(fmt.Sprintf("cert: [%s]", "not-exists")),
			&w,
		)
		r.NotNil(err)
		r.True(os.IsNotExist(errors.Cause(err)))
	}

	{
		w := certWrapper{}
		certFile := "invalid.crt"
		r.Nil(afero.WriteFile(fs, certFile, []byte("xxx"), os.ModePerm))
		err := yaml.Unmarshal(
			[]byte(fmt.Sprintf("cert: [%s]", certFile)),
			&w,
		)
		r.NotNil(err)
		r.EqualError(err, fmt.Sprintf("invalid cert: %s", certFile))
	}
}

func (ts *configTestSuite) TestProxyNew() {
	r := ts.Require()

	{
		validRegexp := ".*"
		useHTTPS := false
		direct := false
		p, err := NewProxy(validRegexp, useHTTPS, direct, "")
		r.Nil(err)
		r.NotNil(p)
		r.Equal(useHTTPS, p.UseHTTPS)
		r.Equal(direct, p.Direct)
		r.Equal(validRegexp, p.Regx.String())
	}

	{
		p, err := NewProxy(`\K`, false, false, "")
		r.Nil(p)
		r.NotNil(err)
		r.True(strings.HasPrefix(err.Error(), "invalid regexp:"))
	}
}

func (ts *configTestSuite) TestProxyMatch() {
	r := ts.Require()
	p, err := NewProxy("blobs/sha256.*", false, false, "")
	r.Nil(err)
	r.NotNil(p)

	for _, match := range []string{"blobs/sha256:xxx", "http://xx/blobs/sha256:xxx"} {
		r.True(p.Match(match))
	}

	for _, unmatch := range []string{"", "blobs", "sha256", "xxx"} {
		r.False(p.Match(unmatch))
	}

}

func (ts *configTestSuite) TestMirrorTLSConfig() {
	r := ts.Require()

	var nilMirror *RegistryMirror
	r.Nil(nilMirror.TLSConfig())

	m := &RegistryMirror{
		Certs: &CertPool{
			CertPool: x509.NewCertPool(),
		},
	}
	r.Equal(m.Certs.CertPool, m.TLSConfig().RootCAs)

	m.Insecure = true
	r.Equal(m.Insecure, m.TLSConfig().InsecureSkipVerify)
}

func (ts *configTestSuite) TestDFGetConfig() {
	c := defaultConfig()
	r := ts.Require()

	{
		flagsConfigs := c.DFGetConfig()
		r.Contains(flagsConfigs.DfgetFlags, "--dfdaemon")
		r.Equal(c.SuperNodes, flagsConfigs.SuperNodes)
		r.Equal(c.RateLimit.String(), flagsConfigs.RateLimit)
		r.Equal(c.DFRepo, flagsConfigs.DFRepo)
		r.Equal(c.DFPath, flagsConfigs.DFPath)

		c.Verbose = true
		flagsConfigs = c.DFGetConfig()
		r.Contains(flagsConfigs.DfgetFlags, "--verbose")
	}

	{
		aRegex, err := NewRegexp("a.registry.com")
		r.Nil(err)
		bRegex, err := NewRegexp("b.registry.com")
		r.Nil(err)
		c.HijackHTTPS = &HijackConfig{
			Hosts: []*HijackHost{
				{
					Regx:     aRegex,
					Insecure: true,
					Certs:    nil,
				},
				{
					Regx:     bRegex,
					Insecure: false,
					Certs: &CertPool{
						CertPool: x509.NewCertPool(),
					},
				},
			},
		}
		r.Equal(c.HijackHTTPS.Hosts, c.DFGetConfig().HostsConfig)
	}

	{
		url, err := NewURL("c.registry.com")
		r.Nil(err)
		c.RegistryMirror = &RegistryMirror{
			Remote: url,
			Certs: &CertPool{
				CertPool: x509.NewCertPool(),
			},
			Insecure: false,
		}
		mirrorConfigs := c.DFGetConfig()
		r.Equal(len(mirrorConfigs.HostsConfig), 3)
		r.Equal(mirrorConfigs.HostsConfig[len(mirrorConfigs.HostsConfig)-1].Regx.String(), c.RegistryMirror.Remote.Host)
		r.Equal(mirrorConfigs.HostsConfig[len(mirrorConfigs.HostsConfig)-1].Certs, c.RegistryMirror.Certs)
		r.Equal(mirrorConfigs.HostsConfig[len(mirrorConfigs.HostsConfig)-1].Insecure, c.RegistryMirror.Insecure)
	}
}

func (ts *configTestSuite) TestSerialization() {
	currentFs := fs
	defer func() { fs = currentFs }()
	fs = afero.NewMemMapFs()
	r := ts.Require()
	r.Nil(afero.WriteFile(fs, "test.crt", []byte(testCrt), os.ModePerm))

	cases := []struct {
		serializer interface {
			Unmarshal([]byte, interface{}) error
			Marshal(interface{}) ([]byte, error)
		}
		success  bool
		text     string
		receiver interface{}
	}{
		{&yamlM{}, true, `.*`, &Regexp{}},
		{&yamlM{}, true, `http://xxx`, &URL{}},
		{&yamlM{}, true, "cert:\n- test.crt", &struct {
			Cert *CertPool `yaml:"cert"`
		}{}},
		{&yamlM{}, false, "cert:\n- none.crt", &struct {
			Cert *CertPool `yaml:"cert"`
		}{}},
		{&jsonM{}, true, `{"reg":".*"}`, &struct {
			Reg *Regexp `json:"reg"`
		}{}},
		{&jsonM{}, false, `{"reg":1}`, &struct {
			Reg *Regexp `json:"reg"`
		}{}},
		{&jsonM{}, true, `{"url":"http://xxx"}`, &struct {
			URL *URL `json:"url"`
		}{}},
		{&jsonM{}, false, `{"url":1}`, &struct {
			URL *URL `json:"url"`
		}{}},
		{&jsonM{}, false, `{"url":":"}`, &struct {
			URL *URL `json:"url"`
		}{}},
		{&jsonM{}, true, `{"cert":["test.crt"]}`, &struct {
			Cert *CertPool `json:"cert"`
		}{}},
		{&jsonM{}, false, `{"cert":"test.crt"}`, &struct {
			Cert *CertPool `json:"cert"`
		}{}},
	}

	for _, c := range cases {
		err := c.serializer.Unmarshal([]byte(c.text), c.receiver)
		if c.success {
			r.Nil(err)
			s, err := c.serializer.Marshal(c.receiver)
			r.Nil(err)
			r.Equal(c.text, strings.TrimSpace(string(s)))
		} else {
			r.NotNilf(err, "%s %+v", c.text, c.receiver)
		}
	}
}

type jsonM struct{}

func (m *jsonM) Marshal(d interface{}) ([]byte, error)      { return json.Marshal(d) }
func (m *jsonM) Unmarshal(text []byte, d interface{}) error { return json.Unmarshal(text, d) }

type yamlM struct{}

func (m *yamlM) Marshal(d interface{}) ([]byte, error)      { return yaml.Marshal(d) }
func (m *yamlM) Unmarshal(text []byte, d interface{}) error { return yaml.Unmarshal(text, d) }

func getCode(err error) int {
	if de, ok := err.(*errortypes.DfError); ok {
		return de.Code
	}
	return 0
}

func defaultConfig() *Properties {
	return &Properties{
		Port:      65001,
		HostIP:    "127.0.0.1",
		DFRepo:    "/tmp",
		DFPath:    "/tmp",
		RateLimit: 20 * rate.MB,
	}
}

func TestConfig(t *testing.T) {
	suite.Run(t, &configTestSuite{})
}
