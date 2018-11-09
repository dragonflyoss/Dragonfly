package client

import (
	"context"

	"github.com/alibaba/Dragonfly/apis/types"
)

// PreheatCreate creates a preheat task.
func (client *APIClient) PreheatCreate(ctx context.Context, config *types.PreheatCreateRequest) (*types.PreheatCreateResponse, error) {
	resp, err := client.post(ctx, "/preheats", nil, config, nil)
	if err != nil {
		return nil, err
	}

	preheat := &types.PreheatCreateResponse{}

	err = decodeBody(preheat, resp.Body)
	ensureCloseReader(resp)

	return preheat, err
}
