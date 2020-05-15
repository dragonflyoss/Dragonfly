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

package bitmap

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	lbm "github.com/openacid/low/bitmap"
)

const (
	// sizeOf64BitsLimit limits the max size of array of bitmap.
	// It is limited by "github.com/openacid/low/bitmap".
	sizeOf64BitsLimit = (math.MaxInt32 >> 6) + 1
)

// BitMap is a struct which provides the Get or Set of bits map.
type BitMap struct {
	lock        sync.RWMutex
	bm          []uint64
	maxBitIndex uint32
}

// NewBitMap generates a BitMap.
func NewBitMap(sizeOf64Bits uint32, allSetBit bool) (*BitMap, error) {
	if sizeOf64Bits > sizeOf64BitsLimit {
		return nil, fmt.Errorf("sizeOf64Bits %d should be range[0, %d]", sizeOf64Bits, sizeOf64BitsLimit)
	}

	bm := make([]uint64, sizeOf64Bits)
	if allSetBit {
		for i := 0; i < int(sizeOf64Bits); i++ {
			bm[i] = math.MaxUint64
		}
	}

	return &BitMap{
		bm:          bm,
		maxBitIndex: (sizeOf64Bits << 6) - 1,
	}, nil
}

// NewBitMapWithNumBits generates a BitMap.
func NewBitMapWithNumBits(numberBits uint32, allSetBit bool) (*BitMap, error) {
	sizeOf64Bits := uint32(numberBits / 64)
	if (numberBits % 64) > 0 {
		sizeOf64Bits++
	}

	bm, err := NewBitMap(sizeOf64Bits, allSetBit)
	if err != nil {
		return nil, err
	}

	bm.maxBitIndex = numberBits - 1
	return bm, nil
}

// RestoreBitMap generate the BitMap by input bytes.
func RestoreBitMap(data []byte) (*BitMap, error) {
	if uint64(len(data)) > (sizeOf64BitsLimit << 3) {
		return nil, fmt.Errorf("sizeOf64Bits %d should be range[0, %d]", uint64(len(data)), sizeOf64BitsLimit<<3)
	}

	bm := DecodeToUintArray(data)

	return &BitMap{
		bm:          bm,
		maxBitIndex: uint32(len(bm)*64 - 1),
	}, nil
}

// Get gets the bits in range [start, end]. if set is true, return the set bits.
// else return the unset bits.
func (b *BitMap) Get(start uint32, end uint32, setBit bool) ([]*BitsRange, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.getWithoutLock(start, end, setBit)
}

// Set sets or cleans the bits in range [start, end]. if setBit is true, set bits. else clean bits.
func (b *BitMap) Set(start uint32, end uint32, setBit bool) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.setWithoutLock(start, end, setBit)
}

func (b *BitMap) getWithoutLock(start uint32, end uint32, setBit bool) ([]*BitsRange, error) {
	if start > end {
		return nil, fmt.Errorf("start %d should not bigger than %d", start, end)
	}

	if end > b.maxBitIndex {
		return nil, fmt.Errorf("end %d should not bigger than %d", end, b.maxBitIndex)
	}

	rs := []*BitsRange{}
	var lastRange *BitsRange

	second64MinIndex := ((start >> 6) + 1) << 6
	first64MaxIndex := end
	if first64MaxIndex >= second64MinIndex {
		first64MaxIndex = second64MinIndex - 1
	}

	last64MinIndex := (end >> 6) << 6

	appendArr, last := b.mergeAndFetchRangeOfUint64(b.bm[start>>6], (start>>6)<<6, setBit, start, first64MaxIndex, lastRange)
	rs = append(rs, appendArr...)
	lastRange = last

	for i := second64MinIndex; i < last64MinIndex; i += 64 {
		appendArr, last := b.mergeAndFetchRangeOfUint64(b.bm[i>>6], (i>>6)<<6, setBit, i, i+63, lastRange)
		rs = append(rs, appendArr...)
		lastRange = last
	}

	// get range of last uint64
	if last64MinIndex >= second64MinIndex {
		appendArr, last := b.mergeAndFetchRangeOfUint64(b.bm[end>>6], (end>>6)<<6, setBit, last64MinIndex, end, lastRange)
		rs = append(rs, appendArr...)
		lastRange = last
	}

	if lastRange != nil {
		rs = append(rs, lastRange)
	}

	return rs, nil
}

// mergeAndFetchRangeOfUint64 will get range of x, and merge prv range if possible.
// return the array of ranges and last range which may merged by next uint64.
func (b *BitMap) mergeAndFetchRangeOfUint64(x uint64, base uint32, setBit bool, limitStart uint32, limitEnd uint32, prv *BitsRange) (appendArr []*BitsRange, last *BitsRange) {
	var (
		start, end           uint32
		startIndex, endIndex uint32
		out                  bool
	)

	appendArr = []*BitsRange{}

	if !setBit {
		x = uint64(^x)
	}

	for {
		if x == 0 || out {
			break
		}

		// start is the value of the trailing zeros of x
		start = uint32(Ctz64(x))
		// remove the trailing zeros of x and counts again
		end = uint32(Ctz64(uint64(^(x >> start)))) + start

		x = (x >> end) << end

		startIndex = start + base
		endIndex = end + base - 1

		if endIndex < limitStart {
			continue
		}

		if startIndex > limitEnd {
			continue
		}

		if startIndex < limitStart {
			startIndex = limitStart
		}

		if endIndex > limitEnd {
			out = true
			endIndex = limitEnd
		}

		if prv != nil {
			if prv.EndIndex+1 == startIndex {
				prv.EndIndex = endIndex
				continue
			}
			appendArr = append(appendArr, prv)
		}

		prv = &BitsRange{
			StartIndex: startIndex,
			EndIndex:   endIndex,
		}
	}

	return appendArr, prv
}

func (b *BitMap) setWithoutLock(start uint32, end uint32, setBit bool) error {
	if start > end {
		return fmt.Errorf("start %d should not bigger than %d", start, end)
	}

	if end > b.maxBitIndex {
		return fmt.Errorf("end %d should not bigger than %d", end, b.maxBitIndex)
	}

	second64MinIndex := ((start >> 6) + 1) << 6
	first64MaxIndex := end
	if first64MaxIndex >= second64MinIndex {
		first64MaxIndex = second64MinIndex - 1
	}

	for i := start; i <= first64MaxIndex; i++ {
		if setBit {
			b.bm[i>>6] = b.bm[i>>6] | lbm.Bit[i&63]
		} else {
			b.bm[i>>6] = b.bm[i>>6] & (^lbm.Bit[i&63])
		}
	}

	last64MinIndex := (end >> 6) << 6
	if last64MinIndex < first64MaxIndex {
		last64MinIndex = first64MaxIndex + 1
	}

	for i := second64MinIndex; i < last64MinIndex; i += 64 {
		if setBit {
			b.bm[i>>6] = math.MaxUint64
		} else {
			b.bm[i>>6] = 0
		}
	}

	for i := last64MinIndex; i <= end; i++ {
		if setBit {
			b.bm[i>>6] = b.bm[i>>6] | lbm.Bit[i&63]
		} else {
			b.bm[i>>6] = b.bm[i>>6] & (^lbm.Bit[i&63])
		}
	}

	return nil
}

func (b *BitMap) Encode() []byte {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return EncodeUintArray(b.bm)
}

// BitsRange shows the range of bitmap.
type BitsRange struct {
	StartIndex uint32
	EndIndex   uint32
}

// EncodeUintArray encodes []uint64 to bytes.
func EncodeUintArray(input []uint64) []byte {
	arrLen := len(input)
	data := make([]byte, arrLen*8)

	bytesIndex := 0
	for i := 0; i < arrLen; i++ {
		binary.LittleEndian.PutUint64(data[bytesIndex:bytesIndex+8], input[i])
		bytesIndex += 8
	}

	return data[:bytesIndex]
}

// DecodeToUintArray decodes input bytes to []uint64.
func DecodeToUintArray(data []byte) []uint64 {
	var (
		bytesIndex int
	)

	arrLen := len(data) / 8
	out := make([]uint64, arrLen)
	for i := 0; i < arrLen; i++ {
		out[i] = binary.LittleEndian.Uint64(data[bytesIndex : bytesIndex+8])
		bytesIndex += 8
	}

	return out
}
