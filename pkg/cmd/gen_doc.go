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

func NewGenDocCommand(name string) *cobra.Command {
	genDocCommand := &GenDocCommand{}
	genDocCommand.cmd = &cobra.Command{
		Use:           "gen-doc",
		Short:         fmt.Sprintf("Generate Document for %s command line tool in MarkDown format", name),
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return genDocCommand.runGenDoc(args)
		},
	}
	genDocCommand.addFlags()
	return genDocCommand.cmd
}

// addFlags adds flags for specific command.
func (g *GenDocCommand) addFlags() {
	flagSet := g.cmd.Flags()

	flagSet.StringVarP(&g.path, "path", "p", "/tmp", "destination path of generated markdown documents")
}

func (g *GenDocCommand) runGenDoc(args []string) error {
	// FIXME: make document path configurable
	if _, err := os.Stat(g.path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path %s does not exist, please check your gen-doc input flag --path", g.path)
		}
		return err
	}
	return doc.GenMarkdownTree(g.cmd.Parent(), g.path)
}
