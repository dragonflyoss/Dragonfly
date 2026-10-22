/*
 *     Copyright 2024 The Dragonfly Authors
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

package redis

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTLSClientConfig(t *testing.T) {
	caCertPath := writeTestCACert(t)

	tests := []struct {
		name               string
		caCert             string
		cert               string
		key                string
		insecureSkipVerify bool
		expect             func(t *testing.T, cfg *tls.Config, err error)
	}{
		{
			name: "no ca and no client cert falls back to system roots",
			expect: func(t *testing.T, cfg *tls.Config, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.NotNil(cfg)
				assert.Nil(cfg.RootCAs)
				assert.False(cfg.InsecureSkipVerify)
			},
		},
		{
			name:               "insecureSkipVerify is honored",
			insecureSkipVerify: true,
			expect: func(t *testing.T, cfg *tls.Config, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.NotNil(cfg)
				assert.True(cfg.InsecureSkipVerify)
			},
		},
		{
			name:   "ca certificate is loaded into the root pool",
			caCert: caCertPath,
			expect: func(t *testing.T, cfg *tls.Config, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.NotNil(cfg)
				assert.NotNil(cfg.RootCAs)
			},
		},
		{
			name:   "missing ca certificate returns an error",
			caCert: filepath.Join(t.TempDir(), "does-not-exist.pem"),
			expect: func(t *testing.T, cfg *tls.Config, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := NewTLSClientConfig(tc.caCert, tc.cert, tc.key, tc.insecureSkipVerify)
			tc.expect(t, cfg, err)
		})
	}
}

// writeTestCACert generates a self-signed CA certificate and writes it as PEM to
// a temporary file, returning the file path.
func writeTestCACert(t *testing.T) string {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "dragonfly-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	assert.NoError(t, err)

	path := filepath.Join(t.TempDir(), "ca.pem")
	f, err := os.Create(path)
	assert.NoError(t, err)
	defer f.Close()

	assert.NoError(t, pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: der}))

	return path
}
