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

package handler

import (
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

// getArgs returns all the arguments of command-line except the program name.
func getArgs(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("access:%s", r.URL.String())

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain;charset=utf-8")
	for index, value := range os.Args {
		if index > 0 {
			if _, err := w.Write([]byte(value + " ")); err != nil {
				logrus.Errorf("failed to respond information: %v", err)
			}
		}

	}
}
