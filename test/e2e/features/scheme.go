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
	"context"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("choose scheme", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.FIt("grpc", func() {
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
  ports:
    - port: 50051
      scheme: grpc
`))
		time.Sleep(2 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)

		err = s.CreateResourceFromString(`
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
 name: grpc-route
spec:
 rules:
 - host: grpc.local
   http:
     paths:
     - backend:
         serviceName: grpc_server_service
         servicePort: 50051
       path: /helloworld.Greeter/SayHello
`)
		assert.Nil(ginkgo.GinkgoT(), err)

		ep, err := s.GetAPISIXEndpoint()
		assert.Nil(ginkgo.GinkgoT(), err)
		grpcConn, err := grpc.DialContext(context.TODO(), ep,
			grpc.WithBlock(),
			grpc.WithInsecure(),
		)
		assert.Nil(ginkgo.GinkgoT(), err)
		cli := helloworld.NewGreeterClient(grpcConn)
		hr := &helloworld.HelloRequest{
			Name: "Alex",
		}
		resp, err := cli.SayHello(context.TODO(), hr)
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Equal(ginkgo.GinkgoT(), resp.Message, "Alex")
	})
})
