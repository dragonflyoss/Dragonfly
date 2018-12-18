package app

import (
	"fmt"

	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:           "version",
	Short:         "Show the current version",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Version: %s\n", version.DFGetVersion)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
