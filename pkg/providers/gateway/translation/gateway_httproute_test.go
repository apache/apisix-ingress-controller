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
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"k8s.io/client-go/kubernetes"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	fakeapisix "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned/fake"
	apisixinformers "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/informers/externalversions"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func refKind(str gatewayv1beta1.Kind) *gatewayv1beta1.Kind {
	return &str
}
func refNamespace(str gatewayv1beta1.Namespace) *gatewayv1beta1.Namespace {
	return &str
}
func refInt32(i int32) *int32 {
	return &i
}
func refPortNumber(i gatewayv1beta1.PortNumber) *gatewayv1beta1.PortNumber {
	return &i
}

func newServiceAndEndpoints(t *testing.T, client kubernetes.Interface, ns, svcName string, ports, targetPorts []int32, address []string) (*corev1.Service, *corev1.Endpoints) {
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{},
		},
	}

	endpoints := &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: ns,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Ports:     []corev1.EndpointPort{},
				Addresses: []corev1.EndpointAddress{},
			},
		},
	}

	for i, port := range ports {
		targetPort := targetPorts[i]
		svc.Spec.Ports = append(svc.Spec.Ports, corev1.ServicePort{
			Name: fmt.Sprintf("port%v", i),
			Port: port,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: targetPort,
			},
		})
		endpoints.Subsets[0].Ports = append(endpoints.Subsets[0].Ports, corev1.EndpointPort{
			Name: fmt.Sprintf("port%v", i),
			Port: targetPort,
		})
	}

	for _, addr := range address {
		endpoints.Subsets[0].Addresses = append(endpoints.Subsets[0].Addresses, corev1.EndpointAddress{
			IP: addr,
		})
	}

	_, err := client.CoreV1().Endpoints(ns).Create(context.Background(), endpoints, metav1.CreateOptions{})
	assert.Nil(t, err)
	_, err = client.CoreV1().Services(ns).Create(context.Background(), svc, metav1.CreateOptions{})
	assert.Nil(t, err)

	return svc, endpoints
}

func mockHTTPRouteTranslator(t *testing.T) (*translator, <-chan struct{}) {

	client := fake.NewSimpleClientset()
	informersFactory := informers.NewSharedInformerFactory(client, 0)
	svcInformer := informersFactory.Core().V1().Services().Informer()
	svcLister := informersFactory.Core().V1().Services().Lister()
	epLister, epInformer := kube.NewEndpointListerAndInformer(informersFactory, false)
	apisixClient := fakeapisix.NewSimpleClientset()
	apisixInformersFactory := apisixinformers.NewSharedInformerFactory(apisixClient, 0)

	newServiceAndEndpoints(t, client, "test", "svc", []int32{80, 443}, []int32{9080, 9443}, []string{"192.168.1.1", "192.168.1.2"})
	newServiceAndEndpoints(t, client, "test", "svc2", []int32{81, 444}, []int32{9081, 9444}, []string{"192.168.1.3", "192.168.1.4"})

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

	stopCh := make(chan struct{})
	defer close(stopCh)
	go svcInformer.Run(stopCh)
	go epInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, svcInformer.HasSynced)

	return tr, processCh
}

func TestTranslateGatewayHTTPRouteExactMatch(t *testing.T) {
	tr, processCh := mockHTTPRouteTranslator(t)
	<-processCh
	<-processCh

	httpRoute := &gatewayv1beta1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "http_route",
			Namespace: "test",
		},
		Spec: gatewayv1beta1.HTTPRouteSpec{
			Hostnames: []gatewayv1beta1.Hostname{
				"example.com",
			},
			Rules: []gatewayv1beta1.HTTPRouteRule{
				{
					Matches: []gatewayv1beta1.HTTPRouteMatch{
						{
							Path: &gatewayv1beta1.HTTPPathMatch{
								Type:  utils.PtrOf(gatewayv1beta1.PathMatchPathPrefix),
								Value: utils.PtrOf("/path"),
							},
							Headers: []gatewayv1beta1.HTTPHeaderMatch{
								{
									Type:  utils.PtrOf(gatewayv1beta1.HeaderMatchExact),
									Name:  "REFERER",
									Value: "api7.com",
								},
							},
							QueryParams: []gatewayv1beta1.HTTPQueryParamMatch{
								{
									Type:  utils.PtrOf(gatewayv1beta1.QueryParamMatchExact),
									Name:  "user",
									Value: "api7",
								},
								{
									Type:  utils.PtrOf(gatewayv1beta1.QueryParamMatchExact),
									Name:  "title",
									Value: "ingress",
								},
							},
							Method: utils.PtrOf(gatewayv1beta1.HTTPMethodGet),
						},
					},
					Filters: []gatewayv1beta1.HTTPRouteFilter{
						// TODO
					},
					BackendRefs: []gatewayv1beta1.HTTPBackendRef{
						{
							BackendRef: gatewayv1beta1.BackendRef{
								BackendObjectReference: gatewayv1beta1.BackendObjectReference{
									Kind:      refKind("Service"),
									Name:      "svc",
									Namespace: refNamespace("test"),
									Port:      refPortNumber(80),
								},
								Weight: refInt32(100), // TODO
							},
							Filters: []gatewayv1beta1.HTTPRouteFilter{
								// TODO
							},
						},
					},
				},
			},
		},
	}

	tctx, err := tr.TranslateGatewayHTTPRouteV1beta1(httpRoute)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(tctx.Routes))
	assert.Equal(t, 1, len(tctx.Upstreams))

	r := tctx.Routes[0]
	u := tctx.Upstreams[0]

	// Metadata
	// FIXME
	assert.NotEqual(t, "", u.ID)
	assert.Equal(t, u.ID, r.UpstreamId)

	// hosts
	assert.Equal(t, 1, len(r.Hosts))
	assert.Equal(t, "example.com", r.Hosts[0])

	// matches
	assert.Equal(t, "/path*", r.Uri)

	assert.Equal(t, 3, len(r.Vars))
	referer := r.Vars[0]
	argUser := r.Vars[1]
	argTitle := r.Vars[2]
	assert.Equal(t, []v1.StringOrSlice{{StrVal: "http_referer"}, {StrVal: "=="}, {StrVal: "api7.com"}}, referer)
	assert.Equal(t, []v1.StringOrSlice{{StrVal: "arg_user"}, {StrVal: "=="}, {StrVal: "api7"}}, argUser)
	assert.Equal(t, []v1.StringOrSlice{{StrVal: "arg_title"}, {StrVal: "=="}, {StrVal: "ingress"}}, argTitle)

	assert.Equal(t, 1, len(r.Methods))
	assert.Equal(t, "GET", r.Methods[0])

	// backend refs
	assert.Equal(t, "http", u.Scheme) // FIXME
	assert.Equal(t, 2, len(u.Nodes))
	assert.Equal(t, "192.168.1.1", u.Nodes[0].Host)
	assert.Equal(t, 9080, u.Nodes[0].Port)
	assert.Equal(t, "192.168.1.2", u.Nodes[1].Host)
	assert.Equal(t, 9080, u.Nodes[1].Port)
}

func TestTranslateGatewayHTTPRouteRegexMatch(t *testing.T) {
	tr, processCh := mockHTTPRouteTranslator(t)
	<-processCh
	<-processCh

	httpRoute := &gatewayv1beta1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "http_route",
			Namespace: "test",
		},
		Spec: gatewayv1beta1.HTTPRouteSpec{
			Hostnames: []gatewayv1beta1.Hostname{
				"example.com",
			},
			Rules: []gatewayv1beta1.HTTPRouteRule{
				{
					Matches: []gatewayv1beta1.HTTPRouteMatch{
						{
							Path: &gatewayv1beta1.HTTPPathMatch{
								Type:  utils.PtrOf(gatewayv1beta1.PathMatchRegularExpression),
								Value: utils.PtrOf("/path"),
							},
							Headers: []gatewayv1beta1.HTTPHeaderMatch{
								{
									Type:  utils.PtrOf(gatewayv1beta1.HeaderMatchRegularExpression),
									Name:  "REFERER",
									Value: "api7.com",
								},
							},
							QueryParams: []gatewayv1beta1.HTTPQueryParamMatch{
								{
									Type:  utils.PtrOf(gatewayv1beta1.QueryParamMatchRegularExpression),
									Name:  "user",
									Value: "api7",
								},
								{
									Type:  utils.PtrOf(gatewayv1beta1.QueryParamMatchRegularExpression),
									Name:  "title",
									Value: "ingress",
								},
							},
							Method: utils.PtrOf(gatewayv1beta1.HTTPMethodGet),
						},
					},
					Filters: []gatewayv1beta1.HTTPRouteFilter{
						// TODO
					},
					BackendRefs: []gatewayv1beta1.HTTPBackendRef{
						{
							BackendRef: gatewayv1beta1.BackendRef{
								BackendObjectReference: gatewayv1beta1.BackendObjectReference{
									Kind:      refKind("Service"),
									Name:      "svc",
									Namespace: refNamespace("test"),
									Port:      refPortNumber(80),
								},
								Weight: refInt32(100), // TODO
							},
							Filters: []gatewayv1beta1.HTTPRouteFilter{
								// TODO
							},
						},
					},
				},
			},
		},
	}

	tctx, err := tr.TranslateGatewayHTTPRouteV1beta1(httpRoute)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(tctx.Routes))
	assert.Equal(t, 1, len(tctx.Upstreams))

	r := tctx.Routes[0]
	u := tctx.Upstreams[0]

	// Metadata
	// FIXME
	assert.NotEqual(t, "", u.ID)
	assert.Equal(t, u.ID, r.UpstreamId)

	// hosts
	assert.Equal(t, 1, len(r.Hosts))
	assert.Equal(t, "example.com", r.Hosts[0])

	// matches
	assert.Equal(t, 4, len(r.Vars))
	uri := r.Vars[0]
	referer := r.Vars[1]
	argUser := r.Vars[2]
	argTitle := r.Vars[3]
	assert.Equal(t, []v1.StringOrSlice{{StrVal: "uri"}, {StrVal: "~~"}, {StrVal: "/path"}}, uri)
	assert.Equal(t, []v1.StringOrSlice{{StrVal: "http_referer"}, {StrVal: "~~"}, {StrVal: "api7.com"}}, referer)
	assert.Equal(t, []v1.StringOrSlice{{StrVal: "arg_user"}, {StrVal: "~~"}, {StrVal: "api7"}}, argUser)
	assert.Equal(t, []v1.StringOrSlice{{StrVal: "arg_title"}, {StrVal: "~~"}, {StrVal: "ingress"}}, argTitle)

	assert.Equal(t, 1, len(r.Methods))
	assert.Equal(t, "GET", r.Methods[0])

	// backend refs
	assert.Equal(t, "http", u.Scheme) // FIXME
	assert.Equal(t, 2, len(u.Nodes))
	assert.Equal(t, "192.168.1.1", u.Nodes[0].Host)
	assert.Equal(t, 9080, u.Nodes[0].Port)
	assert.Equal(t, "192.168.1.2", u.Nodes[1].Host)
	assert.Equal(t, 9080, u.Nodes[1].Port)
}

// TODO: Multiple Rules, Multiple Matches

func TestTranslateGatewayHTTPRouteMultipleBackendRefs(t *testing.T) {
	tr, processCh := mockHTTPRouteTranslator(t)
	<-processCh
	<-processCh

	httpRoute := &gatewayv1beta1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "http_route",
			Namespace: "test",
		},
		Spec: gatewayv1beta1.HTTPRouteSpec{
			Hostnames: []gatewayv1beta1.Hostname{
				"example.com",
			},
			Rules: []gatewayv1beta1.HTTPRouteRule{
				{
					Matches: []gatewayv1beta1.HTTPRouteMatch{
						{
							Path: &gatewayv1beta1.HTTPPathMatch{
								Type:  utils.PtrOf(gatewayv1beta1.PathMatchPathPrefix),
								Value: utils.PtrOf("/path"),
							},
						},
					},
					Filters: []gatewayv1beta1.HTTPRouteFilter{
						// TODO
					},
					BackendRefs: []gatewayv1beta1.HTTPBackendRef{
						{
							BackendRef: gatewayv1beta1.BackendRef{
								BackendObjectReference: gatewayv1beta1.BackendObjectReference{
									Kind:      refKind("Service"),
									Name:      "svc",
									Namespace: refNamespace("test"),
									Port:      refPortNumber(80),
								},
								Weight: refInt32(100), // TODO
							},
							Filters: []gatewayv1beta1.HTTPRouteFilter{
								// TODO
							},
						},
					},
				},
				{
					Matches: []gatewayv1beta1.HTTPRouteMatch{
						{
							Path: &gatewayv1beta1.HTTPPathMatch{
								Type:  utils.PtrOf(gatewayv1beta1.PathMatchPathPrefix),
								Value: utils.PtrOf("/path2"),
							},
						},
					},
					Filters: []gatewayv1beta1.HTTPRouteFilter{
						// TODO
					},
					BackendRefs: []gatewayv1beta1.HTTPBackendRef{
						{
							BackendRef: gatewayv1beta1.BackendRef{
								BackendObjectReference: gatewayv1beta1.BackendObjectReference{
									Kind:      refKind("Service"),
									Name:      "svc2",
									Namespace: refNamespace("test"),
									Port:      refPortNumber(81),
								},
								Weight: refInt32(100), // TODO
							},
							Filters: []gatewayv1beta1.HTTPRouteFilter{
								// TODO
							},
						},
					},
				},
			},
		},
	}

	tctx, err := tr.TranslateGatewayHTTPRouteV1beta1(httpRoute)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(tctx.Routes))
	assert.Equal(t, 2, len(tctx.Upstreams))

	// === route 1 ===
	r := tctx.Routes[0]
	u := tctx.Upstreams[0]

	// Metadata
	// FIXME
	assert.NotEqual(t, "", u.ID)
	assert.Equal(t, u.ID, r.UpstreamId)

	// hosts
	assert.Equal(t, 1, len(r.Hosts))
	assert.Equal(t, "example.com", r.Hosts[0])

	// matches
	assert.Equal(t, "/path*", r.Uri)

	// backend refs
	assert.Equal(t, "http", u.Scheme) // FIXME
	assert.Equal(t, 2, len(u.Nodes))
	assert.Equal(t, "192.168.1.1", u.Nodes[0].Host)
	assert.Equal(t, 9080, u.Nodes[0].Port)
	assert.Equal(t, "192.168.1.2", u.Nodes[1].Host)
	assert.Equal(t, 9080, u.Nodes[1].Port)

	// === route 2 ===
	r = tctx.Routes[1]
	u = tctx.Upstreams[1]

	// Metadata
	// FIXME
	assert.NotEqual(t, "", u.ID)
	assert.Equal(t, u.ID, r.UpstreamId)

	// hosts
	assert.Equal(t, 1, len(r.Hosts))
	assert.Equal(t, "example.com", r.Hosts[0])

	// matches
	assert.Equal(t, "/path2*", r.Uri)

	// backend refs
	assert.Equal(t, "http", u.Scheme) // FIXME
	assert.Equal(t, 2, len(u.Nodes))
	assert.Equal(t, "192.168.1.3", u.Nodes[0].Host)
	assert.Equal(t, 9081, u.Nodes[0].Port)
	assert.Equal(t, "192.168.1.4", u.Nodes[1].Host)
	assert.Equal(t, 9081, u.Nodes[1].Port)
}
