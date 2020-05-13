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

package clientwriter

import (
	"github.com/dragonflyoss/Dragonfly/pkg/protocol"
)

// ClientStream defines how to organize distribution data for range request.
// An instance binds to a range request.
// It may receive a lot of distribution data.
// Developer could add a io.WriteCloser in constructor of instance, and the ClientWriter will
// write request data to io.Writer.
type ClientWriter interface {
	// WriteData writes the distribution data from other peers, it may be called more times.
	PutData(data protocol.DistributionData) error
}
