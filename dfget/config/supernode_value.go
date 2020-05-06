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

package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dragonflyoss/Dragonfly/pkg/algorithm"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/pkg/errors"
)

const weightSeparator = '='

type SupernodesValue struct {
	Nodes *[]*NodeWeight
}

type NodeWeight struct {
	Node   string
	Weight int
}

func NewSupernodesValue(p *[]*NodeWeight, val []*NodeWeight) *SupernodesValue {
	ssv := new(SupernodesValue)
	ssv.Nodes = p
	*ssv.Nodes = val
	return ssv
}

// GetDefaultSupernodesValue returns the default value of supernodes.
// default: ["127.0.0.1:8002=1"]
func GetDefaultSupernodesValue() []*NodeWeight {
	var result = make([]*NodeWeight, 0)
	result = append(result, &NodeWeight{
		Node:   fmt.Sprintf("%s:%d", DefaultSupernodeIP, DefaultSupernodePort),
		Weight: DefaultSupernodeWeight,
	})
	return result
}

// String implements the pflag.Value interface.
func (sv *SupernodesValue) String() string {
	var result []string
	for _, v := range *sv.Nodes {
		result = append(result, v.string())
	}
	return strings.Join(result, ",")
}

// Set implements the pflag.Value interface.
func (sv *SupernodesValue) Set(value string) error {
	nodes, err := ParseNodesString(value)
	if err != nil {
		return err
	}

	*sv.Nodes = nodes
	return nil
}

// Type implements the pflag.Value interface.
func (sv *SupernodesValue) Type() string {
	return "supernodes"
}

// MarshalYAML implements the yaml.Marshaler interface.
func (nw *NodeWeight) MarshalYAML() (interface{}, error) {
	return nw.string(), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (nw *NodeWeight) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}

	nodeWeight, err := string2NodeWeight(value)
	if err != nil {
		return err
	}

	*nw = *nodeWeight
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (nw *NodeWeight) MarshalJSON() ([]byte, error) {
	return json.Marshal(nw.string())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (nw *NodeWeight) UnmarshalJSON(b []byte) error {
	str, _ := strconv.Unquote(string(b))
	nodeWeight, err := string2NodeWeight(str)
	if err != nil {
		return err
	}

	*nw = *nodeWeight
	return nil
}

func (nw *NodeWeight) string() string {
	return fmt.Sprintf("%s%c%d", nw.Node, weightSeparator, nw.Weight)
}

// ParseNodesString parses the value in string type to []*NodeWeight.
func ParseNodesString(value string) ([]*NodeWeight, error) {
	return ParseNodesSlice(strings.Split(value, ","))
}

// ParseNodesSlice parses the value in string slice type to []*NodeWeight.
func ParseNodesSlice(value []string) ([]*NodeWeight, error) {
	nodeWeightSlice := make([]*NodeWeight, 0)
	weightKey := make([]int, 0)

	// split node and weight
	for _, v := range value {
		nodeWeight, err := string2NodeWeight(v)
		if err != nil {
			return nil, errors.Wrapf(errortypes.ErrInvalidValue, "node: %s %v", v, err)
		}

		weightKey = append(weightKey, nodeWeight.Weight)
		nodeWeightSlice = append(nodeWeightSlice, nodeWeight)
	}

	var result []*NodeWeight
	// get the greatest common divisor of the weight slice and
	// divide all weights by the greatest common divisor.
	gcdNumber := algorithm.GCDSlice(weightKey)
	for _, v := range nodeWeightSlice {
		result = append(result, &NodeWeight{
			Node:   v.Node,
			Weight: v.Weight / gcdNumber,
		})
	}

	return result, nil
}

// NodeWeightSlice2StringSlice parses nodeWeight slice to string slice.
// It takes the NodeWeight.Node as the value and every value will be appended the corresponding NodeWeight.Weight times.
func NodeWeightSlice2StringSlice(supernodes []*NodeWeight) []string {
	var nodes []string
	for _, v := range supernodes {
		for i := 0; i < v.Weight; i++ {
			nodes = append(nodes, v.Node)
		}
	}
	return nodes
}

func string2NodeWeight(value string) (*NodeWeight, error) {
	node, weight, err := splitNodeAndWeight(value)
	if err != nil {
		return nil, err
	}

	node, err = handleDefaultPort(node)
	if err != nil {
		return nil, err
	}

	return &NodeWeight{
		Node:   node,
		Weight: weight,
	}, nil
}

// splitNodeAndWeight returns the node address and weight which parsed by the given value.
// If no weight specified, the DefaultSupernodeWeight will be returned as the weight value.
func splitNodeAndWeight(value string) (string, int, error) {
	result := strings.Split(value, string(weightSeparator))
	splitLength := len(result)

	switch splitLength {
	case 1:
		return result[0], DefaultSupernodeWeight, nil
	case 2:
		v, err := strconv.Atoi(result[1])
		if err != nil {
			return "", 0, err
		}
		return result[0], v, nil
	default:
		return "", 0, errortypes.ErrInvalidValue
	}
}

func handleDefaultPort(node string) (string, error) {
	result := strings.Split(node, ":")
	splitLength := len(result)

	if splitLength == 2 {
		if result[0] == "" || result[1] == "" {
			return "", errortypes.ErrInvalidValue
		}
		return node, nil
	}

	if splitLength == 1 {
		return fmt.Sprintf("%s:%d", node, DefaultSupernodePort), nil
	}

	return "", errortypes.ErrInvalidValue
}
