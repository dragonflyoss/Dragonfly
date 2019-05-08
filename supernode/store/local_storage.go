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
	"path"
	"sync"
	"sync/atomic"

	"gopkg.in/yaml.v2"

	"github.com/dragonflyoss/Dragonfly/common/util"
)

// LocalStorageDriver is a const of local storage driver.
const LocalStorageDriver = "local"

var fileMutexLocker sync.Map

func init() {
	Register(LocalStorageDriver, NewLocalStorage)
}

type fileMutex struct {
	count int32
	sync.RWMutex
}

func lock(key string, ro bool) {
	v, _ := fileMutexLocker.LoadOrStore(key, &fileMutex{})
	f := v.(*fileMutex)

	if ro {
		f.RLock()
	} else {
		f.Lock()
	}

	atomic.AddInt32(&f.count, 1)
}

func releaseLock(key string, ro bool) {
	v, ok := fileMutexLocker.Load(key)
	if !ok {
		// return fmt.Errorf("panic error")
	}
	f := v.(*fileMutex)

	atomic.AddInt32(&f.count, -1)

	if ro {
		f.RUnlock()
	} else {
		f.Unlock()
	}

	if f.count < 1 {
		fileMutexLocker.Delete(key)
	}
}

// localStorage is one of the implementations of StorageDriver by locally.
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
	if !path.IsAbs(cfg.BaseDir) {
		return nil, fmt.Errorf("not absolute path: %s", cfg.BaseDir)
	}
	if err := util.CreateDirectory(cfg.BaseDir); err != nil {
		return nil, err
	}

	return &localStorage{
		BaseDir: cfg.BaseDir,
	}, nil
}

// Get the content of key from storage and return in io stream.
func (ls *localStorage) Get(ctx context.Context, raw *Raw, writer io.Writer) error {
	path, _, err := ls.statPath(raw.Key)
	if err != nil {
		return err
	}

	lock(getLockKey(path, raw.Offset), true)
	defer releaseLock(getLockKey(path, raw.Offset), true)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Seek(raw.Offset, 0)
	if raw.Length <= 0 {
		_, err = io.Copy(writer, f)
	} else {
		_, err = io.CopyN(writer, f, raw.Length)
	}

	if err != nil {
		return err
	}
	return nil
}

// GetBytes gets the content of key from storage and return in bytes.
func (ls *localStorage) GetBytes(ctx context.Context, raw *Raw) (data []byte, err error) {
	path, _, err := ls.statPath(raw.Key)
	if err != nil {
		return nil, err
	}

	lock(getLockKey(path, raw.Offset), true)
	defer releaseLock(getLockKey(path, raw.Offset), true)

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
	path, err := ls.preparePath(raw.Key)
	if err != nil {
		return err
	}

	lock(getLockKey(path, raw.Offset), false)
	defer releaseLock(getLockKey(path, raw.Offset), false)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_SYNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Seek(raw.Offset, 0)
	if _, err = io.Copy(f, data); err != nil {
		return err
	}

	return nil
}

// PutBytes puts the content of key from storage with bytes.
func (ls *localStorage) PutBytes(ctx context.Context, raw *Raw, data []byte) error {
	path, err := ls.preparePath(raw.Key)
	if err != nil {
		return err
	}

	lock(getLockKey(path, raw.Offset), false)
	defer releaseLock(getLockKey(path, raw.Offset), false)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_SYNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Seek(raw.Offset, 0)
	if _, err := f.Write(data); err != nil {
		return err
	}

	return nil
}

// Stat determine whether the file exists.
func (ls *localStorage) Stat(ctx context.Context, raw *Raw) (*StorageInfo, error) {
	path, fileInfo, err := ls.statPath(raw.Key)
	if err != nil {
		return nil, err
	}

	sys, ok := util.GetSys(fileInfo)
	if !ok {
		return nil, fmt.Errorf("get create time error")
	}
	return &StorageInfo{
		Path:       path,
		Size:       fileInfo.Size(),
		CreateTime: util.Ctime(sys),
		ModTime:    fileInfo.ModTime(),
	}, nil
}

// Remove deletes a file or dir.
func (ls *localStorage) Remove(ctx context.Context, raw *Raw) error {
	path, _, err := ls.statPath(raw.Key)
	if err != nil {
		return err
	}

	lock(getLockKey(path, raw.Offset), false)
	defer releaseLock(getLockKey(path, raw.Offset), false)

	return os.RemoveAll(path)
}

// helper function

// preparePath gets the target path and creates the upper directory if it does not exist.
func (ls *localStorage) preparePath(key string) (string, error) {
	dir := path.Join(ls.BaseDir, getPrefix(key))

	if err := util.CreateDirectory(dir); err != nil {
		return "", err
	}

	target := path.Join(dir, key)
	return target, nil
}

// statPath determines whether the target file exists and returns an fileMutex if so.
func (ls *localStorage) statPath(key string) (string, os.FileInfo, error) {
	filePath := path.Join(ls.BaseDir, getPrefix(key), key)
	f, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, ErrNotFound
		}
		return "", nil, err
	}

	return filePath, f, nil
}

func getLockKey(path string, offset int64) string {
	return fmt.Sprintf("%s%d", path, offset)
}

func getPrefix(str string) string {
	if len(str) > 3 {
		return string([]byte(str)[:3])
	}
	return str
}
