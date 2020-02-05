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

package proxy

import (
	"crypto"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

type LeafCertSpec struct {
	publicKey crypto.PublicKey

	privateKey crypto.PrivateKey

	signatureAlgorithm x509.SignatureAlgorithm
}

// genLeafCert generates a Leaf TLS certificate and sign it with given CA
func genLeafCert(ca *tls.Certificate, leafCertSpec *LeafCertSpec, host string) (*tls.Certificate, error) {
	now := time.Now().Add(-1 * time.Hour).UTC()
	if !ca.Leaf.IsCA {
		return nil, errors.New("CA cert is not a CA")
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		logrus.Errorf("failed to generate serial number: %s", err)
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: host},
		NotBefore:             now,
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyAgreement,
		BasicConstraintsValid: true,
		SignatureAlgorithm:    leafCertSpec.signatureAlgorithm,
	}
	ip := net.ParseIP(host)
	if ip == nil {
		tmpl.DNSNames = []string{host}
	} else {
		tmpl.IPAddresses = []net.IP{ip}
	}
	newCert, err := x509.CreateCertificate(rand.Reader, tmpl, ca.Leaf, leafCertSpec.publicKey, ca.PrivateKey)
	if err != nil {
		logrus.Errorf("failed to generate leaf cert %s", err)
		return nil, err
	}
	cert := new(tls.Certificate)
	cert.Certificate = append(cert.Certificate, newCert)
	cert.PrivateKey = leafCertSpec.privateKey
	cert.Leaf, _ = x509.ParseCertificate(newCert)
	return cert, nil
}
