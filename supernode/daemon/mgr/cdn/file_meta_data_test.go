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
	"os"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/store"

	"github.com/go-check/check"
	"github.com/prashantv/gostub"
)

var taskID = "00c4e7b174af7ed61c414b36ef82810ac0c98142c03e5748c00e1d1113f3c882"

type CDNFileMetaDataTestSuite struct {
	workHome        string
	content         string
	metaDataManager *fileMetaDataManager

	metaDataPathStub      *gostub.Stubs
	md5DataPathStub       *gostub.Stubs
	currentTimeMillisStub *gostub.Stubs
}

func init() {
	check.Suite(&CDNFileMetaDataTestSuite{})
}

func (s *CDNFileMetaDataTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "supernode-cdn-CDNFileMetaDataTestSuite-")
	s.content = "baseDir: " + s.workHome
	fileStore, err := store.NewStore(store.LocalStorageDriver, store.NewLocalStorage, s.content)
	c.Check(err, check.IsNil)
	s.metaDataManager = newFileMetaDataManager(fileStore)

	s.metaDataPathStub = gostub.Stub(&getMetaDataRawFunc, func(taskID string) *store.Raw {
		return &store.Raw{
			Bucket: "download",
			Key:    taskID + ".meta",
		}
	})
	s.md5DataPathStub = gostub.Stub(&getMd5DataRawFunc, func(taskID string) *store.Raw {
		return &store.Raw{
			Bucket: "download",
			Key:    taskID + ".md5",
		}
	})
	s.currentTimeMillisStub = gostub.Stub(&getCurrentTimeMillisFunc, func() int64 {
		return 0
	})
}

func (s *CDNFileMetaDataTestSuite) TearDownSuite(c *check.C) {
	s.metaDataPathStub.Reset()
	s.md5DataPathStub.Reset()
	s.currentTimeMillisStub.Reset()

	if s.workHome != "" {
		if err := os.RemoveAll(s.workHome); err != nil {
			fmt.Printf("remove path: %s error", s.workHome)
		}
	}
}

func (s *CDNFileMetaDataTestSuite) TestWriteReadFileMetaData(c *check.C) {
	ctx := context.TODO()
	task := &types.TaskInfo{
		ID:             taskID,
		TaskURL:        "http://aa.bb.com",
		PieceSize:      4 * 1024,
		HTTPFileLength: 65565,
		Identifier:     "abc",
	}
	expectedFileMetaData := &fileMetaData{
		TaskID:      task.ID,
		URL:         task.TaskURL,
		PieceSize:   task.PieceSize,
		HTTPFileLen: task.HTTPFileLength,
		Identifier:  task.Identifier,
	}

	// write
	result, err := s.metaDataManager.writeFileMetaDataByTask(ctx, task)
	c.Check(err, check.IsNil)
	c.Check(result, check.DeepEquals, expectedFileMetaData)

	// read
	jsonResult, err := s.metaDataManager.readFileMetaData(ctx, task.ID)
	c.Check(err, check.IsNil)
	c.Check(jsonResult, check.DeepEquals, expectedFileMetaData)

	// update updateFileMetaData
	updatedFileMetaData := &fileMetaData{
		LastModified: 1,
		ETag:         "a275d0ff02eb0e006fa365f2f725b010",
	}
	s.metaDataManager.updateLastModifiedAndETag(ctx, task.ID, updatedFileMetaData.LastModified, updatedFileMetaData.ETag)
	expectedUpdatedFileMetaData := &fileMetaData{
		TaskID:       task.ID,
		URL:          task.TaskURL,
		PieceSize:    task.PieceSize,
		HTTPFileLen:  task.HTTPFileLength,
		Identifier:   task.Identifier,
		LastModified: updatedFileMetaData.LastModified,
		ETag:         updatedFileMetaData.ETag,
	}
	jsonResult, err = s.metaDataManager.readFileMetaData(ctx, task.ID)
	c.Check(err, check.IsNil)
	c.Check(jsonResult, check.DeepEquals, expectedUpdatedFileMetaData)
}

func (s *CDNFileMetaDataTestSuite) TestWriteReadPieceMD5s(c *check.C) {
	ctx := context.TODO()
	pieceMD5s := []string{"91fe186ee566659663232dcd18749cce:1502", "11fe186ee566659663232dcd18749cce:1502"}
	fileMD5 := "7a19c32d2c75345debe9031cfa9b649a"

	err := s.metaDataManager.writePieceMD5s(ctx, taskID, fileMD5, pieceMD5s)
	c.Check(err, check.IsNil)

	result, err := s.metaDataManager.readPieceMD5s(ctx, taskID, fileMD5)
	c.Check(err, check.IsNil)
	c.Check(result, check.DeepEquals, pieceMD5s)
}
