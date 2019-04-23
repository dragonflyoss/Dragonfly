package cdn

import (
	"time"
)

var getCurrentTimeMillisFunc = getCurrentTimeMillis

func getCurrentTimeMillis() int64 {
	return time.Now().UnixNano() / time.Millisecond.Nanoseconds()
}
