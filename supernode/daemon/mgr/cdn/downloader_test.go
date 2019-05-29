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

package cdn

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	errorType "github.com/dragonflyoss/Dragonfly/common/errors"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type CDNDownloadTestSuite struct {
}

func init() {
	check.Suite(&CDNDownloadTestSuite{})
}

func (s *CDNDownloadTestSuite) TestDownload(c *check.C) {
	cm, _ := NewManager(config.NewConfig(), nil, nil)
	bytes := []byte("hello world")
	bytesLength := int64(len(bytes))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeStr := r.Header.Get("Range")
		if cutil.IsEmptyStr(rangeStr) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, string(bytes[:]))
			return
		}

		rangeStruct, err := cutil.GetRangeSE(rangeStr, bytesLength)
		if err != nil {
			if errorType.IsRangeNotSatisfiable(err) {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusPartialContent)
		fmt.Fprint(w, string(bytes[rangeStruct[0].StartIndex:rangeStruct[0].EndIndex+1]))
	}))
	defer ts.Close()

	var cases = []struct {
		headers        map[string]string
		startPieceNum  int
		httpFileLength int64
		pieceContSize  int32

		errCheck           func(error) bool
		exceptedStatusCode int
		exceptedBody       string
	}{
		{
			headers:            map[string]string{"foo": "foo"},
			startPieceNum:      0,
			httpFileLength:     bytesLength,
			pieceContSize:      2,
			errCheck:           errorType.IsNilError,
			exceptedStatusCode: http.StatusOK,
			exceptedBody:       "hello world",
		},
		{
			headers:            map[string]string{"foo": "foo"},
			startPieceNum:      2,
			httpFileLength:     bytesLength,
			pieceContSize:      3,
			errCheck:           errorType.IsNilError,
			exceptedStatusCode: http.StatusPartialContent,
			exceptedBody:       "world",
		},
	}

	for _, v := range cases {
		resp, err := cm.download(context.TODO(), "", ts.URL, v.headers, v.startPieceNum, v.httpFileLength, v.pieceContSize)
		c.Check(v.errCheck(err), check.Equals, true)

		c.Check(resp.StatusCode, check.Equals, v.exceptedStatusCode)

		result, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		c.Check(string(result), check.Equals, string(v.exceptedBody))
	}
}
