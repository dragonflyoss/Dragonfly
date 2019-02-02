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

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/uploader"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Launch a peer server for uploading files.",
	Run: func(cmd *cobra.Command, args []string) {

		initServerLog()

		// launch a peer server as a uploader server
		port, err := uploader.LaunchPeerServer(cfg)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(config.CodeLaunchServerError)
		}
		fmt.Println(port)
		uploader.WaitForShutdown()
	},
}

func init() {
	initServerFlags()
	rootCmd.AddCommand(serverCmd)
}

func initServerFlags() {
	flagSet := serverCmd.Flags()

	flagSet.StringVar(&cfg.RV.SystemDataDir, "data", cfg.RV.SystemDataDir,
		"local directory which stores temporary files for p2p uploading")
	flagSet.StringVar(&cfg.WorkHome, "home", cfg.WorkHome,
		"the work home directory of dfget server")
	flagSet.StringVar(&cfg.RV.LocalIP, "ip", "",
		"IP address that server will listen on")
	flagSet.IntVar(&cfg.RV.PeerPort, "port", 0,
		"port number that server will listen on")
	flagSet.StringVar(&cfg.RV.MetaPath, "meta", cfg.RV.MetaPath,
		"meta file path")

	flagSet.DurationVar(&cfg.RV.DataExpireTime, "expiretime", config.DataExpireTime,
		"caching duration for which cached file keeps no accessed by any process, after this period cache file will be deleted")
	flagSet.DurationVar(&cfg.RV.ServerAliveTime, "alivetime", config.ServerAliveTime,
		"Alive duration for which uploader keeps no accessing by any uploading requests, after this period uploader will automically exit")

	flagSet.BoolVar(&cfg.Verbose, "verbose", false,
		"be verbose")
}

func initServerLog() {
	level := "INFO"
	if cfg.Verbose {
		level = "DEBUG"
	}
	serverLogger, err := util.CreateLogger(filepath.Join(cfg.WorkHome, "logs"),
		"dfserver.log", level, cfg.Sign)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(config.CodeLaunchServerError)
	}
	cfg.ServerLogger = serverLogger
}
