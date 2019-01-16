package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// GenDocCommand is used to implement 'gen-doc' command.
type GenDocCommand struct {
	cmd *cobra.Command

	// path is the destination path of generated markdown documents.
	path string
}

func init() {
	genDocCommand := &GenDocCommand{}
	genDocCommand.cmd = &cobra.Command{
		Use:           "gen-doc",
		Short:         "Generate Document for dfdaemon command line tool with MarkDown format",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return genDocCommand.runGenDoc(args)
		},
	}
	genDocCommand.addFlags()
	rootCmd.AddCommand(genDocCommand.cmd)
}

// addFlags adds flags for specific command.
func (g *GenDocCommand) addFlags() {
	flagSet := g.cmd.Flags()

	flagSet.StringVarP(&g.path, "path", "p", "/tmp", "destination path of generated markdown documents")
}

func (g *GenDocCommand) runGenDoc(args []string) error {
	if _, err := os.Stat(g.path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path %s does not exits, please check your gen-doc input flag --path", g.path)
		}
		return err
	}
	return doc.GenMarkdownTree(g.cmd.Parent(), g.path)
}
