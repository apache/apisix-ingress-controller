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
package kube

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	listerscorev1 "k8s.io/client-go/listers/core/v1"

	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
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
	TranslateUpstreamNodes(*corev1.Endpoints, int32) ([]apisixv1.UpstreamNode, error)
	// TranslateUpstreamConfig translates ApisixUpstreamConfig (part of ApisixUpstream)
	// to APISIX Upstream, it doesn't fill the the Upstream metadata and nodes.
	TranslateUpstreamConfig(config *configv1.ApisixUpstreamConfig) (*apisixv1.Upstream, error)
	// TranslateUpstream composes an upstream according to the
	// given namespace, name (searching Service/Endpoints) and port (filtering Endpoints).
	// The returned Upstream doesn't have metadata info.
	TranslateUpstream(string, string, int32) (*apisixv1.Upstream, error)
}

// TranslatorOptions contains options to help Translator
// work well.
type TranslatorOptions struct {
	EndpointsLister      listerscorev1.EndpointsLister
	ServiceLister        listerscorev1.ServiceLister
	ApisixUpstreamLister listersv1.ApisixUpstreamLister
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

func (t *translator) TranslateUpstreamNodes(endpoints *corev1.Endpoints, port int32) ([]apisixv1.UpstreamNode, error) {
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
	var nodes []apisixv1.UpstreamNode
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
					IP:   addr.IP,
					Port: int(epPort.Port),
					// FIXME Custom node weight
					Weight: _defaultWeight,
				})
			}
		}
	}
	return nodes, nil
}
