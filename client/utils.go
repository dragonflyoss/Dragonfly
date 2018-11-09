package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

func decodeBody(obj interface{}, body io.Reader) error {
	if err := json.NewDecoder(body).Decode(obj); err != nil {
		return fmt.Errorf("failed to decode body: %v", err)
	}

	return nil
}

func ensureCloseReader(resp *Response) {
	if resp != nil && resp.Body != nil {
		// close body ReadCloser to make Transport reuse the connection
		io.CopyN(ioutil.Discard, resp.Body, 512)
		resp.Body.Close()
	}
}
