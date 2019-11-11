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

var (
	// An apiClient is a Dragonfly supernode API client.
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
