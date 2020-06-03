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

package seed

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"

	"github.com/dragonflyoss/Dragonfly/pkg/bitmap"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	lbm "github.com/openacid/low/bitmap"
)

// cacheBuffer interface caches the seed file
type cacheBuffer interface {
	io.WriterAt
	// write close
	io.Closer
	Sync() error

	// ReadStream prepares io.ReadCloser from cacheBuffer.
	ReadStream(off int64, size int64) (io.ReadCloser, error)

	// remove the cache
	Remove() error

	// the cache full size
	FullSize() int64
}

func newFileCacheBuffer(path string, fullSize int64, trunc bool, memoryCache bool, blockOrder uint32) (cb cacheBuffer, err error) {
	var (
		fw *os.File
	)

	_, err = os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	if trunc {
		fw, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	} else {
		fw, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	}

	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			fw.Close()
		}
	}()

	fcb := &fileCacheBuffer{path: path, fw: fw, fullSize: fullSize, memoryCache: memoryCache}
	if memoryCache {
		fcb.blockOrder = blockOrder
		fcb.blockSize = 1 << blockOrder
		blocks := fullSize / int64(fcb.blockSize)
		if (fullSize % int64(fcb.blockSize)) > 0 {
			blocks++
		}

		fcb.blockMeta, err = bitmap.NewBitMapWithNumBits(uint32(blocks), false)
		if err != nil {
			return nil, err
		}

		fcb.memCacheMap = make(map[int64]*bytes.Buffer)
		fcb.maxBlockIndex = uint32(blocks - 1)
	}

	return fcb, nil
}

type fileCacheBuffer struct {
	// the lock protects fields of 'remove', 'memCacheMap'
	sync.RWMutex

	path     string
	fw       *os.File
	remove   bool
	fullSize int64

	// memory cache mode, in the mode, it will store cache in temperate memory.
	// if sync is called, the temperate memory will sync to local fs.
	// in memory cache mode, WriteAt should transfer a block buffer.
	memoryCache   bool
	blockMeta     *bitmap.BitMap
	blockSize     int32
	blockOrder    uint32
	maxBlockIndex uint32

	// memCacheMap caches the buffer, and the buffer should not be changed if added to the map.
	// the key is bytes start index of buffer cache in full cache.
	memCacheMap map[int64]*bytes.Buffer
}

// if in memory mode, the write data buffer will be reused, so don't change the buffer.
func (fcb *fileCacheBuffer) WriteAt(p []byte, off int64) (n int, err error) {
	err = fcb.checkWriteAtValid(off, int64(len(p)))
	if err != nil {
		return
	}

	if fcb.memoryCache {
		fcb.storeMemoryCache(p, off)
		return len(p), nil
	}

	return fcb.fw.WriteAt(p, off)
}

// Close closes the file writer.
func (fcb *fileCacheBuffer) Close() error {
	if err := fcb.Sync(); err != nil {
		return err
	}
	return fcb.fw.Close()
}

func (fcb *fileCacheBuffer) Sync() error {
	fcb.Lock()
	remove := fcb.remove
	fcb.Unlock()

	if remove {
		return io.ErrClosedPipe
	}

	if fcb.memoryCache {
		if err := fcb.syncMemoryCache(); err != nil {
			return err
		}
	}

	return fcb.fw.Sync()
}

func (fcb *fileCacheBuffer) ReadStream(off int64, size int64) (io.ReadCloser, error) {
	off, size, err := fcb.checkReadStreamParam(off, size)
	if err != nil {
		return nil, err
	}

	return fcb.openReadCloser(off, size)
}

func (fcb *fileCacheBuffer) Remove() error {
	fcb.Lock()
	defer fcb.Unlock()

	if fcb.remove {
		return nil
	}

	fcb.remove = true
	return os.Remove(fcb.path)
}

func (fcb *fileCacheBuffer) FullSize() int64 {
	fcb.RLock()
	defer fcb.RUnlock()

	return fcb.fullSize
}

func (fcb *fileCacheBuffer) checkReadStreamParam(off int64, size int64) (int64, int64, error) {
	fcb.RLock()
	defer fcb.RUnlock()

	if fcb.remove {
		return 0, 0, io.ErrClosedPipe
	}

	if off < 0 {
		off = 0
	}

	// Note: if file size if zero, it should be specially handled in the upper caller.
	// In current progress, if size <= 0, it means to read to the end of file.
	// if size <= 0, set range to [off, fullSize-1]
	if size <= 0 {
		size = fcb.fullSize - off
	}

	if off+size > fcb.fullSize {
		return 0, 0, errortypes.NewHTTPError(http.StatusRequestedRangeNotSatisfiable, "out of range")
	}

	return off, size, nil
}

func (fcb *fileCacheBuffer) storeMemoryCache(p []byte, off int64) {
	fcb.Lock()
	defer fcb.Unlock()

	if _, ok := fcb.memCacheMap[off]; ok {
		return
	}

	buf := bytes.NewBuffer(p)
	fcb.memCacheMap[off] = buf

	startBlockIndex := uint32(off >> fcb.blockOrder)
	// set bits in bit map
	fcb.blockMeta.Set(startBlockIndex, startBlockIndex, true)
}

// syncMemoryCache flushes the memory cache to local file.
func (fcb *fileCacheBuffer) syncMemoryCache() error {
	var (
		err error
	)

	var arr []*struct {
		off int64
		buf *bytes.Buffer
	}

	fcb.RLock()
	for off, buf := range fcb.memCacheMap {
		arr = append(arr, &struct {
			off int64
			buf *bytes.Buffer
		}{off: off, buf: buf})
	}
	fcb.RUnlock()

	sort.Slice(arr, func(i, j int) bool {
		return arr[i].off < arr[j].off
	})

	for i := 0; i < len(arr); i++ {
		err = fcb.syncBlockToFile(arr[i].buf.Bytes(), arr[i].off)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fcb *fileCacheBuffer) syncBlockToFile(p []byte, off int64) error {
	n, err := fcb.fw.WriteAt(p, off)
	if err != nil {
		return err
	}

	if n < len(p) {
		return io.ErrShortWrite
	}

	fcb.Lock()
	defer fcb.Unlock()

	blockIndex := uint32(off >> fcb.blockOrder)
	delete(fcb.memCacheMap, off)
	fcb.blockMeta.Set(blockIndex, blockIndex, false)
	return nil
}

func (fcb *fileCacheBuffer) openReadCloser(off int64, size int64) (io.ReadCloser, error) {
	if fcb.memoryCache {
		return fcb.openReadCloserInMemoryCacheMode(off, size)
	}

	fr, err := os.Open(fcb.path)
	if err != nil {
		return nil, err
	}

	return newLimitReadCloser(fr, off, size)
}

// if in memory cache mode, the reader is multi reader in which some data in memory and others in file.
func (fcb *fileCacheBuffer) openReadCloserInMemoryCacheMode(off, size int64) (io.ReadCloser, error) {
	var (
		rds   []io.Reader
		useFr bool
	)

	fr, err := os.Open(fcb.path)
	if err != nil {
		return nil, err
	}

	fcb.RLock()
	defer fcb.RUnlock()

	if len(fcb.memCacheMap) == 0 {
		return newLimitReadCloser(fr, off, size)
	}

	currentOff := off
	var currentBlock int32
	var currentBlockStartBytes, currentBlockEndBytes, useBlockBytes, currentBlockOff, currentBlockOffEnd int64
	for {
		if size <= 0 {
			break
		}

		currentBlock = int32(currentOff >> fcb.blockOrder)
		currentBlockStartBytes = int64(currentBlock) << fcb.blockOrder
		currentBlockEndBytes = currentBlockStartBytes + int64(fcb.blockSize) - 1
		if currentBlockEndBytes >= fcb.fullSize {
			currentBlockEndBytes = fcb.fullSize - 1
		}

		useBlockBytes = currentBlockEndBytes - currentOff + 1
		if useBlockBytes > size {
			useBlockBytes = size
		}

		currentBlockOff = currentOff - currentBlockStartBytes
		currentBlockOffEnd = currentBlockOff + useBlockBytes - 1
		buf, ok := fcb.memCacheMap[currentBlockStartBytes]
		if ok {
			// currentBlock in memory
			b := buf.Bytes()
			rd := bytes.NewReader(b[currentBlockOff : currentBlockOffEnd+1])
			rds = append(rds, rd)
		} else {
			// else currentBlock in file
			rd := io.NewSectionReader(fr, currentOff, useBlockBytes)
			rds = append(rds, rd)
			useFr = true
		}

		size -= useBlockBytes
		currentOff += useBlockBytes
	}

	if !useFr {
		fr.Close()
		fr = nil
	}

	return newMultiReadCloser(rds, fr), nil
}

func (fcb *fileCacheBuffer) checkWriteAtValid(off, size int64) error {
	if !fcb.memoryCache {
		return nil
	}

	if uint64(off)&(lbm.Mask[fcb.blockOrder]) > 0 {
		return fmt.Errorf("memory cache mode, off %d should be align with blockSize %d", off, fcb.blockSize)
	}

	maxIndex := off + size - 1

	if maxIndex >= fcb.fullSize {
		return fmt.Errorf("memory cache mode, max write index %d should be smaller than max block index %d", maxIndex, fcb.fullSize)
	}

	// if last block, the size may smaller than block size
	if uint32(off>>fcb.blockOrder) == fcb.maxBlockIndex {
		return nil
	}

	if size != int64(fcb.blockSize) {
		return fmt.Errorf("memory cache mode, len of bytes %d should be equal to block size %d", size, fcb.blockSize)
	}

	return nil
}

// fileReadCloser provides a selection readCloser of file.
type fileReadCloser struct {
	sr *io.SectionReader
	fr *os.File
}

func newLimitReadCloser(fr *os.File, off int64, size int64) (io.ReadCloser, error) {
	sr := io.NewSectionReader(fr, off, size)
	return &fileReadCloser{
		sr: sr,
		fr: fr,
	}, nil
}

func (lr *fileReadCloser) Read(p []byte) (n int, err error) {
	return lr.sr.Read(p)
}

func (lr *fileReadCloser) Close() error {
	return lr.fr.Close()
}

// multiReadCloser provides multi ReadCloser.
type multiReadCloser struct {
	rds    []io.Reader
	realRd io.Reader
	fr     *os.File
}

func newMultiReadCloser(rds []io.Reader, fr *os.File) *multiReadCloser {
	return &multiReadCloser{
		rds:    rds,
		realRd: io.MultiReader(rds...),
		fr:     fr,
	}
}

func (mr *multiReadCloser) Read(p []byte) (n int, err error) {
	return mr.realRd.Read(p)
}

func (mr *multiReadCloser) Close() error {
	if mr.fr != nil {
		return mr.fr.Close()
	}

	return nil
}
