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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// NewConfigCommand returns cobra.Command for "<component> config" command
func NewConfigCommand(componentName string, defaultComponentConfigFunc func() (interface{}, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: fmt.Sprintf("Manage the configurations of %s", componentName),
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newConfigPrintDefaultCommand(componentName, defaultComponentConfigFunc))
	return cmd
}

// newConfigPrintDefaultCommand returns cobra.Command for "<component> config default" command
func newConfigPrintDefaultCommand(componentName string, defaultComponentConfigFunc func() (interface{}, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "default",
		Short:         fmt.Sprintf("Print the default configurations of %s in yaml format", componentName),
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigPrintDefault(defaultComponentConfigFunc)
		},
	}
	return cmd
}

func runConfigPrintDefault(defaultComponentConfigFunc func() (interface{}, error)) error {
	cfg, err := defaultComponentConfigFunc()
	if err != nil {
		return errors.Wrap(err, "failed to get component default configurations")
	}
	d, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal component default configurations")
	}
	printer.Print(string(d))
	return nil
}
