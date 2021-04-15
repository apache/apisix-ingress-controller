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
package translation

import (
	"context"
	"testing"

	"github.com/apache/apisix-ingress-controller/pkg/id"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	apisixfake "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned/fake"
	apisixinformers "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/informers/externalversions"
)

func TestTranslateTrafficSplitPlugin(t *testing.T) {
	client := fake.NewSimpleClientset()
	informersFactory := informers.NewSharedInformerFactory(client, 0)
	svcInformer := informersFactory.Core().V1().Services().Informer()
	svcLister := informersFactory.Core().V1().Services().Lister()
	epInformer := informersFactory.Core().V1().Endpoints().Informer()
	epLister := informersFactory.Core().V1().Endpoints().Lister()
	apisixClient := apisixfake.NewSimpleClientset()
	apisixInformersFactory := apisixinformers.NewSharedInformerFactory(apisixClient, 0)
	auInformer := apisixInformersFactory.Apisix().V1().ApisixUpstreams().Informer()
	auLister := apisixInformersFactory.Apisix().V1().ApisixUpstreams().Lister()

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

	stopCh := make(chan struct{})
	defer close(stopCh)
	go svcInformer.Run(stopCh)
	go epInformer.Run(stopCh)
	go auInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, svcInformer.HasSynced)

	svc1 := &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-1",
			Namespace: "test",
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.5.3",
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
	endpoints1 := &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-1",
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

	_, err := client.CoreV1().Services("test").Create(context.Background(), svc1, metav1.CreateOptions{})
	assert.Nil(t, err)
	_, err = client.CoreV1().Endpoints("test").Create(context.Background(), endpoints1, metav1.CreateOptions{})
	assert.Nil(t, err)

	<-processCh
	<-processCh

	weight10 := 10
	weight20 := 20
	backends := []*configv2alpha1.ApisixRouteHTTPBackend{
		{
			ServiceName: "svc-1",
			ServicePort: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "port1",
			},
			Weight: &weight10,
		},
		{
			ServiceName: "svc-1",
			ServicePort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 443,
			},
			ResolveGranularity: "service",
			Weight:             &weight20,
		},
	}

	ar1 := &configv2alpha1.ApisixRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApisixRoute",
			APIVersion: "apisix.apache.org/v2alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ar1",
			Namespace: "test",
		},
		Spec: &configv2alpha1.ApisixRouteSpec{
			HTTP: []*configv2alpha1.ApisixRouteHTTP{
				{
					Name: "r1",
					Match: &configv2alpha1.ApisixRouteHTTPMatch{
						Paths: []string{"/*"},
						Hosts: []string{"test.com"},
					},
					Backends: backends,
				},
			},
		},
	}

	tr := &translator{&TranslatorOptions{
		ServiceLister:        svcLister,
		EndpointsLister:      epLister,
		ApisixUpstreamLister: auLister,
	}}
	ups, cfg, err := tr.translateTrafficSplitPlugin(ar1, 30, backends)
	assert.Nil(t, err)

	assert.Len(t, ups, 2)
	assert.Equal(t, ups[0].Name, "test_svc-1_80")
	assert.Len(t, ups[0].Nodes, 2)
	assert.Equal(t, ups[0].Nodes[0].Host, "192.168.1.1")
	assert.Equal(t, ups[0].Nodes[0].Port, 9080)
	assert.Equal(t, ups[0].Nodes[1].Host, "192.168.1.2")
	assert.Equal(t, ups[0].Nodes[1].Port, 9080)

	assert.Equal(t, ups[1].Name, "test_svc-1_443")
	assert.Len(t, ups[1].Nodes, 1)
	assert.Equal(t, ups[1].Nodes[0].Host, "10.0.5.3")
	assert.Equal(t, ups[1].Nodes[0].Port, 443)

	assert.Len(t, cfg.Rules, 1)
	assert.Len(t, cfg.Rules[0].WeightedUpstreams, 3)
	assert.Equal(t, cfg.Rules[0].WeightedUpstreams[0].UpstreamID, id.GenID("test_svc-1_80"))
	assert.Equal(t, cfg.Rules[0].WeightedUpstreams[0].Weight, 10)
	assert.Equal(t, cfg.Rules[0].WeightedUpstreams[1].UpstreamID, id.GenID("test_svc-1_443"))
	assert.Equal(t, cfg.Rules[0].WeightedUpstreams[1].Weight, 20)
	assert.Equal(t, cfg.Rules[0].WeightedUpstreams[2].UpstreamID, "")
	assert.Equal(t, cfg.Rules[0].WeightedUpstreams[2].Weight, 30)
}

func TestTranslateTrafficSplitPluginWithSameUpstreams(t *testing.T) {
	client := fake.NewSimpleClientset()
	informersFactory := informers.NewSharedInformerFactory(client, 0)
	svcInformer := informersFactory.Core().V1().Services().Informer()
	svcLister := informersFactory.Core().V1().Services().Lister()
	epInformer := informersFactory.Core().V1().Endpoints().Informer()
	epLister := informersFactory.Core().V1().Endpoints().Lister()
	apisixClient := apisixfake.NewSimpleClientset()
	apisixInformersFactory := apisixinformers.NewSharedInformerFactory(apisixClient, 0)
	auInformer := apisixInformersFactory.Apisix().V1().ApisixUpstreams().Informer()
	auLister := apisixInformersFactory.Apisix().V1().ApisixUpstreams().Lister()

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

	stopCh := make(chan struct{})
	defer close(stopCh)
	go svcInformer.Run(stopCh)
	go epInformer.Run(stopCh)
	go auInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, svcInformer.HasSynced)

	svc1 := &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-1",
			Namespace: "test",
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.5.3",
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
	endpoints1 := &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-1",
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

	_, err := client.CoreV1().Services("test").Create(context.Background(), svc1, metav1.CreateOptions{})
	assert.Nil(t, err)
	_, err = client.CoreV1().Endpoints("test").Create(context.Background(), endpoints1, metav1.CreateOptions{})
	assert.Nil(t, err)

	<-processCh
	<-processCh

	weigth10 := 10
	weight20 := 20

	backends := []*configv2alpha1.ApisixRouteHTTPBackend{
		{
			ServiceName: "svc-1",
			ServicePort: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "port1",
			},
			Weight: &weigth10,
		},
		{
			ServiceName: "svc-1",
			ServicePort: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "port1",
			},
			Weight: &weight20,
		},
	}

	ar1 := &configv2alpha1.ApisixRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApisixRoute",
			APIVersion: "apisix.apache.org/v2alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ar1",
			Namespace: "test",
		},
		Spec: &configv2alpha1.ApisixRouteSpec{
			HTTP: []*configv2alpha1.ApisixRouteHTTP{
				{
					Name: "r1",
					Match: &configv2alpha1.ApisixRouteHTTPMatch{
						Paths: []string{"/*"},
						Hosts: []string{"test.com"},
					},
					Backends: backends,
				},
			},
		},
	}

	tr := &translator{&TranslatorOptions{
		ServiceLister:        svcLister,
		EndpointsLister:      epLister,
		ApisixUpstreamLister: auLister,
	}}
	ups, cfg, err := tr.translateTrafficSplitPlugin(ar1, 30, backends)
	assert.Nil(t, err)

	// Here ups has two elements, but the duplicated one will be
	// removed in TranslateApisixRouteV2alpha1.
	assert.Len(t, ups, 2)
	assert.Equal(t, ups[0].Name, "test_svc-1_80")
	assert.Len(t, ups[0].Nodes, 2)
	assert.Equal(t, ups[0].Nodes[0].Host, "192.168.1.1")
	assert.Equal(t, ups[0].Nodes[0].Port, 9080)
	assert.Equal(t, ups[0].Nodes[1].Host, "192.168.1.2")
	assert.Equal(t, ups[0].Nodes[1].Port, 9080)

	assert.Equal(t, ups[1].Name, "test_svc-1_80")
	assert.Len(t, ups[1].Nodes, 2)
	assert.Equal(t, ups[1].Nodes[0].Host, "192.168.1.1")
	assert.Equal(t, ups[1].Nodes[0].Port, 9080)
	assert.Equal(t, ups[1].Nodes[1].Host, "192.168.1.2")
	assert.Equal(t, ups[1].Nodes[1].Port, 9080)

	assert.Len(t, cfg.Rules, 1)
	assert.Len(t, cfg.Rules[0].WeightedUpstreams, 3)
	assert.Equal(t, cfg.Rules[0].WeightedUpstreams[0].UpstreamID, id.GenID("test_svc-1_80"))
	assert.Equal(t, cfg.Rules[0].WeightedUpstreams[0].Weight, 10)
	assert.Equal(t, cfg.Rules[0].WeightedUpstreams[1].UpstreamID, id.GenID("test_svc-1_80"))
	assert.Equal(t, cfg.Rules[0].WeightedUpstreams[1].Weight, 20)
	assert.Equal(t, cfg.Rules[0].WeightedUpstreams[2].UpstreamID, "")
	assert.Equal(t, cfg.Rules[0].WeightedUpstreams[2].Weight, 30)
}

func TestTranslateTrafficSplitPluginBadCases(t *testing.T) {
	client := fake.NewSimpleClientset()
	informersFactory := informers.NewSharedInformerFactory(client, 0)
	svcInformer := informersFactory.Core().V1().Services().Informer()
	svcLister := informersFactory.Core().V1().Services().Lister()
	epInformer := informersFactory.Core().V1().Endpoints().Informer()
	epLister := informersFactory.Core().V1().Endpoints().Lister()
	apisixClient := apisixfake.NewSimpleClientset()
	apisixInformersFactory := apisixinformers.NewSharedInformerFactory(apisixClient, 0)
	auInformer := apisixInformersFactory.Apisix().V1().ApisixUpstreams().Informer()
	auLister := apisixInformersFactory.Apisix().V1().ApisixUpstreams().Lister()

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

	stopCh := make(chan struct{})
	defer close(stopCh)
	go svcInformer.Run(stopCh)
	go epInformer.Run(stopCh)
	go auInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, svcInformer.HasSynced)

	svc1 := &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-1",
			Namespace: "test",
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "", // Headless service
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
	endpoints1 := &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-1",
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

	_, err := client.CoreV1().Services("test").Create(context.Background(), svc1, metav1.CreateOptions{})
	assert.Nil(t, err)
	_, err = client.CoreV1().Endpoints("test").Create(context.Background(), endpoints1, metav1.CreateOptions{})
	assert.Nil(t, err)

	<-processCh
	<-processCh

	weight10 := 10
	weight20 := 20

	backends := []*configv2alpha1.ApisixRouteHTTPBackend{
		{
			ServiceName: "svc-2",
			ServicePort: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "port1",
			},
			Weight: &weight10,
		},
		{
			ServiceName: "svc-1",
			ServicePort: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "port1",
			},
			Weight: &weight20,
		},
	}

	ar1 := &configv2alpha1.ApisixRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApisixRoute",
			APIVersion: "apisix.apache.org/v2alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ar1",
			Namespace: "test",
		},
		Spec: &configv2alpha1.ApisixRouteSpec{
			HTTP: []*configv2alpha1.ApisixRouteHTTP{
				{
					Name: "r1",
					Match: &configv2alpha1.ApisixRouteHTTPMatch{
						Paths: []string{"/*"},
						Hosts: []string{"test.com"},
					},
					Backends: backends,
				},
			},
		},
	}

	tr := &translator{&TranslatorOptions{
		ServiceLister:        svcLister,
		EndpointsLister:      epLister,
		ApisixUpstreamLister: auLister,
	}}
	ups, cfg, err := tr.translateTrafficSplitPlugin(ar1, 30, backends)
	assert.Nil(t, ups)
	assert.Nil(t, cfg)
	assert.Equal(t, err.Error(), "service \"svc-2\" not found")

	backends[0].ServiceName = "svc-1"
	backends[1].ServicePort.StrVal = "port-not-found"
	ups, cfg, err = tr.translateTrafficSplitPlugin(ar1, 30, backends)
	assert.Nil(t, ups)
	assert.Nil(t, cfg)
	assert.Equal(t, err.Error(), "service.spec.ports: port not defined")

	backends[1].ServicePort.StrVal = "port2"
	backends[1].ResolveGranularity = "service"
	ups, cfg, err = tr.translateTrafficSplitPlugin(ar1, 30, backends)
	assert.Nil(t, ups)
	assert.Nil(t, cfg)
	assert.Equal(t, err.Error(), "conflict headless service and backend resolve granularity")
}
