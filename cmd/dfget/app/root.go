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
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/common/dflog"
	"github.com/dragonflyoss/Dragonfly/common/errors"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core"
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

// dfgetDescription is used to describe dfget command in details.
var dfgetDescription = `dfget is the client of Dragonfly which takes a role of peer in a P2P network.
When user triggers a file downloading task, dfget will download the pieces of
file from other peers. Meanwhile, it will act as an uploader to support other
peers to download pieces from it if it owns them. In addition, dfget has the
abilities to provide more advanced functionality, such as network bandwidth
limit, transmission encryption and so on.`

var rootCmd = &cobra.Command{
	Use:               "dfget",
	Short:             "client of Dragonfly used to download and upload files",
	Long:              dfgetDescription,
	DisableAutoGenTag: true, // disable displaying auto generation tag in cli docs
	Example:           dfgetExample(),
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
	if err := handleNodes(); err != nil {
		util.Printer.Println(err.Error())
		return err
	}

	checkParameters()
	logrus.Infof("get cmd params:%q", os.Args)

	if err := config.AssertConfig(cfg); err != nil {
		util.Printer.Println(fmt.Sprintf("assert context error: %v", err))
		return err
	}
	logrus.Infof("get init config:%v", cfg)

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
			logrus.Debugf("initProperties[%s] success: %v", v, properties)
			break
		} else {
			logrus.Debugf("initProperties[%s] fail: %v", v, err)
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

// initClientLog initializes dfget client's logger.
// There are two kinds of logger dfget client uses: logfile and console.
// logfile is used to stored generated log in local filesystem,
// while console log will output the dfget client's log in console/terminal for
// debugging usage.
func initClientLog() error {
	logFilePath := path.Join(cfg.WorkHome, "logs", "dfclient.log")

	dflog.InitLog(cfg.Verbose, logFilePath, cfg.Sign)

	// once cfg.Console is set, process should also output log to console
	if cfg.Console {
		dflog.InitConsoleLog(cfg.Verbose, cfg.Sign)
	}
	return nil
}

func initFlags() {
	// pass to server
	flagSet := rootCmd.Flags()

	// url & output
	flagSet.StringVarP(&cfg.URL, "url", "u", "", "URL of user requested downloading file(only HTTP/HTTPs supported)")
	flagSet.StringVarP(&cfg.Output, "output", "o", "",
		"Destination path which is used to store the requested downloading file. It must contain detailed directory and specific filename, for example, '/tmp/file.mp4'")

	// localLimit & totalLimit & timeout
	flagSet.StringVarP(&localLimit, "locallimit", "s", "",
		"network bandwidth rate limit for single download task, in format of 20M/m/K/k")
	flagSet.StringVar(&totalLimit, "totallimit", "",
		"network bandwidth rate limit for the whole host, in format of 20M/m/K/k")
	flagSet.IntVarP(&cfg.Timeout, "timeout", "e", 0,
		"Timeout set for file downloading task. If dfget has not finished downloading all pieces of file before --timeout, the dfget will throw an error and exit")

	// md5 & identifier
	flagSet.StringVarP(&cfg.Md5, "md5", "m", "",
		"md5 value input from user for the requested downloading file to enhance security")
	flagSet.StringVarP(&cfg.Identifier, "identifier", "i", "",
		"The usage of identifier is making different downloading tasks generate different downloading task IDs even if they have the same URLs. conflict with --md5.")

	flagSet.StringVar(&cfg.CallSystem, "callsystem", "",
		"The name of dfget caller which is for debugging. Once set, it will be passed to all components around the request to make debugging easy")
	flagSet.StringVarP(&cfg.Pattern, "pattern", "p", "p2p",
		"download pattern, must be p2p/cdn/source, cdn and source do not support flag --totallimit")
	flagSet.StringVarP(&filter, "filter", "f", "",
		"filter some query params of URL, use char '&' to separate different params"+
			"\neg: -f 'key&sign' will filter 'key' and 'sign' query param"+
			"\nin this way, different but actually the same URLs can reuse the same downloading task")
	flagSet.StringSliceVar(&cfg.Header, "header", nil,
		"http header, eg: --header='Accept: *' --header='Host: abc'")
	flagSet.StringSliceVarP(&cfg.Node, "node", "n", nil,
		"specify the addresses(IP:port) of supernodes")
	flagSet.BoolVar(&cfg.Notbs, "notbs", false,
		"disable back source downloading for requested file when p2p fails to download it")
	flagSet.BoolVar(&cfg.DFDaemon, "dfdaemon", false,
		"identify whether the request is from dfdaemon")
	flagSet.IntVar(&cfg.ClientQueueSize, "clientqueue", config.DefaultClientQueueSize,
		"specify the size of client queue which controls the number of pieces that can be processed simultaneously")

	// others
	flagSet.BoolVarP(&cfg.ShowBar, "showbar", "b", false,
		"show progress bar, it is conflict with '--console'")
	flagSet.BoolVar(&cfg.Console, "console", false,
		"show log on console, it's conflict with '--showbar'")
	flagSet.BoolVar(&cfg.Verbose, "verbose", false,
		"be verbose")

	// pass to server
	flagSet.DurationVar(&cfg.RV.DataExpireTime, "expiretime", config.DataExpireTime,
		"caching duration for which cached file keeps no accessed by any process, after this period cache file will be deleted")
	flagSet.DurationVar(&cfg.RV.ServerAliveTime, "alivetime", config.ServerAliveTime,
		"Alive duration for which uploader keeps no accessing by any uploading requests, after this period uploader will automically exit")

	flagSet.MarkDeprecated("exceed", "please use '--timeout' or '-e' instead")
}

// Helper functions.
func transLimit(limit string) (int, error) {
	if cutil.IsEmptyStr(limit) {
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
	if cutil.IsEmptyStr(filter) {
		return nil
	}
	return strings.Split(filter, "&")
}

func handleNodes() error {
	nodes := make([]string, 0)

	for _, v := range cfg.Node {
		// TODO: check the validity of v.
		if strings.IndexByte(v, ':') > 0 {
			nodes = append(nodes, v)
			continue
		}
		nodes = append(nodes, fmt.Sprintf("%s:%d", v, config.DefaultSupernodePort))
	}
	cfg.Node = nodes
	return nil
}

func resultMsg(cfg *config.Config, end time.Time, e *errors.DfError) string {
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

// dfgetExample shows examples in dfget command, and is used in auto-generated cli docs.
func dfgetExample() string {
	return `
$ dfget -u https://www.taobao.com -o /tmp/test/b.test --notbs --expiretime 20s
--2019-02-02 18:56:34--  https://www.taobao.com
dfget version:0.3.0
workspace:/root/.small-dragonfly sign:96414-1549104994.143
client:127.0.0.1 connected to node:127.0.0.1
start download by dragonfly
download SUCCESS(0) cost:0.026s length:141898 reason:0
`
}
