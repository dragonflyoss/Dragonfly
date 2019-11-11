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

package store

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	// ErrKeyNotFound is an error which will be returned
	// when the key can not be found.
	ErrKeyNotFound = StorageError{codeKeyNotFound, "the key not found"}

	// ErrEmptyKey is an error when the key is empty.
	ErrEmptyKey = StorageError{codeEmptyKey, "the key is empty"}

	// ErrInvalidValue represents the value is invalid.
	ErrInvalidValue = StorageError{codeInvalidValue, "invalid value"}

	// ErrRangeNotSatisfiable represents the length of file is insufficient.
	ErrRangeNotSatisfiable = StorageError{codeRangeNotSatisfiable, "range not satisfiable"}
)

const (
	codeKeyNotFound = iota
	codeEmptyKey
	codeInvalidValue
	codeRangeNotSatisfiable
)

// StorageError represents a storage error.
type StorageError struct {
	Code int
	Msg  string
}

func (s StorageError) Error() string {
	return fmt.Sprintf("{\"Code\":%d,\"Msg\":\"%s\"}", s.Code, s.Msg)
}

// IsNilError checks the error is nil or not.
func IsNilError(err error) bool {
	return err == nil
}

// IsKeyNotFound checks the error is the key cannot be found.
func IsKeyNotFound(err error) bool {
	return checkError(err, codeKeyNotFound)
}

// IsEmptyKey checks the error is the key is empty or nil.
func IsEmptyKey(err error) bool {
	return checkError(err, codeEmptyKey)
}

// IsInvalidValue checks the error is the value is invalid or not.
func IsInvalidValue(err error) bool {
	return checkError(err, codeInvalidValue)
}

// IsRangeNotSatisfiable checks the error is a
// range not exist error or not.
func IsRangeNotSatisfiable(err error) bool {
	return checkError(err, codeRangeNotSatisfiable)
}

func checkError(err error, code int) bool {
	e, ok := errors.Cause(err).(StorageError)
	return ok && e.Code == code
}
