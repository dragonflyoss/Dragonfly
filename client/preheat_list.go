package client

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// PreheatList lists detailed information of preheat tasks.
func (client *APIClient) PreheatList(ctx context.Context, id string) ([]*types.PreheatInfo, error) {
	resp, err := client.get(ctx, "/preheats", nil, nil)
	if err != nil {
		return nil, err
	}

	preheats := []*types.PreheatInfo{}

	err = decodeBody(preheats, resp.Body)
	ensureCloseReader(resp)

	return preheats, err
}
