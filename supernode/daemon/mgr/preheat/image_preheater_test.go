package preheat

import (
	"fmt"
	"testing"

	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

func TestParseLayers(t *testing.T) {
	task := &mgr.PreheatTask{
		URL: "https://registry.cn-hangzhou.aliyuncs.com/v2/yuhai/pod-counter-controller/manifests/latest",
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
	fmt.Println(len(layers))
}
