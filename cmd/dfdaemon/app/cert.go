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
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/certutils"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	duration10years = time.Hour * 24 * 365 * 10
)

// GenCACommand is used to implement 'gen-ca' command.
type GenCACommand struct {
	cmd *cobra.Command

	// config contains the basic fields required for creating a certificate
	config certutils.CertConfig

	// keyOutputPath is the destination path of generated ca.key
	keyOutputPath string
	// certOutputPath is the destination path of generated ca.crt
	certOutputPath string
	// overwrite is a flag to control whether to overwrite the existing CA files
	overwrite bool
}

// NewGenCACommand returns cobra.Command for "gen-ca" command
func NewGenCACommand() *cobra.Command {
	genCACommand := &GenCACommand{}
	genCACommand.cmd = &cobra.Command{
		Use:           "gen-ca",
		Short:         fmt.Sprintf("generate CA files, including ca.key and ca.crt"),
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return genCACommand.runGenCA(args)
		},
	}

	genCACommand.addFlags()
	return genCACommand.cmd
}

// addFlags adds flags for specific command.
func (g *GenCACommand) addFlags() {
	flagSet := g.cmd.Flags()

	flagSet.StringVarP(&g.config.CommonName, "common-name", "", "", "subject common name of the certificate, if not specified, the hostname will be used")
	flagSet.DurationVarP(&g.config.ExpireDuration, "expire-duration", "", duration10years, "expire duration of the certificate")
	flagSet.StringVarP(&g.keyOutputPath, "key-output", "", "/tmp/ca.key", "destination path of generated ca.key file")
	flagSet.StringVarP(&g.certOutputPath, "cert-output", "", "/tmp/ca.crt", "destination path of generated ca.crt file")
	flagSet.BoolVarP(&g.overwrite, "overwrite", "", false, "whether to overwrite the existing CA files")
}

// complete completes all the required options.
func (g *GenCACommand) complete() error {
	if g.config.CommonName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return errors.Wrap(err, "failed to read hostname")
		}
		g.config.CommonName = hostname
	}
	return nil
}

// validate validates the provided options.
func (g *GenCACommand) validate() error {
	if _, err := os.Stat(g.keyOutputPath); !os.IsNotExist(err) && !g.overwrite {
		return fmt.Errorf("path %q already exists, please remove it before generating the newer one or pass --overwrite flag explicitly", g.keyOutputPath)
	}
	if _, err := os.Stat(g.certOutputPath); !os.IsNotExist(err) && !g.overwrite {
		return fmt.Errorf("path %q already exists, please remove it before generating the newer one or pass --overwrite flag explicitly", g.certOutputPath)
	}
	return nil
}

func (g *GenCACommand) runGenCA(args []string) error {
	if err := g.complete(); err != nil {
		return err
	}

	if err := g.validate(); err != nil {
		return err
	}

	caCert, caKey, err := certutils.NewCertificateAuthority(&g.config)
	if err != nil {
		return errors.Wrapf(err, "failed to generate CA certificate")
	}

	if err := certutils.WriteKey(g.keyOutputPath, caKey); err != nil {
		return errors.Wrapf(err, "failed to write ca.key to %v", g.keyOutputPath)
	}

	if err := certutils.WriteCert(g.certOutputPath, caCert); err != nil {
		return errors.Wrapf(err, "failed to write ca.crt to %v", g.certOutputPath)
	}

	return nil
}
