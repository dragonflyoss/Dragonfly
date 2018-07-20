// Copyright 1999-2017 Alibaba Group.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"runtime"

	"github.com/sirupsen/logrus"

	"github.com/alibaba/Dragonfly/cmd/dfdaemon/options"
	"github.com/alibaba/Dragonfly/dfdaemon/initializer"
)

func main() {
	options := options.New()
	options.AddFlags(flag.CommandLine)
	flag.Parse()

	initializer.Init(options)

	// if CommandLine.MaxProcs <= 0, programs run with GOMAXPROCS set to the number of cores available
	if options.MaxProcs > 0 {
		runtime.GOMAXPROCS(options.MaxProcs)
	}

	logrus.Infof("start dfdaemon param:%+v", options)

	fmt.Printf("\nlaunch dfdaemon on port:%d\n", options.Port)

	var err error

	if options.CertFile != "" && options.KeyFile != "" {
		err = http.ListenAndServeTLS(fmt.Sprintf(":%d", options.Port),
			options.CertFile, options.KeyFile, nil)
	} else {
		err = http.ListenAndServe(fmt.Sprintf(":%d", options.Port), nil)
	}

	if err != nil {
		fmt.Printf("%v", err)
		logrus.Fatal(err)
	}
}
