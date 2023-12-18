// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package scaffold

import (
	"fmt"

	"github.com/gruntwork-io/terratest/modules/k8s"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const (
	CoreDNSDeployment = "coredns"
)

var (
	_testBackendDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-backend-deployment-e2e-test
spec:
  replicas: %d
  selector:
    matchLabels:
      app: test-backend-deployment-e2e-test
  strategy:
    rollingUpdate:
      maxSurge: 50%%
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: test-backend-deployment-e2e-test
    spec:
      terminationGracePeriodSeconds: 0
      containers:
        - livenessProbe:
            failureThreshold: 3
            initialDelaySeconds: 2
            periodSeconds: 5
            successThreshold: 1
            tcpSocket:
              port: 80
            timeoutSeconds: 2
          readinessProbe:
            failureThreshold: 3
            initialDelaySeconds: 2
            periodSeconds: 5
            successThreshold: 1
            tcpSocket:
              port: 80
            timeoutSeconds: 2
          image: "localhost:5000/test-backend:dev"
          imagePullPolicy: IfNotPresent
          name: test-backend-deployment-e2e-test
          ports:
            - containerPort: 80
              name: "http"
              protocol: "TCP"
            - containerPort: 443
              name: "https"
              protocol: "TCP"
            - containerPort: 8443
              name: "http-mtls"
              protocol: "TCP"
            - containerPort: 50051
              name: "grpc"
              protocol: "TCP"
            - containerPort: 50052
              name: "grpcs"
              protocol: "TCP"
            - containerPort: 50053
              name: "grpc-mtls"
              protocol: "TCP"
`

	_testTimeoutAndRetryDeploymentWithTimeout = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gobackend-deployment2
spec:
  replicas: 2  # You can adjust the number of replicas as needed
  selector:
    matchLabels:
      app: gobackend
  template:
    metadata:
      labels:
        app: gobackend
    spec:
      containers:
      - name: gobackend
        imagePullPolicy: IfNotPresent
        image: "127.0.0.1:5000/test-timeout:dev"
        command: ["/app/gobackend", "fail"]
        ports:
        - containerPort: 9280
`

	_testTimeoutAndRetryDeploymentWithNoTimeout = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gobackend-deployment1
spec:
  replicas: 1  # You can adjust the number of replicas as needed
  selector:
    matchLabels:
      app: gobackend
  template:
    metadata:
      labels:
        app: gobackend
    spec:
      containers:
      - name: gobackend
        imagePullPolicy: IfNotPresent
        image: "127.0.0.1:5000/test-timeout:dev"
        ports:
        - containerPort: 9280
`

	_testTimeoutAndRetryService = `
apiVersion: v1
kind: Service
metadata:
  name: gobackend-service
spec:
  selector:
    app: gobackend
  ports:
    - protocol: TCP
      port: 9280  
      targetPort: 9280
`

	_testBackendService = `
apiVersion: v1
kind: Service
metadata:
  name: test-backend-service-e2e-test
spec:
  selector:
    app: test-backend-deployment-e2e-test
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 80
    - name: https
      port: 443
      protocol: TCP
      targetPort: 443
    - name: http-mtls
      port: 8443
      protocol: TCP
      targetPort: 8443
    - name: grpc
      port: 50051
      protocol: TCP
      targetPort: 50051
    - name: grpcs
      port: 50052
      protocol: TCP
      targetPort: 50052
    - name: grpc-mtls
      port: 50053
      protocol: TCP
      targetPort: 50053
  type: ClusterIP
`
	_udpDeployment = fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
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
          initialDelaySeconds: 2
          periodSeconds: 10
        ports:    
        - name: dns
          containerPort: 53
          protocol: UDP
`, CoreDNSDeployment)
	_udpService = `
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
`
)

func (s *Scaffold) NewServiceForRetryTimeoutTest() error {
	if err := s.CreateResourceFromString(_testTimeoutAndRetryService); err != nil {
		return err
	}
	return nil
}

func (s *Scaffold) NewDeploymentForRetryTimeoutTest() error {
	if err := s.CreateResourceFromString(_testTimeoutAndRetryDeploymentWithTimeout); err != nil {
		return err
	}
	if err := s.CreateResourceFromString(_testTimeoutAndRetryDeploymentWithNoTimeout); err != nil {
		return err
	}
	return nil
}

func (s *Scaffold) newTestBackend() (*corev1.Service, error) {
	backendDeployment := fmt.Sprintf(s.FormatRegistry(_testBackendDeploymentTemplate), 1)
	if err := s.CreateResourceFromString(backendDeployment); err != nil {
		return nil, err
	}
	if err := s.CreateResourceFromString(_testBackendService); err != nil {
		return nil, err
	}
	svc, err := k8s.GetServiceE(s.t, s.kubectlOptions, "test-backend-service-e2e-test")
	if err != nil {
		return nil, err
	}
	return svc, nil
}

// NewCoreDNSService creates a new UDP backend for testing.
func (s *Scaffold) NewCoreDNSService() *corev1.Service {
	err := s.CreateResourceFromString(_udpDeployment)
	assert.Nil(ginkgo.GinkgoT(), err, "failed to create CoreDNS deployment")

	err = s.CreateResourceFromString(_udpService)
	assert.Nil(ginkgo.GinkgoT(), err, "failed to create CoreDNS service")

	s.EnsureNumEndpointsReady(ginkgo.GinkgoT(), "coredns", 1)

	svc, err := k8s.GetServiceE(s.t, s.kubectlOptions, "coredns")
	assert.Nil(ginkgo.GinkgoT(), err, "failed to get CoreDNS service")

	return svc
}
