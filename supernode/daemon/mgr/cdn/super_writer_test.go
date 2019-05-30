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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/store"

	"github.com/go-check/check"
)

type SuperWriterTestSuite struct {
	workHome string
	config   string
	writer   *superWriter
}

func init() {
	check.Suite(&SuperWriterTestSuite{})
}

func (s *SuperWriterTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "supernode-cdn-SuperWriterTestSuite-")
	s.config = "baseDir: " + s.workHome
	fileStore, err := store.NewStore(store.LocalStorageDriver, store.NewLocalStorage, s.config)
	c.Check(err, check.IsNil)
	s.writer = newSuperWriter(fileStore, nil)
}

func (s *SuperWriterTestSuite) TearDownSuite(c *check.C) {
	if s.workHome != "" {
		if err := os.RemoveAll(s.workHome); err != nil {
			fmt.Printf("remove path: %s error", s.workHome)
		}
	}
}

func (s *SuperWriterTestSuite) TestStartWriter(c *check.C) {
	var pieceContSize = int32(10)
	var pieceSize = pieceContSize + config.PieceWrapSize

	testStr := "hello dragonfly"
	var httpFileLen = int64(len(testStr))
	f := strings.NewReader(testStr)

	task := &types.TaskInfo{
		ID:        "5806501cbcc3bb92f0b645918c5a4b15495a63259e3e0363008f97e186509e9e",
		PieceSize: pieceSize,
	}

	pieceCount := (httpFileLen + int64(pieceContSize-1)) / int64(pieceContSize)
	expectedSize := httpFileLen + pieceCount*int64(config.PieceWrapSize)

	downloadMetadata, err := s.writer.startWriter(context.TODO(), nil, f, task, 0, httpFileLen, pieceContSize)
	c.Check(err, check.IsNil)
	c.Check(downloadMetadata.realFileLength, check.Equals, expectedSize)
	checkFileSize(s.writer.cdnStore, task.ID, expectedSize, c)
}

func (s *SuperWriterTestSuite) TestWriteToFile(c *check.C) {
	var pieceContSize = int32(15)
	var pieceSize = pieceContSize + config.PieceWrapSize

	testStr := "hello dragonfly"

	var bb = bytes.NewBufferString(testStr)

	task := &types.TaskInfo{
		ID:        "5816501cbcc3bb92f0b645918c5a4b15495a63259e3e0363008f97e186509e9e",
		PieceSize: pieceSize,
	}

	err := s.writer.writeToFile(context.TODO(), bb, task.ID, 0, pieceContSize, task.PieceSize, nil)
	c.Check(err, check.IsNil)

	checkFileSize(s.writer.cdnStore, task.ID, int64(pieceSize), c)
}

func checkFileSize(cdnStore *store.Store, taskID string, expectedSize int64, c *check.C) {
	storageInfo, err := cdnStore.Stat(context.TODO(), &store.Raw{
		Bucket: config.DownloadHome,
		Key:    getDownloadKey(taskID),
	})
	c.Check(err, check.IsNil)
	c.Check(storageInfo.Size, check.Equals, expectedSize)
}
