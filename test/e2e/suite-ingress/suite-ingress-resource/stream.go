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

package ingress

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-ingress-resource: ApisixRoute stream Testing", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("stream tcp proxy", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: httpbin-tcp-route
spec:
  stream:
  - name: rule1
    protocol: TCP
    match:
      ingressPort: 9100
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

			err := s.EnsureNumApisixStreamRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			sr, err := s.ListApisixStreamRoutes()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), sr, 1)
			assert.Equal(ginkgo.GinkgoT(), sr[0].ServerPort, int32(9100))

			resp := s.NewAPISIXClientWithTCPProxy().GET("/ip").Expect()
			resp.Body().Contains("origin")

			resp = s.NewAPISIXClientWithTCPProxy().GET("/get").WithHeader("x-my-header", "x-my-value").Expect()
			resp.Body().Contains("x-my-value")
		})
		ginkgo.It("stream udp proxy", func() {
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns
spec:
  replicas: 1
  selector:
    matchLabels:
      app: coredns
  template:
    metadata:
      labels:
        app: coredns
    spec:
      containers:
      - name: coredns
        image: coredns/coredns:1.8.4
        livenessProbe:
          tcpSocket:
            port: 53
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          tcpSocket:
            port: 53
          initialDelaySeconds: 5
          periodSeconds: 10
        ports:    
        - name: dns
          containerPort: 53
          protocol: UDP
`))
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(`
kind: Service
apiVersion: v1
metadata:
  name: coredns
spec:
  selector:
    app: coredns
  type: ClusterIP
  ports:
  - port: 53
    targetPort: 53
    protocol: UDP
`))

			s.EnsureNumEndpointsReady(ginkgo.GinkgoT(), "coredns", 1)

			apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: httpbin-udp-route
spec:
  stream:
  - name: rule1
    protocol: UDP
    match:
      ingressPort: 9200
    backend:
      serviceName: coredns
      servicePort: 53
`)
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

			err := s.EnsureNumApisixStreamRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			sr, err := s.ListApisixStreamRoutes()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), sr, 1)
			assert.Equal(ginkgo.GinkgoT(), sr[0].ServerPort, int32(9200))
			// test dns query
			r := s.DNSResolver()
			host := "httpbin.org"
			_, err = r.LookupIPAddr(context.Background(), host)
			assert.Nil(ginkgo.GinkgoT(), err, "dns query error")
		})
	}
	ginkgo.Describe("suite-ingress-resource: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultV2beta3Scaffold())
	})
	ginkgo.Describe("suite-ingress-resource: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold())
	})
})

var _ = ginkgo.Describe("suite-ingress-resource: ApisixRoute stream Testing SNI with v2", func() {
	s := scaffold.NewDefaultV2Scaffold()

	ginkgo.It("stream route with sni when set host", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-tcp-route
spec:
  stream:
  - name: rule1
    protocol: TCP
    match:
      ingressPort: 9100
      host: a.test.com
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

		err := s.EnsureNumApisixStreamRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		sr, err := s.ListApisixStreamRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), sr, 1)
		assert.Equal(ginkgo.GinkgoT(), sr[0].ServerPort, int32(9100))
		assert.Equal(ginkgo.GinkgoT(), sr[0].SNI, "a.test.com")
	})

	ginkgo.It("no sni in stream route when not set host", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-tcp-route
spec:
  stream:
  - name: rule1
    protocol: TCP
    match:
      ingressPort: 9100
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

		err := s.EnsureNumApisixStreamRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		sr, err := s.ListApisixStreamRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), sr, 1)
		assert.Equal(ginkgo.GinkgoT(), sr[0].ServerPort, int32(9100))
		assert.Equal(ginkgo.GinkgoT(), sr[0].SNI, "")
	})

	ginkgo.It("stream tcp proxy with SNI", func() {
		// create secrets
		host := "a.test.com"
		secret := "server-secret"
		serverCert, serverKey := generateCert(ginkgo.GinkgoT(), []string{host})
		err := s.NewSecret(secret, serverCert.String(), serverKey.String())
		assert.Nil(ginkgo.GinkgoT(), err, "create server cert secret error")

		// create ApisixTls resource
		err = s.NewApisixTls("tls-server", host, secret)
		assert.Nil(ginkgo.GinkgoT(), err, "create ApisixTls error")

		// check ssl in APISIX
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixTlsCreated(1))

		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-tcp-route
spec:
  stream:
  - name: rule1
    protocol: TCP
    match:
      ingressPort: 9110
      host: %s
    backend:
      serviceName: %s
      servicePort: %d
`, host, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

		err = s.EnsureNumApisixStreamRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		client := s.NewAPISIXClientWithTLSOverTCP(host)
		client.GET("/ip").WithHost(host).Expect().Status(http.StatusOK)
	})
})

func generateCert(t ginkgo.GinkgoTInterface, dnsNames []string) (certPemBytes, privPemBytes bytes.Buffer) {
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
