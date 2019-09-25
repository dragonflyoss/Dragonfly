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

package environment

import (
	"fmt"
	"runtime"

	"github.com/dragonflyoss/Dragonfly/client"
)

var (
	// SupernodeListenPort is the port that supernode will listen.
	SupernodeListenPort = 8008

	// SupernodeDownloadPort is the port that supernode will listen.
	SupernodeDownloadPort = 8009

	// DragonflySupernodeBinary is the default binary path.
	DragonflySupernodeBinary = "/usr/local/bin/supernode"

	// DragonflyAddress is the default address dragonfly supernode listens on.
	DragonflyAddress = fmt.Sprintf("tcp://127.0.0.1:%d", SupernodeListenPort)

	// TLSConfig is the default TLS config.
	TLSConfig = client.TLSConfig{}

	// GateWay is the default gateway for test.
	GateWay = "192.168.1.1"

	// Subnet is the default subnet for test.
	Subnet = "192.168.1.0/24"
)

func init() {

}

// IsLinux checks if the OS of test environment is Linux.
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsHubConnected checks if hub address can be connected.
func IsHubConnected() bool {
	// TODO: found a proper way to test if hub address can be connected.
	return true
}
