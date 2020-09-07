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
	"fmt"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/digest"
	dferr "github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

const (
	key = ">I$pg-~AS~sP'rqu_`Oh&lz#9]\"=;nE%"
	dfgetPath = "/usr/local/bin/dfget"
)

type PreheatService struct {
	PreheatPath string
	repository *PreheatTaskRepository
}

func NewPreheatService(homeDir string) *PreheatService {
	return &PreheatService{
		repository: NewPreheatTaskRepository(),
		PreheatPath: filepath.Join(homeDir, "repo", "preheat"),
	}
}

// Get detailed preheat task information
func (svc *PreheatService) 	Get(id string) *mgr.PreheatTask {
	if id == "" {
		return nil
	}
	return svc.repository.Get(id)
}

// Get all preheat tasks
func (svc *PreheatService) 	GetAll() []*mgr.PreheatTask {
	return svc.repository.GetAll()
}

// Delete a preheat task.
func (svc *PreheatService) 	Delete(id string) {
	task := svc.repository.Get(id)
	if task != nil && len(task.Children) > 0 {
		for _, childId := range task.Children {
			svc.repository.Delete(childId)
		}
	}
	svc.repository.Delete(id)
}

// update a preheat task
func (svc *PreheatService) 	Update(id string, task *mgr.PreheatTask) bool {
	return svc.repository.Update(id, task)
}

// create a preheat task
func (svc *PreheatService) 	Create(task *mgr.PreheatTask) (string, error) {
	preheater := GetPreheater(strings.ToLower(task.Type))
	if preheater == nil {
		return "", dferr.New(400, task.Type + " isn't supported")
	}
	task.ID = svc.createTaskID(task.URL, task.Filter, task.Identifier, task.Headers)
	task.StartTime = time.Now().UnixNano() / int64(time.Millisecond)
	task.Status = types.PreheatStatusWAITING
	previous, _ := svc.repository.Add(task)
	if previous != nil && previous.FinishTime > 0 {
		return "", dferr.New(http.StatusAlreadyReported, "preheat task already exists, id:" + task.ID)
	}
	preheater.NewWorker(task, svc).Run()
	return task.ID, nil
}

// execute preheat task
func (svc *PreheatService) 	ExecutePreheat(task *mgr.PreheatTask) (progress *PreheatProgress, err error) {
	targetName := uuid.New()
	targetPath := filepath.Join(svc.PreheatPath, targetName)
	cmd := svc.createCommand(task.URL, task.Headers, task.Filter, task.Identifier, targetPath)
	logrus.Infof("ExecutePreheat CMD: %s", cmd.String())
	err = cmd.Start()
	go cmd.Wait()
	if err != nil {
		err = dferr.New(500, err.Error())
		return
	}
	progress = NewPreheatProgress(targetPath, cmd)
	return
}

func (svc *PreheatService) createTaskID(url, filter, identifier string, header map[string]string) string {
	url = netutils.FilterURLParam(url, strings.Split(filter, "&"))
	sign := identifier
	var id string
	if r, ok := header["Range"]; ok {
		id = fmt.Sprintf("%s%s%s%s%s", key, url, sign, r, key)
	} else {
		id = fmt.Sprintf("%s%s%s%s", key, url, sign, key)
	}
	return digest.Sha256(id)
}

func (svc *PreheatService) createCommand(url string, header map[string]string, filter, identifier, tmpTarget string) *exec.Cmd {
	netRate := 50
	rate := fmt.Sprintf("%dM", netRate / 2)

	args := []string{"-u", url, "-o", tmpTarget, "--callsystem", "dragonfly_preheat", "--totallimit", rate, "-s", rate}
	if (header != nil) {
		for k, v := range header {
			args = append(args, []string{"--header", fmt.Sprintf("%s:%s", k, v)}...)
		}
	}
	if filter != "" {
		args = append(args, []string{"-f", filter}...)
	}
	if identifier != "" {
		args = append(args, []string{"-i", identifier}...)
	}
	return exec.Command(dfgetPath, args...)
}
