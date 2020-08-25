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
package preheat

import (
	"testing"

	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

func TestParseLayers(t *testing.T) {
	task := &mgr.PreheatTask{
		URL: "https://registry.cn-zhangjiakou.aliyuncs.com/v2/acs/alpine/manifests/3.6",
		Headers: map[string]string{},
	}
	worker := &ImageWorker{BaseWorker: newBaseWorker(task, nil, nil)}
	result := IMAGE_MANIFESTS_PATTERN.FindSubmatch([]byte(task.URL))
	if len(result) == 5 {
		worker.protocol = string(result[1])
		worker.domain = string(result[2])
		worker.name = string(result[3])
	}
	layers, err := worker.getLayers(task.URL, task.Headers, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(layers) != 4 {
		t.Fatal("parse layer failed")
	}
}
