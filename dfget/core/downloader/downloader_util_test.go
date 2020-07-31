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

package downloader

import (
	"context"
	"strings"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/mock"
	"github.com/dragonflyoss/Dragonfly/dfget/core/regist"

	"github.com/go-check/check"
	"github.com/golang/mock/gomock"
)

type DownloaderUtilTestSuite struct {
}

func init() {
	check.Suite(&DownloaderUtilTestSuite{})
}

func (s *DownloaderUtilTestSuite) TestCalculateRoutineCount(c *check.C) {
	// file with length of 100MB
	fileLength := int64(101 * 1024 * 1024)
	// piece with size of 4MB
	pieceSize := int32(4 * 1024 * 1024)

	// test for corner case
	routine := calculateRoutineCount(fileLength, -1)
	c.Check(config.StreamWriterRoutineLimit, check.Equals, routine)

	routine = calculateRoutineCount(-1, pieceSize)
	c.Check(config.StreamWriterRoutineLimit, check.Equals, routine)

	routine = calculateRoutineCount(0, pieceSize)
	c.Check(1, check.Equals, routine)

	// test for normal case
	routine = calculateRoutineCount(fileLength, pieceSize)
	c.Check(config.StreamWriterRoutineLimit, check.Equals, routine)

	fileLength = int64(7 * 1024 * 1024)
	routine = calculateRoutineCount(fileLength, pieceSize)
	c.Check(2, check.Equals, routine)
}

func (s *DownloaderUtilTestSuite) TestStartWriter(c *check.C) {
	// initialization
	testController := gomock.NewController(c)
	defer testController.Finish()
	mockUploaderAPI := mock.NewMockUploaderAPI(testController)

	task := StreamDownloadTimeoutTask{
		Config:      newConfig(),
		UploaderAPI: mockUploaderAPI,
		Result:      newRegisterResult(),
	}
	// set the behavior of the mocked object
	mockUploaderAPI.EXPECT().DeliverPieceToUploader("127.0.0.1", 50,
		gomock.Any()).Return(nil).Times(3)

	reader := strings.NewReader("abcde")
	err := task.startWriter(context.TODO(), reader)

	c.Check(err, check.IsNil)
}

// ----------------------------------------------------------------------------
// helper functions

func newConfig() *config.Config {
	return &config.Config{
		RV: config.RuntimeVariable{
			LocalIP:  "127.0.0.1",
			PeerPort: 50,
		},
	}
}

func newRegisterResult() *regist.RegisterResult {
	return &regist.RegisterResult{
		TaskID:     "100",
		FileLength: 5,
		PieceSize:  2 + config.PieceMetaSize,
	}
}
