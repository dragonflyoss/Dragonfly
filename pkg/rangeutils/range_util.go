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

package rangeutils

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	separator         = "-"
	invalidPieceIndex = -1
)

// CalculatePieceSize calculates the size of piece
// according to the parameter range.
func CalculatePieceSize(rangeStr string) int64 {
	startIndex, endIndex, err := ParsePieceIndex(rangeStr)
	if err != nil {
		return 0
	}

	return endIndex - startIndex + 1
}

// CalculatePieceNum calculates the number of piece
// according to the parameter range.
func CalculatePieceNum(rangeStr string) int {
	startIndex, endIndex, err := ParsePieceIndex(rangeStr)
	if err != nil {
		return -1
	}

	pieceSize := endIndex - startIndex + 1

	return int(startIndex / pieceSize)
}

// ParsePieceIndex parses the start and end index ​​according to range string.
// rangeStr: "start-end"
func ParsePieceIndex(rangeStr string) (start, end int64, err error) {
	ranges := strings.Split(rangeStr, separator)
	if len(ranges) != 2 {
		return invalidPieceIndex, invalidPieceIndex, fmt.Errorf("range value(%s) is illegal which should be like 0-45535", rangeStr)
	}

	startIndex, err := strconv.ParseInt(ranges[0], 10, 64)
	if err != nil {
		return invalidPieceIndex, invalidPieceIndex, fmt.Errorf("range(%s) start is not a number", rangeStr)
	}
	endIndex, err := strconv.ParseInt(ranges[1], 10, 64)
	if err != nil {
		return invalidPieceIndex, invalidPieceIndex, fmt.Errorf("range(%s) end is not a number", rangeStr)
	}

	if endIndex < startIndex {
		return invalidPieceIndex, invalidPieceIndex, fmt.Errorf("range(%s) start is larger than end", rangeStr)
	}

	return startIndex, endIndex, nil
}

// CalculateBreakRange calculates the start and end of piece
// with the following formula:
//     start = pieceNum * pieceSize
//     end = rangeLength - 1
// The different with the CalculatePieceRange function is that
// the end is calculated by rangeLength which is passed in by the caller itself.
func CalculateBreakRange(startPieceNum, pieceContSize int, rangeLength int64) (string, error) {
	// This method is to resume downloading from break-point,
	// so there is no need to call this function when the startPieceNum equals to 0.
	// It is recommended to check this value before calling this function.
	if startPieceNum <= 0 {
		return "", fmt.Errorf("startPieceNum is illegal for value: %d", startPieceNum)
	}
	if rangeLength <= 0 {
		return "", fmt.Errorf("rangeLength is illegal for value: %d", rangeLength)
	}

	start := int64(startPieceNum) * int64(pieceContSize)
	end := rangeLength - 1
	if start > end {
		return "", fmt.Errorf("start: %d is larger than end: %d", start, end)

	}
	return getRangeString(start, end), nil
}

// CalculatePieceRange calculates the start and end of piece
// with the following formula:
//     start = pieceNum * pieceSize
//     end = start + pieceSize - 1
func CalculatePieceRange(pieceNum int, pieceSize int32) string {
	startIndex := int64(pieceNum) * int64(pieceSize)
	endIndex := startIndex + int64(pieceSize) - 1
	return getRangeString(startIndex, endIndex)
}

func getRangeString(startIndex, endIndex int64) string {
	return fmt.Sprintf("%s%s%s", strconv.FormatInt(startIndex, 10), separator, strconv.FormatInt(endIndex, 10))
}
