// Copyright 1999-2017 Alibaba Group.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/alibaba/Dragonfly/dfdaemon/global"
)

// GetEnv returns the environments of dfdaemon
func GetEnv(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("access:%s", r.URL.String())

	env := make(map[string]interface{})

	env["dfPattern"] = global.CopyDfPattern()

	env["home"] = global.HomeDir

	env["param"] = global.CommandLine

	w.Write([]byte(fmt.Sprintf("%+v", env)))
}
