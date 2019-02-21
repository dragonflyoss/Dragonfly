package main

import (
	"os"
	"time"

	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cfg = &config.Config{}

func main() {
	var cmdServe = &cobra.Command{
		Use:          "Dragonfly Supernode",
		Long:         "",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSuperNode(cmd)
		},
	}

	setupFlags(cmdServe)
	if err := cmdServe.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

// setupFlags setups flags for command line.
func setupFlags(cmd *cobra.Command) {
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.
	// flagSet := cmd.PersistentFlags()

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	flagSet := cmd.Flags()
	flagSet.IntVar(&cfg.ListenPort, "port", 65001, "ListenPort is the port supernode server listens on")
	flagSet.StringVar(&cfg.HomeDir, "home-dir", "", "HomeDir is working directory of supernode")
	flagSet.StringVar(&cfg.DownloadPath, "download-path", "downloads", "specifies the path where to store downloaded filed from source address")
	flagSet.IntVar(&cfg.SystemReservedBandwidth, "system-bandwidth", 20, "Network rate reserved for system (unit: MB/s)")
	flagSet.IntVar(&cfg.MaxBandwidth, "max-bandwidth", 20, "network rate that supernode can use (unit: MB/s)")
	flagSet.IntVar(&cfg.SchedulerCorePoolSize, "pool-size", 10, "the core pool size of ScheduledExecutorService")
	flagSet.BoolVar(&cfg.EnableProfiler, "profiler", false, "Set if supernode HTTP server setup profiler")
	flagSet.BoolVarP(&cfg.Debug, "debug", "D", false, "Switch daemon log level to DEBUG mode")
	flagSet.IntVar(&cfg.PeerUpLimit, "up-limit", 5, "upload limit for a peer to serve download tasks")
	flagSet.IntVar(&cfg.PeerDownLimit, "down-limit", 5, "download limit for supernode to serve download tasks")

}

// runSuperNode prepares configs, setups essential details and runs supernode daemon.
func runSuperNode(cmd *cobra.Command) error {
	// initialize log.
	initLog()

	logrus.Info("start to run supernode")

	d, err := daemon.New(cfg)
	if err != nil {
		logrus.Errorf("failed to initialize daemon in supernode: %v", err)
		return err
	}

	if err := d.Run(); err != nil {
		logrus.Errorf("failed to run daemon: %v", err)
		return err
	}
	return nil
}

// initLog initializes log Level and log format of daemon.
func initLog() {
	if cfg.Debug {
		logrus.Infof("start daemon at debug level")
		logrus.SetLevel(logrus.DebugLevel)
	}

	formatter := &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	}
	logrus.SetFormatter(formatter)
}
