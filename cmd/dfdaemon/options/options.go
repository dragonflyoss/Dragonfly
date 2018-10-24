package options

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
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

	// Notbs not try back source to download if throw exception.
	Notbs bool

	// MaxProcs the maximum number of CPUs that the dfdaemon can use.
	MaxProcs int

	// Version show version.
	Version bool

	// Verbose indicates whether to be verbose.
	// If set true, log level will be 'debug'.
	Verbose bool

	// Help show help information.
	Help bool

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
}

// New returns the default options.
func New() *Options {
	// assume the dfget binary is at the same directory as this daemon.
	var defaultPath string
	if path, err := exec.LookPath(os.Args[0]); err == nil {
		if absPath, err := filepath.Abs(path); err == nil {
			defaultPath = filepath.Dir(absPath) + "/dfget"
		}

	}

	o := &Options{
		DFRepo:     filepath.Join(os.Getenv("HOME"), ".small-dragonfly/dfdaemon/data/"),
		DfPath:     defaultPath,
		CallSystem: "com_ops_dragonfly",
		URLFilter:  "Signature&Expires&OSSAccessKeyId",
		Notbs:      true,
		Version:    false,
		Verbose:    false,
		Help:       false,
		HostIP:     "127.0.0.1",
		Port:       65001,
		MaxProcs:   4,
	}
	return o
}

// AddFlags add flags to the specified FlagSet.
func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.DFRepo, "localrepo", o.DFRepo, "temp output dir of dfdaemon")
	fs.StringVar(&o.DfPath, "dfpath", o.DfPath, "dfget path")
	fs.StringVar(&o.RateLimit, "ratelimit", o.RateLimit, "net speed limit,format:xxxM/K")
	fs.StringVar(&o.CallSystem, "callsystem", o.CallSystem, "caller name")
	fs.StringVar(&o.URLFilter, "urlfilter", o.URLFilter, "filter specified url fields")
	fs.BoolVar(&o.Notbs, "notbs", o.Notbs, "not try back source to download if throw exception")
	fs.BoolVar(&o.Version, "v", o.Version, "version")
	fs.BoolVar(&o.Verbose, "verbose", o.Verbose, "verbose")
	fs.BoolVar(&o.Help, "h", o.Help, "help")
	fs.StringVar(&o.HostIP, "hostIp", o.HostIP, "dfdaemon host ip, default: 127.0.0.1")
	fs.UintVar(&o.Port, "port", o.Port, "dfdaemon will listen the port")
	fs.StringVar(&o.Registry, "registry", o.Registry, "registry addr(https://abc.xx.x or http://abc.xx.x) and must exist if dfdaemon is used to mirror mode")
	fs.StringVar(&o.DownRule, "rule", o.DownRule, "download the url by P2P if url matches the specified pattern,format:reg1,reg2,reg3")
	fs.StringVar(&o.CertFile, "certpem", o.CertFile, "cert.pem file path")
	fs.StringVar(&o.KeyFile, "keypem", o.KeyFile, "key.pem file path")
	fs.IntVar(&o.MaxProcs, "maxprocs", o.MaxProcs, "the maximum number of CPUs that the dfdaemon can use")
}
