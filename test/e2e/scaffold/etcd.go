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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EtcdServiceName = "etcd-service-e2e-test"
)

var (
	etcdDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: etcd-deployment-e2e-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: etcd-deployment-e2e-test
  strategy:
    rollingUpdate:
      maxSurge: 50%
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: etcd-deployment-e2e-test
    spec:
      terminationGracePeriodSeconds: 0
      containers:
        - env:
          - name: ALLOW_NONE_AUTHENTICATION
            value: "yes"
          livenessProbe:
            failureThreshold: 3
            initialDelaySeconds: 1
            periodSeconds: 2
            successThreshold: 1
            tcpSocket:
              port: 2379
            timeoutSeconds: 2
          readinessProbe:
            failureThreshold: 3
            initialDelaySeconds: 1
            periodSeconds: 2
            successThreshold: 1
            tcpSocket:
              port: 2379
            timeoutSeconds: 2
          image: "localhost:5000/etcd:dev"
          imagePullPolicy: IfNotPresent
          name: etcd-deployment-e2e-test
          ports:
            - containerPort: 2379
              name: "etcd"
              protocol: "TCP"
`

	etcdService = fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: etcd-deployment-e2e-test
  ports:
    - name: etcd-client
      port: 2379
      protocol: TCP
      targetPort: 2379
  type: ClusterIP
`, EtcdServiceName)
)

func (s *Scaffold) newEtcd() (*corev1.Service, error) {
	if err := s.CreateResourceFromString(s.FormatRegistry(etcdDeployment)); err != nil {
		return nil, err
	}
	if err := s.CreateResourceFromString(etcdService); err != nil {
		return nil, err
	}
	svc, err := k8s.GetServiceE(s.t, s.kubectlOptions, EtcdServiceName)
	if err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *Scaffold) waitAllEtcdPodsAvailable() error {
	opts := metav1.ListOptions{
		LabelSelector: "app=etcd-deployment-e2e-test",
	}
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(s.t, s.kubectlOptions, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			ginkgo.GinkgoT().Log("no etcd pods created")
			return false, nil
		}
		for _, item := range items {
			foundPodReady := false
			for _, cond := range item.Status.Conditions {
				if cond.Type != corev1.PodReady {
					continue
				}
				foundPodReady = true
				if cond.Status != "True" {
					return false, nil
				}
			}
			if !foundPodReady {
				return false, nil
			}
		}
		return true, nil
	}
	return waitExponentialBackoff(condFunc)
}

func (s *Scaffold) labelSelector(label string) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: label,
	}
}
