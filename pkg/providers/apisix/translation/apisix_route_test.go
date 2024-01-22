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
package translation

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	fakeapisix "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned/fake"
	apisixinformers "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/informers/externalversions"
	_const "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/const"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestRouteMatchExpr(t *testing.T) {
	tr := &translator{}
	value1 := "text/plain"
	value2 := "gzip"
	value3 := "13"
	value4 := ".*\\.php"
	exprs := []configv2.ApisixRouteHTTPMatchExpr{
		{
			Subject: configv2.ApisixRouteHTTPMatchExprSubject{
				Scope: _const.ScopeHeader,
				Name:  "Content-Type",
			},
			Op:    _const.OpEqual,
			Value: &value1,
		},
		{
			Subject: configv2.ApisixRouteHTTPMatchExprSubject{
				Scope: _const.ScopeHeader,
				Name:  "Content-Encoding",
			},
			Op:    _const.OpNotEqual,
			Value: &value2,
		},
		{
			Subject: configv2.ApisixRouteHTTPMatchExprSubject{
				Scope: _const.ScopeQuery,
				Name:  "ID",
			},
			Op:    _const.OpGreaterThan,
			Value: &value3,
		},
		{
			Subject: configv2.ApisixRouteHTTPMatchExprSubject{
				Scope: _const.ScopeQuery,
				Name:  "ID",
			},
			Op:    _const.OpLessThan,
			Value: &value3,
		},
		{
			Subject: configv2.ApisixRouteHTTPMatchExprSubject{
				Scope: _const.ScopeQuery,
				Name:  "ID",
			},
			Op:    _const.OpRegexMatch,
			Value: &value4,
		},
		{
			Subject: configv2.ApisixRouteHTTPMatchExprSubject{
				Scope: _const.ScopeQuery,
				Name:  "ID",
			},
			Op:    _const.OpRegexMatchCaseInsensitive,
			Value: &value4,
		},
		{
			Subject: configv2.ApisixRouteHTTPMatchExprSubject{
				Scope: _const.ScopeQuery,
				Name:  "ID",
			},
			Op:    _const.OpRegexNotMatch,
			Value: &value4,
		},
		{
			Subject: configv2.ApisixRouteHTTPMatchExprSubject{
				Scope: _const.ScopeQuery,
				Name:  "ID",
			},
			Op:    _const.OpRegexNotMatchCaseInsensitive,
			Value: &value4,
		},
		{
			Subject: configv2.ApisixRouteHTTPMatchExprSubject{
				Scope: _const.ScopeCookie,
				Name:  "domain",
			},
			Op: _const.OpIn,
			Set: []string{
				"a.com",
				"b.com",
			},
		},
		{
			Subject: configv2.ApisixRouteHTTPMatchExprSubject{
				Scope: _const.ScopeCookie,
				Name:  "X-Foo",
			},
			Op: _const.OpIn,
			Set: []string{
				"foo.com",
			},
		},
	}
	results, err := tr.TranslateRouteMatchExprs(exprs)
	assert.Nil(t, err)
	assert.Len(t, results, 10)

	assert.Len(t, results[0], 3)
	assert.Equal(t, "http_content_type", results[0][0].StrVal)
	assert.Equal(t, "==", results[0][1].StrVal)
	assert.Equal(t, "text/plain", results[0][2].StrVal)

	assert.Len(t, results[1], 3)
	assert.Equal(t, "http_content_encoding", results[1][0].StrVal)
	assert.Equal(t, "~=", results[1][1].StrVal)
	assert.Equal(t, "gzip", results[1][2].StrVal)

	assert.Len(t, results[2], 3)
	assert.Equal(t, "arg_ID", results[2][0].StrVal)
	assert.Equal(t, ">", results[2][1].StrVal)
	assert.Equal(t, "13", results[2][2].StrVal)

	assert.Len(t, results[3], 3)
	assert.Equal(t, "arg_ID", results[3][0].StrVal)
	assert.Equal(t, "<", results[3][1].StrVal)
	assert.Equal(t, "13", results[3][2].StrVal)

	assert.Len(t, results[4], 3)
	assert.Equal(t, "arg_ID", results[4][0].StrVal)
	assert.Equal(t, "~~", results[4][1].StrVal)
	assert.Equal(t, ".*\\.php", results[4][2].StrVal)

	assert.Len(t, results[5], 3)
	assert.Equal(t, "arg_ID", results[5][0].StrVal)
	assert.Equal(t, "~*", results[5][1].StrVal)
	assert.Equal(t, ".*\\.php", results[5][2].StrVal)

	assert.Len(t, results[6], 4)
	assert.Equal(t, "arg_ID", results[6][0].StrVal)
	assert.Equal(t, "!", results[6][1].StrVal)
	assert.Equal(t, "~~", results[6][2].StrVal)
	assert.Equal(t, ".*\\.php", results[6][3].StrVal)

	assert.Len(t, results[7], 4)
	assert.Equal(t, "arg_ID", results[7][0].StrVal)
	assert.Equal(t, "!", results[7][1].StrVal)
	assert.Equal(t, "~*", results[7][2].StrVal)
	assert.Equal(t, ".*\\.php", results[7][3].StrVal)

	assert.Len(t, results[8], 3)
	assert.Equal(t, "cookie_domain", results[8][0].StrVal)
	assert.Equal(t, "in", results[8][1].StrVal)
	assert.Equal(t, []string{"a.com", "b.com"}, results[8][2].SliceVal)

	assert.Len(t, results[9], 3)
	assert.Equal(t, "cookie_X-Foo", results[9][0].StrVal)
	assert.Equal(t, "in", results[9][1].StrVal)
	assert.Equal(t, []string{"foo.com"}, results[9][2].SliceVal)
}

func TestTranslateApisixRouteV2WithDuplicatedName(t *testing.T) {
	tr, processCh := mockTranslatorV2(t)
	<-processCh
	<-processCh

	ar := &configv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ar",
			Namespace: "test",
		},
		Spec: configv2.ApisixRouteSpec{
			HTTP: []configv2.ApisixRouteHTTP{
				{
					Name: "rule1",
					Match: configv2.ApisixRouteHTTPMatch{
						Paths: []string{
							"/*",
						},
					},
					Backends: []configv2.ApisixRouteHTTPBackend{
						{
							ServiceName: "svc",
							ServicePort: intstr.IntOrString{
								IntVal: 80,
							},
						},
					},
				},
				{
					Name: "rule1",
					Match: configv2.ApisixRouteHTTPMatch{
						Paths: []string{
							"/*",
						},
					},
					Backends: []configv2.ApisixRouteHTTPBackend{
						{
							ServiceName: "svc",
							ServicePort: intstr.IntOrString{
								IntVal: 80,
							},
						},
					},
				},
			},
		},
	}

	_, err := tr.TranslateRouteV2(ar)
	assert.NotNil(t, err)
	assert.Equal(t, "duplicated route rule name", err.Error())
}

func TestTranslateApisixRouteV2WithEmptyPluginConfigName(t *testing.T) {
	tr, processCh := mockTranslatorV2(t)
	<-processCh
	<-processCh

	ar := &configv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ar",
			Namespace: "test",
		},
		Spec: configv2.ApisixRouteSpec{
			HTTP: []configv2.ApisixRouteHTTP{
				{
					Name: "rule1",
					Match: configv2.ApisixRouteHTTPMatch{
						Paths: []string{
							"/*",
						},
					},
					Backends: []configv2.ApisixRouteHTTPBackend{
						{
							ServiceName: "svc",
							ServicePort: intstr.IntOrString{
								IntVal: 80,
							},
						},
					},
				},
				{
					Name: "rule2",
					Match: configv2.ApisixRouteHTTPMatch{
						Paths: []string{
							"/*",
						},
					},
					Backends: []configv2.ApisixRouteHTTPBackend{
						{
							ServiceName: "svc",
							ServicePort: intstr.IntOrString{
								IntVal: 80,
							},
						},
					},
					PluginConfigName: "test-PluginConfigName-1",
				},
				{
					Name: "rule3",
					Match: configv2.ApisixRouteHTTPMatch{
						Paths: []string{
							"/*",
						},
					},
					Backends: []configv2.ApisixRouteHTTPBackend{
						{
							ServiceName: "svc",
							ServicePort: intstr.IntOrString{
								IntVal: 80,
							},
						},
					},
				},
			},
		},
	}
	res, err := tr.TranslateRouteV2(ar)
	assert.NoError(t, err)
	assert.Len(t, res.PluginConfigs, 0)
	assert.Len(t, res.Routes, 3)
	assert.Equal(t, "", res.Routes[0].PluginConfigId)
	expectedPluginId := id.GenID(apisixv1.ComposePluginConfigName(ar.Namespace, ar.Spec.HTTP[1].PluginConfigName))
	assert.Equal(t, expectedPluginId, res.Routes[1].PluginConfigId)
	assert.Equal(t, "", res.Routes[2].PluginConfigId)
}

func TestTranslateApisixRouteV2WithPluginConfigNamespace(t *testing.T) {
	tr, processCh := mockTranslatorV2(t)
	<-processCh
	<-processCh
	pluginConfigNamespace := "test-2"
	ar := &configv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ar",
			Namespace: "test",
		},
		Spec: configv2.ApisixRouteSpec{
			HTTP: []configv2.ApisixRouteHTTP{
				{
					Name: "rule1",
					Match: configv2.ApisixRouteHTTPMatch{
						Paths: []string{
							"/*",
						},
					},
					Backends: []configv2.ApisixRouteHTTPBackend{
						{
							ServiceName: "svc",
							ServicePort: intstr.IntOrString{
								IntVal: 80,
							},
						},
					},
					PluginConfigName:      "test-PluginConfigName-1",
					PluginConfigNamespace: pluginConfigNamespace,
				},
			},
		},
	}
	res, err := tr.TranslateRouteV2(ar)
	assert.NoError(t, err)
	assert.Len(t, res.PluginConfigs, 0)
	expectedPluginId := id.GenID(apisixv1.ComposePluginConfigName(pluginConfigNamespace, ar.Spec.HTTP[0].PluginConfigName))
	assert.Equal(t, expectedPluginId, res.Routes[0].PluginConfigId)
}

func TestGenerateApisixRouteV2DeleteMark(t *testing.T) {
	tr := &translator{
		&TranslatorOptions{},
		translation.NewTranslator(nil),
	}
	ar := &configv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ar",
			Namespace: "test",
		},
		Spec: configv2.ApisixRouteSpec{
			HTTP: []configv2.ApisixRouteHTTP{
				{
					Name: "rule1",
					Match: configv2.ApisixRouteHTTPMatch{
						Paths: []string{
							"/*",
						},
					},
					Backends: []configv2.ApisixRouteHTTPBackend{
						{
							ServiceName: "svc1",
							ServicePort: intstr.IntOrString{
								IntVal: 81,
							},
						},
					},
					Plugins: []configv2.ApisixRoutePlugin{
						{
							Name:   "plugin-1",
							Enable: true,
							Config: map[string]interface{}{
								"key-1": 123456,
								"key-2": "2121331",
							},
						},
					},
					PluginConfigName: "echo-and-cors-apc",
				},
				{
					Name: "rule2",
					Match: configv2.ApisixRouteHTTPMatch{
						Paths: []string{
							"/*",
						},
					},
					Backends: []configv2.ApisixRouteHTTPBackend{
						{
							ServiceName: "svc2",
							ServicePort: intstr.IntOrString{
								IntVal: 82,
							},
						},
					},
				},
			},
		},
	}

	tx, err := tr.GenerateRouteV2DeleteMark(ar)
	fmt.Println(tx)
	assert.NoError(t, err, "translateRoute not strictly should be no error")
	assert.Equal(t, 2, len(tx.Routes), "There should be 2 routes")
	assert.Equal(t, 2, len(tx.Upstreams), "There should be 2 upstreams")
	assert.Equal(t, "test_ar_rule1", tx.Routes[0].Name, "route1 name error")
	assert.Equal(t, "test_ar_rule2", tx.Routes[1].Name, "route2 name error")
	assert.Equal(t, "test_svc1_81", tx.Upstreams[0].Name, "upstream1 name error")
	assert.Equal(t, "test_svc2_82", tx.Upstreams[1].Name, "upstream2 name error")

	assert.Equal(t, id.GenID("test_ar_rule1"), tx.Routes[0].ID, "route1 id error")
	assert.Equal(t, id.GenID("test_ar_rule2"), tx.Routes[1].ID, "route2 id error")
	assert.Equal(t, id.GenID(apisixv1.ComposePluginConfigName(ar.Namespace, ar.Spec.HTTP[0].PluginConfigName)), tx.Routes[0].PluginConfigId, "route1 PluginConfigId error")
	assert.Equal(t, "", tx.Routes[1].PluginConfigId, "route2 PluginConfigId error ")

	assert.Equal(t, id.GenID("test_svc1_81"), tx.Upstreams[0].ID, "upstream1 id error")
	assert.Equal(t, id.GenID("test_svc2_82"), tx.Upstreams[1].ID, "upstream2 id error")
}

func ptrOf[T interface{}](v T) *T {
	return &v
}

func mockTranslatorV2(t *testing.T) (*translator, <-chan struct{}) {
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

	au := &configv2.ApisixUpstream{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "au",
			Namespace: "test",
		},
		Spec: &configv2.ApisixUpstreamSpec{
			ExternalNodes: []configv2.ApisixUpstreamExternalNode{
				{
					Name:   "httpbin.org",
					Type:   configv2.ExternalTypeDomain,
					Weight: ptrOf(1),
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

	auInformer := apisixInformersFactory.Apisix().V2().ApisixUpstreams().Informer()
	auLister := kube.NewApisixUpstreamLister(
		apisixInformersFactory.Apisix().V2().ApisixUpstreams().Lister(),
	)

	_, err := client.CoreV1().Endpoints("test").Create(context.Background(), endpoints, metav1.CreateOptions{})
	assert.Nil(t, err)
	_, err = client.CoreV1().Services("test").Create(context.Background(), svc, metav1.CreateOptions{})
	assert.Nil(t, err)
	_, err = apisixClient.ApisixV2().ApisixUpstreams("test").Create(context.Background(), au, metav1.CreateOptions{})
	assert.Nil(t, err)

	tr := &translator{
		&TranslatorOptions{
			ServiceLister:        svcLister,
			ApisixUpstreamLister: auLister,
		},
		translation.NewTranslator(&translation.TranslatorOptions{
			ServiceLister:        svcLister,
			EndpointLister:       epLister,
			ApisixUpstreamLister: auLister,
			APIVersion:           config.ApisixV2,
		}),
	}

	processCh := make(chan struct{}, 2)
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
	auInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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

	return tr, processCh
}

func TestTranslateApisixRouteV2WithUpstream(t *testing.T) {
	tr, processCh := mockTranslatorV2(t)
	<-processCh
	<-processCh

	ar := &configv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ar",
			Namespace: "test",
		},
		Spec: configv2.ApisixRouteSpec{
			HTTP: []configv2.ApisixRouteHTTP{
				{
					Name: "rule1",
					Match: configv2.ApisixRouteHTTPMatch{
						Paths: []string{
							"/*",
						},
					},
					Backends: []configv2.ApisixRouteHTTPBackend{
						{
							ServiceName: "svc",
							ServicePort: intstr.IntOrString{
								IntVal: 80,
							},
							Weight: ptrOf(2),
						},
					},
					Upstreams: []configv2.ApisixRouteUpstreamReference{
						{
							Name:   "au",
							Weight: ptrOf(1),
						},
					},
				},
			},
		},
	}

	tctx, err := tr.TranslateRouteV2(ar)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(tctx.Routes))
	r := tctx.Routes[0]

	assert.NotNil(t, r.Plugins["traffic-split"])

	tsCfg, ok := r.Plugins["traffic-split"].(*apisixv1.TrafficSplitConfig)
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, len(tsCfg.Rules))
	assert.NotNil(t, tsCfg.Rules[0])
	assert.NotNil(t, tsCfg.Rules[0].WeightedUpstreams, "weighted upstreams")

	wups := tsCfg.Rules[0].WeightedUpstreams

	upsName := apisixv1.ComposeExternalUpstreamName(ar.Namespace, "au")
	upsID := id.GenID(upsName)
	assert.Equal(t, []apisixv1.TrafficSplitConfigRuleWeightedUpstream{
		{
			// Default
			UpstreamID: "",
			Weight:     2,
		},
		{
			UpstreamID: upsID,
			Weight:     1,
		},
	}, wups)

	assert.Equal(t, 2, len(tctx.Upstreams))
	var ups *apisixv1.Upstream
	for _, u := range tctx.Upstreams {
		if u.ID == upsID {
			ups = u
			break
		}
	}
	assert.NotNil(t, ups)

	// unset useless data
	ups.Desc = ""
	assert.Equal(t, &apisixv1.Upstream{
		Metadata: apisixv1.Metadata{
			ID:   upsID,
			Name: upsName,
			Desc: "",
			Labels: map[string]string{
				"managed-by":  "apisix-ingress-controller",
				"meta_weight": "1",
			},
		},
		Type:   apisixv1.LbRoundRobin,
		HashOn: "",
		Key:    "",
		Checks: nil,
		Nodes: []apisixv1.UpstreamNode{
			{
				Host:   "httpbin.org",
				Port:   80,
				Weight: 1,
			},
		},
		Scheme:  apisixv1.SchemeHTTP,
		Retries: nil,
		Timeout: nil,
		TLS:     nil,
	}, ups)
}

func TestTranslateApisixRouteV2WithUpstreamNoBackendsAndPluginConfigName(t *testing.T) {
	tr, processCh := mockTranslatorV2(t)
	<-processCh
	<-processCh

	ar := &configv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ar",
			Namespace: "test",
		},
		Spec: configv2.ApisixRouteSpec{
			HTTP: []configv2.ApisixRouteHTTP{
				{
					Name: "rule1",
					Match: configv2.ApisixRouteHTTPMatch{
						Paths: []string{
							"/*",
						},
					},
					Upstreams: []configv2.ApisixRouteUpstreamReference{
						{
							Name:   "au",
							Weight: ptrOf(1),
						},
					},
					PluginConfigName: "test-PluginConfigName-1",
				},
			},
		},
	}

	tctx, err := tr.TranslateRouteV2(ar)
	assert.NoError(t, err)
	assert.Len(t, tctx.PluginConfigs, 0)
	assert.Len(t, tctx.Routes, 1)
	expectedPluginId := id.GenID(apisixv1.ComposePluginConfigName(ar.Namespace, ar.Spec.HTTP[0].PluginConfigName))
	assert.Equal(t, expectedPluginId, tctx.Routes[0].PluginConfigId)
}
