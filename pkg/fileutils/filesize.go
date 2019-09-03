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
	"regexp"
	"strconv"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/pkg/errors"
)

// Fsize is a wrapper type which indicates the file size.
type Fsize int64

const (
	B  Fsize = 1
	KB       = 1024 * B
	MB       = 1024 * KB
	GB       = 1024 * MB
)

// fsizeRegex only supports the format G(B)/M(B)/K(B)/B or pure number.
var fsizeRegex = regexp.MustCompile("^([0-9]+)([GMK]B?|B)$")

// MarshalYAML implements the yaml.Marshaler interface.
func (f Fsize) MarshalYAML() (interface{}, error) {
	result := FsizeToString(f)
	return result, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (f *Fsize) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var fsizeStr string
	if err := unmarshal(&fsizeStr); err != nil {
		return err
	}

	fsize, err := StringToFSize(fsizeStr)
	if err != nil {
		return err
	}
	*f = Fsize(fsize)
	return nil
}

// FsizeToString parses a Fsize value into string.
func FsizeToString(fsize Fsize) string {
	var (
		n      = int64(fsize)
		symbol = "B"
		unit   = B
	)
	if n == 0 {
		return "0B"
	}

	switch int64(0) {
	case n % int64(GB):
		symbol = "GB"
		unit = GB
	case n % int64(MB):
		symbol = "MB"
		unit = MB
	case n % int64(KB):
		symbol = "KB"
		unit = KB
	}
	return fmt.Sprintf("%v%v", n/int64(unit), symbol)
}

// StringToFSize parses a string into Fsize.
func StringToFSize(fsize string) (Fsize, error) {
	var n int
	n, err := strconv.Atoi(fsize)
	if err == nil && n >= 0 {
		return Fsize(n), nil
	}
	if n < 0 {
		return 0, errors.Wrapf(errortypes.ErrInvalidValue, "%s is not a negative value fsize", fsize)
	}

	matches := fsizeRegex.FindStringSubmatch(fsize)
	if len(matches) != 3 {
		return 0, errors.Wrapf(errortypes.ErrInvalidValue, "%s and supported format: G(B)/M(B)/K(B)/B or pure number", fsize)
	}
	n, _ = strconv.Atoi(matches[1])
	switch unit := matches[2]; {
	case unit == "G" || unit == "GB":
		n *= int(GB)
	case unit == "M" || unit == "MB":
		n *= int(MB)
	case unit == "K" || unit == "KB":
		n *= int(KB)
	case unit == "B":
		// Value already correct
	default:
		return 0, errors.Wrapf(errortypes.ErrInvalidValue, "%s and supported format: G(B)/M(B)/K(B)/B or pure number", fsize)
	}
	return Fsize(n), nil
}
