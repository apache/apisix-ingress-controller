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
package gateway_translation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	fakeapisix "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned/fake"
	apisixinformers "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/informers/externalversions"
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation"
)

func mockTCPRouteTranslator(t *testing.T) (*translator, <-chan struct{}) {
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "test",
		},
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
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "test",
		},
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

	client := fake.NewSimpleClientset()
	informersFactory := informers.NewSharedInformerFactory(client, 0)
	svcInformer := informersFactory.Core().V1().Services().Informer()
	svcLister := informersFactory.Core().V1().Services().Lister()
	epLister, epInformer := kube.NewEndpointListerAndInformer(informersFactory, false)
	apisixClient := fakeapisix.NewSimpleClientset()
	apisixInformersFactory := apisixinformers.NewSharedInformerFactory(apisixClient, 0)

	_, err := client.CoreV1().Endpoints("test").Create(context.Background(), endpoints, metav1.CreateOptions{})
	assert.Nil(t, err)
	_, err = client.CoreV1().Services("test").Create(context.Background(), svc, metav1.CreateOptions{})
	assert.Nil(t, err)

	tr := &translator{
		&TranslatorOptions{
			KubeTranslator: translation.NewTranslator(&translation.TranslatorOptions{
				EndpointLister: epLister,
				ServiceLister:  svcLister,
				ApisixUpstreamLister: kube.NewApisixUpstreamLister(
					apisixInformersFactory.Apisix().V2beta3().ApisixUpstreams().Lister(),
					apisixInformersFactory.Apisix().V2().ApisixUpstreams().Lister(),
				),
				APIVersion: config.DefaultAPIVersion,
			}),
		},
	}

	processCh := make(chan struct{})
	svcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			processCh <- struct{}{}
		},
	})
	epInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			processCh <- struct{}{}
		},
	})

	stopCh := make(chan struct{}, 2)
	defer close(stopCh)
	go svcInformer.Run(stopCh)
	go epInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, svcInformer.HasSynced)

	return tr, processCh
}

func TestTranslateGatewayTCPRoute(t *testing.T) {
	refKind := func(str gatewayv1alpha2.Kind) *gatewayv1alpha2.Kind {
		return &str
	}
	refNamespace := func(str gatewayv1alpha2.Namespace) *gatewayv1alpha2.Namespace {
		return &str
	}
	refPortNumber := func(i gatewayv1alpha2.PortNumber) *gatewayv1alpha2.PortNumber {
		return &i
	}
	refInt32 := func(i int32) *int32 {
		return &i
	}
	tr, processCh := mockTCPRouteTranslator(t)
	<-processCh
	<-processCh

	tcpRoute := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tcp_route",
			Namespace: "test",
		},
		Spec: gatewayv1alpha2.TCPRouteSpec{
			Rules: []gatewayv1alpha2.TCPRouteRule{
				{
					BackendRefs: []gatewayv1alpha2.BackendRef{
						{
							BackendObjectReference: gatewayv1alpha2.BackendObjectReference{
								Kind:      refKind("Service"),
								Name:      "svc",
								Namespace: refNamespace("test"),
								Port:      refPortNumber(80),
							},
							Weight: refInt32(100),
						},
					},
				},
			},
		},
	}

	tctx, err := tr.TranslateGatewayTCPRouteV1Alpha2(tcpRoute)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(tctx.Routes))
	assert.Equal(t, 1, len(tctx.Upstreams))

	r := tctx.Routes[0]
	u := tctx.Upstreams[0]

	// Metadata
	// FIXME
	assert.NotEqual(t, "", u.ID)
	assert.Equal(t, u.ID, r.UpstreamId)
}
