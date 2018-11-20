package client

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// PreheatInfo get detailed information of a preheat task.
func (client *APIClient) PreheatInfo(ctx context.Context, id string) (*types.PreheatInfo, error) {
	resp, err := client.get(ctx, "/preheats/"+id, nil, nil)
	if err != nil {
		return nil, err
	}

	preheat := &types.PreheatInfo{}

	err = decodeBody(preheat, resp.Body)
	ensureCloseReader(resp)

	return preheat, err
}
