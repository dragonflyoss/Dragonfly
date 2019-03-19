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

// Package errors defines all exceptions happened in dfget's runtime.
package errors

import (
	errHandler "github.com/pkg/errors"
)

var (
	// ErrDataNotFound represents the data cannot be found..
	ErrDataNotFound = SuperError{codeDataNotFound, "data not found"}

	// ErrInvalidValue represents the value is invalid.
	ErrInvalidValue = SuperError{codeInvalidValue, "invalid value"}

	// ErrEmptyValue represents the value is empty or nil.
	ErrEmptyValue = SuperError{codeEmptyValue, "empty value"}

	// ErrNotInitialized represents the object is not initialized.
	ErrNotInitialized = SuperError{codeNotInitialized, "not initialized"}

	// ErrConvertFailed represents failed to convert.
	ErrConvertFailed = SuperError{codeConvertFailed, "convert failed"}

	// ErrRangeNotSatisfiable represents the length of file is insufficient.
	ErrRangeNotSatisfiable = SuperError{codeRangeNotSatisfiable, "range not satisfiable"}

	// ErrSystemError represents the error is a system error..
	ErrSystemError = SuperError{codeSystemError, "system error"}
)

const (
	codeDataNotFound = iota
	codeEmptyValue
	codeInvalidValue
	codeNotInitialized
	codeConvertFailed
	codeRangeNotSatisfiable
	codeSystemError
)

// SuperError represents a error created by supernode.
type SuperError struct {
	Code int
	Msg  string
}

func (s SuperError) Error() string {
	return s.Msg
}

// IsNilError check the error is nil or not.
func IsNilError(err error) bool {
	return err == nil
}

// IsDataNotFound check the error is the data cannot be found.
func IsDataNotFound(err error) bool {
	return checkError(err, codeDataNotFound)
}

// IsEmptyValue check the error is the value is empty or nil.
func IsEmptyValue(err error) bool {
	return checkError(err, codeEmptyValue)
}

// IsInvalidValue check the error is the value is invalid or not.
func IsInvalidValue(err error) bool {
	return checkError(err, codeInvalidValue)
}

// IsNotInitialized check the error is the object is not initialized or not.
func IsNotInitialized(err error) bool {
	return checkError(err, codeNotInitialized)
}

// IsConvertFailed check the error is a conversion error or not.
func IsConvertFailed(err error) bool {
	return checkError(err, codeConvertFailed)
}

// IsRangeNotSatisfiable check the error is a
// range not exist error or not.
func IsRangeNotSatisfiable(err error) bool {
	return checkError(err, codeRangeNotSatisfiable)
}

// IsSystemError check the error is a system error or not.
func IsSystemError(err error) bool {
	return checkError(err, codeSystemError)
}

func checkError(err error, code int) bool {
	e, ok := errHandler.Cause(err).(SuperError)
	return ok && e.Code == code
}
