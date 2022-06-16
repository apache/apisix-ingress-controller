// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
package gateway

import (
	"fmt"
	"net/http"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

func createSSL(s *scaffold.Scaffold) {
	secretName := "test-apisix-tls"
	cert := `-----BEGIN CERTIFICATE-----
MIIFcjCCA1qgAwIBAgIJALDqPppBVXQ3MA0GCSqGSIb3DQEBCwUAMGUxCzAJBgNV
BAYTAkNOMRAwDgYDVQQIDAdKaWFuZ3N1MQ8wDQYDVQQHDAZTdXpob3UxEDAOBgNV
BAoMB2FwaTcuYWkxEDAOBgNVBAsMB2FwaTcuYWkxDzANBgNVBAMMBmp3LmNvbTAg
Fw0yMTA0MDkwNzEyMDBaGA8yMDUxMDQwMjA3MTIwMFowZTELMAkGA1UEBhMCQ04x
EDAOBgNVBAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTEQMA4GA1UECgwHYXBp
Ny5haTEQMA4GA1UECwwHYXBpNy5haTEPMA0GA1UEAwwGancuY29tMIICIjANBgkq
hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwY
Y6sVLGtWoR8gKFSZImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV
0npk/TpZfaCx7zobsfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG
3Fhr0AC067GVYvdwp1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFl
itFFPZkeYG89O/7Ca1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaW
v+xauWnm4hxOzBK7ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415h
M2jMK69aAkQL71xa+66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTl
X4csA+aMHF3v/U7hL/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN
7fRMZKDIHLacSPE0GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXF
w2GqfAFEQbD4wazCh1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVe
v0Yg/OxbbymeTh/hNCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrO
eFuhSMLVblUCAwEAAaMjMCEwHwYDVR0RBBgwFoIIYXBpNi5jb22CCiouYXBpNi5j
b20wDQYJKoZIhvcNAQELBQADggIBAFgeSuMPpriYUrQByO3taiY1+56s+KoAmsyA
LH15n2+thAgorusX2g1Zd7UNBHBtVO8c+ihpQb+2PnmgTTGT70ndpRbV5F6124Mt
Hui/X0kjm76RYd1QKN1VFp0Zo+JVdRa+VhDsXWjO0VetINmFhNINFEJctyeHB8oi
aaDL0wZrevHh47hBqtnrmLl+QVG34aLBRhZ5953leiNvXHUJNaT0nLgf0j9p4esS
b2bx9uP4pFI1T9wcv/TE3K0rQbu/uqGY6MgznXHyi4qIK/I+WCa3fF2UZ5P/5EUM
k2ptQneYkLLUVwwmj8C04bYhYe7Z6jkYYp17ojxIP+ejOY1eiE8SYKNjKinmphrM
5aceqIyfbE4TPqvicNzIggA4yEMPbTA0PC/wqbCf61oMc15hwacdeIaQGiwsM+pf
DTk+GBxp3megx/+0XwTQbguleTlHnaaES+ma0vbl6a1rUK0YAUDUrcfFLv6EFtGD
6EHxFf7gH9sTfc2RiGhKxUhRbyEree+6INAvXy+AymVYmQmKuAFqkDQJ+09bTfm8
bDs+00FijzHFBvC8cIhNffj0qqiv35g+9FTwnE9qpunlrtKG/sMgEXX6m8kL1YQ8
m5DPGhyEZyt5Js2kzzo8TyINPKmrqriYuiD4p4EH13eSRs3ayanQh6ckC7lb+WXq
3IrSc5hO
-----END CERTIFICATE-----`
	key := `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`
	// create kube secret
	err := s.NewKubeTlsSecret(secretName, cert, key)
	assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
	// create ApisixTls resource
	tlsName := "tls-name"
	host := "api6.com"
	err = s.NewApisixTls(tlsName, host, secretName)
	assert.Nil(ginkgo.GinkgoT(), err, "create tls error")

	time.Sleep(10 * time.Second)
	tls, err := s.ListApisixSsl()
	assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
	assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
	assert.Equal(ginkgo.GinkgoT(), cert, tls[0].Cert, "tls cert not expect")
}

var _ = ginkgo.Describe("suite-gateway: TLSRoute", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("Basic with 1 Hosts 1 Rule 1 Match 1 BackendRef", func() {
		createSSL(s)

		host := "api6.com"

		backendSvc, backendPorts := s.DefaultHTTPBackend()
		time.Sleep(time.Second * 6)
		route := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TLSRoute
metadata:
  name: basic-tls-route
spec:
  hostnames: ["%s"]
  rules:
  - backendRefs:
    - name: %s
      port: %d
`, host, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "creating TLSRoute")
		time.Sleep(time.Second * 6)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixStreamRoutesCreated(1), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		sr, err := s.ListApisixStreamRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), sr, 1)
		assert.Equal(ginkgo.GinkgoT(), host, sr[0].SNI)

		client := s.NewAPISIXClientWithTLSOverTCP(host)
		_ = client.GET("/ip").
			Expect().
			Status(http.StatusOK)
		_ = client.GET("/notfound").
			Expect().
			Status(http.StatusNotFound)
	})

	ginkgo.It("Basic with 2 Hosts 1 Rule 1 Match 1 BackendRef", func() {
		createSSL(s)

		host := "api6.com"
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		time.Sleep(time.Second * 6)
		route := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TLSRoute
metadata:
  name: basic-tls-route
spec:
  hostnames: ["anydomain.com", "%s"]
  rules:
  - backendRefs:
    - name: %s
      port: %d
`, host, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "creating TLSRoute")
		time.Sleep(time.Second * 6)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixStreamRoutesCreated(2), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		client := s.NewAPISIXClientWithTLSOverTCP(host)
		_ = client.GET("/ip").
			Expect().
			Status(http.StatusOK)
		_ = client.GET("/notfound").
			Expect().
			Status(http.StatusNotFound)
	})
})
