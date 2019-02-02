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
		"the directory which stores temporary files for p2p uploading")
	flagSet.StringVar(&cfg.WorkHome, "home", cfg.WorkHome,
		"the work home of dfget server")
	flagSet.StringVar(&cfg.RV.LocalIP, "ip", "",
		"the ip that server will listen on")
	flagSet.IntVar(&cfg.RV.PeerPort, "port", 0,
		"the port that server will listen on")
	flagSet.StringVar(&cfg.RV.MetaPath, "meta", cfg.RV.MetaPath,
		"meta file path")
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
