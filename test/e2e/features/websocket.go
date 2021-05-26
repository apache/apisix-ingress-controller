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
	"net/http"
	"net/url"
	"time"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/gorilla/websocket"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = ginkgo.Describe("websocket", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("sanity", func() {
		resources := `
apiVersion: v1
kind: Pod
metadata:
  name: websocket-server
  labels:
    app: websocket-server
spec:
  containers:
  - name: websocket-server
    image: localhost:5000/jmalloc/echo-server:latest
    ports:
    - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: websocket-server-service
spec:
  selector:
    app: websocket-server
  ports:
  - name: ws
    port: 48733
    protocol: TCP
    targetPort: 8080
`
		err := s.CreateResourceFromString(resources)
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(5 * time.Second)

		ar := `
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /echo
   websocket: true
   backends:
   - serviceName: websocket-server-service
     servicePort: 48733
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		dialer := websocket.Dialer{}
		u := url.URL{
			Scheme: "ws",
			Host:   s.APISIXGatewayServiceEndpoint(),
			Path:   "/echo",
		}
		header := http.Header{
			"Host": []string{"httpbin.org"},
		}
		conn, resp, err := dialer.Dial(u.String(), header)
		assert.Nil(ginkgo.GinkgoT(), err, "websocket handshake failure")
		assert.Equal(ginkgo.GinkgoT(), resp.StatusCode, http.StatusSwitchingProtocols)

		assert.Nil(ginkgo.GinkgoT(), conn.WriteMessage(websocket.TextMessage, []byte("hello, I'm gorilla")), "writing message")
		msgType, buf, err := conn.ReadMessage()
		assert.Nil(ginkgo.GinkgoT(), err, "reading message")
		assert.Equal(ginkgo.GinkgoT(), string(buf), "Request served by websocket-server")
		msgType, buf, err = conn.ReadMessage()
		assert.Nil(ginkgo.GinkgoT(), err, "reading message")
		assert.Equal(ginkgo.GinkgoT(), msgType, websocket.TextMessage)
		assert.Equal(ginkgo.GinkgoT(), string(buf), "hello, I'm gorilla")
		assert.Nil(ginkgo.GinkgoT(), conn.Close(), "closing ws connection")
	})
})
