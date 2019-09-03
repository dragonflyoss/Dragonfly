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

package rate

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
)

// Rate wraps int64. It is used to parse the custom rate format
// from YAML and JSON.
// This type should not propagate beyond the scope of input/output processing.
type Rate int64

const (
	B  Rate = 1
	KB      = 1024 * B
	MB      = 1024 * KB
	GB      = 1024 * MB
)

// Set implements pflag/flag.Value
func (d *Rate) Set(s string) error {
	var err error
	*d, err = ParseRate(s)
	return err
}

// Type implements pflag.Value
func (d *Rate) Type() string {
	return "rate"
}

var rateRE = regexp.MustCompile("^([0-9]+)(MB?|m|KB?|k|GB?|g|B)$")

// ParseRate parses a string into a int64.
func ParseRate(rateStr string) (Rate, error) {
	var n int
	n, err := strconv.Atoi(rateStr)
	if err == nil && n >= 0 {
		return Rate(n), nil
	}

	if n < 0 {
		return 0, fmt.Errorf("not a valid rate string: %d, only non-negative values are supported", n)
	}

	matches := rateRE.FindStringSubmatch(rateStr)
	if len(matches) != 3 {
		return 0, fmt.Errorf("not a valid rate string: %q, supported format: G(B)/g/M(B)/m/K(B)/k/B or pure number", rateStr)
	}
	n, _ = strconv.Atoi(matches[1])
	switch unit := matches[2]; {
	case unit == "g" || unit == "G" || unit == "GB":
		n *= int(GB)
	case unit == "m" || unit == "M" || unit == "MB":
		n *= int(MB)
	case unit == "k" || unit == "K" || unit == "KB":
		n *= int(KB)
	case unit == "B":
		// Value already correct
	default:
		return 0, fmt.Errorf("invalid unit in rate string: %q, supported format: G(B)/g/M(B)/m/K(B)/k/B or pure number", unit)
	}
	return Rate(n), nil
}

// String returns the rate with an uppercase unit.
func (d Rate) String() string {
	var (
		n      = int64(d)
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

// MarshalYAML implements the yaml.Marshaler interface.
func (d Rate) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (d *Rate) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	rate, err := ParseRate(s)
	if err != nil {
		return err
	}
	*d = rate
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (d Rate) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *Rate) UnmarshalJSON(b []byte) error {
	str, _ := strconv.Unquote(string(b))
	rate, err := ParseRate(str)
	if err != nil {
		return err
	}
	*d = rate
	return nil
}
