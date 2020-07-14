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
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/cmd"
	"github.com/dragonflyoss/Dragonfly/pkg/dflog"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/rate"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const (
	// SupernodeEnvPrefix is the default environment prefix for Viper.
	// Both BindEnv and AutomaticEnv will use this prefix.
	SupernodeEnvPrefix = "supernode"
)

var (
	supernodeViper = viper.GetViper()
)

// supernodeDescription is used to describe supernode command in details.
var supernodeDescription = `SuperNode is a long-running process with two primary responsibilities:
It's the tracker and scheduler in the P2P network that choose appropriate downloading net-path for each peer.
It's also a CDN server that caches downloaded data from source to avoid downloading the same files from source repeatedly.`

var rootCmd = &cobra.Command{
	Use:               "supernode",
	Short:             "the central control server of Dragonfly used for scheduling and cdn cache",
	Long:              supernodeDescription,
	Args:              cobra.NoArgs,
	DisableAutoGenTag: true, // disable displaying auto generation tag in cli docs
	SilenceUsage:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// load config file.
		if err := readConfigFile(supernodeViper, cmd); err != nil {
			return errors.Wrap(err, "read config file")
		}

		// get config from viper.
		cfg, err := getConfigFromViper(supernodeViper)
		if err != nil {
			return errors.Wrap(err, "get config from viper")
		}

		// create home dir
		if err := fileutils.CreateDirectory(supernodeViper.GetString("base.homeDir")); err != nil {
			return fmt.Errorf("failed to create home dir %s: %v", supernodeViper.GetString("base.homeDir"), err)
		}

		// initialize supernode logger.
		if err := initLog(logrus.StandardLogger(), "app.log", cfg.LogConfig); err != nil {
			return err
		}

		// initialize dfget logger.
		dfgetLogger := logrus.New()
		if err := initLog(dfgetLogger, "dfget.log", cfg.LogConfig); err != nil {
			return err
		}

		// set supernode advertise ip
		if stringutils.IsEmptyStr(cfg.AdvertiseIP) {
			if err := setAdvertiseIP(cfg); err != nil {
				return err
			}
		}
		logrus.Infof("success to init local ip of supernode, use ip: %s", cfg.AdvertiseIP)

		// set up the CIDPrefix
		cfg.SetCIDPrefix(cfg.AdvertiseIP)

		logrus.Debugf("get supernode config: %+v", cfg)
		logrus.Info("start to run supernode")

		d, err := daemon.New(cfg, dfgetLogger)
		if err != nil {
			logrus.Errorf("failed to initialize daemon in supernode: %v", err)
			return err
		}

		// register supernode
		if err := d.RegisterSuperNode(); err != nil {
			logrus.Errorf("failed to register super node: %v", err)
			return err
		}

		return d.Run()
	},
}

func init() {
	setupFlags(rootCmd)

	// add sub commands
	rootCmd.AddCommand(cmd.NewGenDocCommand("supernode"))
	rootCmd.AddCommand(cmd.NewVersionCommand("supernode"))
	rootCmd.AddCommand(cmd.NewConfigCommand("supernode", getDefaultConfig))
}

// setupFlags setups flags for command line.
func setupFlags(cmd *cobra.Command) {
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.
	// flagSet := cmd.PersistentFlags()

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	flagSet := cmd.Flags()

	defaultBaseProperties := config.NewBaseProperties()

	flagSet.String("config", config.DefaultSupernodeConfigFilePath,
		"the path of supernode's configuration file")

	flagSet.String("cdn-pattern", config.CDNPatternLocal,
		"cdn pattern, must be in [\"local\", \"source\"]. Default: local")

	flagSet.Int("port", defaultBaseProperties.ListenPort,
		"listenPort is the port that supernode server listens on")

	flagSet.Int("download-port", defaultBaseProperties.DownloadPort,
		"downloadPort is the port for download files from supernode")

	flagSet.String("home-dir", defaultBaseProperties.HomeDir,
		"homeDir is the working directory of supernode")

	flagSet.Var(&defaultBaseProperties.SystemReservedBandwidth, "system-bandwidth",
		"network rate reserved for system")

	flagSet.Var(&defaultBaseProperties.MaxBandwidth, "max-bandwidth",
		"network rate that supernode can use")

	flagSet.Int("pool-size", defaultBaseProperties.SchedulerCorePoolSize,
		"pool size is the core pool size of ScheduledExecutorService")

	flagSet.Bool("profiler", defaultBaseProperties.EnableProfiler,
		"profiler sets whether supernode HTTP server setups profiler")

	flagSet.BoolP("debug", "D", defaultBaseProperties.Debug,
		"switch daemon log level to DEBUG mode")

	flagSet.Int("up-limit", defaultBaseProperties.PeerUpLimit,
		"upload limit for a peer to serve download tasks")

	flagSet.Int("down-limit", defaultBaseProperties.PeerDownLimit,
		"download limit for supernode to serve download tasks")

	flagSet.String("advertise-ip", "",
		"the supernode ip is the ip we advertise to other peers in the p2p-network")

	flagSet.Duration("fail-access-interval", defaultBaseProperties.FailAccessInterval,
		"fail access interval is the interval time after failed to access the URL")

	flagSet.Duration("gc-initial-delay", defaultBaseProperties.GCInitialDelay,
		"gc initial delay is the delay time from the start to the first GC execution")

	flagSet.Duration("gc-meta-interval", defaultBaseProperties.GCMetaInterval,
		"gc meta interval is the interval time to execute the GC meta")

	flagSet.Duration("task-expire-time", defaultBaseProperties.TaskExpireTime,
		"task expire time is the time that a task is treated expired if the task is not accessed within the time")

	flagSet.Duration("peer-gc-delay", defaultBaseProperties.PeerGCDelay,
		"peer gc delay is the delay time to execute the GC after the peer has reported the offline")

	exitOnError(bindRootFlags(supernodeViper), "bind root command flags")
}

// bindRootFlags binds flags on rootCmd to the given viper instance.
func bindRootFlags(v *viper.Viper) error {
	flags := []struct {
		key  string
		flag string
	}{
		{
			key:  "config",
			flag: "config",
		},
		{
			key:  "base.CDNPattern",
			flag: "cdn-pattern",
		},
		{
			key:  "base.listenPort",
			flag: "port",
		},
		{
			key:  "base.downloadPort",
			flag: "download-port",
		},
		{
			key:  "base.homeDir",
			flag: "home-dir",
		},
		{
			key:  "base.systemReservedBandwidth",
			flag: "system-bandwidth",
		},
		{
			key:  "base.maxBandwidth",
			flag: "max-bandwidth",
		},
		{
			key:  "base.schedulerCorePoolSize",
			flag: "pool-size",
		},
		{
			key:  "base.enableProfiler",
			flag: "profiler",
		},
		{
			key:  "base.debug",
			flag: "debug",
		},
		{
			key:  "base.peerUpLimit",
			flag: "up-limit",
		},
		{
			key:  "base.peerDownLimit",
			flag: "down-limit",
		},
		{
			key:  "base.advertiseIP",
			flag: "advertise-ip",
		},
		{
			key:  "base.failAccessInterval",
			flag: "fail-access-interval",
		},
		{
			key:  "base.gcInitialDelay",
			flag: "gc-initial-delay",
		},
		{
			key:  "base.gcMetaInterval",
			flag: "gc-meta-interval",
		},
		{
			key:  "base.taskExpireTime",
			flag: "task-expire-time",
		},
		{
			key:  "base.peerGCDelay",
			flag: "peer-gc-delay",
		},
	}

	for _, f := range flags {
		if err := v.BindPFlag(f.key, rootCmd.Flag(f.flag)); err != nil {
			return err
		}
	}

	v.SetEnvPrefix(SupernodeEnvPrefix)
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

// getDefaultConfig returns the default configuration of supernode
func getDefaultConfig() (interface{}, error) {
	return getConfigFromViper(viper.GetViper())
}

// getConfigFromViper returns supernode config from the given viper instance
func getConfigFromViper(v *viper.Viper) (*config.Config, error) {
	cfg := config.NewConfig()

	if err := v.Unmarshal(cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
		dc.DecodeHook = decodeWithYAML(
			reflect.TypeOf(time.Second),
			reflect.TypeOf(rate.B),
			reflect.TypeOf(fileutils.B),
		)
	}); err != nil {
		return nil, errors.Wrap(err, "unmarshal yaml")
	}

	// set dynamic configuration
	cfg.DownloadPath = filepath.Join(cfg.HomeDir, "repo", "download")

	return cfg, nil
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

// initLog initializes log Level and log format.
func initLog(logger *logrus.Logger, logPath string, logConfig dflog.LogConfig) error {
	logFilePath := filepath.Join(supernodeViper.GetString("base.homeDir"), "logs", logPath)

	opts := []dflog.Option{
		dflog.WithLogFile(logFilePath, logConfig.MaxSize, logConfig.MaxBackups),
		dflog.WithSign(fmt.Sprintf("%d", os.Getpid())),
		dflog.WithDebug(supernodeViper.GetBool("base.debug")),
	}

	logrus.Debugf("use log file %s", logFilePath)
	if err := dflog.Init(logger, opts...); err != nil {
		return errors.Wrap(err, "init log")
	}

	return nil
}

func setAdvertiseIP(cfg *config.Config) error {
	// use the first non-loop address if the AdvertiseIP is empty
	ipList, err := netutils.GetAllIPs()
	if err != nil {
		return errors.Wrapf(errortypes.ErrSystemError, "failed to get ip list: %v", err)
	}
	if len(ipList) == 0 {
		logrus.Errorf("get empty system's unicast interface addresses")
		return errors.Wrapf(errortypes.ErrSystemError, "Unable to autodetect advertiser ip, please set it via --advertise-ip")
	}

	cfg.AdvertiseIP = ipList[0]

	return nil
}

// Execute will process supernode.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func exitOnError(err error, msg string) {
	if err != nil {
		logrus.Fatalf("%s: %v", msg, err)
	}
}
