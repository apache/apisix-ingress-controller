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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	listerscorev1 "k8s.io/client-go/listers/core/v1"

	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	listersv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v1"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_defaultWeight = 100
)

type translateError struct {
	field  string
	reason string
}

func (te *translateError) Error() string {
	return fmt.Sprintf("%s: %s", te.field, te.reason)
}

// Translator translates Apisix* CRD resources to the description in APISIX.
type Translator interface {
	// TranslateUpstreamNodes translate Endpoints resources to APISIX Upstream nodes
	// according to the give port.
	TranslateUpstreamNodes(*corev1.Endpoints, int32) (apisixv1.UpstreamNodes, error)
	// TranslateUpstreamConfig translates ApisixUpstreamConfig (part of ApisixUpstream)
	// to APISIX Upstream, it doesn't fill the the Upstream metadata and nodes.
	TranslateUpstreamConfig(*configv1.ApisixUpstreamConfig) (*apisixv1.Upstream, error)
	// TranslateUpstream composes an upstream according to the
	// given namespace, name (searching Service/Endpoints) and port (filtering Endpoints).
	// The returned Upstream doesn't have metadata info.
	// It doesn't assign any metadata fields, so it's caller's responsibility to decide
	// the metadata.
	TranslateUpstream(string, string, int32) (*apisixv1.Upstream, error)
	// TranslateIngress composes a couple of APISIX Routes and upstreams according
	// to the given Ingress resource.
	TranslateIngress(kube.Ingress) (*TranslateContext, error)
	// TranslateRouteV1 translates the configv1.ApisixRoute object into several Route
	// and Upstream resources.
	TranslateRouteV1(*configv1.ApisixRoute) (*TranslateContext, error)
	// TranslateRouteV2alpha1 translates the configv2alpha1.ApisixRoute object into several Route
	// and Upstream resources.
	TranslateRouteV2alpha1(*configv2alpha1.ApisixRoute) (*TranslateContext, error)
	// TranslateSSL translates the configv2alpha1.ApisixTls object into the APISIX SSL resource.
	TranslateSSL(*configv1.ApisixTls) (*apisixv1.Ssl, error)
	// TranslateClusterConfig translates the configv2alpha1.ApisixClusterConfig object into the APISIX
	// Global Rule resource.
	TranslateClusterConfig(*configv2alpha1.ApisixClusterConfig) (*apisixv1.GlobalRule, error)
	// TranslateApisixConsumer translates the configv2alpha1.APisixConsumer object into the APISIX Consumer
	// resource.
	TranslateApisixConsumer(*configv2alpha1.ApisixConsumer) (*apisixv1.Consumer, error)
}

// TranslatorOptions contains options to help Translator
// work well.
type TranslatorOptions struct {
	EndpointsLister      listerscorev1.EndpointsLister
	ServiceLister        listerscorev1.ServiceLister
	ApisixUpstreamLister listersv1.ApisixUpstreamLister
	SecretLister         listerscorev1.SecretLister
}

type translator struct {
	*TranslatorOptions
}

// NewTranslator initializes a APISIX CRD resources Translator.
func NewTranslator(opts *TranslatorOptions) Translator {
	return &translator{
		TranslatorOptions: opts,
	}
}

func (t *translator) TranslateUpstreamConfig(au *configv1.ApisixUpstreamConfig) (*apisixv1.Upstream, error) {
	ups := apisixv1.NewDefaultUpstream()
	if err := t.translateUpstreamScheme(au.Scheme, ups); err != nil {
		return nil, err
	}
	if err := t.translateUpstreamLoadBalancer(au.LoadBalancer, ups); err != nil {
		return nil, err
	}
	if err := t.translateUpstreamHealthCheck(au.HealthCheck, ups); err != nil {
		return nil, err
	}
	if err := t.translateUpstreamRetriesAndTimeout(au.Retries, au.Timeout, ups); err != nil {
		return nil, err
	}
	return ups, nil
}

func (t *translator) TranslateUpstream(namespace, name string, port int32) (*apisixv1.Upstream, error) {
	endpoints, err := t.EndpointsLister.Endpoints(namespace).Get(name)
	if err != nil {
		return nil, &translateError{
			field:  "endpoints",
			reason: err.Error(),
		}
	}
	nodes, err := t.TranslateUpstreamNodes(endpoints, port)
	if err != nil {
		return nil, err
	}
	ups := apisixv1.NewDefaultUpstream()
	au, err := t.ApisixUpstreamLister.ApisixUpstreams(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			ups.Nodes = nodes
			return ups, nil
		}
		return nil, &translateError{
			field:  "ApisixUpstream",
			reason: err.Error(),
		}
	}
	upsCfg := &au.Spec.ApisixUpstreamConfig
	for _, pls := range au.Spec.PortLevelSettings {
		if pls.Port == port {
			upsCfg = &pls.ApisixUpstreamConfig
			break
		}
	}
	ups, err = t.TranslateUpstreamConfig(upsCfg)
	if err != nil {
		return nil, err
	}
	ups.Nodes = nodes
	return ups, nil
}

func (t *translator) TranslateUpstreamNodes(endpoints *corev1.Endpoints, port int32) (apisixv1.UpstreamNodes, error) {
	svc, err := t.ServiceLister.Services(endpoints.Namespace).Get(endpoints.Name)
	if err != nil {
		return nil, &translateError{
			field:  "service",
			reason: err.Error(),
		}
	}

	var svcPort *corev1.ServicePort
	for _, exposePort := range svc.Spec.Ports {
		if exposePort.Port == port {
			svcPort = &exposePort
			break
		}
	}
	if svcPort == nil {
		return nil, &translateError{
			field:  "service.spec.ports",
			reason: "port not defined",
		}
	}
	// As nodes is not optional, here we create an empty slice,
	// not a nil slice.
	nodes := make(apisixv1.UpstreamNodes, 0)
	for _, subset := range endpoints.Subsets {
		var epPort *corev1.EndpointPort
		for _, port := range subset.Ports {
			if port.Name == svcPort.Name {
				epPort = &port
				break
			}
		}
		if epPort != nil {
			for _, addr := range subset.Addresses {
				nodes = append(nodes, apisixv1.UpstreamNode{
					Host: addr.IP,
					Port: int(epPort.Port),
					// FIXME Custom node weight
					Weight: _defaultWeight,
				})
			}
		}
	}
	return nodes, nil
}

func (t *translator) TranslateIngress(ing kube.Ingress) (*TranslateContext, error) {
	switch ing.GroupVersion() {
	case kube.IngressV1:
		return t.translateIngressV1(ing.V1())
	case kube.IngressV1beta1:
		return t.translateIngressV1beta1(ing.V1beta1())
	case kube.IngressExtensionsV1beta1:
		return t.translateIngressExtensionsV1beta1(ing.ExtensionsV1beta1())
	default:
		return nil, fmt.Errorf("translator: source group version not supported: %s", ing.GroupVersion())
	}
}
