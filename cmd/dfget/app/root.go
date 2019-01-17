package app

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core"
	"github.com/dragonflyoss/Dragonfly/dfget/errors"
	"github.com/dragonflyoss/Dragonfly/dfget/util"

	errHandler "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	localLimit string
	totalLimit string
	filter     string
)

var cfg = config.NewConfig()

var rootCmd = &cobra.Command{
	Use:               "dfget",
	Short:             "The dfget is the client of Dragonfly.",
	Long:              "The dfget is the client of Dragonfly, a non-interactive P2P downloader.",
	DisableAutoGenTag: true, // disable displaying auto generation tag in cli docs
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDfget()
	},
}

func init() {
	initFlags()
}

// runDfget do some init operations and start to download.
func runDfget() error {
	// initialize logger
	if err := initClientLog(); err != nil {
		util.Printer.Println(fmt.Sprintf("init log error: %v", err))
		return err
	}

	// get config from property files
	initProperties()

	if err := transParams(); err != nil {
		util.Printer.Println(err.Error())
		return err
	}

	checkParameters()
	cfg.ClientLogger.Infof("get cmd params:%q", os.Args)

	if err := config.AssertConfig(cfg); err != nil {
		util.Printer.Println(fmt.Sprintf("assert context error: %v", err))
		return err
	}
	cfg.ClientLogger.Infof("get init config:%v", cfg)

	// enter the core process
	err := core.Start(cfg)
	util.Printer.Println(resultMsg(cfg, time.Now(), err))
	if err != nil {
		os.Exit(err.Code)
	}
	return nil
}

func checkParameters() {
	if len(os.Args) < 2 {
		fmt.Println("Please use the command 'help' to show the help information.")
		os.Exit(0)
	}
}

// load config from property files.
func initProperties() {
	properties := config.NewProperties()
	for _, v := range cfg.ConfigFiles {
		if err := properties.Load(v); err == nil {
			cfg.ClientLogger.Debugf("initProperties[%s] success: %v", v, properties)
			break
		} else {
			cfg.ClientLogger.Debugf("initProperties[%s] fail: %v", v, err)
		}
	}

	if cfg.Node == nil {
		cfg.Node = properties.Nodes
	}

	if cfg.LocalLimit == 0 {
		cfg.LocalLimit = properties.LocalLimit
	}

	if cfg.TotalLimit == 0 {
		cfg.TotalLimit = properties.TotalLimit
	}

	if cfg.ClientQueueSize == 0 {
		cfg.ClientQueueSize = properties.ClientQueueSize
	}
}

// transParams trans the user-friendly parameter formats
// to the format corresponding to the `Config` struct.
func transParams() error {
	cfg.Filter = transFilter(filter)

	var err error
	if cfg.LocalLimit, err = transLimit(localLimit); err != nil {
		return errHandler.Wrapf(errors.ErrConvertFailed, "locallimit: %v", err)
	}

	if cfg.TotalLimit, err = transLimit(totalLimit); err != nil {
		return errHandler.Wrapf(errors.ErrConvertFailed, "totallimit: %v", err)
	}

	return nil
}

// initClientLog init client logger.
func initClientLog() error {
	var (
		logPath  = path.Join(cfg.WorkHome, "logs")
		logLevel = "info"
	)
	if cfg.Verbose {
		logLevel = "debug"
	}

	var err error
	if cfg.ClientLogger, err = util.CreateLogger(logPath, "dfclient.log", logLevel, cfg.Sign); err != nil {
		return err
	}

	if cfg.Console {
		util.AddConsoleLog(cfg.ClientLogger)
	}
	return nil
}

func initFlags() {
	persistFlagSet := rootCmd.PersistentFlags()

	// pass to server
	persistFlagSet.DurationVar(&cfg.RV.DataExpireTime, "expiretime", config.DataExpireTime,
		"server will delete cached files if these files doesn't be modification within this duration")
	persistFlagSet.DurationVar(&cfg.RV.ServerAliveTime, "alivetime", config.ServerAliveTime,
		"server will stop if there is no uploading task within this duration")

	flagSet := rootCmd.Flags()

	// url & output
	flagSet.StringVarP(&cfg.URL, "url", "u", "",
		"will download a file from this url")
	flagSet.StringVarP(&cfg.Output, "output", "o", "",
		"output path that not only contains the dir part but also name part")

	// localLimit & totalLimit & timeout
	flagSet.StringVarP(&localLimit, "locallimit", "s", "",
		"rate limit about a single download task, its format is 20M/m/K/k")
	flagSet.StringVar(&totalLimit, "totallimit", "",
		"rate limit about the whole host, its format is 20M/m/K/k")
	flagSet.IntVarP(&cfg.Timeout, "timeout", "e", 0,
		"download timeout(second)")

	// md5 & identifier
	flagSet.StringVarP(&cfg.Md5, "md5", "m", "",
		"expected file md5")
	flagSet.StringVarP(&cfg.Identifier, "identifier", "i", "",
		"identify download task, it is available merely when md5 param not exist")

	flagSet.StringVar(&cfg.CallSystem, "callsystem", "",
		"system name that executes dfget")

	flagSet.StringVarP(&cfg.Pattern, "pattern", "p", "p2p",
		"download pattern, must be 'p2p' or 'cdn' or 'source'"+
			"\ncdn/source pattern not support 'totallimit' flag")

	flagSet.StringVarP(&filter, "filter", "f", "",
		"filter some query params of url, use char '&' to separate different params"+
			"\neg: -f 'key&sign' will filter 'key' and 'sign' query param"+
			"\nin this way, different urls correspond one same download task that can use p2p mode")

	flagSet.StringSliceVar(&cfg.Header, "header", nil,
		"http header, eg: --header='Accept: *' --header='Host: abc'")

	flagSet.StringSliceVarP(&cfg.Node, "node", "n", nil,
		"specify supnernodes")

	flagSet.BoolVar(&cfg.Notbs, "notbs", false,
		"not back source when p2p fail")
	flagSet.BoolVar(&cfg.DFDaemon, "dfdaemon", false,
		"caller is from dfdaemon")
	// others
	flagSet.BoolVarP(&cfg.ShowBar, "showbar", "b", false,
		"show progress bar, it's conflict with '--console'")
	flagSet.BoolVar(&cfg.Console, "console", false,
		"show log on console, it's conflict with '--showbar'")
	flagSet.BoolVar(&cfg.Verbose, "verbose", false,
		"be verbose")

	flagSet.MarkDeprecated("exceed", "please use '--timeout' or '-e' instead")
}

// Helper functions.
func transLimit(limit string) (int, error) {
	if util.IsEmptyStr(limit) {
		return 0, nil
	}
	l := len(limit)
	i, err := strconv.Atoi(limit[:l-1])

	if err != nil {
		return 0, err
	}

	unit := limit[l-1]
	if unit == 'k' || unit == 'K' {
		return i * 1024, nil
	}
	if unit == 'm' || unit == 'M' {
		return i * 1024 * 1024, nil
	}
	return 0, fmt.Errorf("invalid unit '%c' of '%s', 'KkMm' are supported",
		unit, limit)
}

func transFilter(filter string) []string {
	if util.IsEmptyStr(filter) {
		return nil
	}
	return strings.Split(filter, "&")
}

func resultMsg(cfg *config.Config, end time.Time, e *errors.DFGetError) string {
	if e != nil {
		return fmt.Sprintf("download FAIL(%d) cost:%.3fs length:%d reason:%d error:%v",
			e.Code, end.Sub(cfg.StartTime).Seconds(), cfg.RV.FileLength,
			cfg.BackSourceReason, e)
	}
	return fmt.Sprintf("download SUCCESS(0) cost:%.3fs length:%d reason:%d",
		end.Sub(cfg.StartTime).Seconds(), cfg.RV.FileLength, cfg.BackSourceReason)
}

// Execute will process dfget.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}
