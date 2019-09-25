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

func ensureCloseReader(resp *Response) error {
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()

		// Close body ReadCloser to make Transport reuse the connection.
		_, err := io.CopyN(ioutil.Discard, resp.Body, 512)
		return err
	}
	return nil
}
