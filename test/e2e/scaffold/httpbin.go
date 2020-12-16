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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (s *Scaffold) newHTTPBIN() (*appsv1.Deployment, *corev1.Service, error) {
	desc := &deploymentDesc{
		name:            "httpbin-deloyment-e2e-test",
		namespace:       s.namespace,
		image:           s.opts.HTTPBINImage,
		ports:           []int32{80},
		replica:         1,
		probe:           &corev1.Probe{
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(80),
				},
			},
			InitialDelaySeconds: 2,
			TimeoutSeconds:      2,
			PeriodSeconds:       5,
		},
	}

	d, err := ensureDeployment(s.clientset, newDeployment(desc))
	if err != nil {
		return nil, nil, err
	}

	svcDesc := &serviceDesc{
		name:      "httpbin-service-e2e-test",
		namespace: s.namespace,
		selector:  d.Spec.Selector.MatchLabels,
		ports:     []corev1.ServicePort{
			{
				Name: "http",
				Protocol: corev1.ProtocolTCP,
				Port: 80,
				TargetPort: intstr.FromInt(80),
			},
		},
	}
	svc, err := ensureService(s.clientset, newService(svcDesc))
	if err != nil {
		return nil, nil, err
	}

	return d, svc, err
}
