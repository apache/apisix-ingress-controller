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
package crd

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"

	"github.com/stretchr/testify/assert"

	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
)

func TestTranslateUpstreamConfig(t *testing.T) {
	tr := &translator{}

	au := &configv1.ApisixUpstreamConfig{
		LoadBalancer: nil,
		Scheme:       apisixv1.SchemeGRPC,
	}

	ups, err := tr.TranslateUpstreamConfig(au)
	assert.Nil(t, err, "checking upstream config translating")
	assert.Equal(t, ups.Type, apisixv1.LbRoundRobin)
	assert.Equal(t, ups.Scheme, apisixv1.SchemeGRPC)

	au = &configv1.ApisixUpstreamConfig{
		LoadBalancer: &configv1.LoadBalancer{
			Type:   apisixv1.LbConsistentHash,
			HashOn: apisixv1.HashOnHeader,
			Key:    "user-agent",
		},
		Scheme: apisixv1.SchemeHTTP,
	}
	ups, err = tr.TranslateUpstreamConfig(au)
	assert.Nil(t, err, "checking upstream config translating")
	assert.Equal(t, ups.Type, apisixv1.LbConsistentHash)
	assert.Equal(t, ups.Key, "user-agent")
	assert.Equal(t, ups.HashOn, apisixv1.HashOnHeader)
	assert.Equal(t, ups.Scheme, apisixv1.SchemeHTTP)

	au = &configv1.ApisixUpstreamConfig{
		LoadBalancer: &configv1.LoadBalancer{
			Type:   apisixv1.LbConsistentHash,
			HashOn: apisixv1.HashOnHeader,
			Key:    "user-agent",
		},
		Scheme: "dns",
	}
	_, err = tr.TranslateUpstreamConfig(au)
	assert.Error(t, err, &translateError{
		field:  "scheme",
		reason: "invalid value",
	})

	au = &configv1.ApisixUpstreamConfig{
		LoadBalancer: &configv1.LoadBalancer{
			Type: "hash",
		},
	}
	_, err = tr.TranslateUpstreamConfig(au)
	assert.Error(t, err, &translateError{
		field:  "loadbalancer.type",
		reason: "invalid value",
	})

	au = &configv1.ApisixUpstreamConfig{
		LoadBalancer: &configv1.LoadBalancer{
			Type:   apisixv1.LbConsistentHash,
			HashOn: "arg",
		},
	}
	_, err = tr.TranslateUpstreamConfig(au)
	assert.Error(t, err, &translateError{
		field:  "loadbalancer.hashOn",
		reason: "invalid value",
	})
}

func TestTranslateUpstreamNodes(t *testing.T) {
	tr := &translator{}
	svc := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "port1",
					Port: 80,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 9080,
					},
				},
				{
					Name: "port2",
					Port: 443,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 9443,
					},
				},
			},
		},
	}
	endpoints := &corev1.Endpoints{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Subsets: []corev1.EndpointSubset{
			{
				Ports: []corev1.EndpointPort{
					{
						Name: "port1",
						Port: 9080,
					},
					{
						Name: "port2",
						Port: 9443,
					},
				},
				Addresses: []corev1.EndpointAddress{
					{IP: "192.168.1.1"},
					{IP: "192.168.1.2"},
				},
			},
		},
	}
	nodes, err := tr.translateUptreamNodes(svc, endpoints, 10080)
	assert.Nil(t, nodes)
	assert.Equal(t, err, &translateError{
		field:  "service.spec.ports",
		reason: "port not defined",
	})

	nodes, err = tr.translateUptreamNodes(svc, endpoints, 80)
	assert.Nil(t, err)
	assert.Equal(t, nodes, []apisixv1.UpstreamNode{
		{
			IP:     "192.168.1.1",
			Port:   9080,
			Weight: 100,
		},
		{
			IP:     "192.168.1.2",
			Port:   9080,
			Weight: 100,
		},
	})

	nodes, err = tr.translateUptreamNodes(svc, endpoints, 443)
	assert.Nil(t, err)
	assert.Equal(t, nodes, []apisixv1.UpstreamNode{
		{
			IP:     "192.168.1.1",
			Port:   9443,
			Weight: 100,
		},
		{
			IP:     "192.168.1.2",
			Port:   9443,
			Weight: 100,
		},
	})
}
