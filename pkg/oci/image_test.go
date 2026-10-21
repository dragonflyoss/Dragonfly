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
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseImage(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		opts   []ParseOption
		expect func(t *testing.T, ref *Reference, err error)
	}{
		{
			name:  "short image reference",
			image: "nginx:latest",
			expect: func(t *testing.T, ref *Reference, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal("https", ref.Scheme)
				assert.Equal("registry-1.docker.io", ref.Registry)
				assert.Equal("library/nginx", ref.Repository)
				assert.Equal("latest", ref.Reference)
			},
		},
		{
			name:  "image reference without tag",
			image: "docker.io/library/nginx",
			expect: func(t *testing.T, ref *Reference, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal("registry-1.docker.io", ref.Registry)
				assert.Equal("library/nginx", ref.Repository)
				assert.Equal("latest", ref.Reference)
			},
		},
		{
			name:  "image reference with digest",
			image: "registry.example.com/library/nginx@sha256:b2c366cce7e68013d5441c6326d5a3e1b12aeb5ed58564d0fd3fa089bc29cb6e",
			expect: func(t *testing.T, ref *Reference, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal("registry.example.com", ref.Registry)
				assert.Equal("library/nginx", ref.Repository)
				assert.Equal("sha256:b2c366cce7e68013d5441c6326d5a3e1b12aeb5ed58564d0fd3fa089bc29cb6e", ref.Reference)
			},
		},
		{
			name:  "image reference with registry port",
			image: "localhost:5000/myrepo:v1.0.0",
			expect: func(t *testing.T, ref *Reference, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal("localhost:5000", ref.Registry)
				assert.Equal("myrepo", ref.Repository)
				assert.Equal("v1.0.0", ref.Reference)
			},
		},
		{
			name:  "image reference with plain http",
			image: "localhost:5000/myrepo:v1.0.0",
			opts:  []ParseOption{WithPlainHTTP(true)},
			expect: func(t *testing.T, ref *Reference, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal("http", ref.Scheme)
			},
		},
		{
			name:  "invalid image reference",
			image: "invalid image reference!!",
			expect: func(t *testing.T, ref *Reference, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.Error(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ref, err := ParseImage(tc.image, tc.opts...)
			tc.expect(t, ref, err)
		})
	}
}

func TestNewHTTPClient(t *testing.T) {
	rootCAs := x509.NewCertPool()

	tests := []struct {
		name               string
		rootCAs            *x509.CertPool
		insecureSkipVerify bool
		expect             func(t *testing.T, tlsConfig *tls.Config)
	}{
		{
			name: "default client verifies the registry certificate",
			expect: func(t *testing.T, tlsConfig *tls.Config) {
				assert := assert.New(t)
				assert.False(tlsConfig.InsecureSkipVerify)
				assert.Nil(tlsConfig.RootCAs)
			},
		},
		{
			name:    "verify the registry certificate against the given root cas",
			rootCAs: rootCAs,
			expect: func(t *testing.T, tlsConfig *tls.Config) {
				assert := assert.New(t)
				assert.False(tlsConfig.InsecureSkipVerify)
				assert.Equal(rootCAs, tlsConfig.RootCAs)
			},
		},
		{
			name:               "skip the registry certificate verification",
			insecureSkipVerify: true,
			expect: func(t *testing.T, tlsConfig *tls.Config) {
				assert := assert.New(t)
				assert.True(tlsConfig.InsecureSkipVerify)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			client := NewHTTPClient(tc.rootCAs, tc.insecureSkipVerify)
			transport, ok := client.Transport.(*http.Transport)
			assert.True(ok)
			tc.expect(t, transport.TLSClientConfig)
		})
	}
}

func TestDefaultHTTPClient(t *testing.T) {
	assert := assert.New(t)

	transport, ok := DefaultHTTPClient().Transport.(*http.Transport)
	assert.True(ok)
	assert.False(transport.TLSClientConfig.InsecureSkipVerify)
}

func TestResolve(t *testing.T) {
	manifest := `{
		"schemaVersion": 2,
		"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
		"config": {
			"mediaType": "application/vnd.docker.container.image.v1+json",
			"size": 7023,
			"digest": "sha256:b5b2b2c507a0944348e0303114d8d93aaaa081732b86451d9bce1f432a537bc7"
		},
		"layers": [
			{
				"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
				"size": 32654,
				"digest": "sha256:e692418e4cbaf90ca69d05a66403747baa33ee08806650b51fab815ad7fc331f"
			}
		]
	}`

	var server *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=%q,service=\"registry\"", server.URL+"/token"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"token":"test-token"}`)
	})
	mux.HandleFunc("/v2/library/nginx/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=%q,service=\"registry\"", server.URL+"/token"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		fmt.Fprint(w, manifest)
	})
	server = httptest.NewServer(mux)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	issuedHeader := make(http.Header)
	issuedHeader.Set("Authorization", "Bearer test-token")

	manifestDigest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(manifest)))

	tests := []struct {
		name   string
		opts   []ResolveOption
		expect func(t *testing.T, manifestURLs, blobURLs []string, token string, err error)
	}{
		{
			name: "resolve with token challenge",
			opts: []ResolveOption{WithHTTPClient(&http.Client{Transport: http.DefaultTransport})},
			expect: func(t *testing.T, manifestURLs, blobURLs []string, token string, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal([]string{
					fmt.Sprintf("http://%s/v2/library/nginx/manifests/%s", serverURL.Host, manifestDigest),
				}, manifestURLs)
				assert.Equal([]string{
					fmt.Sprintf("http://%s/v2/library/nginx/blobs/sha256:b5b2b2c507a0944348e0303114d8d93aaaa081732b86451d9bce1f432a537bc7", serverURL.Host),
					fmt.Sprintf("http://%s/v2/library/nginx/blobs/sha256:e692418e4cbaf90ca69d05a66403747baa33ee08806650b51fab815ad7fc331f", serverURL.Host),
				}, blobURLs)
				assert.Equal("Bearer test-token", token)
			},
		},
		{
			name: "resolve with issued authorization header",
			opts: []ResolveOption{
				WithHTTPClient(&http.Client{Transport: http.DefaultTransport}),
				WithHeader(issuedHeader),
			},
			expect: func(t *testing.T, manifestURLs, blobURLs []string, token string, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal([]string{
					fmt.Sprintf("http://%s/v2/library/nginx/manifests/%s", serverURL.Host, manifestDigest),
				}, manifestURLs)
				assert.Equal([]string{
					fmt.Sprintf("http://%s/v2/library/nginx/blobs/sha256:b5b2b2c507a0944348e0303114d8d93aaaa081732b86451d9bce1f432a537bc7", serverURL.Host),
					fmt.Sprintf("http://%s/v2/library/nginx/blobs/sha256:e692418e4cbaf90ca69d05a66403747baa33ee08806650b51fab815ad7fc331f", serverURL.Host),
				}, blobURLs)
				assert.Equal("Bearer test-token", token)
			},
		},
		{
			name: "invalid platform",
			opts: []ResolveOption{
				WithHTTPClient(&http.Client{Transport: http.DefaultTransport}),
				WithPlatform("linux-amd64"),
			},
			expect: func(t *testing.T, manifestURLs, blobURLs []string, token string, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.Error(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			ref, err := ParseImage(serverURL.Host+"/library/nginx:latest", WithPlainHTTP(true))
			assert.NoError(err)

			manifestURLs, blobURLs, token, err := Resolve(context.Background(), ref, tc.opts...)
			tc.expect(t, manifestURLs, blobURLs, token, err)
		})
	}
}

func TestResolveManifestList(t *testing.T) {
	manifest := `{
		"schemaVersion": 2,
		"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
		"config": {
			"mediaType": "application/vnd.docker.container.image.v1+json",
			"size": 7023,
			"digest": "sha256:b5b2b2c507a0944348e0303114d8d93aaaa081732b86451d9bce1f432a537bc7"
		},
		"layers": [
			{
				"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
				"size": 32654,
				"digest": "sha256:e692418e4cbaf90ca69d05a66403747baa33ee08806650b51fab815ad7fc331f"
			}
		]
	}`
	manifestDigest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(manifest)))
	manifestList := fmt.Sprintf(`{
		"schemaVersion": 2,
		"mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
		"manifests": [
			{
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"size": %d,
				"digest": "%s",
				"platform": {
					"architecture": "amd64",
					"os": "linux"
				}
			},
			{
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"size": 7143,
				"digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"platform": {
					"architecture": "arm64",
					"os": "linux"
				}
			}
		]
	}`, len(manifest), manifestDigest)

	var server *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=%q,service=\"registry\"", server.URL+"/token"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"token":"test-token"}`)
	})
	mux.HandleFunc("/v2/library/nginx/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.list.v2+json")
		fmt.Fprint(w, manifestList)
	})
	mux.HandleFunc("/v2/library/nginx/manifests/"+manifestDigest, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		fmt.Fprint(w, manifest)
	})
	server = httptest.NewServer(mux)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		platform string
		expect   func(t *testing.T, manifestURLs, blobURLs []string, token string, err error)
	}{
		{
			name:     "resolve matched platform manifest from manifest list",
			platform: "linux/amd64",
			expect: func(t *testing.T, manifestURLs, blobURLs []string, token string, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal([]string{
					fmt.Sprintf("http://%s/v2/library/nginx/manifests/%s", serverURL.Host, manifestDigest),
				}, manifestURLs)
				assert.Equal([]string{
					fmt.Sprintf("http://%s/v2/library/nginx/blobs/sha256:b5b2b2c507a0944348e0303114d8d93aaaa081732b86451d9bce1f432a537bc7", serverURL.Host),
					fmt.Sprintf("http://%s/v2/library/nginx/blobs/sha256:e692418e4cbaf90ca69d05a66403747baa33ee08806650b51fab815ad7fc331f", serverURL.Host),
				}, blobURLs)
				assert.Equal("Bearer test-token", token)
			},
		},
		{
			name:     "no matching manifest for platform",
			platform: "linux/riscv64",
			expect: func(t *testing.T, manifestURLs, blobURLs []string, token string, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.Error(err)
			},
		},
		{
			name:     "matched platform manifest is not served by the registry",
			platform: "linux/arm64",
			expect: func(t *testing.T, manifestURLs, blobURLs []string, token string, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			ref, err := ParseImage(serverURL.Host+"/library/nginx:latest", WithPlainHTTP(true))
			assert.NoError(err)

			manifestURLs, blobURLs, token, err := Resolve(context.Background(), ref,
				WithHTTPClient(&http.Client{Transport: http.DefaultTransport}),
				WithPlatform(tc.platform),
			)
			tc.expect(t, manifestURLs, blobURLs, token, err)
		})
	}
}
