package options

import (
	"flag"
	"reflect"
	"testing"
)

func TestAddFlags(t *testing.T) {
	f := flag.NewFlagSet("addflagstest", flag.ContinueOnError)
	options := New()
	options.AddFlags(f)

	args := []string{
		"--localrepo=/tmp/repo",
		"--dfpath=/usr/bin/dfget",
		"--ratelimit=1M",
		"--callsystem=addflagstest",
		"--urlfilter=filter",
		"--notbs=false",
		"--v=false",
		"--verbose=true",
		"--h=false",
		"--hostIp=127.0.0.1",
		"--port=65001",
		"--registry=https://index.docker.io",
		"--rule=testrule",
		"--certpem=/etc/dfdaemon/cert.pem",
		"--keypem=/etc/dfdaemon/key.pem",
		"--maxprocs=1",
	}

	f.Parse(args)

	// This is the desired options parsed from args
	expected := &Options{
		DfPath:     "/usr/bin/dfget",
		DFRepo:     "/tmp/repo",
		RateLimit:  "1M",
		CallSystem: "addflagstest",
		URLFilter:  "filter",
		Notbs:      false,
		MaxProcs:   1,
		Version:    false,
		Verbose:    true,
		Help:       false,
		HostIP:     "127.0.0.1",
		Port:       65001,
		Registry:   "https://index.docker.io",
		DownRule:   "testrule",
		CertFile:   "/etc/dfdaemon/cert.pem",
		KeyFile:    "/etc/dfdaemon/key.pem",
	}

	if !reflect.DeepEqual(expected, options) {
		t.Errorf("Got different run options than expected.")
	}
}
