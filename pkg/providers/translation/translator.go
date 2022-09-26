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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	listerscorev1 "k8s.io/client-go/listers/core/v1"

	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/pod"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	DefaultWeight = 100
)

type TranslateError struct {
	Field  string
	Reason string
}

func (te *TranslateError) Error() string {
	return fmt.Sprintf("%s: %s", te.Field, te.Reason)
}

type Translator interface {
	// TranslateUpstreamConfigV2beta3 translates ApisixUpstreamConfig (part of ApisixUpstream)
	// to APISIX Upstream, it doesn't fill the the Upstream metadata and nodes.
	TranslateUpstreamConfigV2beta3(*configv2beta3.ApisixUpstreamConfig) (*apisixv1.Upstream, error)
	// TranslateUpstreamConfigV2 translates ApisixUpstreamConfig (part of ApisixUpstream)
	// to APISIX Upstream, it doesn't fill the the Upstream metadata and nodes.
	TranslateUpstreamConfigV2(*configv2.ApisixUpstreamConfig, *apisixv1.Upstream) error
	// TranslateService translates the K8s Service to APISIX Upstream nodes (cluster ip)
	TranslateService(*corev1.Service, intstr.IntOrString) (apisixv1.UpstreamNodes, error)
	// TranslateEndpoint translates the K8s Endpoint to APISIX Upstream nodes (pod ip)
	// according to the given port. Extra labels can be passed to filter the ultimate
	// upstream nodes.
	TranslateEndpoint(kube.Endpoint, intstr.IntOrString, types.Labels) (apisixv1.UpstreamNodes, error)
	// TranslateUpstream composes an upstream according to the
	// given namespace, name (searching Service/Endpoints) and port (filtering Endpoints).
	// The returned Upstream doesn't have metadata info.
	// It doesn't assign any metadata fields, so it's caller's responsibility to decide
	// the metadata.
	// Note the subset is used to filter the ultimate node list, only pods whose labels
	// matching the subset labels (defined in ApisixUpstream) will be selected.
	// When the subset is not found, the node list will be empty. When the subset is empty,
	// all pods IP will be filled.
	// ResolveGranularity determines the granularity of the generated nodes of upstream. It supports service/endpoint.
	TranslateUpstream(namespace string, name string, subset string, resolveGranularity string, port intstr.IntOrString) (*apisixv1.Upstream, error)
	// TranslateUpstreamNodes translates the K8s Endpoint to APISIX Upstream nodes
	// according to the given port. Extra labels can be passed to filter the ultimate
	// upstream nodes.
	TranslateUpstreamNodes(namespace string, name string, resolveGranularity string, port intstr.IntOrString, labels types.Labels) (apisixv1.UpstreamNodes, error)
}

// TranslatorOptions contains options to help Translator
// work well.
type TranslatorOptions struct {
	APIVersion string

	EndpointLister       kube.EndpointLister
	ServiceLister        listerscorev1.ServiceLister
	SecretLister         listerscorev1.SecretLister
	PodLister            listerscorev1.PodLister
	ApisixUpstreamLister kube.ApisixUpstreamLister

	PodProvider pod.Provider
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
