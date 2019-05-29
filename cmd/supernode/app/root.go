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

import "C"
import (
	"fmt"
	"os"
	"path"
	"reflect"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/dragonflyoss/Dragonfly/common/dflog"
	errorType "github.com/dragonflyoss/Dragonfly/common/errors"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon"
)

var (
	configFilePath = config.DefaultSupernodeConfigFilePath
	cfg            = config.NewConfig()
	options        = NewOptions()
)

var rootCmd = &cobra.Command{
	Use:          "Dragonfly Supernode",
	Long:         "",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSuperNode()
	},
}

func init() {
	setupFlags(rootCmd, options)
}

// setupFlags setups flags for command line.
func setupFlags(cmd *cobra.Command, opt *Options) {
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.
	// flagSet := cmd.PersistentFlags()

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	flagSet := cmd.Flags()

	flagSet.StringVar(&configFilePath, "config", configFilePath,
		"the path of supernode's configuration file")

	flagSet.IntVar(&opt.ListenPort, "port", opt.ListenPort,
		"ListenPort is the port supernode server listens on")

	flagSet.IntVar(&opt.DownloadPort, "download-port", opt.DownloadPort,
		"DownloadPort is the port for download files from supernode")

	flagSet.StringVar(&opt.HomeDir, "home-dir", opt.HomeDir,
		"HomeDir is working directory of supernode")

	flagSet.StringVar(&opt.DownloadPath, "download-path", opt.DownloadPath,
		"specifies the path where to store downloaded filed from source address")

	flagSet.IntVar(&opt.SystemReservedBandwidth, "system-bandwidth", opt.SystemReservedBandwidth,
		"Network rate reserved for system (unit: MB/s)")

	flagSet.IntVar(&opt.MaxBandwidth, "max-bandwidth", opt.MaxBandwidth,
		"network rate that supernode can use (unit: MB/s)")

	flagSet.IntVar(&opt.SchedulerCorePoolSize, "pool-size", opt.SchedulerCorePoolSize,
		"the core pool size of ScheduledExecutorService")

	flagSet.BoolVar(&opt.EnableProfiler, "profiler", opt.EnableProfiler,
		"Set if supernode HTTP server setup profiler")

	flagSet.BoolVarP(&opt.Debug, "debug", "D", opt.Debug,
		"Switch daemon log level to DEBUG mode")

	flagSet.IntVar(&opt.PeerUpLimit, "up-limit", opt.PeerUpLimit,
		"upload limit for a peer to serve download tasks")

	flagSet.IntVar(&opt.PeerDownLimit, "down-limit", opt.PeerDownLimit,
		"download limit for supernode to serve download tasks")

	flagSet.StringVar(&opt.AdvertiseIP, "advertise-ip", "",
		"the supernode ip that we advertise to other peer in the p2p-network")
}

// runSuperNode prepares configs, setups essential details and runs supernode daemon.
func runSuperNode() error {
	// initialize log.
	if err := initLog(); err != nil {
		return err
	}

	if err := initConfig(); err != nil {
		return err
	}

	// set supernode advertise ip
	if cutil.IsEmptyStr(cfg.AdvertiseIP) {
		if err := setAdvertiseIP(); err != nil {
			return err
		}
	}
	logrus.Infof("success to init local ip of supernode, use ip: %s", cfg.AdvertiseIP)

	// set up the CIDPrefix
	cfg.SetCIDPrefix(cfg.AdvertiseIP)

	logrus.Info("start to run supernode")

	d, err := daemon.New(cfg)
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
}

// initLog initializes log Level and log format of daemon.
func initLog() error {
	logPath := path.Join(options.HomeDir, "logs", "app.log")
	err := dflog.InitLog(options.Debug, logPath, fmt.Sprintf("%d", os.Getpid()))
	if err != nil {
		logrus.Errorf("failed to initialize logs: %v", err)
	}
	return err
}

// initConfig load configuration from config file.
// The properties in config file will be covered by the value that comes from
// command line parameters.
func initConfig() error {
	if err := cfg.Load(configFilePath); err != nil {
		logrus.Errorf("failed to init properties: %v", err)
		if configFilePath != config.DefaultSupernodeConfigFilePath ||
			!os.IsNotExist(err) {
			return err
		}
		cfg.BaseProperties = options.BaseProperties
		return nil
	}

	opt := getPureOptionFromCLI()
	choosePropValue(opt.BaseProperties, cfg.BaseProperties)
	return nil
}

func setAdvertiseIP() error {
	// use the first non-loop address if the AdvertiseIP is empty
	ipList, err := cutil.GetAllIPs()
	if err != nil {
		return errors.Wrapf(errorType.ErrSystemError, "failed to get ip list: %v", err)
	}
	if len(ipList) == 0 {
		logrus.Debugf("get empty system's unicast interface addresses")
		return nil
	}

	cfg.AdvertiseIP = ipList[0]

	return nil
}

func choosePropValue(cliProp, cfgProp *config.BaseProperties) {
	if cliProp == nil || cfgProp == nil {
		return
	}
	var inited = func(v reflect.Value) bool {
		switch v.Kind() {
		case reflect.String:
			return v.String() != ""
		case reflect.Bool:
			return v.Bool()
		case reflect.Int:
			return v.Int() != 0
		}
		return false
	}

	cliV := reflect.ValueOf(cliProp).Elem()
	cfgV := reflect.ValueOf(cfgProp).Elem()

	for i := cliV.NumField() - 1; i >= 0; i-- {
		v := cliV.Field(i)
		if inited(v) {
			cfgV.Field(i).Set(v)
		}
	}
}

func getPureOptionFromCLI() *Options {
	cmd := &cobra.Command{}
	opt := &Options{&config.BaseProperties{}}
	setupFlags(cmd, opt)
	cmd.ParseFlags(os.Args)
	return opt
}

// Execute will process supernode.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}
