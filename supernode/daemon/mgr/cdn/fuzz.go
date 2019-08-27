// +build gofuzz

package cdn

import (
	"bytes"
	"context"
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

func Fuzz(data []byte) int {
	// Don't spam output with parse failures
	logrus.SetOutput(ioutil.Discard)
	r := bytes.NewReader(data)
	sr := newSuperReader()
	_, err := sr.readFile(context.Background(), r, true, true)
	if err != nil {
		return 0
	}
	return 1
}
