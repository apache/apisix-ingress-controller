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
          image: "bitnami/etcd:3.4.14-debian-10-r0"
          imagePullPolicy: IfNotPresent
          name: etcd-deployment-e2e-test
          ports:
            - containerPort: 2379
              name: "etcd"
              protocol: "TCP"
`

	etcdService = `
apiVersion: v1
kind: Service
metadata:
  name: etcd-service-e2e-test
spec:
  selector:
    app: etcd-deployment-e2e-test
  ports:
    - name: etcd-client
      port: 2379
      protocol: TCP
      targetPort: 2379
  type: ClusterIP
`
)

func (s *Scaffold) newEtcd() (*corev1.Service, error) {
	if err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, etcdDeployment); err != nil {
		return nil, err
	}
	if err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, etcdService); err != nil {
		return nil, err
	}
	svc, err := k8s.GetServiceE(s.t, s.kubectlOptions, "etcd-service-e2e-test")
	if err != nil {
		return nil, err
	}
	s.EtcdServiceFQDN = fmt.Sprintf("etcd-service-e2e-test.%s.svc.cluster.local", svc.Namespace)
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

func (s *Scaffold) etcdSelector() metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: "app=etcd-deployment-e2e-test",
	}
}
