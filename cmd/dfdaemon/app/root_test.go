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

package app

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
)

type rootTestSuite struct {
	suite.Suite
}

func (ts *rootTestSuite) TestConfigNotFound() {
	r := ts.Require()
	v := viper.New()
	fs := afero.NewMemMapFs()
	v.SetFs(fs)

	r.False(afero.Exists(fs, rootCmd.Flag("config").DefValue))
	r.Nil(bindRootFlags(v))
	r.Nil(readConfigFile(v, rootCmd))

	fake := generateFakeFilename(fs)
	r.False(afero.Exists(fs, fake))
	rootCmd.Flags().Set("config", fake)
	r.Equal(v.GetString("config"), fake)
	r.True(os.IsNotExist(errors.Cause(readConfigFile(v, rootCmd))))
}

func (ts *rootTestSuite) TestNodeFlag() {
	r := ts.Require()
	fs := afero.NewMemMapFs()

	fs.Create("/dfget")

	configName := "dfdaemon.yml"
	file, err := fs.Create(configName)
	r.Nil(err)
	file.WriteString("supernodes:\n- 127.0.0.1:6666")
	file.Close()

	// flag not set, should use config file
	{
		v := viper.New()
		v.SetFs(fs)
		v.Set("dfpath", "/")
		rootCmd.Flags().Set("config", configName)
		r.Nil(bindRootFlags(v))
		r.Nil(readConfigFile(v, rootCmd))
		cfg, err := getConfigFromViper(rootCmd, v)
		r.Nil(err)
		r.Equal([]string{"127.0.0.1:6666"}, cfg.SuperNodes)
	}

	// flag not set, config file doesn't exist, should be nil
	{
		v := viper.New()
		v.SetFs(fs)
		v.Set("dfpath", "/")
		rootCmd.Flags().Set("config", "xxx")
		r.Nil(bindRootFlags(v))
		r.NotNil(readConfigFile(v, rootCmd))
		cfg, err := getConfigFromViper(rootCmd, v)
		r.Nil(err)
		r.EqualValues([]string(nil), cfg.SuperNodes)
	}

	// when --node flag is set, should always use the flag
	rootCmd.Flags().Set("node", "127.0.0.1:7777")

	{
		v := viper.New()
		v.SetFs(fs)
		v.Set("dfpath", "/")
		rootCmd.Flags().Set("config", "xxx")
		r.Nil(bindRootFlags(v))
		r.NotNil(readConfigFile(v, rootCmd))
		cfg, err := getConfigFromViper(rootCmd, v)
		r.Nil(err)
		r.Equal([]string{"127.0.0.1:7777"}, cfg.SuperNodes)
	}

	{
		v := viper.New()
		v.SetFs(fs)
		v.Set("dfpath", "/")
		rootCmd.Flags().Set("config", configName)
		r.Nil(bindRootFlags(v))
		r.Nil(readConfigFile(v, rootCmd))
		cfg, err := getConfigFromViper(rootCmd, v)
		r.Nil(err)
		r.Equal([]string{"127.0.0.1:7777"}, cfg.SuperNodes)
	}
}

func generateFakeFilename(fs afero.Fs) string {
	for i := 0; i < 100; i++ {
		d := fmt.Sprintf("/dftest-%d-%d", time.Now().UnixNano(), rand.Int())
		_, err := fs.Stat(d)
		if os.IsNotExist(err) {
			return d
		}
	}

	panic("failed to generate fake dir")
}

func (ts *rootTestSuite) TestBindRootFlags() {
	r := ts.Require()
	v := viper.New()
	r.Nil(bindRootFlags(v))
	r.Equal(v.GetString("registry"), v.GetString("registry_mirror.remote"))
}

func (ts *rootTestSuite) TestAutomaticEnv() {
	r := ts.Require()
	v := viper.New()
	r.Nil(bindRootFlags(v))

	maxprocsEnvKey := strings.ToUpper(DFDaemonEnvPrefix + "_maxprocs")
	workHomeEnvKey := strings.ToUpper(DFDaemonEnvPrefix + "_workHome")
	registryEnvKey := strings.ToUpper(DFDaemonEnvPrefix + "_registry_mirror.remote")

	os.Setenv(maxprocsEnvKey, "17")
	os.Setenv(workHomeEnvKey, "/dragonfly/home")
	os.Setenv(registryEnvKey, "https://dragonfly.io")

	r.Equal(17, v.GetInt("maxprocs"))
	r.Equal("/dragonfly/home", v.GetString("workHome"))
	r.Equal("https://dragonfly.io", v.GetString("registry_mirror.remote"))

	os.Unsetenv(maxprocsEnvKey)
	os.Unsetenv(workHomeEnvKey)
	os.Unsetenv(registryEnvKey)
}

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

// TestDecodeWithYAML tests if config.URL and config.Regexp are decoded correctly.
func (ts *rootTestSuite) TestDecodeWithYAML() {
	r := ts.Require()
	v := viper.New()
	r.Nil(bindRootFlags(v))
	// Sets dfrepo and dfpath to pass the checks for directories
	v.Set("dfrepo", "/tmp")
	v.Set("dfpath", "/tmp")

	mockURL := "http://xxxx"
	v.Set("registry_mirror.remote", mockURL)

	mockRegx := "test.*"
	v.Set("proxies", []interface{}{
		map[string]string{"regx": mockRegx},
	})

	f, err := ioutil.TempFile("", "")
	r.Nil(err)
	defer os.RemoveAll(f.Name())
	_, err = f.WriteString(testCrt)
	r.Nil(err)
	err = f.Close()
	r.Nil(err)
	v.Set("registry_mirror.certs", []string{f.Name()})

	cfg, err := getConfigFromViper(rootCmd, v)
	r.Nil(err)
	r.NotNil(cfg.RegistryMirror.Remote)
	r.Equal(mockURL, cfg.RegistryMirror.Remote.String())
	r.Len(cfg.Proxies, 1)
	r.Equal(mockRegx, cfg.Proxies[0].Regx.String())
	r.NotNil(cfg.RegistryMirror.Certs)
	r.NotNil(cfg.RegistryMirror.Certs.CertPool)
}

func TestRootCommand(t *testing.T) {
	suite.Run(t, &rootTestSuite{})
}
