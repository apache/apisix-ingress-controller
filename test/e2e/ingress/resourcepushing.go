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
	"fmt"
	"net/http"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("ApisixRoute Testing", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("create and then scale upstream pods to 2 ", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
	apiVersion: apisix.apache.org/v1
	kind: ApisixRoute
	metadata:
	name: httpbin-route
	spec:
	rules:
	- host: httpbin.com
	  http:
	    paths:
	    - backend:
	        serviceName: %s
	        servicePort: %d
	      path: /ip
	`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))

		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(2), "scaling number of httpbin instancess")
		assert.Nil(ginkgo.GinkgoT(), s.WaitAllHTTPBINPoddsAvailable(), "waiting for all httpbin pods ready")
		// TODO When ingress controller can feedback the lifecycle of CRDs to the
		// status field, we can poll it rather than sleeping.
		time.Sleep(10 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "list upstreams error")
		assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 2, "upstreams nodes not expect")
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
	})

	ginkgo.It("create and then remove", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
	apiVersion: apisix.apache.org/v1
	kind: ApisixRoute
	metadata:
	 name: httpbin-route
	spec:
	 rules:
	 - host: httpbin.com
	   http:
	     paths:
	     - backend:
	         serviceName: %s
	         servicePort: %d
	       path: /ip
	`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute), "creating ApisixRoute")
		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

		// remove
		assert.Nil(ginkgo.GinkgoT(), s.RemoveResourceByString(apisixRoute))

		// TODO When ingress controller can feedback the lifecycle of CRDs to the
		// status field, we can poll it rather than sleeping.
		time.Sleep(10 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "list upstreams error")
		assert.Len(ginkgo.GinkgoT(), ups, 0, "upstreams nodes not expect")
	})

	ginkgo.It("create route with SSL ", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 rules:
 - host: test.com
   http:
     paths:
     - backend:
         serviceName: %s
         servicePort: %d
       path: /ip
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))

		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(2), "scaling number of httpbin instancess")
		assert.Nil(ginkgo.GinkgoT(), s.WaitAllHTTPBINPoddsAvailable(), "waiting for all httpbin pods ready")
		// TODO When ingress controller can feedback the lifecycle of CRDs to the
		// status field, we can poll it rather than sleeping.
		time.Sleep(10 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "list upstreams error")
		assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 2, "upstreams nodes not expect")
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "test.com").Expect().Status(http.StatusOK).Body().Raw()

		secretName := "test-atls"
		cert := `-----BEGIN CERTIFICATE-----
MIIDRjCCAi4CCQDpQavPMiEljDANBgkqhkiG9w0BAQsFADBkMQswCQYDVQQGEwJD
TjEQMA4GA1UECAwHSmlhbmdzdTEPMA0GA1UEBwwGU3V6aG91MQ0wCwYDVQQKDAR0
ZXN0MRAwDgYDVQQLDAdzZWN0aW9uMREwDwYDVQQDDAh0ZXN0LmNvbTAgFw0yMTAx
MjUxNTU4MTVaGA8yMDUxMDExODE1NTgxNVowZDELMAkGA1UEBhMCQ04xEDAOBgNV
BAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTENMAsGA1UECgwEdGVzdDEQMA4G
A1UECwwHc2VjdGlvbjERMA8GA1UEAwwIdGVzdC5jb20wggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDAUxxDYS4kItHE52CBbbVPdes4E48rRE+UyESkWRX+
2owaJQv0t84ruPkxrIso41K6il+5kV/QQdOSF1dchIX/A2c4bNuVJocqGocdH76m
JeJI0y2ZPFn+I4GmYfiHb40igxLs1cOF9yoU0NoEbOWuEOKEsEgzBMG9scUAC6/z
3yiuHQxpb9SF9vW5QxSKZ+4kjfXwe+Iyg97/xRNPEZCqM2aMG8DnrrIq6vKxySvL
WuQ7xB+0VDSSrhfEK/36+IlPteKFY4ftIbGoG/A3BjSIiWL54BTKghSt8YewdHwF
SqM48T3ORqqsko8XP/4izCY5e3XmUjRYUcbKaO+Ne6o3AgMBAAEwDQYJKoZIhvcN
AQELBQADggEBAB5ebWgexQmFpx/4dlMhwNNRTTG5TkT1jRv9dnJD+WfLQP4w+5Eq
PLVdK7/ZY21KN95GF4hga1i93V3vV6UzXzTA5AcN59qYONdbK86pU4En67k8a9w9
Rx0N1EWDIb6RRPN/RHglMi1iX1aiUWUPaUZHMAKibAb6QSJp5eUUO+xO1dV6SJmE
5vy7snz0HCh2cKLQ38olU9c4CzZ64I16tMktsegnqdZU3UMmGGXRfUqOTDJ/Ojey
U5V3ClQvXX5N43pnvUCdUdoKfGRekUugtq4jh6fNAB4XzxIgPvnKgIAbKJu1Hmlq
3bjOw7adAvt5MP5VHzGrQk8Zd0Udi6pjHDs=
-----END CERTIFICATE-----`
		key := `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAwFMcQ2EuJCLRxOdggW21T3XrOBOPK0RPlMhEpFkV/tqMGiUL
9LfOK7j5MayLKONSuopfuZFf0EHTkhdXXISF/wNnOGzblSaHKhqHHR++piXiSNMt
mTxZ/iOBpmH4h2+NIoMS7NXDhfcqFNDaBGzlrhDihLBIMwTBvbHFAAuv898orh0M
aW/Uhfb1uUMUimfuJI318HviMoPe/8UTTxGQqjNmjBvA566yKurysckry1rkO8Qf
tFQ0kq4XxCv9+viJT7XihWOH7SGxqBvwNwY0iIli+eAUyoIUrfGHsHR8BUqjOPE9
zkaqrJKPFz/+IswmOXt15lI0WFHGymjvjXuqNwIDAQABAoIBACiD2ZpgKIY4R5SB
YZUidWWN48VmaWyl8VXYco1krvuHMqh3UXN2HRqc1vId9Rrh+JWPfObstxB6LYXD
IQY+bLPyFZaPaBqdiS/Xcssx2snJhUfuJNb5HcQp2rAgR2jQmRzCHASEa7IXBWhp
LuRrxF7M88scD7mfsOizQFroG2L2K4akVdiceuj6aVl1b4nUO2F77lyCf8cZfLn7
rqplxVok1w+Gm7xuDiOGwc+d2sbX3UvgOF+e6HvOwmcplDA2ivToiwMzy0Cx455J
9evR1J3VLLDeaUkJP/KaxifN9A5XFbX8jPX4DwFSbbgeqNhx/Vj31ev0xlk1KjjZ
D2WQqoECgYEA/yzlyePbHzkD7KI3jQDjBJv8vQ7JUZPVw0eUJlOfFz1uhtginPhe
ECYb3vUKUdhLCQUyihc2MZ0bjfi/h0XwXHfgFeS6EWI0q2WkDm+qGoEPlAIgYbKm
LWHVoCAZmo+T8DdH2H+UZM92DsoEaEQH5DSw7OBEDdl3xVtZwI10tyECgYEAwPI3
qKMhwKijCVCfvbIFR3nR8F8kJuWjUdY4/4970w/K5XtGoBPGElIm+0qHZsldu58M
CRmzPbBsHSxT9lvUAvJlxBpJcNKNtZiDmNYDWTEW5Gb7/1nS6R3/oaW20Vv+DOsl
ZDjojDnfSoSOmgDVIrNw66yqro/YfuSrkWA5rlcCgYA+ou+48gR8kotDD8KhCwGu
xPdyFOoX6zkCmVRlYAtiMgMqeG1uqIy2XBRlUzL4SiaJDUyNlwsHfLAh1lh1RRau
LALGfQGreLbDB80QehqALQP86dS3BppB84zzpE2Eog/HXFp3a0GqyT4KfU49pc1m
GAUB8D7kQ2hh+n16hX6L4QKBgA0eavpkXR8kWDGB8dqMCB/cAJI/Zc3fP0OJNUbr
Epg/MqR3xU2NCqKkQ1JCtwIeHulq3v6faLiBDljNcsgFZlzs7k5vGx84sbnvLMNv
ibq+w7ez6N5r1RNUntT2139Uqelm85vk4qrmJHCEos2F0PgTC1J64wALd8To92Fj
EYjxAoGAX5EXRcMkxd5joBIrpJHNdt8FTF+xYdXsJVVKRQx1nPNWtYZTOyV6Q1n+
b3ZwsKwPbcl7PnObbrH6F2K8Wh0F1HkPf/ASuYz7TNqQ2F4kMb+WVqzHNhSQQZU4
6sTYxzl0LQKZBQZn+UK6awMpAfuZgR1XBSG8v/n0Gd49WRrZCcE=
-----END RSA PRIVATE KEY-----`
		// create secret
		err = s.NewSecret(secretName, cert, key)
		assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
		// create ApisixTls resource
		tlsName := "tls-name"
		host := "test.com"
		err = s.NewApisixTls(tlsName, host, secretName)
		assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
		// check ssl in APISIX
		time.Sleep(10 * time.Second)
		tls, err := s.ListApisixTls()
		assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
		assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
		// FIXME
		s.NewAPISIXHttpsClient().GET("/ip").WithHeader("Host", "test.com").Expect().Status(http.StatusOK).Body().Raw()
	})
})
