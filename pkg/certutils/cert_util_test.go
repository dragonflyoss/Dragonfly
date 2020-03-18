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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type CertUtilTestSuite struct{}

func init() {
	check.Suite(&CertUtilTestSuite{})
}

func (suite *CertUtilTestSuite) TestNewCertificateAuthority(c *check.C) {
	config := &CertConfig{
		CommonName:     "dfdaemon",
		ExpireDuration: time.Hour * 24,
	}
	cert, key, err := NewCertificateAuthority(config)
	c.Assert(err, check.IsNil)
	c.Assert(cert, check.NotNil)
	c.Assert(key, check.NotNil)
	c.Assert(cert.IsCA, check.Equals, true)
}

func (suite *CertUtilTestSuite) TestWriteKey(c *check.C) {
	tmpdir, err := ioutil.TempDir("", "")
	c.Assert(err, check.IsNil)

	defer os.RemoveAll(tmpdir)

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	c.Assert(err, check.IsNil)

	path := fmt.Sprintf("%s/ca.key", tmpdir)
	err = WriteKey(path, caKey)
	c.Assert(err, check.IsNil)
}

func (suite *CertUtilTestSuite) TestWriteCert(c *check.C) {
	tmpdir, err := ioutil.TempDir("", "")
	c.Assert(err, check.IsNil)

	defer os.RemoveAll(tmpdir)

	caCert := &x509.Certificate{}
	path := fmt.Sprintf("%s/ca.crt", tmpdir)
	err = WriteCert(path, caCert)
	c.Assert(err, check.IsNil)
}
