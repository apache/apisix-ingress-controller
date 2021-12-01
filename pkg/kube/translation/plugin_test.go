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

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	apisixfake "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned/fake"
	apisixinformers "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/informers/externalversions"
)

func TestTranslateTrafficSplitPlugin(t *testing.T) {
	client := fake.NewSimpleClientset()
	informersFactory := informers.NewSharedInformerFactory(client, 0)
	svcInformer := informersFactory.Core().V1().Services().Informer()
	svcLister := informersFactory.Core().V1().Services().Lister()
	epLister, epInformer := kube.NewEndpointListerAndInformer(informersFactory, false)
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
		EndpointLister:       epLister,
		ApisixUpstreamLister: auLister,
	}}
	ctx := &TranslateContext{
		upstreamMap: make(map[string]struct{}),
	}
	cfg, err := tr.translateTrafficSplitPlugin(ctx, ar1.Namespace, 30, backends)
	assert.Nil(t, err)

	assert.Len(t, ctx.Upstreams, 2)
	assert.Equal(t, "test_svc-1_80", ctx.Upstreams[0].Name)
	assert.Len(t, ctx.Upstreams[0].Nodes, 2)
	assert.Equal(t, "192.168.1.1", ctx.Upstreams[0].Nodes[0].Host)
	assert.Equal(t, 9080, ctx.Upstreams[0].Nodes[0].Port)
	assert.Equal(t, "192.168.1.2", ctx.Upstreams[0].Nodes[1].Host)
	assert.Equal(t, 9080, ctx.Upstreams[0].Nodes[1].Port)

	assert.Equal(t, "test_svc-1_443", ctx.Upstreams[1].Name)
	assert.Len(t, ctx.Upstreams[1].Nodes, 1)
	assert.Equal(t, "10.0.5.3", ctx.Upstreams[1].Nodes[0].Host)
	assert.Equal(t, 443, ctx.Upstreams[1].Nodes[0].Port)

	assert.Len(t, cfg.Rules, 1)
	assert.Len(t, cfg.Rules[0].WeightedUpstreams, 3)
	assert.Equal(t, id.GenID("test_svc-1_80"), cfg.Rules[0].WeightedUpstreams[0].UpstreamID)
	assert.Equal(t, 10, cfg.Rules[0].WeightedUpstreams[0].Weight)
	assert.Equal(t, id.GenID("test_svc-1_443"), cfg.Rules[0].WeightedUpstreams[1].UpstreamID)
	assert.Equal(t, 20, cfg.Rules[0].WeightedUpstreams[1].Weight)
	assert.Equal(t, "", cfg.Rules[0].WeightedUpstreams[2].UpstreamID)
	assert.Equal(t, 30, cfg.Rules[0].WeightedUpstreams[2].Weight)
}

func TestTranslateTrafficSplitPluginWithSameUpstreams(t *testing.T) {
	client := fake.NewSimpleClientset()
	informersFactory := informers.NewSharedInformerFactory(client, 0)
	svcInformer := informersFactory.Core().V1().Services().Informer()
	svcLister := informersFactory.Core().V1().Services().Lister()
	epLister, epInformer := kube.NewEndpointListerAndInformer(informersFactory, false)
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
		EndpointLister:       epLister,
		ApisixUpstreamLister: auLister,
	}}
	ctx := &TranslateContext{upstreamMap: make(map[string]struct{})}
	cfg, err := tr.translateTrafficSplitPlugin(ctx, ar1.Namespace, 30, backends)
	assert.Nil(t, err)

	assert.Len(t, ctx.Upstreams, 1)
	assert.Equal(t, "test_svc-1_80", ctx.Upstreams[0].Name)
	assert.Len(t, ctx.Upstreams[0].Nodes, 2)
	assert.Equal(t, "192.168.1.1", ctx.Upstreams[0].Nodes[0].Host)
	assert.Equal(t, 9080, ctx.Upstreams[0].Nodes[0].Port)
	assert.Equal(t, "192.168.1.2", ctx.Upstreams[0].Nodes[1].Host)
	assert.Equal(t, 9080, ctx.Upstreams[0].Nodes[1].Port)

	assert.Len(t, cfg.Rules, 1)
	assert.Len(t, cfg.Rules[0].WeightedUpstreams, 3)
	assert.Equal(t, id.GenID("test_svc-1_80"), cfg.Rules[0].WeightedUpstreams[0].UpstreamID)
	assert.Equal(t, 10, cfg.Rules[0].WeightedUpstreams[0].Weight)
	assert.Equal(t, id.GenID("test_svc-1_80"), cfg.Rules[0].WeightedUpstreams[1].UpstreamID)
	assert.Equal(t, 20, cfg.Rules[0].WeightedUpstreams[1].Weight)
	assert.Equal(t, "", cfg.Rules[0].WeightedUpstreams[2].UpstreamID)
	assert.Equal(t, 30, cfg.Rules[0].WeightedUpstreams[2].Weight)
}

func TestTranslateTrafficSplitPluginBadCases(t *testing.T) {
	client := fake.NewSimpleClientset()
	informersFactory := informers.NewSharedInformerFactory(client, 0)
	svcInformer := informersFactory.Core().V1().Services().Informer()
	svcLister := informersFactory.Core().V1().Services().Lister()
	epLister, epInformer := kube.NewEndpointListerAndInformer(informersFactory, false)
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
		EndpointLister:       epLister,
		ApisixUpstreamLister: auLister,
	}}
	ctx := &TranslateContext{upstreamMap: make(map[string]struct{})}
	cfg, err := tr.translateTrafficSplitPlugin(ctx, ar1.Namespace, 30, backends)
	assert.Nil(t, cfg)
	assert.Len(t, ctx.Upstreams, 0)
	assert.NotNil(t, err)
	assert.Equal(t, "service \"svc-2\" not found", err.Error())

	backends[0].ServiceName = "svc-1"
	backends[1].ServicePort.StrVal = "port-not-found"
	ctx = &TranslateContext{upstreamMap: make(map[string]struct{})}
	cfg, err = tr.translateTrafficSplitPlugin(ctx, ar1.Namespace, 30, backends)
	assert.Nil(t, cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "service.spec.ports: port not defined", err.Error())

	backends[1].ServicePort.StrVal = "port2"
	backends[1].ResolveGranularity = "service"
	ctx = &TranslateContext{upstreamMap: make(map[string]struct{})}
	cfg, err = tr.translateTrafficSplitPlugin(ctx, ar1.Namespace, 30, backends)
	assert.Nil(t, cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "conflict headless service and backend resolve granularity", err.Error())
}

func TestTranslateConsumerKeyAuthPluginWithInPlaceValue(t *testing.T) {
	keyAuth := &configv2alpha1.ApisixConsumerKeyAuth{
		Value: &configv2alpha1.ApisixConsumerKeyAuthValue{Key: "abc"},
	}
	cfg, err := (&translator{}).translateConsumerKeyAuthPlugin("default", keyAuth)
	assert.Nil(t, err)
	assert.Equal(t, "abc", cfg.Key)
}

func TestTranslateConsumerKeyAuthWithSecretRef(t *testing.T) {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "abc-key-auth",
		},
		Data: map[string][]byte{
			"key": []byte("abc"),
		},
	}
	client := fake.NewSimpleClientset()
	informersFactory := informers.NewSharedInformerFactory(client, 0)
	secretInformer := informersFactory.Core().V1().Secrets().Informer()
	secretLister := informersFactory.Core().V1().Secrets().Lister()
	processCh := make(chan struct{})
	stopCh := make(chan struct{})
	secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(_ interface{}) {
			processCh <- struct{}{}
		},
		UpdateFunc: func(_, _ interface{}) {
			processCh <- struct{}{}
		},
	})
	go secretInformer.Run(stopCh)

	tr := &translator{
		&TranslatorOptions{
			SecretLister: secretLister,
		},
	}
	_, err := client.CoreV1().Secrets("default").Create(context.Background(), sec, metav1.CreateOptions{})
	assert.Nil(t, err)

	<-processCh

	keyAuth := &configv2alpha1.ApisixConsumerKeyAuth{
		SecretRef: &corev1.LocalObjectReference{Name: "abc-key-auth"},
	}
	cfg, err := tr.translateConsumerKeyAuthPlugin("default", keyAuth)
	assert.Nil(t, err)
	assert.Equal(t, "abc", cfg.Key)

	cfg, err = tr.translateConsumerKeyAuthPlugin("default2", keyAuth)
	assert.Nil(t, cfg)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")

	delete(sec.Data, "key")
	_, err = client.CoreV1().Secrets("default").Update(context.Background(), sec, metav1.UpdateOptions{})
	assert.Nil(t, err)
	<-processCh

	cfg, err = tr.translateConsumerKeyAuthPlugin("default", keyAuth)
	assert.Nil(t, cfg)
	assert.Equal(t, _errKeyNotFoundOrInvalid, err)

	close(processCh)
	close(stopCh)
}

func TestTranslateConsumerBasicAuthPluginWithInPlaceValue(t *testing.T) {
	basicAuth := &configv2alpha1.ApisixConsumerBasicAuth{
		Value: &configv2alpha1.ApisixConsumerBasicAuthValue{
			Username: "jack",
			Password: "jacknice",
		},
	}
	cfg, err := (&translator{}).translateConsumerBasicAuthPlugin("default", basicAuth)
	assert.Nil(t, err)
	assert.Equal(t, "jack", cfg.Username)
	assert.Equal(t, "jacknice", cfg.Password)
}

func TestTranslateConsumerBasicAuthWithSecretRef(t *testing.T) {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "jack-basic-auth",
		},
		Data: map[string][]byte{
			"username": []byte("jack"),
			"password": []byte("jacknice"),
		},
	}
	client := fake.NewSimpleClientset()
	informersFactory := informers.NewSharedInformerFactory(client, 0)
	secretInformer := informersFactory.Core().V1().Secrets().Informer()
	secretLister := informersFactory.Core().V1().Secrets().Lister()
	processCh := make(chan struct{})
	stopCh := make(chan struct{})
	secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(_ interface{}) {
			processCh <- struct{}{}
		},
		UpdateFunc: func(_, _ interface{}) {
			processCh <- struct{}{}
		},
	})
	go secretInformer.Run(stopCh)

	tr := &translator{
		&TranslatorOptions{
			SecretLister: secretLister,
		},
	}
	_, err := client.CoreV1().Secrets("default").Create(context.Background(), sec, metav1.CreateOptions{})
	assert.Nil(t, err)

	<-processCh

	basicAuth := &configv2alpha1.ApisixConsumerBasicAuth{
		SecretRef: &corev1.LocalObjectReference{Name: "jack-basic-auth"},
	}
	cfg, err := tr.translateConsumerBasicAuthPlugin("default", basicAuth)
	assert.Nil(t, err)
	assert.Equal(t, "jack", cfg.Username)
	assert.Equal(t, "jacknice", cfg.Password)

	cfg, err = tr.translateConsumerBasicAuthPlugin("default2", basicAuth)
	assert.Nil(t, cfg)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")

	delete(sec.Data, "password")
	_, err = client.CoreV1().Secrets("default").Update(context.Background(), sec, metav1.UpdateOptions{})
	assert.Nil(t, err)
	<-processCh

	cfg, err = tr.translateConsumerBasicAuthPlugin("default", basicAuth)
	assert.Nil(t, cfg)
	assert.Equal(t, _errPasswordNotFoundOrInvalid, err)

	delete(sec.Data, "username")
	_, err = client.CoreV1().Secrets("default").Update(context.Background(), sec, metav1.UpdateOptions{})
	assert.Nil(t, err)
	<-processCh

	cfg, err = tr.translateConsumerBasicAuthPlugin("default", basicAuth)
	assert.Nil(t, cfg)
	assert.Equal(t, _errUsernameNotFoundOrInvalid, err)

	close(processCh)
	close(stopCh)
}
