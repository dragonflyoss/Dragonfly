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

	"github.com/dragonflyoss/Dragonfly/pkg/rate"
)

/* the reason of backing to source */
const (
	BackSourceReasonNone          = 0
	BackSourceReasonRegisterFail  = 1
	BackSourceReasonMd5NotMatch   = 2
	BackSourceReasonDownloadError = 3
	BackSourceReasonNoSpace       = 4
	BackSourceReasonInitError     = 5
	BackSourceReasonWriteError    = 6
	BackSourceReasonHostSysError  = 7
	BackSourceReasonNodeEmpty     = 8
	BackSourceReasonSourceError   = 10
	BackSourceReasonUserSpecified = 100
	ForceNotBackSourceAddition    = 1000
)

/* download pattern */
const (
	PatternP2P    = "p2p"
	PatternCDN    = "cdn"
	PatternSource = "source"
)

/* properties */
const (
	DefaultYamlConfigFile  = "/etc/dragonfly/dfget.yml"
	DefaultIniConfigFile   = "/etc/dragonfly.conf"
	DefaultLocalLimit      = 20 * rate.MB
	DefaultMinRate         = 64 * rate.KB
	DefaultClientQueueSize = 6
	DefaultSupernodeWeight = 1
)

/* http headers */
const (
	StrRange         = "Range"
	StrContentLength = "Content-Length"
	StrContentType   = "Content-Type"
	StrUserAgent     = "User-Agent"

	StrTaskFileName = "taskFileName"
	StrClientID     = "cid"
	StrTaskID       = "taskID"
	StrSuperNode    = "superNode"
	StrRateLimit    = "rateLimit"
	StrPieceNum     = "pieceNum"
	StrPieceSize    = "pieceSize"
	StrDataDir      = "dataDir"
	StrTotalLimit   = "totalLimit"
	StrCDNSource    = "cdnSource"

	StrBytes   = "bytes"
	StrPattern = "pattern"
)

/* piece meta */
const (
	// PieceHeadSize every piece starts with a piece head which has 4 bytes,
	// its value is:
	//    real data size | (piece size << 4)
	// And it's written with big-endian into the first four bytes of piece data.
	PieceHeadSize = 4

	// PieceTailSize every piece ends with a piece tail which has 1 byte,
	// its value is: 0x7f
	PieceTailSize = 1

	// PieceMetaSize piece meta is constructed with piece head and tail,
	// its size is 5 bytes.
	PieceMetaSize = PieceHeadSize + PieceTailSize

	// PieceTailChar the value of piece tail
	PieceTailChar = byte(0x7f)
)

/* others */
const (
	DefaultTimestampFormat = "2006-01-02 15:04:05"
	SchemaHTTP             = "http"

	ServerPortLowerLimit = 15000
	ServerPortUpperLimit = 65000

	RangeNotSatisfiableDesc = "range not satisfiable"
	AddrUsedDesc            = "address already in use"

	PeerHTTPPathPrefix = "/peer/file/"
	CDNPathPrefix      = "/qtdown/"

	LocalHTTPPathCheck  = "/check/"
	LocalHTTPPathClient = "/client/"
	LocalHTTPPathRate   = "/rate/"
	LocalHTTPPing       = "/server/ping"

	DataExpireTime         = 3 * time.Minute
	ServerAliveTime        = 5 * time.Minute
	DefaultDownloadTimeout = 5 * time.Minute

	DefaultSupernodeSchema = "http"
	DefaultSupernodeIP     = "127.0.0.1"
	DefaultSupernodePort   = 8002
)

/* errors code */
const (
	// CodeLaunchServerError represents failed to launch a peer server.
	CodeLaunchServerError = 1100 + iota

	// CodePrepareError represents failed to prepare before downloading.
	CodePrepareError

	// CodeGetUserError represents failed to get current user.
	CodeGetUserError

	// CodeRegisterError represents failed to register to supernode.
	CodeRegisterError

	// CodeDownloadError represents failed to download file.
	CodeDownloadError
)

const (
	RangeSeparator = "-"
)
