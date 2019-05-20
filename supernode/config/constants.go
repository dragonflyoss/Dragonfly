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

package config

const (
	// DefaultSupernodeConfigFilePath the default supernode config path.
	DefaultSupernodeConfigFilePath = "/etc/dragonfly/supernode.yml"

	// SuperNodeCIdPrefix is a string as the prefix of the supernode.
	SuperNodeCIdPrefix = "cdnnode:"
)

// PieceStatus code
const (
	PieceSEMISUC = -3
	PieceWAITING = -1
	PieceRUNNING = 0
	PieceSUCCESS = 1
	PieceFAILED  = 2
)

const (
	// FailCountLimit indicates the limit of fail count as a client.
	FailCountLimit = 5

	// EliminationLimit indicates limit of fail count as a server.
	EliminationLimit = 5

	// PeerUpLimit indicates the limit of the load count as a server.
	PeerUpLimit = 5

	// PeerDownLimit indicates the limit of the download task count as a client.
	PeerDownLimit = 4
)
