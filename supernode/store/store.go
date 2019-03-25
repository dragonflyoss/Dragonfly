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
	"strings"
)

// Store is a wrapper of the storage which implements the interface of StorageDriver.
type Store struct {
	// name is a unique identifier, you can also name it ID.
	driverName string
	// config is used to init storage driver.
	config interface{}
	// driver holds a storage which implements the interface of StorageDriver
	driver StorageDriver
}

// NewStore create a new Store instance.
func NewStore(name string, cfg interface{}) (*Store, error) {
	// determine whether the driver has been registered
	initer, ok := driverFactory[name]
	if !ok {
		return nil, fmt.Errorf("unregisterd storage driver : %s", name)
	}

	// init driver with specific config
	driver, err := initer(cfg)
	if err != nil {
		return nil, fmt.Errorf("init storage driver failed %s: %v", name, cfg)
	}

	return &Store{
		driverName: name,
		config:     cfg,
		driver:     driver,
	}, nil
}

// Get the data from the storage driver in io stream.
func (s *Store) Get(ctx context.Context, raw *Raw, writer io.Writer) error {
	if err := isEmptyKey(raw.key); err != nil {
		return err
	}
	return s.driver.Get(ctx, raw, writer)
}

// GetBytes gets the data from the storage driver in bytes.
func (s *Store) GetBytes(ctx context.Context, raw *Raw) ([]byte, error) {
	if err := isEmptyKey(raw.key); err != nil {
		return nil, err
	}
	return s.driver.GetBytes(ctx, raw)
}

// Put puts data into the storage in io stream.
func (s *Store) Put(ctx context.Context, raw *Raw, data io.Reader) error {
	if err := isEmptyKey(raw.key); err != nil {
		return err
	}
	return s.driver.Put(ctx, raw, data)
}

// PutBytes puts data into the storage in bytes.
func (s *Store) PutBytes(ctx context.Context, raw *Raw, data []byte) error {
	if err := isEmptyKey(raw.key); err != nil {
		return err
	}
	return s.driver.PutBytes(ctx, raw, data)
}

// Remove the data from the storage based on raw information.
func (s *Store) Remove(ctx context.Context, raw *Raw) error {
	if err := isEmptyKey(raw.key); err != nil {
		return err
	}
	return s.driver.Remove(ctx, raw)
}

// Stat determine whether the data exists based on raw information.
// If that, and return some info that in the form of struct StorageInfo.
// If not, return the ErrNotFound.
func (s *Store) Stat(ctx context.Context, raw *Raw) (*StorageInfo, error) {
	if err := isEmptyKey(raw.key); err != nil {
		return nil, err
	}
	return s.driver.Stat(ctx, raw)
}

func isEmptyKey(str string) error {
	if strings.TrimSpace(str) == "" {
		return ErrEmptyKey
	}
	return nil
}
