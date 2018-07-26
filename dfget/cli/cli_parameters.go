/*
 * Copyright 1999-2018 Alibaba Group.
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

package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	cfg "github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/dfget/util"
	"github.com/alibaba/Dragonfly/version"
	"github.com/spf13/pflag"
)

var cliOut io.Writer = os.Stderr

func setupFlags(args []string) {
	// url & output
	pflag.StringVarP(&cfg.Ctx.URL, "url", "u", "",
		"will download a file from this url")
	pflag.StringVarP(&cfg.Ctx.Output, "output", "o", "",
		"output path that not only contains the dir part but also name part")

	// localLimit & totalLimit & timeout
	localLimit := pflag.StringP("locallimit", "s", "",
		"rate limit about a single download task, its format is 20M/m/K/k")
	totalLimit := pflag.String("totallimit", "",
		"rate limit about the whole host, its format is 20M/m/K/k")
	pflag.IntVarP(&cfg.Ctx.Timeout, "timeout", "e", 0,
		"download timeout(second)")
	pflag.IntVar(&cfg.Ctx.Timeout, "exceed", 0,
		"download timeout(second)")

	// md5 & identifier
	pflag.StringVarP(&cfg.Ctx.Md5, "md5", "m", "",
		"expected file md5")
	pflag.StringVarP(&cfg.Ctx.Identifier, "identifier", "i", "",
		"identify download task, it is available merely when md5 param not exist")

	pflag.StringVar(&cfg.Ctx.CallSystem, "callsystem", "",
		"system name that executes dfget")

	pflag.StringVarP(&cfg.Ctx.Pattern, "pattern", "p", "p2p",
		"download pattern, must be 'p2p' or 'cdn' or 'source'"+
			"\ncdn/source pattern not support 'totallimit' flag")

	filter := pflag.StringP("filter", "f", "",
		"filter some query params of url, use char '&' to separate different params"+
			"\neg: -f 'key&sign' will filter 'key' and 'sign' query param"+
			"\nin this way, different urls correspond one same download task that can use p2p mode")

	pflag.StringSliceVar(&cfg.Ctx.Header, "header", nil,
		"http header, eg: --header='Accept: *' --header='Host: abc'")

	pflag.StringSliceVarP(&cfg.Ctx.Node, "node", "n", nil,
		"specify supnernodes")

	pflag.BoolVar(&cfg.Ctx.Notbs, "notbs", false,
		"not back source when p2p fail")
	pflag.BoolVar(&cfg.Ctx.DFDaemon, "dfdaemon", false,
		"caller is from dfdaemon")

	// others
	pflag.BoolVarP(&cfg.Ctx.Version, "version", "v", false,
		"show version")
	pflag.BoolVarP(&cfg.Ctx.ShowBar, "showbar", "b", false,
		"show progress bar, it's conflict with '--console'")
	pflag.BoolVar(&cfg.Ctx.Console, "console", false,
		"show log on console, it's conflict with '--showbar")
	pflag.BoolVar(&cfg.Ctx.Verbose, "verbose", false,
		"be verbose")
	pflag.BoolVarP(&cfg.Ctx.Help, "help", "h", false,
		"show help information")

	flags := pflag.CommandLine
	flags.SortFlags = false
	flags.MarkDeprecated("exceed", "please use '--timeout' or '-e' instead")
	flags.Parse(args)

	// be compatible with dfget python version
	var err error
	cfg.Ctx.LocalLimit, err = transLimit(*localLimit)
	util.PanicIfError(err, "convert locallimit error")
	cfg.Ctx.TotalLimit, err = transLimit(*totalLimit)
	util.PanicIfError(err, "convert totallimit error")

	cfg.Ctx.Filter = transFilter(*filter)
}

// Usage shows the usage of this program.
func Usage() {
	fmt.Fprintln(cliOut, "Dragonfly is a file distribution system based p2p.")
	fmt.Fprintf(cliOut, "Usage of %s[%s]:\n", os.Args[0], version.DFGetVersion)
	fmt.Fprintf(cliOut, "%s\n", pflag.CommandLine.FlagUsages())
}

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
