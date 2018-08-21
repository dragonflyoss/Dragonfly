/*
 * Copyright 1999-2018 Alibaba Group.
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

/* the response code from supernode */
const (
	// HTTPError represents that there is an error between client and server.
	HTTPError = -100
	// Success represents the request is success.
	Success       = 200
	ResultFail    = 500
	ResultSuc     = 501
	ResultInvalid = 502
	// ResultSemiSuc represents the result is partial successful.
	ResultSemiSuc = 503
)

/* report status of task to supernode */
const (
	TaskStatusStart   = 700
	TaskStatusRunning = 701
	TaskStatusFinish  = 702
)

/* the task code get from supernode */
const (
	TaskCodeFinish          = 600
	TaskCodeContinue        = 601
	TaskCodeWait            = 602
	TaskCodeLimited         = 603
	TaskCodeSuperFail       = 604
	TaskCodeUnknownError    = 605
	TaskCodeTaskConflict    = 606
	TaskCodeURLNotReachable = 607
	TaskCodeNeedAuth        = 608
	TaskCodeWaitAuth        = 609
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
	DefaultYamlConfigFile  = "/etc/dragonfly.yaml"
	DefaultIniConfigFile   = "/etc/dragonfly.conf"
	DefaultNode            = "127.0.0.1"
	DefaultLocalLimit      = 20 * 1024 * 1024
	DefaultClientQueueSize = 6
)

/* others */
const (
	DefaultTimestampFormat = "2006-01-02 15:04:05"
	SchemaHTTP             = "http"

	ServerPortLowerLimit = 15000
	ServerPortUpperLimit = 65000

	RangeNotExistDesc = "range not satisfiable"
	AddrUsedDesc      = "address already in use"

	PeerHTTPPathPrefix = "/peer/file/"
	CDNPathPrefix      = "/qtdown/"

	LocalHTTPPathCheck  = "/check/"
	LocalHTTPPathClient = "/client/"
	LocalHTTPPathRate   = "/rate/"
)
