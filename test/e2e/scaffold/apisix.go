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
	"strings"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	_apisixConfigMap = `
kind: ConfigMap
apiVersion: v1
metadata:
  name: apisix-gw-config.yaml
data:
  config.yaml: |
%s
`
	_apisixDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apisix-deployment-e2e-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: apisix-deployment-e2e-test
  strategy:
    rollingUpdate:
      maxSurge: 50%
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: apisix-deployment-e2e-test
    spec:
      terminationGracePeriodSeconds: 0
      containers:
        - livenessProbe:
            failureThreshold: 3
            initialDelaySeconds: 1
            periodSeconds: 2
            successThreshold: 1
            tcpSocket:
              port: 9080
            timeoutSeconds: 2
          readinessProbe:
            failureThreshold: 3
            initialDelaySeconds: 1
            periodSeconds: 2
            successThreshold: 1
            tcpSocket:
              port: 9080
            timeoutSeconds: 2
          image: "localhost:5000/apache/apisix:dev"
          imagePullPolicy: IfNotPresent
          name: apisix-deployment-e2e-test
          ports:
            - containerPort: 9080
              name: "http"
              protocol: "TCP"
            - containerPort: 9180
              name: "http-admin"
              protocol: "TCP"
            - containerPort: 9443
              name: "https"
              protocol: "TCP"
          volumeMounts:
            - mountPath: /usr/local/apisix/conf/config.yaml
              name: apisix-config-yaml-configmap
              subPath: config.yaml
      volumes:
        - configMap:
            name: apisix-gw-config.yaml
          name: apisix-config-yaml-configmap
`
	_apisixService = `
apiVersion: v1
kind: Service
metadata:
  name: apisix-service-e2e-test
spec:
  selector:
    app: apisix-deployment-e2e-test
  ports:
    - name: http
      port: 9080
      protocol: TCP
      targetPort: 9080
    - name: http-admin
      port: 9180
      protocol: TCP
      targetPort: 9180
    - name: https
      port: 9443
      protocol: TCP
      targetPort: 9443
    - name: tcp
      port: 9100
      protocol: TCP
      targetPort: 9100
    - name: udp
      port: 9200
      protocol: UDP
      targetPort: 9200
    - name: http-control
      port: 9090
      protocol: TCP
      targetPort: 9090
  type: NodePort
`
)

func (s *Scaffold) newAPISIX() (*corev1.Service, error) {
	data, err := s.renderConfig(s.opts.APISIXConfigPath)
	if err != nil {
		return nil, err
	}
	data = indent(data)
	configData := fmt.Sprintf(_apisixConfigMap, data)
	if err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, configData); err != nil {
		return nil, err
	}
	if err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, _apisixDeployment); err != nil {
		return nil, err
	}
	if err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, _apisixService); err != nil {
		return nil, err
	}

	svc, err := k8s.GetServiceE(s.t, s.kubectlOptions, "apisix-service-e2e-test")
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func indent(data string) string {
	list := strings.Split(data, "\n")
	for i := 0; i < len(list); i++ {
		list[i] = "    " + list[i]
	}
	return strings.Join(list, "\n")
}

func (s *Scaffold) waitAllAPISIXPodsAvailable() error {
	opts := metav1.ListOptions{
		LabelSelector: "app=apisix-deployment-e2e-test",
	}
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(s.t, s.kubectlOptions, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			ginkgo.GinkgoT().Log("no apisix pods created")
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
