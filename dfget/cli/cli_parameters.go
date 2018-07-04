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
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"
)

// cliParameters holds all the parameters passed by CLI.
type cliParameters struct {
	URL        string   `json:"url"`
	Output     string   `json:"output"`
	LocalLimit string   `json:"locallimit,omitempty"`
	TotalLimit string   `json:"totallimit,omitempty"`
	Timeout    int      `json:"timeout,omitempty"`
	Md5        string   `json:"md5,omitempty"`
	Identifier string   `json:"identifier,omitempty"`
	CallSystem string   `json:"callsystem,omitempty"`
	Filter     string   `json:"filter,omitempty"`
	Pattern    string   `json:"pattern,omitempty"`
	Header     []string `json:"header,omitempty"`
	Node       []string `json:"node,omitempty"`
	Notbs      bool     `json:"notbs,omitempty"`
	DFDaemon   bool     `json:"dfdaemon,omitempty"`
	Version    bool     `json:"version,omitempty"`
	ShowBar    bool     `json:"showbar,omitempty"`
	Console    bool     `json:"console,omitempty"`
	Verbose    bool     `json:"verbose,omitempty"`
	Help       bool     `json:"help,omitempty"`
}

// Params is the parameters passed by CLI.
var Params = new(cliParameters)

var cliOut io.Writer = os.Stderr

func (p *cliParameters) String() string {
	js, _ := json.Marshal(Params)
	return fmt.Sprintf("%s", js)
}

func initParameters(args []string) {
	// url & output
	pflag.StringVarP(&Params.URL, "url", "u", "",
		"will download a file from this url")
	pflag.StringVarP(&Params.Output, "output", "o", "",
		"output path that not only contains the dir part but also name part")

	// localLimit & totalLimit & timeout
	pflag.StringVarP(&Params.LocalLimit, "locallimit", "s", "20M",
		"rate limit about a single download task, its format is 20M/m/K/k")
	pflag.StringVar(&Params.TotalLimit, "totallimit", "",
		"rate limit about the whole host, its format is 20M/m/K/k")
	pflag.IntVarP(&Params.Timeout, "timeout", "e", 0,
		"download timeout(second)")
	pflag.IntVar(&Params.Timeout, "exceed", 0,
		"download timeout(second)")

	// md5 & identifier
	pflag.StringVarP(&Params.Md5, "md5", "m", "",
		"expected file md5")
	pflag.StringVarP(&Params.Identifier, "identifier", "i", "",
		"identify download task, it is available merely when md5 param not exist")

	pflag.StringVar(&Params.CallSystem, "callsystem", "",
		"system name that executes dfget")

	pflag.StringVarP(&Params.Filter, "filter", "f", "",
		"filter some query params of url, use char '&' to separate different params"+
			"\neg: -f 'key&sign' will filter 'key' and 'sign' query param"+
			"\nin this way, different urls correspond one same download task that can use p2p mode")

	pflag.StringVarP(&Params.Pattern, "pattern", "p", "p2p",
		"download pattern, must be 'p2p' or 'cdn'"+
			"\ncdn pattern not support 'totallimit' flag")

	pflag.StringSliceVar(&Params.Header, "header", nil,
		"http header, eg: --header='Accept: *' --header='Host: abc'")

	pflag.StringSliceVarP(&Params.Node, "node", "n", nil,
		"specify supnernodes")

	pflag.BoolVar(&Params.Notbs, "notbs", false,
		"not back source when p2p fail")
	pflag.BoolVar(&Params.DFDaemon, "dfdaemon", false,
		"caller is from dfdaemon")

	// others
	pflag.BoolVarP(&Params.Version, "version", "v", false,
		"show version")
	pflag.BoolVarP(&Params.ShowBar, "showbar", "b", false,
		"show progress bar")
	pflag.BoolVar(&Params.Console, "console", false,
		"show log on console")
	pflag.BoolVar(&Params.Verbose, "verbose", false,
		"be verbose")
	pflag.BoolVarP(&Params.Help, "help", "h", false,
		"show help information")

	flags := pflag.CommandLine
	flags.SortFlags = false
	flags.MarkDeprecated("exceed", "please use '--timeout' or '-e' instead")
	flags.Parse(args)
}

// Usage shows the usage of this program.
func Usage() {
	fmt.Fprintln(cliOut, "Dragonfly is a file distribution system based p2p.")
	fmt.Fprintf(cliOut, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(cliOut, "%s\n", pflag.CommandLine.FlagUsages())
}
