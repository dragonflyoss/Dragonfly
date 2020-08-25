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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

func init() {
	RegisterPreheater("image", &ImagePreheat{BasePreheater:new(BasePreheater)})
}

var IMAGE_MANIFESTS_PATTERN, _ = regexp.Compile("^(.*)://(.*)/v2/(.*)/manifests/(.*)")

type ImagePreheat struct {
	*BasePreheater
}

func (p *ImagePreheat) Type() string {
	return "image"
}

/**
 * Create a worker to preheat the task.
 */
func (p *ImagePreheat) NewWorker(task *mgr.PreheatTask, service *PreheatService) IWorker {
	worker := &ImageWorker{BaseWorker: newBaseWorker(task, p, service)}
	worker.worker = worker
	p.addWorker(task.ID, worker)
	result := IMAGE_MANIFESTS_PATTERN.FindSubmatch([]byte(task.URL))
	if len(result) == 5 {
		worker.protocol = string(result[1])
		worker.domain = string(result[2])
		worker.name = string(result[3])
	}
	return worker
}

type ImageWorker struct {
	*BaseWorker
	progress *PreheatProgress
	protocol string
	domain   string
	name     string
}

func (w *ImageWorker) preRun() bool {
	err := w.preheatLayers()
	if err != nil {
		w.failed(err.Error())
		return false
	}
	return true
}

func (w *ImageWorker) query() chan error {
	result := make(chan error, 1)
	go func() {
		time.Sleep(time.Second * 2)
		for w.isRunning() {
			running := len(w.Task.Children)
			for _, child := range w.Task.Children {
				childTask := w.PreheatService.Get(child)
				if childTask == nil {
					continue
				}
				if childTask.FinishTime > 0 {
					running--
				}
				if childTask.Status == types.PreheatStatusFAILED {
					errMsg := childTask.URL + " " + childTask.ErrorMsg
					w.Preheater.Cancel(w.Task.ID)
					result <- errors.New(errMsg)
					logrus.Errorf("PreheatImage Task [%s] prehead failed for %s", w.Task.ID, errMsg)
					return
				}
			}

			if running <= 0 {
				w.succeed()
				w.Preheater.Cancel(w.Task.ID)
				result <- nil
				return
			}
			time.Sleep(time.Second * 10)
		}
	}()
	return result
}

func (w *ImageWorker) preheatLayers() (err error) {
	task := w.Task
	layers, err := w.getLayers(task.URL, task.Headers, true)
	if err != nil {
		return
	}

	children := make([]string, 0)
	for _, layer := range layers {
		logrus.Debugf("preheat layer:%s parentId:%s", layer.url, task.ID)
		if layer.url == "" {
			continue
		}
		child := new(mgr.PreheatTask)
		child.ParentID = task.ID
		child.URL = layer.url
		child.Type = "file"
		child.Headers = layer.headers
		child.ID, err = w.PreheatService.Create(child)
		if err != nil {
			return
		}
		if child.ID != "" {
			children = append(children, child.ID)
		}
	}

	w.Task.Children = children
	w.Task.Status = types.PreheatStatusRUNNING
	w.PreheatService.Update(w.Task.ID, w.Task)
	return
}

func (w *ImageWorker) getLayers(url string, header map[string]string, retryIfUnAuth bool) (layers []*Layer, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	for k, v := range header {
		req.Header.Add(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode/100 != 2 {
		if retryIfUnAuth {
			token := w.getAuthToken(resp.Header)
			if token != "" {
				authHeader := map[string]string{"Authorization": "Bearer " + token}
				return w.getLayers(url, authHeader, false)
			}
		}
		err = fmt.Errorf("%s %s", resp.Status, string(body))
		return
	}

	layers = w.parseLayers(body, header)

	return
}

func (w *ImageWorker) getAuthToken(header http.Header) (token string) {
	if len(header) == 0 {
		return
	}
	var values []string
	for k, v := range header {
		if strings.ToLower(k) == "www-authenticate"  {
			values = v
		}
	}
	if values == nil {
		return
	}
	authUrl := w.authUrl(values)
	if len(authUrl) == 0 {
		return
	}
	resp, err := http.Get(authUrl)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	if result["token"] != nil {
		token = fmt.Sprintf("%v", result["token"])
	}
	return
}

func (w *ImageWorker) authUrl(wwwAuth []string) string {
	// Bearer realm="<auth-service-url>",service="<service>",scope="repository:<name>:pull"
	if len(wwwAuth) == 0 {
		return ""
	}
	polished := make([]string, 0)
	for _, it := range wwwAuth {
		polished = append(polished, strings.ReplaceAll(it, "\"", ""))
	}
	fileds := strings.Split(polished[0], ",")
	host := strings.Split(fileds[0], "=")[1]
	query := strings.Join(fileds[1:], "&")
	return fmt.Sprintf("%s?%s", host, query)
}

func (w *ImageWorker) parseLayers(body []byte, header map[string]string) (layers []*Layer) {
	var meta = make(map[string]interface{})
	json.Unmarshal(body, &meta)
	schemaVersion := fmt.Sprintf("%v", meta["schemaVersion"])
	var layerDigest []string
	if schemaVersion == "1" {
		layerDigest = w.parseLayerDigest(meta, "fsLayers", "blobSum")
	} else {
		mediaType := fmt.Sprintf("%s", meta["mediaType"])
		switch mediaType {
		case "application/vnd.docker.distribution.manifest.list.v2+json", "application/vnd.oci.image.index.v1+json":
			manifestDigest := w.parseLayerDigest(meta, "manifests", "digest")
			for _, digest := range manifestDigest {
				list, _ := w.getLayers(w.manifestUrl(digest), header, false)
				layers = append(layers, list...)
			}
			return
		default:
			layerDigest = w.parseLayerDigest(meta, "layers", "digest")
		}
	}

	for _, digest := range layerDigest {
		layers = append(layers, &Layer{
			digest:  digest,
			url:     w.layerUrl(digest),
			headers: header,
		})
	}

	return
}

func (w *ImageWorker) layerUrl(digest string) string {
	return fmt.Sprintf("%s://%s/v2/%s/blobs/%s", w.protocol, w.domain, w.name, digest)
}

func (w *ImageWorker) manifestUrl(digest string) string {
	return fmt.Sprintf("%s://%s/v2/%s/manifests/%s", w.protocol, w.domain, w.name, digest)
}

func (w *ImageWorker) parseLayerDigest(meta map[string]interface{}, layerKey string, digestKey string) (layers []string) {
	data := meta[layerKey]
	if data == nil {
		return
	}
	array, _ := data.([]interface{})
	if array == nil {
		return
	}
	for _, it := range array {
		layer, _ := it.(map[string]interface{})
		if layer != nil {
			layers = append(layers, fmt.Sprintf("%v", layer[digestKey]))
		}
	}
	return layers
}

type Layer struct {
	digest  string
	url     string
	headers map[string]string
}
