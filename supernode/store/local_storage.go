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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	statutils "github.com/dragonflyoss/Dragonfly/pkg/stat"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// LocalStorageDriver is a const of local storage driver.
const LocalStorageDriver = "local"

var fileLocker = util.NewLockerPool()

func init() {
	Register(LocalStorageDriver, NewLocalStorage)
}

func lock(path string, offset int64, ro bool) {
	if offset != -1 {
		fileLocker.GetLock(getLockKey(path, -1), true)
	}

	fileLocker.GetLock(getLockKey(path, offset), ro)
}

func unLock(path string, offset int64, ro bool) {
	if offset != -1 {
		fileLocker.ReleaseLock(getLockKey(path, -1), true)
	}

	fileLocker.ReleaseLock(getLockKey(path, offset), ro)
}

// localStorage is one of the implementations of StorageDriver using local file system.
type localStorage struct {
	// BaseDir is the dir that local storage driver will store content based on it.
	BaseDir string `yaml:"baseDir"`
}

// NewLocalStorage performs initialization for localStorage and return a StorageDriver.
func NewLocalStorage(conf string) (StorageDriver, error) {
	// type assertion for config
	cfg := &localStorage{}
	if err := yaml.Unmarshal([]byte(conf), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	// prepare the base dir
	if !filepath.IsAbs(cfg.BaseDir) {
		return nil, fmt.Errorf("not absolute path: %s", cfg.BaseDir)
	}
	if err := fileutils.CreateDirectory(cfg.BaseDir); err != nil {
		return nil, err
	}

	return &localStorage{
		BaseDir: cfg.BaseDir,
	}, nil
}

// Get the content of key from storage and return in io stream.
func (ls *localStorage) Get(ctx context.Context, raw *Raw) (io.Reader, error) {
	path, info, err := ls.statPath(raw.Bucket, raw.Key)
	if err != nil {
		return nil, err
	}

	if err := checkGetRaw(raw, info.Size()); err != nil {
		return nil, err
	}

	r, w := io.Pipe()
	go func() {
		defer w.Close()

		lock(path, raw.Offset, true)
		defer unLock(path, raw.Offset, true)

		f, err := os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()

		f.Seek(raw.Offset, 0)
		var reader io.Reader
		reader = f
		if raw.Length > 0 {
			reader = io.LimitReader(f, raw.Length)
		}

		buf := make([]byte, 256*1024)
		io.CopyBuffer(w, reader, buf)
	}()

	return r, nil
}

// GetBytes gets the content of key from storage and return in bytes.
func (ls *localStorage) GetBytes(ctx context.Context, raw *Raw) (data []byte, err error) {
	path, info, err := ls.statPath(raw.Bucket, raw.Key)
	if err != nil {
		return nil, err
	}

	if err := checkGetRaw(raw, info.Size()); err != nil {
		return nil, err
	}

	lock(path, raw.Offset, true)
	defer unLock(path, raw.Offset, true)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	f.Seek(raw.Offset, 0)
	if raw.Length <= 0 {
		data, err = ioutil.ReadAll(f)
	} else {
		data = make([]byte, raw.Length)
		_, err = f.Read(data)
	}

	if err != nil {
		return nil, err
	}
	return data, nil
}

// Put reads the content from reader and put it into storage.
func (ls *localStorage) Put(ctx context.Context, raw *Raw, data io.Reader) error {
	if err := checkPutRaw(raw); err != nil {
		return err
	}

	path, err := ls.preparePath(raw.Bucket, raw.Key)
	if err != nil {
		return err
	}

	if data == nil {
		return nil
	}

	lock(path, raw.Offset, false)
	defer unLock(path, raw.Offset, false)

	f, err := fileutils.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Seek(raw.Offset, 0)
	if raw.Length > 0 {
		if _, err = io.CopyN(f, data, raw.Length); err != nil {
			return err
		}
		return nil
	}

	buf := make([]byte, 256*1024)
	if _, err = io.CopyBuffer(f, data, buf); err != nil {
		return err
	}

	return nil
}

// PutBytes puts the content of key from storage with bytes.
func (ls *localStorage) PutBytes(ctx context.Context, raw *Raw, data []byte) error {
	if err := checkPutRaw(raw); err != nil {
		return err
	}

	path, err := ls.preparePath(raw.Bucket, raw.Key)
	if err != nil {
		return err
	}

	lock(path, raw.Offset, false)
	defer unLock(path, raw.Offset, false)

	f, err := fileutils.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Seek(raw.Offset, 0)
	if raw.Length == 0 {
		if _, err := f.Write(data); err != nil {
			return err
		}
		return nil
	}

	if _, err := f.Write(data[:raw.Length]); err != nil {
		return err
	}
	return nil
}

// Stat determines whether the file exists.
func (ls *localStorage) Stat(ctx context.Context, raw *Raw) (*StorageInfo, error) {
	_, fileInfo, err := ls.statPath(raw.Bucket, raw.Key)
	if err != nil {
		return nil, err
	}

	sys, ok := fileutils.GetSys(fileInfo)
	if !ok {
		return nil, fmt.Errorf("get create time error")
	}
	return &StorageInfo{
		Path:       filepath.Join(raw.Bucket, raw.Key),
		Size:       fileInfo.Size(),
		CreateTime: statutils.Ctime(sys),
		ModTime:    fileInfo.ModTime(),
	}, nil
}

// Remove deletes a file or dir.
func (ls *localStorage) Remove(ctx context.Context, raw *Raw) error {
	path, _, err := ls.statPath(raw.Bucket, raw.Key)
	if err != nil {
		return err
	}

	lock(path, -1, false)
	defer unLock(path, -1, false)

	return os.RemoveAll(path)
}

// GetAvailSpace returns the available disk space in B.
func (ls *localStorage) GetAvailSpace(ctx context.Context, raw *Raw) (fileutils.Fsize, error) {
	path, _, err := ls.statPath(raw.Bucket, raw.Key)
	if err != nil {
		return 0, err
	}

	lock(path, -1, true)
	defer unLock(path, -1, true)
	return fileutils.GetFreeSpace(path)
}

// Walk walks the file tree rooted at root which determined by raw.Bucket and raw.Key,
// calling walkFn for each file or directory in the tree, including root.
func (ls *localStorage) Walk(ctx context.Context, raw *Raw) error {
	path, _, err := ls.statPath(raw.Bucket, raw.Key)
	if err != nil {
		return err
	}

	lock(path, -1, true)
	defer unLock(path, -1, true)

	return filepath.Walk(path, raw.WalkFn)
}

// helper function

// preparePath gets the target path and creates the upper directory if it does not exist.
func (ls *localStorage) preparePath(bucket, key string) (string, error) {
	dir := filepath.Join(ls.BaseDir, bucket)

	if err := fileutils.CreateDirectory(dir); err != nil {
		return "", err
	}

	target := filepath.Join(dir, key)
	return target, nil
}

// statPath determines whether the target file exists and returns an fileMutex if so.
func (ls *localStorage) statPath(bucket, key string) (string, os.FileInfo, error) {
	filePath := filepath.Join(ls.BaseDir, bucket, key)
	f, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, errors.Wrapf(ErrKeyNotFound, "bucket(%s) key(%s)", bucket, key)
		}
		return "", nil, err
	}

	return filePath, f, nil
}

func getLockKey(path string, offset int64) string {
	return fmt.Sprintf("%s:%d", path, offset)
}

func checkGetRaw(raw *Raw, fileLength int64) error {
	if fileLength < raw.Offset {
		return errors.Wrapf(ErrRangeNotSatisfiable, "the offset: %d is lager than the file length: %d", raw.Offset, fileLength)
	}

	if raw.Length < 0 {
		return errors.Wrapf(ErrInvalidValue, "the length: %d is not a positive integer", raw.Length)
	}

	if fileLength < (raw.Offset + raw.Length) {
		return errors.Wrapf(ErrRangeNotSatisfiable, "the offset: %d and length: %d is lager than the file length: %d", raw.Offset, raw.Length, fileLength)
	}
	return nil
}

func checkPutRaw(raw *Raw) error {
	if raw.Length < 0 {
		return errors.Wrapf(ErrInvalidValue, "the length: %d should not be a negative integer", raw.Length)
	}
	return nil
}
