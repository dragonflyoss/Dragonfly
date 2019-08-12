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

	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/spf13/cobra"
)

// versionDescription is used to describe version command in detail and auto generate command doc.
var versionDescription = "Display the version and build information of Dragonfly dfdaemon, " +
	"including GoVersion, OS, Arch, Version, BuildDate and GitCommit."

var versionCmd = &cobra.Command{
	Use:           "version",
	Short:         "Show the current version of dfdaemon",
	Long:          versionDescription,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(version.Print("dfdaemon"))
		return nil
	},
	Example: versionExample(),
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// versionExample shows examples in version command, and is used in auto-generated cli docs.
func versionExample() string {
	return `dfdaemon version  0.4.1
  Git commit:     6fd5c8f
  Build date:     20190717-15:57:52
  Go version:     go1.12.6
  OS/Arch:        linux/amd64
`
}
