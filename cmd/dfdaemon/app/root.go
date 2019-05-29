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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/dragonflyoss/Dragonfly/cmd/dfdaemon/app/options"
	"github.com/dragonflyoss/Dragonfly/dfdaemon"
	g "github.com/dragonflyoss/Dragonfly/dfdaemon/global"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/initializer"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	opt = options.NewOption()
)

var rootCmd = &cobra.Command{
	Use:               "dfdaemon",
	Short:             "The dfdaemon is a proxy that intercepts image download requests.",
	Long:              "The dfdaemon is a proxy between container engine and registry used for pulling images.",
	DisableAutoGenTag: true, // disable displaying auto generation tag in cli docs
	RunE: func(cmd *cobra.Command, args []string) error {
		initOption(opt)

		s, err := dfdaemon.NewFromConfig(*g.Properties, *opt)
		if err != nil {
			logrus.Fatal(err)
			return err
		}

		return s.Start()
	},
}

func init() {
	opt.AddFlags(rootCmd.Flags())
}

// initOption do some initialization for running dfdaemon.
func initOption(opt *options.Options) {
	if opt.DfPath == "" {
		if path, err := exec.LookPath(os.Args[0]); err == nil {
			if absPath, err := filepath.Abs(path); err == nil {
				// assume the dfget binary is at the same directory as this daemon.
				opt.DfPath = filepath.Dir(absPath) + "/dfget"
			}
		}
	}

	initializer.Init(opt)

	// if Options.MaxProcs <= 0, programs run with GOMAXPROCS set to the number of cores available.
	if opt.MaxProcs > 0 {
		runtime.GOMAXPROCS(opt.MaxProcs)
	}
}

// Execute will process dfdaemon.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}
