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

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:           "server",
	Short:         "Launch a peer server to upload files.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Server: %v\n", config.Ctx)
		fmt.Printf("Server: %v\n", config.Ctx.RV.String())
	},
}

func init() {
	initServerFlags()
	rootCmd.AddCommand(serverCmd)
}

func initServerFlags() ()  {
	serverCmd.PersistentFlags().StringVar(&config.Ctx.RV.SystemDataDir, "data", config.Ctx.RV.SystemDataDir,
		"the directory which stores temporary files for p2p uploading")
	serverCmd.PersistentFlags().StringVar(&config.Ctx.WorkHome, "home", config.Ctx.WorkHome,
		"the work home of dfget server")
	serverCmd.PersistentFlags().StringVar(&config.Ctx.RV.LocalIP, "ip", "",
		"the ip that server will listen on")
	serverCmd.PersistentFlags().IntVar(&config.Ctx.RV.PeerPort, "port", 0,
		"the port that server will listen on")
	serverCmd.PersistentFlags().StringVar(&config.Ctx.RV.MetaPath, "meta", config.Ctx.RV.MetaPath,
		"meta file path")
}
