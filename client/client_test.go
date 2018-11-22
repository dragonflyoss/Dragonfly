package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAPIClient(t *testing.T) {
	assert := assert.New(t)
	kvs := map[string]bool{
		"foobar":                 true,
		"https://localhost:2476": false,
		"http://localhost:2476":  false,
	}

	for host, expectError := range kvs {
		cli, err := NewAPIClient(host, TLSConfig{})
		if expectError {
			assert.Error(err, fmt.Sprintf("test data: %v", host))
		} else {
			assert.NoError(err, fmt.Sprintf("test data %v: %v", host, err))
		}

		t.Logf("client info %+v", cli)
	}
}
