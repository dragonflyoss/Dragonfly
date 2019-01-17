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
	"fmt"

	errHandler "github.com/pkg/errors"
)

var (
	// ErrInvalidValue represents the value is invalid.
	ErrInvalidValue = &DFGetError{codeInvalidValue, "invalid value"}

	// ErrNotInitialized represents the object is not initialized.
	ErrNotInitialized = &DFGetError{codeNotInitialized, "not initialized"}

	// ErrConvertFailed represents failed to convert.
	ErrConvertFailed = &DFGetError{codeConvertFailed, "convert failed"}
)

const (
	codeInvalidValue = iota
	codeNotInitialized
	codeConvertFailed
)

// New function creates a DFGetError.
func New(code int, msg string) *DFGetError {
	return &DFGetError{
		Code: code,
		Msg:  msg,
	}
}

// Newf function creates a DFGetError with a message according to
// a format specifier.
func Newf(code int, format string, a ...interface{}) *DFGetError {
	return &DFGetError{
		Code: code,
		Msg:  fmt.Sprintf(format, a...),
	}
}

// DFGetError represents a error created by dfget.
type DFGetError struct {
	Code int
	Msg  string
}

func (e *DFGetError) Error() string {
	return fmt.Sprintf("{\"Code\":%d,\"Msg\":\"%s\"}", e.Code, e.Msg)
}

// IsNilError check the error is nil or not.
func IsNilError(err error) bool {
	if err == nil {
		return true
	}
	return false
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

func checkError(err error, code int) bool {
	errCause := errHandler.Cause(err)
	if errTemp, ok := errCause.(*DFGetError); ok && errTemp.Code == code {
		return true
	}
	return false
}
