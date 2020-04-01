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

package types

import "github.com/dragonflyoss/Dragonfly/apis/types"

// FetchP2PNetworkInfoRequest is send to supernode to fetch p2p network info
type FetchP2PNetworkInfoRequest struct {
	// the urls is to filter the peer node, the url should be match with taskURL in TaskInfo
	Urls []string `json:"urls"`
}

// FetchP2PNetworkInfoResponse is send to supernode to fetch p2p network info
type FetchP2PNetworkInfoResponse struct {
	*BaseResponse
	Data *FetchNetworkInfoDataResponse `json:"data,omitempty"`
}

type FetchNetworkInfoDataResponse struct {
	Nodes []*types.Node `json:"nodes"`
}
