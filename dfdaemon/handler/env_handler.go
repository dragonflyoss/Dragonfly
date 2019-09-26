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
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// getEnv returns the environments of dfdaemon.
func getEnv(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("access:%s", r.URL.String())
	if err := json.NewEncoder(w).Encode(ensureStringKey(viper.AllSettings())); err != nil {
		logrus.Errorf("failed to encode env json: %v", err)
	}
}

// ensureStringKey recursively ensures all maps in the given interface are string,
// to make the result marshalable by json. This is meant to be used with viper
// settings, so only maps and slices are handled.
func ensureStringKey(obj interface{}) interface{} {
	rt, rv := reflect.TypeOf(obj), reflect.ValueOf(obj)
	switch rt.Kind() {
	case reflect.Map:
		res := make(map[string]interface{})
		for _, k := range rv.MapKeys() {
			res[fmt.Sprintf("%v", k.Interface())] = ensureStringKey(rv.MapIndex(k).Interface())
		}
		return res
	case reflect.Slice:
		res := make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			res[i] = ensureStringKey(rv.Index(i).Interface())
		}
		return res
	}
	return obj
}
