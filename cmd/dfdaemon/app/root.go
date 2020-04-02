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
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfdaemon"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/constant"
	"github.com/dragonflyoss/Dragonfly/pkg/cmd"
	dferr "github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/rate"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const (
	// DFDaemonEnvPrefix is the default environment prefix for Viper.
	// Both BindEnv and AutomaticEnv will use this prefix.
	DFDaemonEnvPrefix = "dfdaemon"
)

var rootCmd = &cobra.Command{
	Use:               "dfdaemon",
	Short:             "The dfdaemon is a proxy that intercepts image download requests.",
	Long:              "The dfdaemon is a proxy between container engine and registry used for pulling images.",
	DisableAutoGenTag: true, // disable displaying auto generation tag in cli docs
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := readConfigFile(viper.GetViper(), cmd); err != nil {
			return errors.Wrap(err, "read config file")
		}

		cfg, err := getConfigFromViper(cmd, viper.GetViper())
		if err != nil {
			return errors.Wrap(err, "get config from viper")
		}

		if err := initDfdaemon(cfg); err != nil {
			return errors.Wrap(err, "init dfdaemon")
		}

		cfgJSON, _ := json.Marshal(cfg)
		logrus.Infof("using config: %s", cfgJSON)

		s, err := dfdaemon.NewFromConfig(*cfg)
		if err != nil {
			return errors.Wrap(err, "create dfdaemon from config")
		}
		// if stream mode, launch peer server in dfdaemon progress
		if cfg.StreamMode {
			go dfdaemon.LaunchPeerServer(*cfg)
		}
		return s.Start()
	},
}

func init() {
	executable, err := exec.LookPath(os.Args[0])
	exitOnError(err, "exec.LookPath")
	self, err := filepath.Abs(executable)
	exitOnError(err, "get absolute exec path")
	defaultDfgetPath := filepath.Join(filepath.Dir(self), "dfget")

	rf := rootCmd.Flags()

	rf.String("config", constant.DefaultConfigPath, "the path of dfdaemon's configuration file")

	rf.Bool("verbose", false, "verbose")
	rf.Int("maxprocs", 4, "the maximum number of CPUs that the dfdaemon can use")

	// http server config
	rf.String("hostIp", "127.0.0.1", "dfdaemon host ip, default: 127.0.0.1")
	rf.Uint("port", 65001, "dfdaemon will listen the port")
	rf.Uint("peerPort", 0, "peerserver will listen the port")
	rf.Bool("streamMode", false, "dfdaemon will run in stream mode")
	rf.String("certpem", "", "cert.pem file path")
	rf.String("keypem", "", "key.pem file path")

	rf.String("registry", "https://index.docker.io", "registry mirror url, which will override the registry mirror settings in the config file if presented")

	// dfget download config
	rf.String("localrepo", "", "temp output dir of dfdaemon")
	rf.String("workHome", filepath.Join(os.Getenv("HOME"), ".small-dragonfly"), "the work home directory of dfdaemon.")
	rf.String("dfpath", defaultDfgetPath, "dfget path")
	rf.Var(netutils.NetLimit(), "ratelimit", "net speed limit")
	rf.StringSlice("node", nil, "specify the addresses(host:port) of supernodes that will be passed to dfget.")

	exitOnError(bindRootFlags(viper.GetViper()), "bind root command flags")

	// add sub commands
	rootCmd.AddCommand(NewGenCACommand())
	rootCmd.AddCommand(cmd.NewGenDocCommand("dfdaemon"))
	rootCmd.AddCommand(cmd.NewVersionCommand("dfdaemon"))
	rootCmd.AddCommand(cmd.NewConfigCommand("dfdaemon", getDefaultConfig))
}

// bindRootFlags binds flags on rootCmd to the given viper instance.
func bindRootFlags(v *viper.Viper) error {
	if err := v.BindPFlags(rootCmd.Flags()); err != nil {
		return err
	}
	if err := v.BindPFlag("registry_mirror.remote", rootCmd.Flag("registry")); err != nil {
		return err
	}
	v.SetEnvPrefix(DFDaemonEnvPrefix)
	v.AutomaticEnv()

	return nil
}

// readConfigFile reads config file into the given viper instance. If we're
// reading the default configuration file and the file does not exist, nil will
// be returned.
func readConfigFile(v *viper.Viper, cmd *cobra.Command) error {
	v.SetConfigFile(v.GetString("config"))
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		// when the default config file is not found, ignore the error
		if os.IsNotExist(err) && !cmd.Flag("config").Changed {
			return nil
		}
		return err
	}

	return nil
}

func exitOnError(err error, msg string) {
	if err != nil {
		logrus.Fatalf("%s: %v", msg, err)
	}
}

// Execute runs dfdaemon.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("dfdaemon failed: %v", err)
		if e, ok := errors.Cause(err).(*dferr.DfError); ok {
			os.Exit(e.Code)
		} else {
			os.Exit(1)
		}
	}
}

// getDefaultConfig returns the default configuration of dfdaemon
func getDefaultConfig() (interface{}, error) {
	return getConfigFromViper(rootCmd, viper.GetViper())
}

// getConfigFromViper returns dfdaemon config from the given viper instance
func getConfigFromViper(cmd *cobra.Command, v *viper.Viper) (*config.Properties, error) {
	// override supernodes in config file if --node is specified in cli.
	// use default value if no supernodes is configured in config file
	if cmd.Flags().Lookup("node").Changed ||
		len(v.GetStringSlice("supernodes")) == 0 {
		v.Set("supernodes", v.GetStringSlice("node"))
	}

	var cfg config.Properties
	if err := v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
		dc.DecodeHook = decodeWithYAML(
			reflect.TypeOf(config.Regexp{}),
			reflect.TypeOf(config.URL{}),
			reflect.TypeOf(config.CertPool{}),
			reflect.TypeOf(time.Second),
			reflect.TypeOf(rate.B),
		)
	}); err != nil {
		return nil, errors.Wrap(err, "unmarshal yaml")
	}

	// use `{WorkHome}/dfdaemon/data/` as repo dir if localrepo is not configured.
	if cfg.DFRepo == "" {
		cfg.DFRepo = filepath.Join(cfg.WorkHome, "dfdaemon/data/")
	}

	return &cfg, cfg.Validate()
}

// decodeWithYAML returns a mapstructure.DecodeHookFunc to decode the given
// types by unmarshalling from yaml text.
func decodeWithYAML(types ...reflect.Type) mapstructure.DecodeHookFunc {
	return func(f, t reflect.Type, data interface{}) (interface{}, error) {
		for _, typ := range types {
			if t == typ {
				b, _ := yaml.Marshal(data)
				v := reflect.New(t)
				return v.Interface(), yaml.Unmarshal(b, v.Interface())
			}
		}
		return data, nil
	}
}
