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
	"path"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/uploader"
	"github.com/dragonflyoss/Dragonfly/dfget/util"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Launch a peer server for uploading files.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer()
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
	flagSet.DurationVar(&cfg.RV.DataExpireTime, "expiretime", config.DataExpireTime,
		"server will delete cached files if these files doesn't be modification within this duration")
	flagSet.DurationVar(&cfg.RV.ServerAliveTime, "alivetime", config.ServerAliveTime,
		"server will stop if there is no uploading task within this duration")
	flagSet.BoolVar(&cfg.Verbose, "verbose", false,
		"be verbose")
}

func runServer() error {
	initServerLog()
	// launch a peer server as a uploader server
	port, err := uploader.LaunchPeerServer(cfg)
	if err != nil {
		return err
	}
	fmt.Println(port)
	uploader.WaitForShutdown()
	return nil
}

func initServerLog() error {
	logFilePath := path.Join(cfg.WorkHome, "logs", "dfserver.log")
	return util.InitLog(cfg.Verbose, logFilePath, cfg.Sign)
}
