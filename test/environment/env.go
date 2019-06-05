package environment

import (
	"fmt"
	"runtime"

	"github.com/dragonflyoss/Dragonfly/client"
)

// UseJavaVersion whether to run integration test with java version supernode
var UseJavaVersion bool

var (
	// SupernodeListenPort is the port that supernode will listen.
	SupernodeListenPort = 8008

	// DragonflySupernodeBinary is default binary
	DragonflySupernodeBinary = "/usr/local/bin/supernode"

	// DragonflyAddress is default address dragonfly supernode listens on.
	DragonflyAddress = fmt.Sprintf("tcp://127.0.0.1:%d", SupernodeListenPort)

	// TLSConfig is default tls config
	TLSConfig = client.TLSConfig{}

	// GateWay default gateway for test
	GateWay = "192.168.1.1"

	// Subnet default subnet for test
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
