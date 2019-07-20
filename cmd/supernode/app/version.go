package app

import (
	"fmt"

	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/spf13/cobra"
)

// versionDescription is used to describe version command in detail and auto generate command doc.
var versionDescription = "Display the version and build information of Dragonfly supernode， " +
	"including GoVersion, OS, Arch, Version, BuildDate and GitCommit."

var versionCmd = &cobra.Command{
	Use:           "version",
	Short:         "Show the current version of supernode",
	Long:          versionDescription,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(version.Print("supernode"))
		return nil
	},
	Example: versionExample(),
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// versionExample shows examples in version command, and is used in auto-generated cli docs.
func versionExample() string {
	return `supernode version  0.4.1
  Git commit:     6fd5c8f
  Build date:     20190717-15:57:52
  Go version:     go1.12.6
  OS/Arch:        linux/amd64
`
}
