package cdn

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
)

// download the file from the original address and
// set the "Range" header to the undownloaded file range.
func (cm *Manager) download(ctx context.Context, taskID, url string, headers map[string]string, startPieceNum int, httpFileLength int64, pieceContSize int32) (*http.Response, error) {
	logrus.Infof("start to download for taskId:%s, fileUrl:%s", taskID, url)
	return nil, nil
}
