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
	"context"
	"fmt"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-ingress-resource: ApisixRoute stream Testing", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("stream tcp proxy", func() {
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
apiVersion: apisix.apache.org/v2
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
			host := "httpbun.org"
			_, err = r.LookupIPAddr(context.Background(), host)
			assert.Nil(ginkgo.GinkgoT(), err, "dns query error")
		})
	}
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
		hostA := "a.test.com"
		secretA := "server-secret-a"
		serverCertA, serverKeyA := s.GenerateCert(ginkgo.GinkgoT(), []string{hostA})
		err := s.NewSecret(secretA, serverCertA.String(), serverKeyA.String())
		assert.Nil(ginkgo.GinkgoT(), err, "create server cert secret 'a' error")

		hostB := "b.test.com"
		secretB := "server-secret-b"
		serverCertB, serverKeyB := s.GenerateCert(ginkgo.GinkgoT(), []string{hostB})
		err = s.NewSecret(secretB, serverCertB.String(), serverKeyB.String())
		assert.Nil(ginkgo.GinkgoT(), err, "create server cert secret 'b' error")

		// create ApisixTls resource
		err = s.NewApisixTls("tls-server-a", hostA, secretA)
		assert.Nil(ginkgo.GinkgoT(), err, "create ApisixTls 'a' error")

		err = s.NewApisixTls("tls-server-b", hostB, secretB)
		assert.Nil(ginkgo.GinkgoT(), err, "create ApisixTls 'b' error")

		// check ssl in APISIX
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixTlsCreated(2))

		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-tcp-route
spec:
  stream:
  - name: ruleA
    protocol: TCP
    match:
      ingressPort: 9110
      host: %s
    backend:
      serviceName: %s
      servicePort: %d
  - name: ruleB
    protocol: TCP
    match:
      ingressPort: 9110
      host: %s
    backend:
      serviceName: %s
      servicePort: %d
`, hostA, backendSvc, backendSvcPort[0], hostB, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

		err = s.EnsureNumApisixStreamRoutesCreated(2)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		// SNI: hostA should return certificate with hostA in Subject Alternative Name only
		connA, err := s.DialTLSOverTcp(hostA)
		assert.Nil(ginkgo.GinkgoT(), err)
		defer connA.Close()

		assert.Equal(ginkgo.GinkgoT(), connA.ConnectionState().PeerCertificates[0].DNSNames, []string{hostA})

		// SNI: hostB should return certificate with hostB in Subject Alternative Name only
		connB, err := s.DialTLSOverTcp(hostB)
		assert.Nil(ginkgo.GinkgoT(), err)
		defer connB.Close()

		assert.Equal(ginkgo.GinkgoT(), connB.ConnectionState().PeerCertificates[0].DNSNames, []string{hostB})
	})
})
