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

package util

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"gopkg.in/yaml.v2"
)

// BufferSize define the buffer size when reading and writing file
const BufferSize = 8 * 1024 * 1024

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
	if !PathExist(filePath) {
		return fmt.Errorf("delete file:%s error, file not exist", filePath)
	}
	if IsDir(filePath) {
		return fmt.Errorf("delete file:%s error, is a directory instead of a file", filePath)
	}
	return os.Remove(filePath)
}

// DeleteFiles deletes all the given files.
func DeleteFiles(filePaths ...string) {
	if len(filePaths) > 0 {
		for _, f := range filePaths {
			DeleteFile(f)
		}
	}
}

// OpenFile open a file. If the parent directory of the file isn't exist,
// it will create the directory.
func OpenFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	if PathExist(path) {
		return os.OpenFile(path, flag, perm)
	}
	if err := CreateDirectory(filepath.Dir(path)); err != nil {
		return nil, err
	}

	return os.OpenFile(path, flag, perm)
}

// Link creates a hard link pointing to src named linkName for a file.
func Link(src string, linkName string) error {
	if PathExist(linkName) {
		if IsDir(linkName) {
			return fmt.Errorf("link %s to %s: error, link name already exists and is a directory", linkName, src)
		}
		if err := DeleteFile(linkName); err != nil {
			return err
		}

	}
	return os.Link(src, linkName)
}

// SymbolicLink creates target as a symbolic link to src.
func SymbolicLink(src string, target string) error {
	// TODO Add verifications.
	return os.Symlink(src, target)
}

// CopyFile copies the file src to dst.
func CopyFile(src string, dst string) (err error) {
	var (
		s *os.File
		d *os.File
	)
	if !IsRegularFile(src) {
		return fmt.Errorf("copy file:%s error, is not a regular file", src)
	}
	if s, err = os.Open(src); err != nil {
		return err
	}
	defer s.Close()

	if PathExist(dst) {
		return fmt.Errorf("copy file:%s error, dst file already exists", dst)
	}

	if d, err = OpenFile(dst, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0755); err != nil {
		return err
	}
	defer d.Close()

	buf := make([]byte, BufferSize)
	for {
		n, err := s.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 || err == io.EOF {
			break
		}
		if _, err := d.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}

// MoveFile moves the file src to dst.
func MoveFile(src string, dst string) error {
	if !IsRegularFile(src) {
		return fmt.Errorf("move file:%s error, is not a regular file", src)
	}
	if PathExist(dst) && !IsDir(dst) {
		if err := DeleteFile(dst); err != nil {
			return err
		}
	}
	return os.Rename(src, dst)
}

// MoveFileAfterCheckMd5 will check whether the file's md5 is equals to the param md5
// before move the file src to dst.
func MoveFileAfterCheckMd5(src string, dst string, md5 string) error {
	if !IsRegularFile(src) {
		return fmt.Errorf("move file with md5 check:%s error, is not a "+
			"regular file", src)
	}
	m := Md5Sum(src)
	if m != md5 {
		return fmt.Errorf("move file with md5 check:%s error, md5 of srouce "+
			"file doesn't match against the given md5 value", src)
	}
	return MoveFile(src, dst)
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

// IsRegularFile reports whether the file is a regular file.
// If the given file is a symbol link, it will follow the link.
func IsRegularFile(name string) bool {
	f, e := os.Stat(name)
	if e != nil {
		return false
	}
	return f.Mode().IsRegular()
}

// Md5Sum generate md5 for a given file
func Md5Sum(name string) string {
	if !IsRegularFile(name) {
		return ""
	}
	f, err := os.Open(name)
	if err != nil {
		return ""
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, BufferSize)
	h := md5.New()

	_, err = io.Copy(h, r)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// GetSys returns the underlying data source of the os.FileInfo.
func GetSys(info os.FileInfo) (*syscall.Stat_t, bool) {
	sys, ok := info.Sys().(*syscall.Stat_t)
	return sys, ok
}

// LoadYaml load yaml config file.
func LoadYaml(path string, out interface{}) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if err = yaml.Unmarshal(content, out); err != nil {
		return fmt.Errorf("path:%s err:%s", path, err)
	}
	return nil
}
