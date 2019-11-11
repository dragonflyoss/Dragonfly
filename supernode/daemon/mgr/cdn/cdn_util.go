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
	"fmt"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/timeutils"
)

var getCurrentTimeMillisFunc = timeutils.GetCurrentTimeMillis

// getContentLengthByHeader calculates the piece content length by piece header.
func getContentLengthByHeader(pieceHeader uint32) int32 {
	return int32(pieceHeader & 0xffffff)
}

func getPieceHeader(dataSize, pieceSize int32) uint32 {
	return uint32(dataSize | (pieceSize << 4))
}

func getUpdateTaskInfoWithStatusOnly(cdnStatus string) *types.TaskInfo {
	return getUpdateTaskInfo(cdnStatus, "", 0)
}

func getUpdateTaskInfo(cdnStatus, realMD5 string, fileLength int64) *types.TaskInfo {
	return &types.TaskInfo{
		CdnStatus:  cdnStatus,
		FileLength: fileLength,
		RealMd5:    realMD5,
	}
}

func getPieceMd5Value(pieceMd5Sum string, pieceLength int32) string {
	return fmt.Sprintf("%s:%d", pieceMd5Sum, pieceLength)
}
