package app

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/dragonflyoss/Dragonfly/cmd/dfdaemon/app/options"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/initializer"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	opt = &options.Options{}
)

var rootCmd = &cobra.Command{
	Use:   "dfdaemon",
	Short: "The dfdaemon is a proxy that intercepts image download requests.",
	Long:  "The dfdaemon is a proxy between container engine and registry used for pulling images.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemon()
	},
}

func init() {
	opt.AddFlags(rootCmd.Flags())
}

// start to run dfdaemon server.
func runDaemon() error {
	initOption(opt)

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
