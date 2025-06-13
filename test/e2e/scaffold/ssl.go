// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scaffold

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
)

const (
	_secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: %s
data:
  cert: %s
  key: %s
`
	_kubeTlsSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: %s
data:
  tls.crt: %s
  tls.key: %s
`
	_clientCASecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: %s
data:
  cert: %s
`
)

// NewSecret new a k8s secret
func (s *Scaffold) NewSecret(name, cert, key string) error {
	certBase64 := base64.StdEncoding.EncodeToString([]byte(cert))
	keyBase64 := base64.StdEncoding.EncodeToString([]byte(key))
	secret := fmt.Sprintf(_secretTemplate, name, certBase64, keyBase64)
	if err := s.CreateResourceFromString(secret); err != nil {
		return err
	}
	return nil
}

// NewKubeTlsSecret new a kube style tls secret
func (s *Scaffold) NewKubeTlsSecret(name, cert, key string) error {
	certBase64 := base64.StdEncoding.EncodeToString([]byte(cert))
	keyBase64 := base64.StdEncoding.EncodeToString([]byte(key))
	secret := fmt.Sprintf(_kubeTlsSecretTemplate, name, certBase64, keyBase64)
	if err := s.CreateResourceFromString(secret); err != nil {
		return err
	}
	return nil
}

// NewClientCASecret new a k8s secret
func (s *Scaffold) NewClientCASecret(name, cert, key string) error {
	certBase64 := base64.StdEncoding.EncodeToString([]byte(cert))
	secret := fmt.Sprintf(_clientCASecretTemplate, name, certBase64)
	if err := s.CreateResourceFromString(secret); err != nil {
		return err
	}
	return nil
}

func (s *Scaffold) GenerateCert(t GinkgoTInterface, dnsNames []string) (certPemBytes, privPemBytes bytes.Buffer) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(err).ToNot(HaveOccurred())
	pub := priv.Public()

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	Expect(err).ToNot(HaveOccurred())

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,

		DNSNames: dnsNames,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, pub, priv)
	Expect(err).ToNot(HaveOccurred())
	err = pem.Encode(&certPemBytes, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	Expect(err).ToNot(HaveOccurred())

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	Expect(err).ToNot(HaveOccurred())
	err = pem.Encode(&privPemBytes, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	Expect(err).ToNot(HaveOccurred())

	return certPemBytes, privPemBytes
}

// GenerateMACert used for generate MutualAuthCerts
func (s *Scaffold) GenerateMACert(
	t GinkgoTInterface,
	dnsNames []string,
) (
	caCertBytes, serverCertBytes, serverKeyBytes, clientCertBytes, clientKeyBytes bytes.Buffer,
) {
	// CA cert
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(err).ToNot(HaveOccurred())
	caPub := caKey.Public()

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	Expect(err).ToNot(HaveOccurred())

	caTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   dnsNames[0] + "-ca",
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	caTemplate.IsCA = true
	caTemplate.KeyUsage |= x509.KeyUsageCertSign

	caBytes, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, caPub, caKey)
	Expect(err).ToNot(HaveOccurred())
	err = pem.Encode(&caCertBytes, &pem.Block{Type: "CERTIFICATE", Bytes: caBytes})
	Expect(err).ToNot(HaveOccurred())

	// Server cert
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(err).ToNot(HaveOccurred())

	serialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
	Expect(err).ToNot(HaveOccurred())

	serverTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   dnsNames[0],
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	serverBytes, err := x509.CreateCertificate(rand.Reader, &serverTemplate, &caTemplate, &serverKey.PublicKey, caKey)
	Expect(err).ToNot(HaveOccurred())
	err = pem.Encode(&serverCertBytes, &pem.Block{Type: "CERTIFICATE", Bytes: serverBytes})
	Expect(err).ToNot(HaveOccurred())
	serverKeyBytesD, err := x509.MarshalPKCS8PrivateKey(serverKey)
	Expect(err).ToNot(HaveOccurred())
	err = pem.Encode(&serverKeyBytes, &pem.Block{Type: "PRIVATE KEY", Bytes: serverKeyBytesD})
	Expect(err).ToNot(HaveOccurred())

	// Client cert
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(err).ToNot(HaveOccurred())

	serialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
	Expect(err).ToNot(HaveOccurred())

	clientTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   dnsNames[0] + "-client",
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	clientBytes, err := x509.CreateCertificate(rand.Reader, &clientTemplate, &caTemplate, &clientKey.PublicKey, caKey)
	Expect(err).ToNot(HaveOccurred())
	err = pem.Encode(&clientCertBytes, &pem.Block{Type: "CERTIFICATE", Bytes: clientBytes})
	Expect(err).ToNot(HaveOccurred())
	clientKeyBytesD, err := x509.MarshalPKCS8PrivateKey(clientKey)
	Expect(err).ToNot(HaveOccurred())
	err = pem.Encode(&clientKeyBytes, &pem.Block{Type: "PRIVATE KEY", Bytes: clientKeyBytesD})
	Expect(err).ToNot(HaveOccurred())

	return caCertBytes, serverCertBytes, serverKeyBytes, clientCertBytes, clientKeyBytes
}
