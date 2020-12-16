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
	"io/ioutil"
	"net/url"
	"strings"
	"text/template"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	_apisixConfigConfigMap = "apisix-gw-config.yaml"
)

func (s *Scaffold) apisixServiceURL() string {
	// TODO remove these hacks after we create NodePort serivce.
	// For now, we forward service port to local.
	u := url.URL{
		Scheme: "http",
		Host:   "127.0.0.1",
	}
	return u.String()
}

func (s *Scaffold) readAPISIXConfigFromFile(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	t := template.Must(template.New(path).Parse(string(data)))
	if err := t.Execute(&buf, s); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *Scaffold) newAPISIX() (*appsv1.Deployment, *corev1.Service, error) {
	data, err := s.readAPISIXConfigFromFile(s.opts.APISIXConfigPath)
	if err != nil {
		return nil, nil, err
	}
	defaultData, err := s.readAPISIXConfigFromFile(s.opts.APISIXDefaultConfigPath)
	if err != nil {
		return nil, nil, err
	}
	cmData := map[string]string{
		"config.yaml":         data,
		"config-default.yaml": defaultData,
	}
	if err := createConfigMap(s.clientset, _apisixConfigConfigMap, s.namespace, cmData); err != nil {
		return nil, nil, err
	}
	desc := &deploymentDesc{
		name:      "apisix-deployment-e2e-test",
		namespace: s.namespace,
		image:     s.opts.APISIXImage,
		ports:     []int32{9080, 9180},
		replica:   1,
		probe: &corev1.Probe{
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(9080),
				},
			},
			InitialDelaySeconds: 2,
			TimeoutSeconds:      2,
			PeriodSeconds:       5,
		},
		volumeMounts: []corev1.VolumeMount{
			{
				Name:      "apisix-config-yaml-configmap",
				MountPath: "/usr/local/apisix/conf/config.yaml",
				SubPath:   "config.yaml",
			},
			{
				Name:      "apisix-config-yaml-configmap",
				MountPath: "/usr/local/apisix/conf/config-default.yaml",
				SubPath:   "config-default.yaml",
			},
		},
		volumes: []corev1.Volume{
			{
				Name: "apisix-config-yaml-configmap",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: _apisixConfigConfigMap,
						},
					},
				},
			},
		},
	}

	d, err := ensureDeployment(s.clientset, newDeployment(desc))
	if err != nil {
		return nil, nil, err
	}

	svcDesc := &serviceDesc{
		name:      "apisix-service-e2e-test",
		namespace: s.namespace,
		selector:  d.Spec.Selector.MatchLabels,
		ports: []corev1.ServicePort{
			{
				Protocol:   corev1.ProtocolTCP,
				Name:       "apisix-dp",
				Port:       9080,
				TargetPort: intstr.FromInt(9080),
			},
			{
				Protocol:   corev1.ProtocolTCP,
				Name:       "apisix-cp",
				Port:       9180,
				TargetPort: intstr.FromInt(9180),
			},
		},
	}

	svc, err := ensureService(s.clientset, newService(svcDesc))
	if err != nil {
		return nil, nil, err
	}
	return d, svc, nil
}
