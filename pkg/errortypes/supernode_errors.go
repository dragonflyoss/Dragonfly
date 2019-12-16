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

// Package errortypes defines all exceptions happened in supernode's runtime.
package errortypes

var (
	// ErrSystemError represents the error is a system error.
	ErrSystemError = DfError{codeSystemError, "system error"}

	// ErrCDNFail represents the cdn status is fail.
	ErrCDNFail = DfError{codeCDNFail, "cdn status is fail"}

	// ErrCDNWait represents the cdn status is wait.
	ErrCDNWait = DfError{codeCDNWait, "cdn status is wait"}

	// ErrPeerWait represents the peer should wait.
	ErrPeerWait = DfError{codePeerWait, "peer should wait"}

	// ErrUnknownError represents the error should not happen
	// and the cause of that is unknown.
	ErrUnknownError = DfError{codeUnknownError, "unknown error"}

	// PeerContinue represents the peer should wait.
	PeerContinue = DfError{codePeerContinue, "peer continue"}

	// ErrURLNotReachable represents the url is a not reachable.
	ErrURLNotReachable = DfError{codeURLNotReachable, "url not reachable"}

	// ErrTaskIDDuplicate represents the task id is in conflict.
	ErrTaskIDDuplicate = DfError{codeTaskIDDuplicate, "taskId conflict"}

	// ErrAuthenticationRequired represents the authentication is required.
	ErrAuthenticationRequired = DfError{codeAuthenticationRequired, "authentication required"}
)

// IsSystemError checks the error is a system error or not.
func IsSystemError(err error) bool {
	return checkError(err, codeSystemError)
}

// IsCDNFail checks the error is CDNFail or not.
func IsCDNFail(err error) bool {
	return checkError(err, codeCDNFail)
}

// IsCDNWait checks the error is CDNWait or not.
func IsCDNWait(err error) bool {
	return checkError(err, codeCDNWait)
}

// IsPeerWait checks the error is PeerWait or not.
func IsPeerWait(err error) bool {
	return checkError(err, codePeerWait)
}

// IsUnknowError checks the error is UnknowError or not.
func IsUnknowError(err error) bool {
	return checkError(err, codeUnknownError)
}

// IsPeerContinue checks the error is PeerContinue or not.
func IsPeerContinue(err error) bool {
	return checkError(err, codePeerContinue)
}

// IsURLNotReachable checks the error is a url not reachable or not.
func IsURLNotReachable(err error) bool {
	return checkError(err, codeURLNotReachable)
}

// IsTaskIDDuplicate checks the error is a TaskIDDuplicate error or not.
func IsTaskIDDuplicate(err error) bool {
	return checkError(err, codeTaskIDDuplicate)
}

// IsAuthenticationRequired checks the error is an AuthenticationRequired error or not.
func IsAuthenticationRequired(err error) bool {
	return checkError(err, codeAuthenticationRequired)
}
