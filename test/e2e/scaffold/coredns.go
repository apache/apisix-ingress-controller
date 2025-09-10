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
