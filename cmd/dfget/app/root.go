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
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core"
	"github.com/dragonflyoss/Dragonfly/pkg/cmd"
	"github.com/dragonflyoss/Dragonfly/pkg/dflog"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/printer"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type propertiesResult struct {
	prop     *config.Properties
	fileName string
	err      error
}

var filter string

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
	rootCmd.AddCommand(cmd.NewGenDocCommand("dfget"))
	rootCmd.AddCommand(cmd.NewVersionCommand("dfget"))
}

// runDfget does some init operations and starts to download.
func runDfget() error {
	// get config from property files
	propResults, err := initProperties()
	if err != nil {
		return err
	}

	// initialize logger
	if err := initClientLog(); err != nil {
		return err
	}

	// TODO: make the result print of initproperties elegant.
	// print property load result
	for _, propRst := range propResults {
		if propRst.err != nil {
			logrus.Debugf("initProperties[%s] fail: %v", propRst.fileName, propRst.err)
			continue
		}
		logrus.Debugf("initProperties[%s] success: %v", propRst.fileName, propRst.prop)
	}

	cfg.Filter = transFilter(filter)

	if err := checkParameters(); err != nil {
		return err
	}
	logrus.Infof("get cmd params:%q", os.Args)

	if err := config.AssertConfig(cfg); err != nil {
		return errors.Wrap(err, "failed to assert context")
	}
	logrus.Infof("get init config:%v", cfg)

	// enter the core process
	dfError := core.Start(cfg)
	printer.Println(resultMsg(cfg, time.Now(), dfError))
	if dfError != nil {
		os.Exit(dfError.Code)
	}
	return nil
}

func checkParameters() error {
	if len(os.Args) < 2 {
		return errortypes.New(-1, "Please use the command 'help' to show the help information.")
	}
	return nil
}

// initProperties loads config from property files.
func initProperties() ([]*propertiesResult, error) {
	var results []*propertiesResult
	properties := config.NewProperties()
	for _, v := range cfg.ConfigFiles {
		err := properties.Load(v)
		if err == nil {
			break
		}
		results = append(results, &propertiesResult{
			prop:     properties,
			fileName: v,
			err:      err,
		})
	}

	supernodes := cfg.Supernodes
	if supernodes == nil {
		supernodes = properties.Supernodes
	}
	if supernodes != nil {
		cfg.Nodes = config.NodeWeightSlice2StringSlice(supernodes)
	}

	if cfg.LocalLimit == 0 {
		cfg.LocalLimit = properties.LocalLimit
	}

	if cfg.MinRate == 0 {
		cfg.MinRate = properties.MinRate
	}

	if cfg.TotalLimit == 0 {
		cfg.TotalLimit = properties.TotalLimit
	}

	if cfg.ClientQueueSize == 0 {
		cfg.ClientQueueSize = properties.ClientQueueSize
	}

	currentUser, err := user.Current()
	if err != nil {
		printer.Println(fmt.Sprintf("get user error: %s", err))
		os.Exit(config.CodeGetUserError)
	}
	cfg.User = currentUser.Username
	if cfg.WorkHome == "" {
		cfg.WorkHome = properties.WorkHome
		if cfg.WorkHome == "" {
			cfg.WorkHome = filepath.Join(currentUser.HomeDir, ".small-dragonfly")
		}
	}
	cfg.RV.MetaPath = filepath.Join(cfg.WorkHome, "meta", "host.meta")
	cfg.RV.SystemDataDir = filepath.Join(cfg.WorkHome, "data")
	cfg.RV.FileLength = -1

	return results, nil
}

// initClientLog initializes dfget client's logger.
// There are two kinds of logger dfget client uses: logfile and console.
// logfile is used to stored generated log in local filesystem,
// while console log will output the dfget client's log in console/terminal for
// debugging usage.
func initClientLog() error {
	if cfg.LogConfig.Path == "" {
		cfg.LogConfig.Path = filepath.Join(cfg.WorkHome, "logs", "dfclient.log")
	}

	opts := []dflog.Option{
		dflog.WithLogFile(cfg.LogConfig.Path, cfg.LogConfig.MaxSize, cfg.LogConfig.MaxBackups),
		dflog.WithSign(cfg.Sign),
		dflog.WithDebug(cfg.Verbose),
	}

	// Once cfg.Console is set, process should also output log to console.
	if cfg.Console {
		opts = append(opts, dflog.WithConsole())
	}

	return dflog.Init(logrus.StandardLogger(), opts...)
}

func initFlags() {
	// pass to server
	flagSet := rootCmd.Flags()

	// url & output
	flagSet.StringVarP(&cfg.URL, "url", "u", "", "URL of user requested downloading file(only HTTP/HTTPs supported)")
	flagSet.StringVarP(&cfg.Output, "output", "o", "",
		"destination path which is used to store the requested downloading file. It must contain detailed directory and specific filename, for example, '/tmp/file.mp4'")

	// localLimit & minRate & totalLimit & timeout
	flagSet.VarP(&cfg.LocalLimit, "locallimit", "s",
		"network bandwidth rate limit for single download task, in format of G(B)/g/M(B)/m/K(B)/k/B, pure number will also be parsed as Byte")
	flagSet.Var(&cfg.MinRate, "minrate",
		"minimal network bandwidth rate for downloading a file, in format of G(B)/g/M(B)/m/K(B)/k/B, pure number will also be parsed as Byte")
	flagSet.Var(&cfg.TotalLimit, "totallimit",
		"network bandwidth rate limit for the whole host, in format of G(B)/g/M(B)/m/K(B)/k/B, pure number will also be parsed as Byte")
	flagSet.DurationVarP(&cfg.Timeout, "timeout", "e", 0,
		"timeout set for file downloading task. If dfget has not finished downloading all pieces of file before --timeout, the dfget will throw an error and exit")

	// md5 & identifier
	flagSet.StringVarP(&cfg.Md5, "md5", "m", "",
		"md5 value input from user for the requested downloading file to enhance security")
	flagSet.StringVarP(&cfg.Identifier, "identifier", "i", "",
		"the usage of identifier is making different downloading tasks generate different downloading task IDs even if they have the same URLs. conflict with --md5.")
	flagSet.StringVar(&cfg.CallSystem, "callsystem", "",
		"the name of dfget caller which is for debugging. Once set, it will be passed to all components around the request to make debugging easy")
	flagSet.StringSliceVar(&cfg.Cacerts, "cacerts", nil,
		"the cacert file which is used to verify remote server when supernode interact with the source.")
	flagSet.StringVarP(&cfg.Pattern, "pattern", "p", "p2p",
		"download pattern, must be p2p/cdn/source, cdn and source do not support flag --totallimit")
	flagSet.StringVarP(&filter, "filter", "f", "",
		"filter some query params of URL, use char '&' to separate different params"+
			"\neg: -f 'key&sign' will filter 'key' and 'sign' query param"+
			"\nin this way, different but actually the same URLs can reuse the same downloading task")
	flagSet.StringArrayVar(&cfg.Header, "header", nil,
		"http header, eg: --header='Accept: *' --header='Host: abc'")
	flagSet.VarP(config.NewSupernodesValue(&cfg.Supernodes, nil), "node", "n",
		"specify the addresses(host:port=weight) of supernodes where the host is necessary, the port(default: 8002) and the weight(default:1) are optional. And the type of weight must be integer")
	flagSet.BoolVar(&cfg.Notbs, "notbs", false,
		"disable back source downloading for requested file when p2p fails to download it")
	flagSet.BoolVar(&cfg.DFDaemon, "dfdaemon", false,
		"identify whether the request is from dfdaemon")
	flagSet.BoolVar(&cfg.Insecure, "insecure", false,
		"identify whether supernode should skip secure verify when interact with the source.")
	flagSet.IntVar(&cfg.ClientQueueSize, "clientqueue", config.DefaultClientQueueSize,
		"specify the size of client queue which controls the number of pieces that can be processed simultaneously")

	// others
	flagSet.BoolVarP(&cfg.ShowBar, "showbar", "b", false,
		"show progress bar, it is conflict with '--console'")
	flagSet.BoolVar(&cfg.Console, "console", false,
		"show log on console, it's conflict with '--showbar'")
	flagSet.BoolVar(&cfg.Verbose, "verbose", false,
		"be verbose")
	flagSet.StringVar(&cfg.WorkHome, "home", cfg.WorkHome,
		"the work home directory of dfget")

	// pass to peer server which as a uploader server
	flagSet.StringVar(&cfg.RV.LocalIP, "ip", "",
		"IP address that server will listen on")
	flagSet.IntVar(&cfg.RV.PeerPort, "port", 0,
		"port number that server will listen on")
	flagSet.DurationVar(&cfg.RV.DataExpireTime, "expiretime", config.DataExpireTime,
		"caching duration for which cached file keeps no accessed by any process, after this period cache file will be deleted")
	flagSet.DurationVar(&cfg.RV.ServerAliveTime, "alivetime", config.ServerAliveTime,
		"alive duration for which uploader keeps no accessing by any uploading requests, after this period uploader will automatically exit")

	flagSet.MarkDeprecated("exceed", "please use '--timeout' or '-e' instead")
}

func transFilter(filter string) []string {
	if stringutils.IsEmptyStr(filter) {
		return nil
	}
	return strings.Split(filter, "&")
}

func resultMsg(cfg *config.Config, end time.Time, e *errortypes.DfError) string {
	if e != nil {
		return fmt.Sprintf("download FAIL(%d) cost:%.3fs length:%d reason:%d error:%v",
			e.Code, end.Sub(cfg.StartTime).Seconds(), cfg.RV.FileLength,
			cfg.BackSourceReason, e)
	}
	return fmt.Sprintf("download SUCCESS cost:%.3fs length:%d reason:%d",
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
workspace:/root/.small-dragonfly
sign:96414-1549104994.143
client:127.0.0.1 connected to node:127.0.0.1
start download by dragonfly...
download SUCCESS cost:0.026s length:141898 reason:0
`
}
