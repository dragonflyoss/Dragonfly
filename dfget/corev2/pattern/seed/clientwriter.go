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

package seed

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/corev2/basic"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/clientwriter"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/common"
	"github.com/dragonflyoss/Dragonfly/pkg/protocol"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"

	"github.com/sirupsen/logrus"
)

type ClientWriter struct {
	sync.RWMutex
	wc      io.WriteCloser
	addData bool
	running bool
	closed  bool

	dataQ  queue.Queue
	notify *common.Notify
}

func NewClientWriter() clientwriter.ClientWriter {
	return &ClientWriter{
		dataQ:  queue.NewQueue(0),
		notify: common.NewNotify(),
	}
}

func (cw *ClientWriter) PutData(data protocol.DistributionData) error {
	if cw.isClosed() {
		return errors.New("closed writer")
	}

	if !cw.isRunning() {
		return fmt.Errorf("writer not running")
	}

	dataT := data.Type().String()
	if dataT != SeedDataType && dataT != "EOF" {
		return fmt.Errorf("data type should be seed or eof")
	}

	cw.dataQ.Put(data)
	return nil
}

func (cw *ClientWriter) Run(ctx context.Context, wc io.WriteCloser) (basic.Notify, error) {
	cw.Lock()
	defer cw.Unlock()
	if cw.running {
		return nil, fmt.Errorf("ClientWriter is running")
	}

	cw.wc = wc
	cw.running = true
	go cw.run(ctx)

	return cw.notify, nil
}

func (cw *ClientWriter) run(ctx context.Context) {
	var (
		waitClose bool
	)

	for {
		select {
		case <-ctx.Done():
			// context done.
			cw.finish(context.DeadlineExceeded)
			return
		default:
		}

		data, ok := cw.dataQ.PollTimeout(time.Second * 2)
		if !ok {
			continue
		}

		dd := data.(protocol.DistributionData)
		if dd.Type().String() == "EOF" {
			cw.finish(nil)
			return
		}

		// if waitClose sets, only EOF data could reach here.
		if waitClose {
			continue
		}

		rd, err := dd.Content(ctx)
		if err != nil {
			logrus.Errorf("failed to get content from distribution data: %v", err)
			continue
		}

		_, err = io.Copy(cw.wc, rd)
		if err != nil {
			if err == io.EOF {
				waitClose = true
				continue
			}
			cw.finish(err)
			return
		}
	}
}

func (cw *ClientWriter) finish(err error) {
	success := true
	cErr := cw.wc.Close()
	if cErr != nil {
		logrus.Errorf("failed to close write closer: %v", cErr)
		success = false
	}

	if err != nil {
		success = false
	} else {
		err = cErr
	}
	cw.setClosed()
	cw.notify.Finish(common.NewNotifyResult(success, err, nil))
}

func (cw *ClientWriter) isClosed() bool {
	cw.RLock()
	defer cw.RUnlock()

	return cw.closed
}

func (cw *ClientWriter) setClosed() {
	cw.Lock()
	defer cw.Unlock()

	cw.closed = true
}

func (cw *ClientWriter) isRunning() bool {
	cw.RLock()
	defer cw.RUnlock()

	return cw.running
}
