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

package app

import (
	"github.com/dragonflyoss/Dragonfly/supernode/config"
)

// NewOptions returns a default options instant.
func NewOptions() *Options {
	return &Options{
		BaseProperties: config.NewBaseProperties(),
	}
}

// Options store the values that comes from command line parameters.
type Options struct {
	*config.BaseProperties
}
