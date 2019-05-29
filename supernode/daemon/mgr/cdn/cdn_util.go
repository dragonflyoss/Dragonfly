package cdn

import (
	"time"
)

var getCurrentTimeMillisFunc = getCurrentTimeMillis

func getCurrentTimeMillis() int64 {
	return time.Now().UnixNano() / time.Millisecond.Nanoseconds()
}

// getContentLengthByHeader calculates the piece content length by piece header.
func getContentLengthByHeader(pieceHeader uint32) int32 {
	return int32(pieceHeader & 0xffffff)
}

func getPieceHeader(dataSize, pieceSize int32) uint32 {
	return uint32(dataSize | (pieceSize << 4))
}
