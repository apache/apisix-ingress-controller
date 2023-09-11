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
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type TranslatorOptions struct {
	Apisix           apisix.APISIX
	ClusterName      string
	IngressClassName string

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

	// TranslateRouteV2 translates the configv2.ApisixRoute object into several Route,
	// Upstream and PluginConfig resources.
	TranslateRouteV2(*configv2.ApisixRoute) (*translation.TranslateContext, error)
	// GenerateRouteV2DeleteMark translates the configv2.ApisixRoute object into several Route,
	// Upstream and PluginConfig resources not strictly, only used for delete event.
	GenerateRouteV2DeleteMark(*configv2.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateOldRoute get route and stream_route objects from cache
	// Build upstream and plugin_config through route and stream_route
	TranslateOldRoute(kube.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateSSLV2 translates the configv2.ApisixTls object into the APISIX SSL resource.
	TranslateSSLV2(*configv2.ApisixTls) (*apisixv1.Ssl, error)
	// TranslateClusterConfigV2 translates the configv2.ApisixClusterConfig object into the APISIX
	// Global Rule resource.
	TranslateClusterConfigV2(*configv2.ApisixClusterConfig) (*apisixv1.GlobalRule, error)
	// TranslateApisixConsumerV2 translates the configv2.APisixConsumer object into the APISIX Consumer
	// resource.
	TranslateApisixConsumerV2(ac *configv2.ApisixConsumer) (*apisixv1.Consumer, error)
	// TranslatePluginConfigV2 translates the configv2.ApisixPluginConfig object into several PluginConfig
	// resources.
	TranslatePluginConfigV2(*configv2.ApisixPluginConfig) (*translation.TranslateContext, error)
	// GeneratePluginConfigV2DeleteMark translates the configv2.ApisixPluginConfig object into several PluginConfig
	// resources not strictly, only used for delete event.
	GeneratePluginConfigV2DeleteMark(*configv2.ApisixPluginConfig) (*translation.TranslateContext, error)

	TranslateRouteMatchExprs(nginxVars []configv2.ApisixRouteHTTPMatchExpr) ([][]apisixv1.StringOrSlice, error)

	// TranslateApisixUpstreamExternalNodes translates an ApisixUpstream with external nodes to APISIX nodes.
	TranslateApisixUpstreamExternalNodes(au *configv2.ApisixUpstream) ([]apisixv1.UpstreamNode, error)

	TranslateGlobalRule(kube.ApisixGlobalRule) (*translation.TranslateContext, error)
}

func NewApisixTranslator(opts *TranslatorOptions, t translation.Translator) ApisixTranslator {
	return &translator{
		TranslatorOptions: opts,
		Translator:        t,
	}
}
