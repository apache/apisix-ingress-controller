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
package scaffold

import (
	"context"
	"fmt"

	"github.com/gruntwork-io/terratest/modules/k8s"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/gateway/versioned"
)

var (
	_udpRouteTemplate = `
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: UDPRoute
metadata:
  name: %s
spec:
  rules:
  - backendRefs:
    - name: %s
      port: %d
`
)

func (s *Scaffold) getGatewayClientset() (*gatewayclientset.Clientset, error) {
	var err error
	var config *rest.Config

	if s.kubectlOptions.InClusterAuth {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		kubeConfigPath, err := s.kubectlOptions.GetConfigPath(s.t)
		if err != nil {
			return nil, err
		}
		config, err = k8s.LoadApiClientConfigE(kubeConfigPath, s.kubectlOptions.ContextName)
		if err != nil {
			config, err = rest.InClusterConfig()
			if err != nil {
				return nil, err
			}
		}
	}

	clientset, err := gatewayclientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil

}

func (s *Scaffold) CreateUDPRoute(name string, backendName string, backendPort int32) *gatewayv1alpha2.UDPRoute {
	udpRoute := fmt.Sprintf(_udpRouteTemplate, name, backendName, backendPort)
	err := s.CreateResourceFromString(udpRoute)
	assert.Nil(ginkgo.GinkgoT(), err, "create UDPRoute failed")
	client, err := s.getGatewayClientset()
	assert.Nil(ginkgo.GinkgoT(), err, "get GatewayClientset failed")
	route, err := client.GatewayV1alpha2().UDPRoutes(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
	assert.Nil(ginkgo.GinkgoT(), err, "get UDPRoute failed")
	return route
}
