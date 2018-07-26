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

package util

import (
	"fmt"
	"os"
)

// CreateDirectory creates directory recursively.
func CreateDirectory(dirPath string) error {
	f, e := os.Stat(dirPath)
	if e != nil && os.IsNotExist(e) {
		return os.MkdirAll(dirPath, 0755)
	}
	if e == nil && !f.IsDir() {
		return fmt.Errorf("create dir:%s error, not a directory", dirPath)
	}
	return e
}

// DeleteFile deletes a file not a directory.
func DeleteFile(filePath string) error {
	return nil
}

// DeleteFiles deletes all the given files.
func DeleteFiles(filePaths ...string) {

}

// OpenFile open a file. If the file isn't exist, it will create the file.
// If the directory isn't exist, it will create the directory.
func OpenFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	return nil, nil
}

// Link creates a hard link pointing to src named linkName.
func Link(src string, linkName string) error {
	return nil
}

// CopyFile copies the file src to dst.
func CopyFile(src string, dst string) error {
	return nil
}

// MoveFile moves the file src to dst.
func MoveFile(src string, dst string) error {
	return nil
}

// MoveFileAfterCheckMd5 will check whether the file's md5 is equals to the param md5
// before move the file src to dst.
func MoveFileAfterCheckMd5(src string, dst string, md5 string) error {
	return nil
}

// PathExist reports whether the path is exist.
// Any error get from os.Stat, it will return false.
func PathExist(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// IsDir reports whether the path is a directory.
func IsDir(name string) bool {
	f, e := os.Stat(name)
	if e != nil {
		return false
	}
	return f.IsDir()
}
