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

package cmd

import (
	"fmt"

	"github.com/dragonflyoss/Dragonfly/pkg/printer"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/spf13/cobra"
)

// versionDescriptionTmp is used to describe version command in detail and auto generate command doc.
var versionDescriptionTmp = "Display the version and build information of Dragonfly %s, " +
	"including GoVersion, OS, Arch, Version, BuildDate and GitCommit."

// NewVersionCommand returns cobra.Command for "<component> version" command
func NewVersionCommand(name string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "version",
		Short:         fmt.Sprintf("Show the current version of %s", name),
		Long:          fmt.Sprintf(versionDescriptionTmp, name),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer.Printf(version.Print(name))
			return nil
		},
		Example: versionExample(name),
	}
	return cmd
}

// versionExample shows examples in version command, and is used in auto-generated cli docs.
func versionExample(name string) string {
	return fmt.Sprintf(`%s version  0.4.1
  Git commit:     6fd5c8f
  Build date:     20190717-15:57:52
  Go version:     go1.12.10
  OS/Arch:        linux/amd64
`, name)
}
