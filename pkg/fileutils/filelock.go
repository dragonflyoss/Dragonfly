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

package fileutils

import (
	"fmt"
	"os"
	"syscall"

	"github.com/pkg/errors"
)

// FileLock defines a file lock implemented by syscall.Flock
type FileLock struct {
	fileName string
	fd       *os.File
}

// NewFileLock create a FileLock instance
func NewFileLock(name string) *FileLock {
	return &FileLock{
		fileName: name,
	}
}

// Lock locks file.
// If the file is already locked, the calling goroutine blocks until the file is unlocked.
// If lock has been invoked without unlock, lock again will return an error.
func (l *FileLock) Lock() error {
	var (
		fd  *os.File
		err error
	)

	if l.fd != nil {
		return fmt.Errorf("file %s has already been locked", l.fileName)
	}

	if fd, err = os.Open(l.fileName); err != nil {
		return err
	}
	l.fd = fd

	if err := syscall.Flock(int(l.fd.Fd()), syscall.LOCK_EX); err != nil {
		return errors.Wrapf(err, "file %s lock failed", l.fileName)
	}
	return nil
}

// Unlock unlocks file.
// If lock has not been invoked before unlock, unlock will return an error.
func (l *FileLock) Unlock() error {
	if l.fd == nil {
		return fmt.Errorf("file %s descriptor is nil", l.fileName)
	}
	fd := l.fd
	l.fd = nil

	defer fd.Close()
	if err := syscall.Flock(int(fd.Fd()), syscall.LOCK_UN); err != nil {
		return errors.Wrapf(err, "file %s unlock failed", l.fileName)
	}
	return nil
}
