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
	"errors"
	"net"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/api7/ingress-controller/pkg/types"
)

const (
	_serviceAccount     = "ingress-apisix-e2e-test-service-account"
	_clusterRoleBinding = "ingress-apisix-e2e-test-clusterrolebinding"
	// TODO Customize cluster role, do not use cluster-admin
	_clusterRole = "apisix-view-clusterrole"
)

func (s *Scaffold) newIngressAPISIXController() (*appsv1.Deployment, error) {
	if err := createServiceAccount(s.clientset, _serviceAccount, s.namespace); err != nil {
		return nil, err
	}
	if err := createClusterRoleBinding(s.clientset, _clusterRoleBinding, s.namespace, _serviceAccount, _clusterRole); err != nil {
		return nil, err
	}

	var (
		cmd  []string
		port int
	)

	zeroTime := types.TimeDuration{
		Duration: time.Duration(0),
	}

	cmd = append(cmd, "/ingress-apisix/apisix-ingress-controller")
	cmd = append(cmd, "ingress")
	if s.opts.IngressAPISIXConfig.LogLevel != "" {
		cmd = append(cmd, "--log-level", s.opts.IngressAPISIXConfig.LogLevel)
	}
	if s.opts.IngressAPISIXConfig.LogOutput != "" {
		cmd = append(cmd, "--log-output", s.opts.IngressAPISIXConfig.LogOutput)
	}
	if s.opts.IngressAPISIXConfig.HTTPListen != "" {
		cmd = append(cmd, "--http-listen", s.opts.IngressAPISIXConfig.HTTPListen)
	}
	if s.opts.IngressAPISIXConfig.Kubernetes.Kubeconfig != "" {
		cmd = append(cmd, "--kubeconfig", s.opts.IngressAPISIXConfig.Kubernetes.Kubeconfig)
	}
	if s.opts.IngressAPISIXConfig.Kubernetes.ResyncInterval != zeroTime {
		cmd = append(cmd, "--resync-interval", s.opts.IngressAPISIXConfig.Kubernetes.ResyncInterval.String())
	}
	if s.opts.IngressAPISIXConfig.APISIX.BaseURL == "" {
		return nil, errors.New("missing APISIX base URL configuration")
	}
	cmd = append(cmd, "--apisix-base-url", s.opts.IngressAPISIXConfig.APISIX.BaseURL)

	if s.opts.IngressAPISIXConfig.HTTPListen != "" {
		_, rawPort, err := net.SplitHostPort(s.opts.IngressAPISIXConfig.HTTPListen)
		if err != nil {
			return nil, err
		}
		port, err = strconv.Atoi(rawPort)
		if err != nil {
			return nil, err
		}
	} else {
		port = 8080 // Default port of ingress apisix api server.
	}

	desc := &deploymentDesc{
		name:      "ingress-apisix-controller-deployment-e2e-test",
		namespace: s.namespace,
		image:     s.opts.IngressAPISIXImage,
		ports:     []int32{8080},
		replica:   1,
		command:   cmd,
		probe: &corev1.Probe{
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(port),
				},
			},
			InitialDelaySeconds: 2,
			TimeoutSeconds:      2,
			PeriodSeconds:       5,
		},
		serviceAccount: _serviceAccount,
	}

	return ensureDeployment(s.clientset, newDeployment(desc))
}
