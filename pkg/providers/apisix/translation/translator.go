// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package translation

import (
	listerscorev1 "k8s.io/client-go/listers/core/v1"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type TranslatorOptions struct {
	Apisix      apisix.APISIX
	ClusterName string

	ApisixUpstreamLister kube.ApisixUpstreamLister
	ServiceLister        listerscorev1.ServiceLister
	SecretLister         listerscorev1.SecretLister
}

type translator struct {
	*TranslatorOptions
	translation.Translator
}

type ApisixTranslator interface {
	translation.Translator

	// TranslateRouteV2beta2 translates the configv2beta2.ApisixRoute object into several Route,
	// and Upstream resources.
	TranslateRouteV2beta2(*configv2beta2.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateRouteV2beta2NotStrictly translates the configv2beta2.ApisixRoute object into several Route,
	// and Upstream  resources not strictly, only used for delete event.
	TranslateRouteV2beta2NotStrictly(*configv2beta2.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateRouteV2beta3 translates the configv2beta3.ApisixRoute object into several Route,
	// Upstream and PluginConfig resources.
	TranslateRouteV2beta3(*configv2beta3.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateRouteV2beta3NotStrictly translates the configv2beta3.ApisixRoute object into several Route,
	// Upstream and PluginConfig resources not strictly, only used for delete event.
	TranslateRouteV2beta3NotStrictly(*configv2beta3.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateRouteV2 translates the configv2.ApisixRoute object into several Route,
	// Upstream and PluginConfig resources.
	TranslateRouteV2(*configv2.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateRouteV2NotStrictly translates the configv2.ApisixRoute object into several Route,
	// Upstream and PluginConfig resources not strictly, only used for delete event.
	TranslateRouteV2NotStrictly(*configv2.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateOldRoute get route and stream_route objects from cache
	// Build upstream and plugin_config through route and stream_route
	TranslateOldRoute(kube.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateSSLV2Beta3 translates the configv2beta3.ApisixTls object into the APISIX SSL resource.
	TranslateSSLV2Beta3(*configv2beta3.ApisixTls) (*apisixv1.Ssl, error)
	// TranslateSSLV2 translates the configv2.ApisixTls object into the APISIX SSL resource.
	TranslateSSLV2(*configv2.ApisixTls) (*apisixv1.Ssl, error)
	// TranslateClusterConfig translates the configv2beta3.ApisixClusterConfig object into the APISIX
	// Global Rule resource.
	TranslateClusterConfigV2beta3(*configv2beta3.ApisixClusterConfig) (*apisixv1.GlobalRule, error)
	// TranslateClusterConfigV2 translates the configv2.ApisixClusterConfig object into the APISIX
	// Global Rule resource.
	TranslateClusterConfigV2(*configv2.ApisixClusterConfig) (*apisixv1.GlobalRule, error)
	// TranslateApisixConsumer translates the configv2beta3.APisixConsumer object into the APISIX Consumer
	// resource.
	TranslateApisixConsumerV2beta3(*configv2beta3.ApisixConsumer) (*apisixv1.Consumer, error)
	// TranslateApisixConsumerV2 translates the configv2beta3.APisixConsumer object into the APISIX Consumer
	// resource.
	TranslateApisixConsumerV2(ac *configv2.ApisixConsumer) (*apisixv1.Consumer, error)
	// TranslatePluginConfigV2beta3 translates the configv2.ApisixPluginConfig object into several PluginConfig
	// resources.
	TranslatePluginConfigV2beta3(*configv2beta3.ApisixPluginConfig) (*translation.TranslateContext, error)
	// TranslatePluginConfigV2beta3NotStrictly translates the configv2beta3.ApisixPluginConfig object into several PluginConfig
	// resources not strictly, only used for delete event.
	TranslatePluginConfigV2beta3NotStrictly(*configv2beta3.ApisixPluginConfig) (*translation.TranslateContext, error)
	// TranslatePluginConfigV2 translates the configv2.ApisixPluginConfig object into several PluginConfig
	// resources.
	TranslatePluginConfigV2(*configv2.ApisixPluginConfig) (*translation.TranslateContext, error)
	// TranslatePluginConfigV2NotStrictly translates the configv2.ApisixPluginConfig object into several PluginConfig
	// resources not strictly, only used for delete event.
	TranslatePluginConfigV2NotStrictly(*configv2.ApisixPluginConfig) (*translation.TranslateContext, error)

	TranslateRouteMatchExprs(nginxVars []configv2.ApisixRouteHTTPMatchExpr) ([][]apisixv1.StringOrSlice, error)

	// TranslateApisixUpstreamExternalNodes translates an ApisixUpstream with external nodes to APISIX nodes.
	TranslateApisixUpstreamExternalNodes(au *configv2.ApisixUpstream) ([]apisixv1.UpstreamNode, error)
}

func NewApisixTranslator(opts *TranslatorOptions, t translation.Translator) ApisixTranslator {
	return &translator{
		TranslatorOptions: opts,
		Translator:        t,
	}
}
