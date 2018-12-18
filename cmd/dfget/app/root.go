package app

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	cfg "github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core"
	errType "github.com/dragonflyoss/Dragonfly/dfget/errors"
	"github.com/dragonflyoss/Dragonfly/dfget/util"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	localLimit string
	totalLimit string
	filter     string
)

var rootCmd = &cobra.Command{
	Use:   "dfget",
	Short: "The dfget is the client of Dragonfly.",
	Long:  "The dfget is the client of Dragonfly, a non-interactive P2P downloader.",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig(args)
		cfg.Ctx.ClientLogger.Infof("cmd params:%q", args)

		// start download file process.
		err := core.Start(cfg.Ctx)
		util.Printer.Println(resultMsg(cfg.Ctx, time.Now(), err))
		if err != nil {
			os.Exit(err.Code)
		}
		os.Exit(0)
	},
}

func init() {
	initFlags()
}

func checkParameters() {
	if len(os.Args) < 2 {
		fmt.Println("Please use the command 'help' to show the help information.")
		os.Exit(0)
	}
}

func initConfig(args []string) {
	// Step1: transProperties.
	err := transProperties()
	if err != nil {
		util.Printer.Println(fmt.Sprintf("trans properties error: %v", err))
		os.Exit(cfg.CodePrepareError)
	}

	// Step2: check parameters are vaild.
	checkParameters()

	// Step3: init the logger client/server.
	if err := initLog(); err != nil {
		util.Printer.Println(fmt.Sprintf("init log error: %v", err))
		os.Exit(cfg.CodeInitLogError)
	}

	// Step4: initProperties form config files.
	initProperties()

	// Step5: assert Context.
	if err := cfg.AssertContext(cfg.Ctx); err != nil {
		util.Printer.Println(fmt.Sprintf("assert context error: %v", err))
		os.Exit(cfg.CodeAssertContextError)
	}

	cfg.Ctx.ClientLogger.Infof("context:%s", cfg.Ctx)
}

func transProperties() error {
	locallyLimit, err := transLimit(localLimit)
	if err != nil {
		return errors.Wrapf(errType.ErrConvertFailed, "locallimit: %v", err)
	}
	cfg.Ctx.LocalLimit = locallyLimit

	totallyLimit, err := transLimit(totalLimit)
	if err != nil {
		return errors.Wrapf(errType.ErrConvertFailed, "totallimit: %v", err)
	}
	cfg.Ctx.TotalLimit = totallyLimit

	cfg.Ctx.Filter = transFilter(filter)
	return nil
}

// get configurations from config files.
func initProperties() {
	for _, v := range cfg.Ctx.ConfigFiles {
		if err := cfg.Props.Load(v); err == nil {
			cfg.Ctx.ClientLogger.Debugf("initProperties[%s] success: %v", v, cfg.Props)
			break
		} else {
			cfg.Ctx.ClientLogger.Debugf("initProperties[%s] fail: %v", v, err)
		}
	}

	if cfg.Ctx.Node == nil {
		cfg.Ctx.Node = cfg.Props.Nodes
	}

	if cfg.Ctx.LocalLimit == 0 {
		cfg.Ctx.LocalLimit = cfg.Props.LocalLimit
	}

	if cfg.Ctx.TotalLimit == 0 {
		cfg.Ctx.TotalLimit = cfg.Props.TotalLimit
	}

	if cfg.Ctx.ClientQueueSize == 0 {
		cfg.Ctx.ClientQueueSize = cfg.Props.ClientQueueSize
	}
}

func initLog() error {
	var (
		logPath  = path.Join(cfg.Ctx.WorkHome, "logs")
		logLevel = "info"
	)
	if cfg.Ctx.Verbose {
		logLevel = "debug"
	}

	// init client logger.
	clientLogger, err := util.CreateLogger(logPath, "dfclient.log", logLevel, cfg.Ctx.Sign)
	if err != nil {
		return err
	}
	cfg.Ctx.ClientLogger = clientLogger

	// show log on console.
	if cfg.Ctx.Console {
		util.AddConsoleLog(cfg.Ctx.ClientLogger)
	}

	// if the pattern equals p2p, then init the p2p server logger.
	if cfg.Ctx.Pattern == "p2p" {
		serverLogger, err := util.CreateLogger(logPath, "dfserver.log", logLevel, cfg.Ctx.Sign)
		if err != nil {
			return err
		}
		cfg.Ctx.ServerLogger = serverLogger
	}
	return nil
}

func initFlags() {
	// url & output
	rootCmd.PersistentFlags().StringVarP(&cfg.Ctx.URL, "url", "u", "",
		"will download a file from this url")
	rootCmd.PersistentFlags().StringVarP(&cfg.Ctx.Output, "output", "o", "",
		"output path that not only contains the dir part but also name part")

	// localLimit & totalLimit & timeout
	rootCmd.PersistentFlags().StringVarP(&localLimit, "locallimit", "s", "",
		"rate limit about a single download task, its format is 20M/m/K/k")
	rootCmd.PersistentFlags().StringVarP(&totalLimit, "totallimit", "", "",
		"rate limit about the whole host, its format is 20M/m/K/k")
	rootCmd.PersistentFlags().IntVarP(&cfg.Ctx.Timeout, "timeout", "e", 0,
		"download timeout(second)")

	// md5 & identifier
	rootCmd.PersistentFlags().StringVarP(&cfg.Ctx.Md5, "md5", "m", "",
		"expected file md5")
	rootCmd.PersistentFlags().StringVarP(&cfg.Ctx.Identifier, "identifier", "i", "",
		"identify download task, it is available merely when md5 param not exist")

	rootCmd.PersistentFlags().StringVar(&cfg.Ctx.CallSystem, "callsystem", "",
		"system name that executes dfget")

	rootCmd.PersistentFlags().StringVarP(&cfg.Ctx.Pattern, "pattern", "p", "p2p",
		"download pattern, must be 'p2p' or 'cdn' or 'source'"+
			"\ncdn/source pattern not support 'totallimit' flag")

	rootCmd.PersistentFlags().StringVarP(&filter, "filter", "f", "",
		"filter some query params of url, use char '&' to separate different params"+
			"\neg: -f 'key&sign' will filter 'key' and 'sign' query param"+
			"\nin this way, different urls correspond one same download task that can use p2p mode")

	rootCmd.PersistentFlags().StringSliceVar(&cfg.Ctx.Header, "header", nil,
		"http header, eg: --header='Accept: *' --header='Host: abc'")

	rootCmd.PersistentFlags().StringSliceVarP(&cfg.Ctx.Node, "node", "n", nil,
		"specify supnernodes")

	rootCmd.PersistentFlags().BoolVar(&cfg.Ctx.Notbs, "notbs", false,
		"not back source when p2p fail")
	rootCmd.PersistentFlags().BoolVar(&cfg.Ctx.DFDaemon, "dfdaemon", false,
		"caller is from dfdaemon")

	// others
	rootCmd.PersistentFlags().BoolVarP(&cfg.Ctx.ShowBar, "showbar", "b", false,
		"show progress bar, it's conflict with '--console'")
	rootCmd.PersistentFlags().BoolVar(&cfg.Ctx.Console, "console", false,
		"show log on console, it's conflict with '--showbar'")
	rootCmd.PersistentFlags().BoolVar(&cfg.Ctx.Verbose, "verbose", false,
		"be verbose")

	rootCmd.PersistentFlags().MarkDeprecated("exceed", "please use '--timeout' or '-e' instead")
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

func resultMsg(ctx *cfg.Context, end time.Time, e *errType.DFGetError) string {
	if e != nil {
		return fmt.Sprintf("download FAIL(%d) cost:%.3fs length:%d reason:%d error:%v",
			e.Code, end.Sub(ctx.StartTime).Seconds(), ctx.RV.FileLength,
			ctx.BackSourceReason, e)
	}
	return fmt.Sprintf("download SUCCESS(0) cost:%.3fs length:%d reason:%d",
		end.Sub(ctx.StartTime).Seconds(), ctx.RV.FileLength, ctx.BackSourceReason)
}

// Execute will process dfget.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}
