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

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
)

func TestPreheatInfoError(t *testing.T) {
	serverErr := "Server error"
	client := &APIClient{
		HTTPCli: newMockClient(errorMockResponse(http.StatusInternalServerError, serverErr)),
	}

	preheatTaskID := "asdfghjkl345678"
	_, err := client.PreheatInfo(context.Background(), preheatTaskID)
	if err == nil {
		t.Fatalf("expected a %s, got no error", serverErr)
	}
	if !strings.Contains(err.Error(), serverErr) {
		t.Fatalf("expected an error contains %s, got %v", serverErr, err)
	}
}

func TestPreheatInfo(t *testing.T) {
	id := "1234567890"
	startTime := strfmt.DateTime(time.Now())
	finishTime := strfmt.DateTime(time.Now().Add(time.Duration(time.Minute)))
	status := types.PreheatStatusSUCCESS

	expectedURL := fmt.Sprintf("/preheats/%s", id)

	httpClient := newMockClient(func(req *http.Request) (*http.Response, error) {
		if !strings.HasPrefix(req.URL.Path, expectedURL) {
			return nil, fmt.Errorf("expected URL '%s', got '%s'", expectedURL, req.URL)
		}
		info := types.PreheatInfo{
			ID:         id,
			StartTime:  startTime,
			FinishTime: finishTime,
			Status:     types.PreheatStatusSUCCESS,
		}
		b, err := json.Marshal(info)
		if err != nil {
			return nil, err
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(b))),
		}, nil
	})

	client := &APIClient{
		HTTPCli: httpClient,
	}

	info, err := client.PreheatInfo(context.Background(), id)
	if err != nil {
		t.Fatalf("expect no error, got %v", err)
	}

	assert.Equal(t, info.ID, id)
	assert.Equal(t, info.StartTime.String(), startTime.String())
	assert.Equal(t, info.FinishTime.String(), finishTime.String())
	assert.Equal(t, info.Status, status)
}
