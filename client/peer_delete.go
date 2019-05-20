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
	"context"
)

// PeerDelete deletes the specified peer in supernode.
func (client *APIClient) PeerDelete(ctx context.Context, id string) error {
	resp, err := client.delete(ctx, "/peers/"+id, nil, nil)
	if err != nil {
		return err
	}
	ensureCloseReader(resp)
	return nil
}
