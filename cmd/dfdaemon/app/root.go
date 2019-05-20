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
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/dragonflyoss/Dragonfly/cmd/dfdaemon/app/options"
	g "github.com/dragonflyoss/Dragonfly/dfdaemon/global"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/initializer"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/proxy"

	"github.com/pkg/errors"
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
		return runDaemon()
	},
}

func init() {
	opt.AddFlags(rootCmd.Flags())
}

func runTransparentProxy(opt *options.Options) error {
	opts := []proxy.Option{
		proxy.WithRules(g.Properties.Proxies),
		proxy.WithHTTPSHosts(g.Properties.HijackHTTPS.Hosts...),
	}
	if g.Properties.HijackHTTPS.Cert != "" && g.Properties.HijackHTTPS.Key != "" {
		opts = append(opts, proxy.WithCertFromFile(
			g.Properties.HijackHTTPS.Cert,
			g.Properties.HijackHTTPS.Key,
		))
	}
	tp, err := proxy.New(opts...)
	if err != nil {
		return errors.Wrap(err, "failed to create transparent proxy")
	}
	s := http.Server{
		Addr:    fmt.Sprintf(":%d", opt.ProxyPort),
		Handler: tp,
	}
	logrus.Infof("launch dfdaemon transparent proxy on %s:%d", opt.HostIP, opt.ProxyPort)
	go func() {
		logrus.Fatalf("Transparent proxy stopped: %v", s.ListenAndServe())
	}()
	return nil
}

// start to run dfdaemon server.
func runDaemon() error {
	initOption(opt)

	if err := runTransparentProxy(opt); err != nil {
		return err
	}

	logrus.Infof("start dfdaemon param: %+v", opt)

	var err error
	if opt.CertFile != "" && opt.KeyFile != "" {
		logrus.Infof("launch dfdaemon https server on %s:%d", opt.HostIP, opt.Port)
		err = http.ListenAndServeTLS(fmt.Sprintf(":%d", opt.Port),
			opt.CertFile, opt.KeyFile, nil)
	} else {
		logrus.Infof("launch dfdaemon http server on %s:%d", opt.HostIP, opt.Port)
		err = http.ListenAndServe(fmt.Sprintf(":%d", opt.Port), nil)
	}

	if err != nil {
		logrus.Fatal(err)
		return err
	}
	return nil
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
