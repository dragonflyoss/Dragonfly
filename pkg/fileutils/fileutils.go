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
	"bufio"
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"gopkg.in/yaml.v2"
)

// BufferSize defines the buffer size when reading and writing file.
const BufferSize = 8 * 1024 * 1024

// CreateDirectory creates directory recursively.
func CreateDirectory(dirPath string) error {
	f, e := os.Stat(dirPath)
	if e != nil {
		if os.IsNotExist(e) {
			return os.MkdirAll(dirPath, 0755)
		}
		return fmt.Errorf("failed to create dir %s: %v", dirPath, e)
	}
	if !f.IsDir() {
		return fmt.Errorf("failed to create dir %s: dir path already exists and is not a directory", dirPath)
	}
	return e
}

// DeleteFile deletes a file not a directory.
func DeleteFile(filePath string) error {
	if !PathExist(filePath) {
		return fmt.Errorf("failed to delete file %s: file not exist", filePath)
	}
	if IsDir(filePath) {
		return fmt.Errorf("failed to delete file %s: file path is a directory rather than a file", filePath)
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

// OpenFile opens a file. If the parent directory of the file isn't exist,
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
			return fmt.Errorf("failed to link %s to %s: link name already exists and is a directory", linkName, src)
		}
		if err := DeleteFile(linkName); err != nil {
			return fmt.Errorf("failed to link %s to %s when deleting target file: %v", linkName, src, err)
		}

	}
	return os.Link(src, linkName)
}

// SymbolicLink creates target as a symbolic link to src.
func SymbolicLink(src string, target string) error {
	if !PathExist(src) {
		return fmt.Errorf("failed to symlink %s to %s: src no such file or directory", target, src)
	}
	if PathExist(target) {
		if IsDir(target) {
			return fmt.Errorf("failed to symlink %s to %s: link name already exists and is a directory", target, src)
		}
		if err := DeleteFile(target); err != nil {
			return fmt.Errorf("failed to symlink %s to %s when deleting target file: %v", target, src, err)
		}

	}
	return os.Symlink(src, target)
}

// CopyFile copies the file src to dst.
func CopyFile(src string, dst string) (err error) {
	var (
		s *os.File
		d *os.File
	)
	if !IsRegularFile(src) {
		return fmt.Errorf("failed to copy %s to %s: src is not a regular file", src, dst)
	}
	if s, err = os.Open(src); err != nil {
		return fmt.Errorf("failed to copy %s to %s when opening source file: %v", src, dst, err)
	}
	defer s.Close()

	if PathExist(dst) {
		return fmt.Errorf("failed to copy %s to %s: dst file already exists", src, dst)
	}

	if d, err = OpenFile(dst, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0755); err != nil {
		return fmt.Errorf("failed to copy %s to %s when opening destination file: %v", src, dst, err)
	}
	defer d.Close()

	buf := make([]byte, BufferSize)
	for {
		n, err := s.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to copy %s to %s when reading src file: %v", src, dst, err)
		}
		if n == 0 || err == io.EOF {
			break
		}
		if _, err := d.Write(buf[:n]); err != nil {
			return fmt.Errorf("failed to copy %s to %s when writing dst file: %v", src, dst, err)
		}
	}
	return nil
}

// MoveFile moves the file src to dst.
func MoveFile(src string, dst string) error {
	if !IsRegularFile(src) {
		return fmt.Errorf("failed to move %s to %s: src is not a regular file", src, dst)
	}
	if PathExist(dst) && !IsDir(dst) {
		if err := DeleteFile(dst); err != nil {
			return fmt.Errorf("failed to move %s to %s when deleting dst file: %v", src, dst, err)
		}
	}
	return os.Rename(src, dst)
}

// MoveFileAfterCheckMd5 will check whether the file's md5 is equals to the param md5
// before move the file src to dst.
func MoveFileAfterCheckMd5(src string, dst string, md5 string) error {
	if !IsRegularFile(src) {
		return fmt.Errorf("failed to move file with md5 check %s to %s: src is not a regular file", src, dst)
	}
	m := Md5Sum(src)
	if m != md5 {
		return fmt.Errorf("failed to move file with md5 check %s to %s: md5 of source file doesn't match against the given md5 value", src, dst)
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

// Md5Sum generates md5 for a given file.
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

	return GetMd5Sum(h, nil)
}

// GetMd5Sum gets md5 sum as a string and appends the current hash to b.
func GetMd5Sum(md5 hash.Hash, b []byte) string {
	return fmt.Sprintf("%x", md5.Sum(b))
}

// GetSys returns the underlying data source of the os.FileInfo.
func GetSys(info os.FileInfo) (*syscall.Stat_t, bool) {
	sys, ok := info.Sys().(*syscall.Stat_t)
	return sys, ok
}

// LoadYaml loads yaml config file.
func LoadYaml(path string, out interface{}) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to load yaml %s when reading file: %v", path, err)
	}
	if err = yaml.Unmarshal(content, out); err != nil {
		return fmt.Errorf("failed to load yaml %s: %v", path, err)
	}
	return nil
}

// GetFreeSpace gets the free disk space of the path.
func GetFreeSpace(path string) (Fsize, error) {
	fs := syscall.Statfs_t{}
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, err
	}

	return Fsize(fs.Bavail * uint64(fs.Bsize)), nil
}

// IsEmptyDir check whether the directory is empty.
func IsEmptyDir(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	if _, err = f.Readdirnames(1); err == io.EOF {
		return true, nil
	}
	return false, err
}
