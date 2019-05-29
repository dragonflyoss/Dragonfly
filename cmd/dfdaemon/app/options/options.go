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

package options

import (
	"os"
	"os/exec"
	"path/filepath"

	flag "github.com/spf13/pflag"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/constant"
)

// Options is the configuration
type Options struct {
	// DfPath `dfget` path.
	DfPath string

	// DFRepo the default value is `$HOME/.small-dragonfly/dfdaemon/data/`.
	DFRepo string

	// RateLimit limit net speed,
	// format:xxxM/K.
	RateLimit string

	// Call system name.
	CallSystem string

	// Filter specified url fields.
	URLFilter string

	// Notbs indicates whether to not back source to download when p2p fails.
	Notbs bool

	// MaxProcs the maximum number of CPUs that the dfdaemon can use.
	MaxProcs int

	// Version show version.
	Version bool

	// Verbose indicates whether to be verbose.
	// If set true, log level will be 'debug'.
	Verbose bool

	// HostIP dfdaemon host ip, default: 127.0.0.1.
	HostIP string

	// Port that dfdaemon will listen, default: 65001.
	Port uint

	// Registry addr and must exist if dfdaemon is used to mirror mode,
	// format: https://xxx.xx.x:port or http://xxx.xx.x:port.
	Registry string

	// The regex download the url by P2P if url matches,
	// format:reg1,reg2,reg3.
	DownRule string

	// Cert file path,
	CertFile string

	// Key file path.
	KeyFile string

	// TrustHosts includes the trusted hosts which dfdaemon forward their
	// requests directly when dfdaemon is used to http_proxy mode.
	TrustHosts []string

	// ConfigPath is the path of dfdaemon's configuration file.
	// default value is: /etc/dragonfly/dfdaemon.yml
	ConfigPath string

	// SupernodeList specify supernode list.
	SupernodeList []string
}

// NewOption returns the default options.
func NewOption() *Options {
	// assume the dfget binary is at the same directory as this daemon.
	var defaultPath string
	if path, err := exec.LookPath(os.Args[0]); err == nil {
		if absPath, err := filepath.Abs(path); err == nil {
			defaultPath = filepath.Dir(absPath) + "/dfget"
		}
	}

	o := &Options{
		DFRepo: filepath.Join(os.Getenv("HOME"), ".small-dragonfly/dfdaemon/data/"),
		DfPath: defaultPath,
	}
	return o
}

// AddFlags add flags to the specified FlagSet.
func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.BoolVarP(&o.Version, "version", "v", false, "version")
	fs.BoolVar(&o.Verbose, "verbose", false, "verbose")
	fs.IntVar(&o.MaxProcs, "maxprocs", 4, "the maximum number of CPUs that the dfdaemon can use")

	// http server config
	fs.StringVar(&o.HostIP, "hostIp", "127.0.0.1", "dfdaemon host ip, default: 127.0.0.1")
	fs.UintVar(&o.Port, "port", 65001, "dfdaemon will listen the port")
	fs.StringVar(&o.CertFile, "certpem", "", "cert.pem file path")
	fs.StringVar(&o.KeyFile, "keypem", "", "key.pem file path")

	// dfget download config
	fs.StringVar(&o.DFRepo, "localrepo", o.DFRepo, "temp output dir of dfdaemon")
	fs.StringVar(&o.CallSystem, "callsystem", "com_ops_dragonfly", "caller name")
	fs.StringVar(&o.DfPath, "dfpath", o.DfPath, "dfget path")
	fs.StringVar(&o.RateLimit, "ratelimit", "", "net speed limit,format:xxxM/K")
	fs.StringVar(&o.URLFilter, "urlfilter", "Signature&Expires&OSSAccessKeyId", "filter specified url fields")
	fs.StringVar(&o.Registry, "registry", "", "registry mirror url, which will override the registry mirror settings in the config file if presented (if not configured through config file or the cli, https://index.docker.io is the default)")
	fs.StringVar(&o.DownRule, "rule", "", "download the url by P2P if url matches the specified pattern,format:reg1,reg2,reg3")
	fs.BoolVar(&o.Notbs, "notbs", true, "not try back source to download if throw exception")
	fs.StringSliceVar(&o.TrustHosts, "trust-hosts", o.TrustHosts, "list of trusted hosts which dfdaemon forward their requests directly, comma separated.")
	fs.StringSliceVar(&o.SupernodeList, "node", o.SupernodeList, "specify the addresses(IP:port) of supernodes that will be passed to dfget.")

	fs.StringVar(&o.ConfigPath, "config", constant.DefaultConfigPath,
		"the path of dfdaemon's configuration file")
}
