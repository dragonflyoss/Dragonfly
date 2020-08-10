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
