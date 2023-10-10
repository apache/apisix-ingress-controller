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

package annotations

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-annotations: annotations.networking/v1 websocket", func() {
	s := scaffold.NewDefaultScaffold()
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
    image: localhost:5000/echo-server:dev
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
		err := s.CreateResourceFromString(s.FormatRegistry(resources))
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(5 * time.Second)

		ing := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/enable-websocket: 'true'
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /echo
        pathType: ImplementationSpecific
        backend:
          service:
            name: websocket-server-service
            port:
              number: 48733
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
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

var _ = ginkgo.Describe("suite-annotations: annotations.networking/v1beta1 with websocket", func() {
	s := scaffold.NewDefaultScaffold()
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
    image: localhost:5000/echo-server:dev
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
		err := s.CreateResourceFromString(s.FormatRegistry(resources))
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(5 * time.Second)

		ing := `
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: ingress-v1beta1
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/enable-websocket: 'true'
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /echo
        pathType: Exact
        backend:
          serviceName: websocket-server-service
          servicePort: 48733
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
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
