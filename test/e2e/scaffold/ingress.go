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
	"fmt"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	_serviceAccount     = "ingress-apisix-e2e-test-service-account"
	_clusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ingress-apisix-e2e-test-clusterrolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: ingress-apisix-e2e-test-service-account
  namespace: %s
`
	_ingressAPISIXDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ingress-apisix-controller-deployment-e2e-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ingress-apisix-controller-deployment-e2e-test
  strategy:
    rollingUpdate:
      maxSurge: 50%
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: ingress-apisix-controller-deployment-e2e-test
    spec:
      terminationGracePeriodSeconds: 0
      containers:
        - livenessProbe:
            failureThreshold: 3
            initialDelaySeconds: 1
            periodSeconds: 2
            successThreshold: 1
            tcpSocket:
              port: 8080
            timeoutSeconds: 2
          readinessProbe:
            failureThreshold: 3
            initialDelaySeconds: 1
            periodSeconds: 2
            successThreshold: 1
            tcpSocket:
              port: 8080
            timeoutSeconds: 2
          image: "viewking/apisix-ingress-controller:dev"
          imagePullPolicy: IfNotPresent
          name: ingress-apisix-controller-deployment-e2e-test
          ports:
            - containerPort: 8080
              name: "http"
              protocol: "TCP"
          command:
            - /ingress-apisix/apisix-ingress-controller
            - ingress
            - --log-level
            - debug
            - --log-output
            - stdout
            - --http-listen
            - :8080
            - --apisix-base-url
            - http://apisix-service-e2e-test:9180/apisix/admin
      serviceAccount: ingress-apisix-e2e-test-service-account
`
)

func (s *Scaffold) newIngressAPISIXController() error {
	if err := k8s.CreateServiceAccountE(s.t, s.kubectlOptions, _serviceAccount); err != nil {
		return err
	}

	crb := fmt.Sprintf(_clusterRoleBinding, s.namespace)
	if err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, crb); err != nil {
		return err
	}
	if err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, _ingressAPISIXDeployment); err != nil {
		return err
	}
	return nil
}

func (s *Scaffold) waitAllIngressControllerPodsAvailable() error {
	opts := metav1.ListOptions{
		LabelSelector: "app=ingress-apisix-controller-deployment-e2e-test",
	}
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(s.t, s.kubectlOptions, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			ginkgo.GinkgoT().Log("no ingress-apisix-controller pods created")
			return false, nil
		}
		for _, item := range items {
			for _, cond := range item.Status.Conditions {
				if cond.Type != corev1.PodReady {
					continue
				}
				if cond.Status != "True" {
					return false, nil
				}
			}
		}
		return true, nil
	}
	return waitExponentialBackoff(condFunc)
}
