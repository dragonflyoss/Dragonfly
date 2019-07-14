package request

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/dragonflyoss/Dragonfly/client"
	"github.com/dragonflyoss/Dragonfly/test/environment"
)

// Option defines a type used to update http.Request.
type Option func(*http.Request) error

// WithContext sets the ctx of http.Request.
func WithContext(ctx context.Context) Option {
	return func(r *http.Request) error {
		r2 := r.WithContext(ctx)
		*r = *r2
		return nil
	}
}

// WithHeader sets the Header of http.Request.
func WithHeader(key string, value string) Option {
	return func(r *http.Request) error {
		r.Header.Add(key, value)
		return nil
	}
}

// WithQuery sets the query field in URL.
func WithQuery(query url.Values) Option {
	return func(r *http.Request) error {
		r.URL.RawQuery = query.Encode()
		return nil
	}
}

// WithRawData sets the input data with raw data
func WithRawData(data io.ReadCloser) Option {
	return func(r *http.Request) error {
		r.Body = data
		return nil
	}
}

// WithJSONBody encodes the input data to JSON and sets it to the body in http.Request
func WithJSONBody(obj interface{}) Option {
	return func(r *http.Request) error {
		b := bytes.NewBuffer([]byte{})

		if obj != nil {
			err := json.NewEncoder(b).Encode(obj)

			if err != nil {
				return err
			}
		}
		r.Body = ioutil.NopCloser(b)
		r.Header.Set("Content-Type", "application/json")
		return nil
	}
}

// DecodeBody decodes body to obj.
func DecodeBody(obj interface{}, body io.ReadCloser) error {
	defer body.Close()
	return json.NewDecoder(body).Decode(obj)
}

// Delete sends request to the default pouchd server with custom request options.
func Delete(endpoint string, opts ...Option) (*http.Response, error) {
	apiClient, err := newAPIClient(environment.DragonflyAddress, environment.TLSConfig)
	if err != nil {
		return nil, err
	}

	fullPath := apiClient.BaseURL() + apiClient.GetAPIPath(endpoint, url.Values{})
	req, err := newRequest(http.MethodDelete, fullPath, opts...)
	if err != nil {
		return nil, err
	}
	return apiClient.HTTPCli.Do(req)
}

// Debug sends request to the default pouchd server to get the debug info.
//
// NOTE: without any version information.
func Debug(endpoint string) (*http.Response, error) {
	apiClient, err := newAPIClient(environment.DragonflyAddress, environment.TLSConfig)
	if err != nil {
		return nil, err
	}

	fullPath := apiClient.BaseURL() + endpoint
	req, err := newRequest(http.MethodGet, fullPath)
	if err != nil {
		return nil, err
	}
	return apiClient.HTTPCli.Do(req)
}

// Get sends request to the default pouchd server with custom request options.
func Get(endpoint string, opts ...Option) (*http.Response, error) {
	apiClient, err := newAPIClient(environment.DragonflyAddress, environment.TLSConfig)
	if err != nil {
		return nil, err
	}

	fullPath := apiClient.BaseURL() + apiClient.GetAPIPath(endpoint, url.Values{})
	req, err := newRequest(http.MethodGet, fullPath, opts...)
	if err != nil {
		return nil, err
	}
	return apiClient.HTTPCli.Do(req)
}

// Post sends post request to pouchd.
func Post(endpoint string, opts ...Option) (*http.Response, error) {
	apiClient, err := newAPIClient(environment.DragonflyAddress, environment.TLSConfig)
	if err != nil {
		return nil, err
	}

	fullPath := apiClient.BaseURL() + apiClient.GetAPIPath(endpoint, url.Values{})
	req, err := newRequest(http.MethodPost, fullPath, opts...)
	if err != nil {
		return nil, err
	}

	// By default, if Content-Type in header is not set, set it to application/json
	if req.Header.Get("Content-Type") == "" {
		WithHeader("Content-Type", "application/json")(req)
	}
	return apiClient.HTTPCli.Do(req)
}

// newAPIClient return new HTTP client with tls.
//
// FIXME: Could we make some functions exported in alibaba/pouch/client?
func newAPIClient(host string, tls client.TLSConfig) (*client.APIClient, error) {
	commonAPIClient, err := client.NewAPIClient(host, tls)
	if err != nil {
		return nil, err
	}
	apiClient := commonAPIClient.(*client.APIClient)
	return apiClient, nil
}

// newRequest creates request targeting on specific host/path by method.
func newRequest(method, url string, opts ...Option) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		err := opt(req)
		if err != nil {
			return nil, err
		}
	}
	return req, nil
}
