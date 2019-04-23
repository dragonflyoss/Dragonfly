package cdn

import (
	"context"
	"fmt"
	"net/http"

	errorType "github.com/dragonflyoss/Dragonfly/common/errors"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// download downloads the file from the original address and
// sets the "Range" header to the undownloaded file range.
//
// If the returned error is nil, the Response will contain a non-nil
// Body which the caller is expected to close.
func (cm *Manager) download(ctx context.Context, taskID, url string, headers map[string]string,
	startPieceNum int, httpFileLength int64, pieceContSize int32) (*http.Response, error) {
	var checkCode = http.StatusOK

	if startPieceNum > 0 {
		breakRange, err := util.CalculateBreakRange(startPieceNum, int(pieceContSize), httpFileLength)
		if err != nil {
			return nil, errors.Wrapf(errorType.ErrInvalidValue, "failed to calculate the breakRange: %v", err)
		}

		if headers == nil {
			headers = make(map[string]string)
		}
		headers["Range"] = cutil.ConstuctRangeStr(breakRange)
		checkCode = http.StatusPartialContent
	}

	logrus.Infof("start to download for taskId(%s) with fileUrl: %s header: %v checkCode: %d", taskID, url, headers, checkCode)
	return getWithURL(url, headers, checkCode)
}

func getWithURL(url string, headers map[string]string, checkCode int) (*http.Response, error) {
	// TODO: add timeout
	resp, err := cutil.HTTPGetWithHeaders(url, headers)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == checkCode {
		return resp, nil
	}
	return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}
