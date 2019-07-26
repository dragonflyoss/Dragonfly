package cdn

import (
	"fmt"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
)

var getCurrentTimeMillisFunc = cutil.GetCurrentTimeMillis

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
