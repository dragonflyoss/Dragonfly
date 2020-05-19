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
}

const (
	HTTPTransport = "http.transport"
	TLSConfig     = "tls.config"
)

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

// ClientOpt is the argument of NewProtocolClient.
// ClientOpt supports some opt by key, such as "http.transport", "tls.config".
// if not set, default opt will be used.
type ClientOpt struct {
	opt map[string]interface{}
}

func NewClientOpt() *ClientOpt {
	return &ClientOpt{
		opt: make(map[string]interface{}),
	}
}

func (opt *ClientOpt) Set(key string, value interface{}) error {
	switch key {
	case HTTPTransport:
		if _, ok := value.(*http.Transport); !ok {
			return errortypes.ErrConvertFailed
		}
		break
	case TLSConfig:
		if _, ok := value.(*tls.Config); !ok {
			return errortypes.ErrConvertFailed
		}
		break
	default:
		return fmt.Errorf("not support")
	}

	opt.opt[key] = value
	return nil
}

func (opt *ClientOpt) Get(key string) interface{} {
	v, ok := opt.opt[key]
	if !ok {
		return nil
	}

	return v
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

func (cb *ClientBuilder) NewProtocolClient(clientOpt interface{}) (protocol.Client, error) {
	var (
		transport = DefaultTransport
		tlsConfig *tls.Config
	)

	if clientOpt != nil {
		opt, ok := clientOpt.(*ClientOpt)
		if !ok {
			return nil, errortypes.ErrConvertFailed
		}

		tran := opt.Get(HTTPTransport)
		if tran != nil {
			transport = tran.(*http.Transport)
		}

		config := opt.Get(TLSConfig)
		if config != nil {
			tlsConfig = config.(*tls.Config)
		}

		// set tls config to transport
		if config != nil {
			if transport == DefaultTransport {
				transport = newDefaultTransport()
			}

			transport.TLSClientConfig = tlsConfig
		}
	}

	if cb.supportHTTPS {
		if transport.TLSClientConfig == nil || transport.DialTLS == nil {
			return nil, fmt.Errorf("in https mode, tls should be set")
		}
	}

	return &Client{
		client:    &http.Client{Transport: transport},
		transport: transport,
	}, nil
}
