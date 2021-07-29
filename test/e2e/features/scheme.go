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
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("choose scheme", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(`
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: grpc-server-service
spec:
  portLevelSettings:
    - port: 50051
      scheme: grpc
`))
		err = s.CreateResourceFromString(`
apiVersion: apisix.apache.org/v2alpha1
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
    backend:
       serviceName: grpc-server-service
       servicePort: 50051
`)
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(2 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Scheme, "grpc")

		// TODO enable the following test cases once APISIX supports HTTP/2 in plain.
		//ep, err := s.GetAPISIXEndpoint()
		//assert.Nil(ginkgo.GinkgoT(), err)
		//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		//defer cancel()
		//dialFunc := func(ctx context.Context, addr string) (net.Conn, error) {
		//	return (&net.Dialer{}).DialContext(ctx, "tcp", addr)
		//}
		//
		//grpcConn, err := grpc.DialContext(ctx, ep,
		//	grpc.WithBlock(),
		//	grpc.WithInsecure(),
		//	grpc.WithContextDialer(dialFunc),
		//)
		//assert.Nil(ginkgo.GinkgoT(), err)
		//defer grpcConn.Close()
		//cli := helloworld.NewGreeterClient(grpcConn)
		//hr := &helloworld.HelloRequest{
		//	Name: "Alex",
		//}
		//resp, err := cli.SayHello(context.TODO(), hr)
		//assert.Nil(ginkgo.GinkgoT(), err)
		//assert.Equal(ginkgo.GinkgoT(), resp.Message, "Alex")
	})
})
