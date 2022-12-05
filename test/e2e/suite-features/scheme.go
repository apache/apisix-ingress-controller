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

package features

import (
	"os"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/apache/apisix-ingress-controller/test/e2e/testbackend/client"
)

var _ = ginkgo.Describe("suite-features: choose scheme", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		ginkgo.It("grpc", func() {
			err := s.CreateResourceFromString(`
apiVersion: v1
kind: Pod
metadata:
  name: grpc-server
  labels:
    app: grpc-server
spec:
  containers:
  - name: grcp-server
    image: docker.io/tokers/grpc_server_example:latest
---
apiVersion: v1
kind: Service
metadata:
  name: grpc-server-service
spec:
  selector:
    app: grpc-server
  ports:
  - name: grpc
    port: 50051
    protocol: TCP
    targetPort: 50051
`)
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixUpstream
metadata:
  name: grpc-server-service
spec:
  portLevelSettings:
    - port: 50051
      scheme: grpc
`))
			err = s.CreateVersionedApisixResource(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
 name: grpc-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - grpc.local
      paths:
      - /helloworld.Greeter/SayHello
    backends:
    -  serviceName: grpc-server-service
       servicePort: 50051
`)
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(2 * time.Second)
			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), ups, 1)
			assert.Equal(ginkgo.GinkgoT(), ups[0].Scheme, "grpc")

			// TODO enable the following test cases once APISIX supports HTTP/2 in plain.
			// ep, err := s.GetAPISIXEndpoint()
			// assert.Nil(ginkgo.GinkgoT(), err)
			// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			// defer cancel()
			// dialFunc := func(ctx context.Context, addr string) (net.Conn, error) {
			//	return (&net.Dialer{}).DialContext(ctx, "tcp", addr)
			// }
			//
			// grpcConn, err := grpc.DialContext(ctx, ep,
			//	grpc.WithBlock(),
			//	grpc.WithInsecure(),
			//	grpc.WithContextDialer(dialFunc),
			// )
			// assert.Nil(ginkgo.GinkgoT(), err)
			// defer grpcConn.Close()
			// cli := helloworld.NewGreeterClient(grpcConn)
			// hr := &helloworld.HelloRequest{
			//	Name: "Alex",
			// }
			// resp, err := cli.SayHello(context.TODO(), hr)
			// assert.Nil(ginkgo.GinkgoT(), err)
			// assert.Equal(ginkgo.GinkgoT(), resp.Message, "Alex")
		})

		ginkgo.It("grpcs", func() {
			grpcSecret := `grpc-secret`
			f, err := os.ReadFile("testbackend/tls/server.pem")
			assert.NoError(ginkgo.GinkgoT(), err, "read server cert")
			serverCert := string(f)

			f, err = os.ReadFile("testbackend/tls/server.key")
			assert.NoError(ginkgo.GinkgoT(), err, "read server key")
			serverKey := string(f)

			err = s.NewSecret(grpcSecret, serverCert, serverKey)
			assert.NoError(ginkgo.GinkgoT(), err, "create server cert secret")

			assert.NoError(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixUpstream
metadata:
  name: test-backend-service-e2e-test
spec:
  scheme: grpcs
`))

			assert.NoError(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(`
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
       servicePort: 50052
`))

			assert.NoError(ginkgo.GinkgoT(), s.NewApisixTls("grpc-secret", "e2e.apisix.local", "grpc-secret"))

			time.Sleep(2 * time.Second)
			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), ups, 1)
			assert.Equal(ginkgo.GinkgoT(), ups[0].Scheme, "grpcs")

			ca, err := os.ReadFile("testbackend/tls/ca.pem")
			assert.NoError(ginkgo.GinkgoT(), err, "read ca cert")
			assert.NoError(ginkgo.GinkgoT(), client.RequestHello(s.GetAPISIXHTTPSEndpoint(), ca), "request apisix using grpc protocol")
		})
	}

	ginkgo.Describe("suite-features: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultV2beta3Scaffold)
	})
	ginkgo.Describe("suite-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
