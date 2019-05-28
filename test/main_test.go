package main

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/dragonflyoss/Dragonfly/client"
	"github.com/dragonflyoss/Dragonfly/test/environment"

	"github.com/go-check/check"
)

func init() {
	flag.BoolVar(&environment.UseJavaVersion, "java-version", false, "whether to run integration test with java version supernode")
}

var (
	// A apiClient is a Dragonfly supernode API client.
	apiClient *client.APIClient
)

// TestMain will do initializes and run all the cases.
func TestMain(m *testing.M) {
	var err error

	flag.Parse()
	commonAPIClient, err := client.NewAPIClient(environment.DragonflyAddress, environment.TLSConfig)
	if err != nil {
		fmt.Printf("fail to initializes dragonfly supernode API client: %v", err)
		os.Exit(1)
	}
	apiClient = commonAPIClient.(*client.APIClient)

	os.Exit(m.Run())
}

// Test is the entrypoint.
func Test(t *testing.T) {
	check.TestingT(t)
}
