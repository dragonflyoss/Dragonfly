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

// Package errors defines all exceptions happened in supernode's runtime.
package errors

var (
	// ErrSystemError represents the error is a system error..
	ErrSystemError = DfError{codeSystemError, "system error"}

	// ErrCDNFail represents the cdn status is fail.
	ErrCDNFail = DfError{codeCDNFail, "cdn status is fail"}

	// ErrCDNWait represents the cdn status is wait.
	ErrCDNWait = DfError{codeCDNWait, "cdn status is wait"}

	// ErrPeerWait represents the peer should wait.
	ErrPeerWait = DfError{codePeerWait, "peer should wait"}
)

// IsSystemError check the error is a system error or not.
func IsSystemError(err error) bool {
	return checkError(err, codeSystemError)
}

// IsCDNFail check the error is CDNFail or not.
func IsCDNFail(err error) bool {
	return checkError(err, codeCDNFail)
}

// IsCDNWait check the error is CDNWait or not.
func IsCDNWait(err error) bool {
	return checkError(err, codeCDNWait)
}

// IsPeerWait check the error is PeerWait or not.
func IsPeerWait(err error) bool {
	return checkError(err, codePeerWait)
}
