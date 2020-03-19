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

package certutils

import (
	"crypto"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

const (
	privateKeyBlockType  = "PRIVATE KEY"
	certificateBlockType = "CERTIFICATE"
	rsaKeySize           = 2048
)

var organization = []string{"dragonfly"}

// CertConfig contains the basic fields required for creating a certificate
type CertConfig struct {
	// CommonName is the subject name of the certificate
	CommonName string
	// ExpireDuration is the duration the certificate can be valid
	ExpireDuration time.Duration
}

// NewCertificateAuthority creates new certificate and private key for the certificate authority
func NewCertificateAuthority(config *CertConfig) (*x509.Certificate, crypto.Signer, error) {
	key, err := NewPrivateKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create private key while generating CA certificate")
	}

	cert, err := NewSelfSignedCACert(key, config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create self-signed CA certificate")
	}

	return cert, key, nil
}

// NewPrivateKey creates an RSA private key
func NewPrivateKey() (crypto.Signer, error) {
	return rsa.GenerateKey(cryptorand.Reader, rsaKeySize)
}

// NewSelfSignedCACert creates a CA certificate
func NewSelfSignedCACert(key crypto.Signer, config *CertConfig) (*x509.Certificate, error) {
	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:   config.CommonName,
			Organization: organization,
		},
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(config.ExpireDuration).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// WriteKey stores the given key at the given location
func WriteKey(path string, key crypto.Signer) error {
	if key == nil {
		return errors.New("private key cannot be nil when writing to file")
	}

	if err := writeKeyToDisk(path, encodeKeyPEM(key)); err != nil {
		return errors.Wrapf(err, "unable to write private key to file %s", path)
	}

	return nil
}

// WriteCert stores the given certificate at the given location
func WriteCert(path string, cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("certificate cannot be nil when writing to file")
	}

	if err := writeCertToDisk(path, encodeCertPEM(cert)); err != nil {
		return errors.Wrapf(err, "unable to write certificate to file %s", path)
	}

	return nil
}

// writeKeyToDisk writes the pem-encoded key data to keyPath.
// The key file will be created with file mode 0600.
// If the key file already exists, it will be overwritten.
// The parent directory of the keyPath will be created as needed with file mode 0755.
func writeKeyToDisk(keyPath string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(keyPath), os.FileMode(0755)); err != nil {
		return err
	}
	return ioutil.WriteFile(keyPath, data, os.FileMode(0600))
}

// writeCertToDisk writes the pem-encoded certificate data to certPath.
// The certificate file will be created with file mode 0644.
// If the certificate file already exists, it will be overwritten.
// The parent directory of the certPath will be created as needed with file mode 0755.
func writeCertToDisk(certPath string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(certPath), os.FileMode(0755)); err != nil {
		return err
	}
	return ioutil.WriteFile(certPath, data, os.FileMode(0644))
}

// encodeKeyPEM returns PEM-endcoded key data
func encodeKeyPEM(privateKey crypto.PrivateKey) []byte {
	t := privateKey.(*rsa.PrivateKey)
	block := pem.Block{
		Type:  privateKeyBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(t),
	}
	return pem.EncodeToMemory(&block)
}

// encodeCertPEM returns PEM-endcoded certificate data
func encodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  certificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}
