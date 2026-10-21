/*
 *     Copyright 2026 The Dragonfly Authors
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

package oci

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"

	nethttp "d7y.io/dragonfly/v2/pkg/net/http"
)

// defaultRegistryTimeout is the default timeout for registry requests.
const defaultRegistryTimeout = 1 * time.Minute

// DefaultHTTPClient returns the default http client for registry requests, the
// registry certificate is verified against the system root CAs.
func DefaultHTTPClient() *http.Client {
	return NewHTTPClient(nil, false)
}

// NewHTTPClient returns a http client for registry requests. The registry
// certificate is verified against rootCAs, or against the system root CAs when
// rootCAs is nil, unless insecureSkipVerify is set.
func NewHTTPClient(rootCAs *x509.CertPool, insecureSkipVerify bool) *http.Client {
	return &http.Client{
		Timeout: defaultRegistryTimeout,
		Transport: &http.Transport{
			DialContext:         nethttp.NewSafeDialer().DialContext,
			TLSClientConfig:     &tls.Config{RootCAs: rootCAs, InsecureSkipVerify: insecureSkipVerify},
			MaxIdleConns:        400,
			MaxIdleConnsPerHost: 20,
			MaxConnsPerHost:     50,
			IdleConnTimeout:     120 * time.Second,
		},
	}
}

// parseOptions holds the configurable settings for ParseImage.
type parseOptions struct {
	plainHTTP bool
}

// ParseOption configures ParseImage.
type ParseOption func(o *parseOptions)

// WithPlainHTTP uses the http scheme instead of https for the registry.
func WithPlainHTTP(plainHTTP bool) ParseOption {
	return func(o *parseOptions) {
		o.plainHTTP = plainHTTP
	}
}

// ParseImage parses an image reference (e.g., "docker.io/library/nginx:latest")
// into a registry reference. The reference is normalized with docker
// conventions, so "nginx:latest" resolves to "docker.io/library/nginx:latest".
func ParseImage(image string, opts ...ParseOption) (*Reference, error) {
	options := &parseOptions{}
	for _, opt := range opts {
		opt(options)
	}

	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return nil, fmt.Errorf("invalid image reference: %w", err)
	}
	named = reference.TagNameOnly(named)

	var tag string
	switch ref := named.(type) {
	case reference.Digested:
		tag = ref.Digest().String()
	case reference.Tagged:
		tag = ref.Tag()
	default:
		return nil, fmt.Errorf("invalid image reference: %s", image)
	}

	registry := reference.Domain(named)
	if registry == "docker.io" {
		registry = "registry-1.docker.io"
	}

	scheme := "https"
	if options.plainHTTP {
		scheme = "http"
	}

	return &Reference{
		Scheme:     scheme,
		Registry:   registry,
		Repository: reference.Path(named),
		Reference:  tag,
	}, nil
}

// resolveOptions holds the configurable settings for Resolve.
type resolveOptions struct {
	username   string
	password   string
	platform   string
	header     http.Header
	httpClient *http.Client
}

// ResolveOption configures Resolve.
type ResolveOption func(o *resolveOptions)

// WithAuth sets the username and password for registry authentication,
// anonymous access is used when they are empty.
func WithAuth(username, password string) ResolveOption {
	return func(o *resolveOptions) {
		o.username = username
		o.password = password
	}
}

// WithPlatform sets the target platform in the format "os/arch"
// (e.g., "linux/amd64"), the current platform is used when it is empty.
func WithPlatform(platform string) ResolveOption {
	return func(o *resolveOptions) {
		o.platform = platform
	}
}

// WithHeader sets the headers forwarded to the manifest requests. An
// Authorization header is used as the issued token to access the v2 API
// directly, without going through v2 authentication.
func WithHeader(header http.Header) ResolveOption {
	return func(o *resolveOptions) {
		o.header = header
	}
}

// WithHTTPClient sets the http client for registry requests, used to customize
// TLS and dialing behavior.
func WithHTTPClient(client *http.Client) ResolveOption {
	return func(o *resolveOptions) {
		o.httpClient = client
	}
}

// Resolve resolves the manifest of the reference (including multi-platform image indexes) from
// the registry and returns the manifest urls referenced by the digests of the resolved manifests,
// the blob urls (config and layers) and the authorization token for downloading them.
func Resolve(ctx context.Context, ref *Reference, opts ...ResolveOption) (manifestURLs, blobURLs []string, token string, err error) {
	options := &resolveOptions{header: make(http.Header)}
	for _, opt := range opts {
		opt(options)
	}

	platform := platforms.DefaultSpec()
	if options.platform != "" {
		platform, err = platforms.Parse(options.platform)
		if err != nil {
			return nil, nil, "", fmt.Errorf("invalid platform format %q, expected \"os/arch\" (e.g., \"linux/amd64\"): %w", options.platform, err)
		}
	}

	if options.httpClient == nil {
		options.httpClient = DefaultHTTPClient()
	}

	var authOpts []Option
	if issuedToken := options.header.Get("Authorization"); issuedToken != "" {
		authOpts = append(authOpts, WithIssuedToken(issuedToken))
	}

	client, err := NewAuthClient(ref, options.httpClient, options.username, options.password, authOpts...)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to authenticate with registry: %w", err)
	}

	manifests, manifestURLs, err := client.ResolveManifests(ctx, ref, options.header.Clone(), platform)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to pull image manifest: %w", err)
	}

	if len(manifests) == 0 {
		return nil, nil, "", fmt.Errorf("no matching manifest for platform %s", platforms.Format(platform))
	}

	for _, manifest := range manifests {
		for _, desc := range manifest.References() {
			blobURLs = append(blobURLs, ref.BlobURL(desc.Digest.String()))
		}
	}

	return manifestURLs, blobURLs, client.AuthToken(), nil
}
