// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
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

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
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
	_api6tlsTemplate = `
apiVersion: %s
kind: ApisixTls
metadata:
  name: %s
spec:
  %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`
	_api6tlsWithClientCATemplate = `
apiVersion: %s
kind: ApisixTls
metadata:
  name: %s
spec:
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
  client:
    caSecret:
      name: %s
      namespace: %s
    depth: 10
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

// NewApisixTls new a ApisixTls CRD
func (s *Scaffold) NewApisixTls(name, host, secretName string, ingressClassName ...string) error {
	var ingClassName string
	if len(ingressClassName) > 0 {
		ingClassName = "ingressClassName: " + ingressClassName[0]
	}
	tls := fmt.Sprintf(_api6tlsTemplate, s.opts.ApisixResourceVersion, name, ingClassName, host, secretName, s.kubectlOptions.Namespace)
	if err := s.CreateResourceFromString(tls); err != nil {
		return err
	}
	return nil
}

// NewApisixTlsWithClientCA new a ApisixTls CRD
func (s *Scaffold) NewApisixTlsWithClientCA(name, host, secretName, clientCASecret string) error {
	tls := fmt.Sprintf(_api6tlsWithClientCATemplate, s.opts.ApisixResourceVersion, name, host, secretName, s.kubectlOptions.Namespace, clientCASecret, s.kubectlOptions.Namespace)
	if err := s.CreateResourceFromString(tls); err != nil {
		return err
	}
	return nil
}

// DeleteApisixTls remove ApisixTls CRD
func (s *Scaffold) DeleteApisixTls(name string, host, secretName string) error {
	tls := fmt.Sprintf(_api6tlsTemplate, s.opts.ApisixResourceVersion, name, "", host, secretName, s.kubectlOptions.Namespace)
	if err := k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, tls); err != nil {
		return err
	}
	return nil
}

func (s *Scaffold) GenerateCert(t ginkgo.GinkgoTInterface, dnsNames []string) (certPemBytes, privPemBytes bytes.Buffer) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	pub := priv.Public()

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	assert.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,

		DNSNames: dnsNames,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, pub, priv)
	assert.NoError(t, err)
	err = pem.Encode(&certPemBytes, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	assert.NoError(t, err)

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	assert.NoError(t, err)
	err = pem.Encode(&privPemBytes, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	assert.NoError(t, err)

	return
}
