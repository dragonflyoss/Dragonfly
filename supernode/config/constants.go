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

import (
	"time"
)

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

const (
	// DefaultPieceSize 4M
	DefaultPieceSize = 4 * 1024 * 1024

	// DefaultPieceSizeLimit 15M
	DefaultPieceSizeLimit = 15 * 1024 * 1024

	// PieceHeadSize 4 bytes
	PieceHeadSize = 4

	// PieceWrapSize 4 bytes head and 1 byte tail
	PieceWrapSize = PieceHeadSize + 1

	// PieceTailChar the value of piece tail
	PieceTailChar = byte(0x7f)
)

const (
	// CDNWriterRoutineLimit 4
	CDNWriterRoutineLimit = 4
)

const (
	// SubsystemSupernode represents metrics from supernode
	SubsystemSupernode = "supernode"
	// SubsystemDfget represents metrics from dfget
	SubsystemDfget = "dfget"
)

const (
	// DefaultFailAccessInterval is the interval time after failed to access the URL.
	DefaultFailAccessInterval = 3 * time.Minute

	// DefaultGCInitialDelay is the delay time from the start to the first GC execution.
	DefaultGCInitialDelay = 6 * time.Second

	// DefaultGCMetaInterval is the interval time to execute the GC meta.
	DefaultGCMetaInterval = 2 * time.Minute

	// DefaultTaskExpireTime when a task is not accessed within the taskExpireTime,
	// and it will be treated to be expired.
	DefaultTaskExpireTime = 3 * time.Minute

	// DefaultPeerGCDelay is the delay time to execute the GC after the peer has reported the offline.
	DefaultPeerGCDelay = 3 * time.Minute
)
