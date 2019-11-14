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
	"math/rand"
	"os"
	"path/filepath"
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

func (ts *rootTestSuite) TestPortFlag() {
	r := ts.Require()
	fs := afero.NewMemMapFs()

	configName := "supernode.yml"
	file, err := fs.Create(configName)
	r.Nil(err)
	file.WriteString("base:\n    listenPort: 8888")
	file.Close()

	// flag not set, should use config file
	{
		v := viper.New()
		v.SetFs(fs)
		rootCmd.Flags().Set("config", configName)
		r.Nil(bindRootFlags(v))
		r.Nil(readConfigFile(v, rootCmd))
		cfg, err := getConfigFromViper(v)
		r.Nil(err)
		r.Equal(8888, cfg.ListenPort)
	}

	// flag not set, config file doesn't exist, should use default
	{
		v := viper.New()
		v.SetFs(fs)
		rootCmd.Flags().Set("config", "xxx")
		r.Nil(bindRootFlags(v))
		r.NotNil(readConfigFile(v, rootCmd))
		cfg, err := getConfigFromViper(v)
		r.Nil(err)
		r.Equal(8002, cfg.ListenPort)
	}

	// when --port flag is set, should always use the flag
	rootCmd.Flags().Set("port", "9999")

	{
		v := viper.New()
		v.SetFs(fs)
		rootCmd.Flags().Set("config", "xxx")
		r.Nil(bindRootFlags(v))
		r.NotNil(readConfigFile(v, rootCmd))
		cfg, err := getConfigFromViper(v)
		r.Nil(err)
		r.Equal(9999, cfg.ListenPort)
	}

	{
		v := viper.New()
		v.SetFs(fs)
		rootCmd.Flags().Set("config", configName)
		r.Nil(bindRootFlags(v))
		r.Nil(readConfigFile(v, rootCmd))
		cfg, err := getConfigFromViper(v)
		r.Nil(err)
		r.Equal(9999, cfg.ListenPort)
	}
}

func (ts *rootTestSuite) TestHomeDirFlag() {
	r := ts.Require()

	defaultHomeDir := filepath.Join(string(filepath.Separator), "home", "admin", "supernode")
	defaultDownloadPath := filepath.Join(defaultHomeDir, "repo", "download")

	// flag not set, should use default value
	{
		v := viper.New()
		r.Nil(bindRootFlags(v))
		r.NotNil(readConfigFile(v, rootCmd))
		cfg, err := getConfigFromViper(v)
		r.Nil(err)
		r.Equal(defaultHomeDir, cfg.HomeDir)
		r.Equal(defaultDownloadPath, cfg.DownloadPath)
	}
	// when --home-dir flag is set, the DownloadPath should follow the update in default
	{
		v := viper.New()
		rootCmd.Flags().Set("home-dir", "/test-home")
		r.Nil(bindRootFlags(v))
		r.NotNil(readConfigFile(v, rootCmd))
		cfg, err := getConfigFromViper(v)
		r.Nil(err)
		r.Equal("/test-home", cfg.HomeDir)
		r.Equal("/test-home/repo/download", cfg.DownloadPath)
	}
}

func (ts *rootTestSuite) TestAutomaticEnv() {
	r := ts.Require()
	v := viper.New()
	r.Nil(bindRootFlags(v))

	listenPortEnvKey := strings.ToUpper(SupernodeEnvPrefix + "_base.listenPort")
	homeDirEnvKey := strings.ToUpper(SupernodeEnvPrefix + "_base.homeDir")
	debugEnvKey := strings.ToUpper(SupernodeEnvPrefix + "_base.debug")
	failAccessIntervalEnvKey := strings.ToUpper(SupernodeEnvPrefix + "_base.failAccessInterval")

	os.Setenv(listenPortEnvKey, "2019")
	os.Setenv(homeDirEnvKey, "/dragonfly/home")
	os.Setenv(debugEnvKey, "true")
	os.Setenv(failAccessIntervalEnvKey, "10m30s")
	expectFailAccessInterval, _ := time.ParseDuration("10m30s")

	r.Equal(2019, v.GetInt("base.listenPort"))
	r.Equal("/dragonfly/home", v.GetString("base.homeDir"))
	r.Equal(true, v.GetBool("base.debug"))
	r.Equal(expectFailAccessInterval, v.GetDuration("base.failAccessInterval"))

	os.Unsetenv(listenPortEnvKey)
	os.Unsetenv(homeDirEnvKey)
	os.Unsetenv(debugEnvKey)
	os.Unsetenv(failAccessIntervalEnvKey)
}

func generateFakeFilename(fs afero.Fs) string {
	for i := 0; i < 100; i++ {
		d := fmt.Sprintf("/supernode-test-%d-%d", time.Now().UnixNano(), rand.Int())
		_, err := fs.Stat(d)
		if os.IsNotExist(err) {
			return d
		}
	}

	panic("failed to generate fake dir")
}

func TestRootCommand(t *testing.T) {
	suite.Run(t, &rootTestSuite{})
}
