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
	"io/ioutil"
	"net/http"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/apache/apisix-ingress-controller/test/e2e/testbackend/client"
)

var _ = ginkgo.Describe("suite-ingress: ApisixUpstreams mTLS test", func() {
	clientSecret := `client-secret`

	f, err := ioutil.ReadFile("testbackend/tls/client.pem")
	assert.NoError(ginkgo.GinkgoT(), err, "read client cert")
	clientCert := string(f)

	f, err = ioutil.ReadFile("testbackend/tls/client.key")
	assert.NoError(ginkgo.GinkgoT(), err, "read client key")
	clientKey := string(f)

	s := scaffold.NewDefaultV2Scaffold()
	ginkgo.It("create ApisixUpstreams with http mTLS", func() {
		// create client secret
		err := s.NewSecret(clientSecret, clientCert, clientKey)
		assert.NoError(ginkgo.GinkgoT(), err, "create client cert secret")

		err = s.NewApisixUpstreamsWithMTLS("test-backend-service-e2e-test", "https", clientSecret)
		assert.NoError(ginkgo.GinkgoT(), err, "create ApisixUpstreams with client secret")
		err = s.CreateResourceFromString(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: upstream-is-mtls.httpbin.local
spec:
  http:
  - backends:
    - serviceName: test-backend-service-e2e-test
      servicePort: 8443
    match:
      hosts:
      - 'upstream-is-mtls.httpbin.local'
      paths:
      - /*
    plugins:
    - name: proxy-rewrite
      enable: true
      config:
        host: 'e2e.apisix.local'
    name: upstream-is-mtls
`)
		assert.NoError(ginkgo.GinkgoT(), err, "create ApisixRoute for backend that require mTLS")
		time.Sleep(10 * time.Second)
		apisixRoutes, err := s.ListApisixRoutes()
		assert.NoError(ginkgo.GinkgoT(), err, "list routes error")
		assert.Len(ginkgo.GinkgoT(), apisixRoutes, 1, "route number not expect")

		s.NewAPISIXClient().GET("/hello").WithHeader("Host", "upstream-is-mtls.httpbin.local").Expect().Status(http.StatusOK).Body().Raw()
	})

	ginkgo.It("create ApisixUpstreams with grpc mTLS", func() {
		// create grpc secret for apisix grpc route
		grpcSecret := `grpc-secret`
		f, err := ioutil.ReadFile("testbackend/tls/server.pem")
		assert.NoError(ginkgo.GinkgoT(), err, "read server cert")
		serverCert := string(f)

		f, err = ioutil.ReadFile("testbackend/tls/server.key")
		assert.NoError(ginkgo.GinkgoT(), err, "read server key")
		serverKey := string(f)

		err = s.NewSecret(grpcSecret, serverCert, serverKey)
		assert.NoError(ginkgo.GinkgoT(), err, "create server cert secret")

		// create client secret
		err = s.NewSecret(clientSecret, clientCert, clientKey)
		assert.NoError(ginkgo.GinkgoT(), err, "create client cert secret")

		err = s.NewApisixUpstreamsWithMTLS("test-backend-service-e2e-test", "grpcs", clientSecret)
		assert.NoError(ginkgo.GinkgoT(), err, "create ApisixUpstreams with client secret")

		assert.NoError(ginkgo.GinkgoT(), s.CreateResourceFromString(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
 name: grpcs-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - e2e.apisix.local
      paths:
      - /helloworld.Greeter/SayHello
    backends:
    -  serviceName: test-backend-service-e2e-test
       servicePort: 50053
`))

		assert.NoError(ginkgo.GinkgoT(), s.CreateResourceFromString(fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixTls
metadata:
  name: grpc-secret
spec:
  hosts:
    - "e2e.apisix.local"
  secret:
    name: grpc-secret
    namespace: %s
`, s.Namespace())))

		assert.NoError(ginkgo.GinkgoT(), err, "create ApisixRoute for backend that require mTLS")
		time.Sleep(10 * time.Second)
		apisixRoutes, err := s.ListApisixRoutes()
		assert.NoError(ginkgo.GinkgoT(), err, "list routes error")
		assert.Len(ginkgo.GinkgoT(), apisixRoutes, 1, "route number not expect")

		ca, err := ioutil.ReadFile("testbackend/tls/ca.pem")
		assert.NoError(ginkgo.GinkgoT(), err, "read ca cert")
		assert.NoError(ginkgo.GinkgoT(), client.RequestHello(s.GetAPISIXHTTPSEndpoint(), ca), "request apisix using grpc protocol")
	})
})
