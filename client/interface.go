package client

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// CommonAPIClient defines common methods of api client
type CommonAPIClient interface {
	PreheatAPIClient
}

// PreheatAPIClient defines methods of Container client.
type PreheatAPIClient interface {
	PreheatCreate(ctx context.Context, config *types.PreheatCreateRequest) (*types.PreheatCreateResponse, error)
}
