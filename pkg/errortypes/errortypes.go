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

// Package errortypes defines all exceptions happened in dragonfly.
package errortypes

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	// ErrDataNotFound represents the data cannot be found.
	ErrDataNotFound = DfError{codeDataNotFound, "data not found"}

	// ErrEmptyValue represents the value is empty or nil.
	ErrEmptyValue = DfError{codeEmptyValue, "empty value"}

	// ErrInvalidValue represents the value is invalid.
	ErrInvalidValue = DfError{codeInvalidValue, "invalid value"}

	// ErrNotInitialized represents the object is not initialized.
	ErrNotInitialized = DfError{codeNotInitialized, "not initialized"}

	// ErrConvertFailed represents failed to convert.
	ErrConvertFailed = DfError{codeConvertFailed, "convert failed"}

	// ErrRangeNotSatisfiable represents the length of file is insufficient.
	ErrRangeNotSatisfiable = DfError{codeRangeNotSatisfiable, "range not satisfiable"}
)

const (
	codeDataNotFound = iota
	codeEmptyValue
	codeInvalidValue
	codeNotInitialized
	codeConvertFailed
	codeRangeNotSatisfiable

	// supernode
	codeSystemError
	codeCDNFail
	codeCDNWait
	codePeerWait
	codeUnknownError
	codePeerContinue
	codeURLNotReachable
	codeTaskIDDuplicate
	codeAuthenticationRequired
)

// DfError represents a Dragonfly error.
type DfError struct {
	Code int
	Msg  string
}

type ErrAssertFunc func(err error) bool

// New function creates a DfError.
func New(code int, msg string) *DfError {
	return &DfError{
		Code: code,
		Msg:  msg,
	}
}

// Newf function creates a DfError with a message according to
// a format specifier.
func Newf(code int, format string, a ...interface{}) *DfError {
	return &DfError{
		Code: code,
		Msg:  fmt.Sprintf(format, a...),
	}
}

func (s DfError) Error() string {
	return fmt.Sprintf("{\"Code\":%d,\"Msg\":\"%s\"}", s.Code, s.Msg)
}

// IsNilError checks the error is nil or not.
func IsNilError(err error) bool {
	return err == nil
}

// IsDataNotFound checks the error is the data cannot be found.
func IsDataNotFound(err error) bool {
	return checkError(err, codeDataNotFound)
}

// IsEmptyValue checks the error is the value is empty or nil.
func IsEmptyValue(err error) bool {
	return checkError(err, codeEmptyValue)
}

// IsInvalidValue checks the error is the value is invalid or not.
func IsInvalidValue(err error) bool {
	return checkError(err, codeInvalidValue)
}

// IsNotInitialized checks the error is the object is not initialized or not.
func IsNotInitialized(err error) bool {
	return checkError(err, codeNotInitialized)
}

// IsConvertFailed checks the error is a conversion error or not.
func IsConvertFailed(err error) bool {
	return checkError(err, codeConvertFailed)
}

// IsRangeNotSatisfiable checks the error is a
// range not exist error or not.
func IsRangeNotSatisfiable(err error) bool {
	return checkError(err, codeRangeNotSatisfiable)
}

func checkError(err error, code int) bool {
	e, ok := errors.Cause(err).(DfError)
	return ok && e.Code == code
}

type HTTPError struct {
	Code int
	Msg  string
}

// NewHTTPError function creates a HTTPError.
func NewHTTPError(code int, msg string) *HTTPError {
	return &HTTPError{
		Code: code,
		Msg:  msg,
	}
}

// Error function implements the interface of error.Error().
func (s HTTPError) Error() string {
	return fmt.Sprintf("{\"Code\":%d,\"Msg\":\"%s\"}", s.Code, s.Msg)
}

// HTTPCode return the http code.
func (s HTTPError) HTTPCode() int {
	return s.Code
}
