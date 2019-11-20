// +build gofuzz

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

package cdn

import (
	"bytes"
	"context"
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

func Fuzz(data []byte) int {
	// Don't spam output with parse failures
	logrus.SetOutput(ioutil.Discard)
	r := bytes.NewReader(data)
	sr := newSuperReader()
	_, err := sr.readFile(context.Background(), r, true, true)
	if err != nil {
		return 0
	}
	return 1
}
