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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("support ingress https", func() {
	s := scaffold.NewDefaultV2Scaffold()

	rootCA := `-----BEGIN CERTIFICATE-----
MIIF9zCCA9+gAwIBAgIUFKuzAJZgm/fsFS6JDrd+lcpVZr8wDQYJKoZIhvcNAQEL
BQAwgZwxCzAJBgNVBAYTAkNOMREwDwYDVQQIDAhaaGVqaWFuZzERMA8GA1UEBwwI
SGFuZ3pob3UxGDAWBgNVBAoMD0FQSVNJWC1UZXN0LUNBXzEYMBYGA1UECwwPQVBJ
U0lYX0NBX1JPT1RfMRUwEwYDVQQDDAxBUElTSVguUk9PVF8xHDAaBgkqhkiG9w0B
CQEWDXRlc3RAdGVzdC5jb20wHhcNMjEwNTI3MTMzNjI4WhcNMjIwNTI3MTMzNjI4
WjCBnDELMAkGA1UEBhMCQ04xETAPBgNVBAgMCFpoZWppYW5nMREwDwYDVQQHDAhI
YW5nemhvdTEYMBYGA1UECgwPQVBJU0lYLVRlc3QtQ0FfMRgwFgYDVQQLDA9BUElT
SVhfQ0FfUk9PVF8xFTATBgNVBAMMDEFQSVNJWC5ST09UXzEcMBoGCSqGSIb3DQEJ
ARYNdGVzdEB0ZXN0LmNvbTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIB
ALJR0lQW/IBqQTE/Oa0Pi4LlmlYUSGnqtFNqiZyOF0PjVzNeqoD9JDPiM1QRyC8p
NCd5L/QhtUIMMx0RlDI9DkJ3ALIWdrPIZlwpveDJf4KtW7cz+ea46A6QQwB6xcyV
xWnqEBkiea7qrEE8NakZOMjgkqkN2/9klg6XyA5FWfvszxtuIHtjcy2Kq8bMC0jd
k7CqEZe4ct6s2wlcI8t8s9prvMDm8gcX66x4Ah+C2/W+C3lTpMDgGqRqSPyCW7na
Wgn0tWmTSf1iybwYMydhC+zpM1QJLvfDyqjp1wJhziR5ttVe2Xc+tDC24s+u16yZ
R93IO0M4lLNjvEKJcMltXyRzrcjvLXOhw3KirSHNL1KfrBEl74lb+DV5eU4pIFCj
cu18gms5FBYs9tpLujwpHDc2MU+zCvRmSPvUA4yCyoXqom3uiSo3g3ymW9IM8dC8
+Bd1GdM6JbpBukvQybc5TQXo1M75I9iEoQa5tQxAfQ/dfwMjOK7skogowBouOuLv
BEFKy3Vd57IWWZXC4p/74M6N4fGYTgHY5FQE3R4Y2phk/eaEm1jS1UPuC98QuTfL
rGuFOIBmK5euOm8uT5m9hnrouG2ZcxEdzHYfjsGDGrLzA0FLu+wtMNBKM4NhsNCa
d+fycLg7jgxWhaLvD5DfkV7WFQlz5LUceYIwYOyhD/chAgMBAAGjLzAtMAwGA1Ud
EwQFMAMBAf8wHQYDVR0RBBYwFIISbXRscy5odHRwYmluLmxvY2FsMA0GCSqGSIb3
DQEBCwUAA4ICAQCNtBmoAc5tv3H38sj9qhTmabvp9RIzZYrQSEcN+A2i3a8FVYAM
YaugZDXDcTycoWn6rcgblUDneow3NiqZ57yYZmN+e4mE3+Q1sGepV7LoRkHDUT8w
jAJndcZ/xxJmgH6B7dImTAPsvLGR7E7gffMH+aKCdnkG9x5Vm+cuBwSEBndiHGfr
yw5cXO6cMUq8M6zJrk2V+1BAucXW2rgLTWy6UTTGD56cgUtbStRO6muOKoElDLbW
mSj2rNv/evakQkV8dgKVRFgh2NQKYKpXmveMaE6xtFFf/dd9OhDFjUh/ksxn94FT
xj/wkhXCEPl+t7tENhr2tNyLbCOVcFzqoi7IyoWKxxZQfvArfj4SmahK8E/BXB/T
4PEmn8kZAxaW7RmGcaekm8MTqGlhCJ3tVJAI2vcYRdd9ZHbXE1jr/4xj0I/Lzglo
O8v5fd4zHyV1SuZ5AH3XbUd7ndl9yDoN2WSqK9Nd9bws3yrf+GwjJAT1InnDvLg1
stWM8I+9FZiDFL255/+iAN0jYcGu9i4TNvC+o6qQ1p85i1OHPJZu6wtUWMgDJN46
uwW3ZLh9sZV6OnhbQJBQaUmcgaPJUQqbXNQmpmpc0NUjET/ltFRZ2hlyvvpf7wwF
2DLY1HRAknQ69DuT6xpYz1aKZqrlkbCWlMMvdosOg6f7+4NxdYJ/rBeS6Q==
-----END CERTIFICATE-----
`

	serverCertSecret := `server-secret`
	serverCert := `-----BEGIN CERTIFICATE-----
MIIF/TCCA+WgAwIBAgIUBbUP7Gk0WAb/JhYYcBBgZEgmhbEwDQYJKoZIhvcNAQEL
BQAwgZwxCzAJBgNVBAYTAkNOMREwDwYDVQQIDAhaaGVqaWFuZzERMA8GA1UEBwwI
SGFuZ3pob3UxGDAWBgNVBAoMD0FQSVNJWC1UZXN0LUNBXzEYMBYGA1UECwwPQVBJ
U0lYX0NBX1JPT1RfMRUwEwYDVQQDDAxBUElTSVguUk9PVF8xHDAaBgkqhkiG9w0B
CQEWDXRlc3RAdGVzdC5jb20wHhcNMjEwNTI3MTMzNjI5WhcNMjIwNTI3MTMzNjI5
WjCBpTELMAkGA1UEBhMCQ04xETAPBgNVBAgMCFpoZWppYW5nMREwDwYDVQQHDAhI
YW5nemhvdTEcMBoGA1UECgwTQVBJU0lYLVRlc3QtU2VydmVyXzEXMBUGA1UECwwO
QVBJU0lYX1NFUlZFUl8xGzAZBgNVBAMMEm10bHMuaHR0cGJpbi5sb2NhbDEcMBoG
CSqGSIb3DQEJARYNdGVzdEB0ZXN0LmNvbTCCAiIwDQYJKoZIhvcNAQEBBQADggIP
ADCCAgoCggIBAMZRPG8gUrOGM4awnV6D8Ds0Xb6jVbiGkx+1YsvPx5oIE4AswJ0l
y6zqhBFnpQozFG63KfsCA6U36/Dty3rIbJzsbO7YaOMJItoiQgqdqF2nrmPpmpCQ
uLGKaVvriRCD55NEmFQPshlRfcU5/EEreNKbRve3zEKHRpCDBZ2Myvrpt3CCVy6D
MbLllbjUvaedrnQxlmI5d7x3UCe4Eunq8vn7c0p4frA1n8TxbX0M4Yr9g3YEEqCv
Q3/9jU4hI5CvujCp+u79EavJZfsaEv3RYgHkoEh7q+OEkUajWXKj4WynizraWsHv
+LvK9pfI300p1HSKK4FqonvW79anRNbK+8BqV4Wt5aBeFU/rW2jHtJxcl1OLRrrh
wftCP5W7vSjvJes5wPDZjDEyv8WP1Aa6yWeGHHtIwrAHPr7556F/JAQS6IPBQQ5U
X45DD2aNXME9xZKdBtyMovItjZm31UUsvoF+YtpAOmbEkX4lMznUO3XZJjM5HWSq
WYyzmFsw+pJEwhXRo4GfSfCHfiZQ6imTLJ7OsZzo9bvmxyfI0kVLe3h3iCe+qYeT
f5AJU6v5vv3thCtfgbxYP2P8b+0MIrfr05e6dCDXbIri1z+nprzWYmyCrZ6H4hVk
DzMktFUlkqenvnsJ2iOV2AZw0Hlk2bwe4zSumzqoIp8Yk/kxbfxhQqr5AgMBAAGj
LDAqMAkGA1UdEwQCMAAwHQYDVR0RBBYwFIISbXRscy5odHRwYmluLmxvY2FsMA0G
CSqGSIb3DQEBCwUAA4ICAQCDDfETCEpWB/KRQZo2JF8n4NEDTeraQ85M3H5luJHp
NdJO4oYq3n8B149ep4FcEYdO20pV+TMeMNWXMfhoRIpGx95JrLuLg6qnw6eNdErn
YupHMC2OEoEWVcmI052LDJcXuKsTXQvU4OeEL2dX4OtNJ+mRODLyh40cg7dA3wry
kGLiprRlLQtiX8pSDG30qPZexL1LcFzBQajriG05QUrJW6Rvbq1JTIlyp7E1T86f
Xljq0Hdzqxy+FklYcAW5ZAxgkQlMmVdTlvDXlD/hQLEQIHGHiW6OMLp8WrnJP6b0
D2HqWmOwuEzqSgXSK0N89rpiWP1FKCpyiKVcsawDNfOpePVuthommVEc2PxacyHf
UCC9V0MS0ZzQ63Tnz2Tja8C6/kMyVX226KQKhcoDxDoS0mQrI96/VXcglwP5hMjF
joth1T1qRVu6+NQmvFPaNjbzWJ+j1R99bnYGihPeLdqDSUxNosV3ULG8T4aN6+f8
hApiqg2dkLJQr8zWf6vWXMlREdPEovb2F7P0Lfn0VeOSRXDUIdqcoRHONi8bWMRs
fjPtGW00Tv8Jg21c9vc8Zh/t1w3wkXQhqYiBMt5cYe6WueIlXdjF7ikSRWAHTwlw
Bfzv/vMftLnbySPovCzQ1PF55D01EWRk0o6PRwUDLfzTQoV+bDKx82LxKtZBtQEX
uw==
-----END CERTIFICATE-----
`
	serverKey := `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`
	ginkgo.It("should support ingress v1beta1 with tls", func() {
		// create secrets
		err := s.NewSecret(serverCertSecret, serverCert, serverKey)
		assert.Nil(ginkgo.GinkgoT(), err, "create server cert secret error")

		// create ingress
		host := "mtls.httpbin.local"
		// create route
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: httpbin-ingress-https
  annotations:
    kubernetes.io/ingress.class: apisix
spec:
  tls:
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: /*
        backend:
          serviceName: %s
          servicePort: %d
`, host, serverCertSecret, host, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
		time.Sleep(10 * time.Second)

		apisixRoutes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err, "list routes error")
		assert.Len(ginkgo.GinkgoT(), apisixRoutes, 1, "route number not expect")

		apisixSsls, err := s.ListApisixSsl()
		assert.Nil(ginkgo.GinkgoT(), err, "list SSLs error")
		assert.Len(ginkgo.GinkgoT(), apisixSsls, 1, "SSL number should be 1")
		assert.Equal(ginkgo.GinkgoT(), id.GenID(s.Namespace()+"_httpbin-ingress-https-tls"), apisixSsls[0].ID, "SSL name")
		assert.Equal(ginkgo.GinkgoT(), apisixSsls[0].Snis, []string{host}, "SSL configuration")

		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM([]byte(rootCA))
		assert.True(ginkgo.GinkgoT(), ok, "Append cert to CA pool")

		s.NewAPISIXHttpsClientWithCertificates(host, true, caCertPool, []tls.Certificate{}).
			GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK)
	})

	ginkgo.It("should support ingress v1 with tls", func() {
		// create secrets
		err := s.NewSecret(serverCertSecret, serverCert, serverKey)
		assert.Nil(ginkgo.GinkgoT(), err, "create server cert secret error")

		// create ingress
		host := "mtls.httpbin.local"
		// create route
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-ingress-https
  annotations:
    kubernetes.io/ingress.class: apisix
spec:
  tls:
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: %d
`, host, serverCertSecret, host, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
		time.Sleep(10 * time.Second)

		apisixRoutes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err, "list routes error")
		assert.Len(ginkgo.GinkgoT(), apisixRoutes, 1, "route number not expect")

		apisixSsls, err := s.ListApisixSsl()
		assert.Nil(ginkgo.GinkgoT(), err, "list SSLs error")
		assert.Len(ginkgo.GinkgoT(), apisixSsls, 1, "SSL number should be 1")
		assert.Equal(ginkgo.GinkgoT(), id.GenID(s.Namespace()+"_httpbin-ingress-https-tls"), apisixSsls[0].ID, "SSL name")
		assert.Equal(ginkgo.GinkgoT(), apisixSsls[0].Snis, []string{host}, "SSL configuration")

		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM([]byte(rootCA))
		assert.True(ginkgo.GinkgoT(), ok, "Append cert to CA pool")

		s.NewAPISIXHttpsClientWithCertificates(host, true, caCertPool, []tls.Certificate{}).
			GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK)
	})

	ginkgo.It("should support ingress v1 with kube style tls secret", func() {
		// create secrets
		err := s.NewKubeTlsSecret(serverCertSecret, serverCert, serverKey)
		assert.Nil(ginkgo.GinkgoT(), err, "create server cert secret error")

		// create ingress
		host := "mtls.httpbin.local"
		// create route
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-ingress-https
  annotations:
    kubernetes.io/ingress.class: apisix
spec:
  tls:
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: %d
`, host, serverCertSecret, host, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
		time.Sleep(10 * time.Second)

		apisixRoutes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err, "list routes error")
		assert.Len(ginkgo.GinkgoT(), apisixRoutes, 1, "route number not expect")

		apisixSsls, err := s.ListApisixSsl()
		assert.Nil(ginkgo.GinkgoT(), err, "list SSLs error")
		assert.Len(ginkgo.GinkgoT(), apisixSsls, 1, "SSL number should be 1")
		assert.Equal(ginkgo.GinkgoT(), id.GenID(s.Namespace()+"_httpbin-ingress-https-tls"), apisixSsls[0].ID, "SSL name")
		assert.Equal(ginkgo.GinkgoT(), apisixSsls[0].Snis, []string{host}, "SSL configuration")

		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM([]byte(rootCA))
		assert.True(ginkgo.GinkgoT(), ok, "Append cert to CA pool")

		s.NewAPISIXHttpsClientWithCertificates(host, true, caCertPool, []tls.Certificate{}).
			GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK)
	})
})

var _ = ginkgo.Describe("support ingress.networking/v1", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("path exact match", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		// Exact path, doesn't match /ip/aha
		_ = s.NewAPISIXClient().GET("/ip/aha").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound)
		// Mismatched host
		_ = s.NewAPISIXClient().GET("/ip/aha").WithHeader("Host", "a.httpbin.org").Expect().Status(http.StatusNotFound)
	})

	ginkgo.It("path prefix match", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /status
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/status/500").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusInternalServerError)
		_ = s.NewAPISIXClient().GET("/status/504").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusGatewayTimeout)
		_ = s.NewAPISIXClient().GET("/statusaaa").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound).Body().Contains("404 Route Not Found")
		// Mismatched host
		_ = s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "a.httpbin.org").Expect().Status(http.StatusNotFound)
	})

	ginkgo.It("path regex match", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/use-regex: 'true'
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /anything/.*/ok
        pathType: ImplementationSpecific
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/anything/aaa/ok").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/anything/aaa/notok").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound).Body().Contains("404 Route Not Found")
		_ = s.NewAPISIXClient().GET("/statusaaa").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound).Body().Contains("404 Route Not Found")
		// Mismatched host
		_ = s.NewAPISIXClient().GET("/anything/aaa/ok").WithHeader("Host", "a.httpbin.org").Expect().Status(http.StatusNotFound)
	})
})

var _ = ginkgo.Describe("support ingress.networking/v1beta1", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("path exact match", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		// Exact path, doesn't match /ip/aha
		_ = s.NewAPISIXClient().GET("/ip/aha").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound)
		// Mismatched host
		_ = s.NewAPISIXClient().GET("/ip/aha").WithHeader("Host", "a.httpbin.org").Expect().Status(http.StatusNotFound)
	})

	ginkgo.It("path prefix match", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /status
        pathType: Prefix
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/status/500").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusInternalServerError)
		_ = s.NewAPISIXClient().GET("/status/504").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusGatewayTimeout)
		_ = s.NewAPISIXClient().GET("/statusaaa").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound).Body().Contains("404 Route Not Found")
		// Mismatched host
		_ = s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "a.httpbin.org").Expect().Status(http.StatusNotFound)
	})

	ginkgo.It("path regex match", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/use-regex: 'true'
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /anything/.*/ok
        pathType: ImplementationSpecific
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/anything/aaa/ok").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/anything/aaa/notok").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound).Body().Contains("404 Route Not Found")
		_ = s.NewAPISIXClient().GET("/statusaaa").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound).Body().Contains("404 Route Not Found")
		// Mismatched host
		_ = s.NewAPISIXClient().GET("/anything/aaa/ok").WithHeader("Host", "a.httpbin.org").Expect().Status(http.StatusNotFound)
	})
})

var _ = ginkgo.Describe("support ingress.extensions/v1beta1", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("path exact match", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
  name: ingress-ext-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		// Exact path, doesn't match /ip/aha
		_ = s.NewAPISIXClient().GET("/ip/aha").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound)
		// Mismatched host
		_ = s.NewAPISIXClient().GET("/ip/aha").WithHeader("Host", "a.httpbin.org").Expect().Status(http.StatusNotFound)
	})

	ginkgo.It("path prefix match", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /status
        pathType: Prefix
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/status/500").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusInternalServerError)
		_ = s.NewAPISIXClient().GET("/status/504").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusGatewayTimeout)
		_ = s.NewAPISIXClient().GET("/statusaaa").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound).Body().Contains("404 Route Not Found")
		// Mismatched host
		_ = s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "a.httpbin.org").Expect().Status(http.StatusNotFound)
	})

	ginkgo.It("path regex match", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/use-regex: 'true'
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /anything/.*/ok
        pathType: ImplementationSpecific
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/anything/aaa/ok").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/anything/aaa/notok").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound).Body().Contains("404 Route Not Found")
		_ = s.NewAPISIXClient().GET("/statusaaa").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound).Body().Contains("404 Route Not Found")
		// Mismatched host
		_ = s.NewAPISIXClient().GET("/anything/aaa/ok").WithHeader("Host", "a.httpbin.org").Expect().Status(http.StatusNotFound)
	})
})

var _ = ginkgo.Describe("support ingress.networking/v1 with headless service backend", func() {
	s := scaffold.NewDefaultScaffold()

	const _httpHeadlessService = `
apiVersion: v1
kind: Service
metadata:
  name: httpbin-headless-service-e2e-test
spec:
  selector:
    app: httpbin-deployment-e2e-test
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 80
  type: ClusterIP
  clusterIP: None
`

	var (
		backendSvc  string
		backendPort []int32
	)
	ginkgo.BeforeEach(func() {
		err := s.CreateResourceFromString(_httpHeadlessService)
		assert.Nil(ginkgo.GinkgoT(), err, "creating headless service")
		svc, err := s.GetServiceByName("httpbin-headless-service-e2e-test")
		assert.Nil(ginkgo.GinkgoT(), err, "get headless service")
		getSvcNameAndPorts := func(svc *corev1.Service) (string, []int32) {
			var ports []int32
			for _, p := range svc.Spec.Ports {
				ports = append(ports, p.Port)
			}
			return svc.Name, ports
		}

		backendSvc, backendPort = getSvcNameAndPorts(svc)
	})

	ginkgo.It("path exact match", func() {
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		// Exact path, doesn't match /ip/aha
		_ = s.NewAPISIXClient().GET("/ip/aha").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound)
		// Mismatched host
		_ = s.NewAPISIXClient().GET("/ip/aha").WithHeader("Host", "a.httpbin.org").Expect().Status(http.StatusNotFound)
	})

	ginkgo.It("path prefix match", func() {
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /status
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/status/500").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusInternalServerError)
		_ = s.NewAPISIXClient().GET("/status/504").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusGatewayTimeout)
		_ = s.NewAPISIXClient().GET("/statusaaa").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound).Body().Contains("404 Route Not Found")
		// Mismatched host
		_ = s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "a.httpbin.org").Expect().Status(http.StatusNotFound)
	})

	ginkgo.It("path regex match", func() {
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/use-regex: 'true'
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /anything/.*/ok
        pathType: ImplementationSpecific
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/anything/aaa/ok").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/anything/aaa/notok").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound).Body().Contains("404 Route Not Found")
		_ = s.NewAPISIXClient().GET("/statusaaa").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound).Body().Contains("404 Route Not Found")
		// Mismatched host
		_ = s.NewAPISIXClient().GET("/anything/aaa/ok").WithHeader("Host", "a.httpbin.org").Expect().Status(http.StatusNotFound)
	})
})
