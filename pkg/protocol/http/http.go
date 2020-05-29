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

package http

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/protocol"
)

var (
	// DefaultTransport is default implementation of http.Transport.
	DefaultTransport = newDefaultTransport()

	// DefaultClient is default implementation of Client.
	DefaultClient = &Client{
		client:    &http.Client{Transport: DefaultTransport},
		transport: DefaultTransport,
	}
)

const (
	// http protocol name
	ProtocolHTTPName = "http"

	// https protocol name
	ProtocolHTTPSName = "https"
)

func init() {
	protocol.RegisterProtocol(ProtocolHTTPName, &ClientBuilder{})
	protocol.RegisterProtocol(ProtocolHTTPSName, &ClientBuilder{supportHTTPS: true})

	protocol.RegisterMapInterfaceOptFunc(ProtocolHTTPName, WithMapInterface)
	protocol.RegisterMapInterfaceOptFunc(ProtocolHTTPSName, WithMapInterface)
}

func newDefaultTransport() *http.Transport {
	// copy from http.DefaultTransport
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

var _ protocol.Client = &Client{}

// Client is an implementation of protocol.Client for http protocol.
type Client struct {
	client    *http.Client
	transport http.RoundTripper
}

func (cli *Client) GetResource(url string, md protocol.Metadata) protocol.Resource {
	var (
		hd *Headers
	)

	if md != nil {
		h, ok := md.(*Headers)
		if ok {
			hd = h
		}
	}

	return &Resource{
		url:    url,
		hd:     hd,
		client: cli,
	}
}

// ClientBuilder is an implementation of protocol.ClientBuilder for http protocol.
type ClientBuilder struct {
	supportHTTPS bool
}

// WithTransport allows to set transport for http protocol of  protocol.Client.
func WithTransport(transport *http.Transport) func(protocol.Client) {
	return func(client protocol.Client) {
		cli := client.(*Client)
		cli.transport = transport
	}
}

// WithTLS allows to set tls config for http protocol of  protocol.Client.
func WithTLS(config *tls.Config) func(protocol.Client) error {
	return func(client protocol.Client) error {
		cli := client.(*Client)
		if cli.transport == DefaultTransport {
			cli.transport = newDefaultTransport()
		}

		tran, ok := cli.transport.(*http.Transport)
		if !ok {
			return fmt.Errorf("transport could not be converted to http.Transport")
		}

		tran.TLSClientConfig = config
		return nil
	}
}

// WithMapInterface allows to set some options by map interface.
// Supported:
// key: "tls.config", value: *tls.Config
// key: "http.transport" value: *http.Transport
func WithMapInterface(opt map[string]interface{}) func(client protocol.Client) error {
	return func(client protocol.Client) error {
		var (
			transport *http.Transport
			config    *tls.Config
			ok        bool
		)

		cli := client.(*Client)
		for k, v := range opt {
			switch k {
			case "tls.config":
				config, ok = v.(*tls.Config)
				if !ok {
					return errortypes.ErrConvertFailed
				}
			case "http.transport":
				transport, ok = v.(*http.Transport)
				if !ok {
					return errortypes.ErrConvertFailed
				}
			default:
			}
		}

		if transport == nil {
			transport = DefaultTransport
		}

		if config != nil {
			if transport == DefaultTransport {
				transport = newDefaultTransport()
			}

			transport.TLSClientConfig = config
		}

		cli.transport = transport
		cli.client = &http.Client{
			Transport: transport,
		}

		return nil
	}
}

func (cb *ClientBuilder) NewProtocolClient(opts ...func(client protocol.Client) error) (protocol.Client, error) {
	cli := &Client{
		transport: DefaultTransport,
	}

	for _, opt := range opts {
		if err := opt(cli); err != nil {
			return nil, err
		}
	}

	cli.client = &http.Client{
		Transport: cli.transport,
	}

	if cb.supportHTTPS {
		tran, ok := cli.transport.(*http.Transport)
		if ok {
			if tran.TLSClientConfig == nil && tran.DialTLS == nil {
				return nil, fmt.Errorf("in https mode, tls should be set")
			}
		}
	}

	return cli, nil
}
