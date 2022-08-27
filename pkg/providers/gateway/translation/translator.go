// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
package gateway_translation

import (
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/pkg/providers/gateway/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
)

type TranslatorOptions struct {
	KubeTranslator translation.Translator
}

type translator struct {
	*TranslatorOptions
}

type Translator interface {
	// TranslateGatewayV1Alpha2 translates Gateway to internal configurations
	TranslateGatewayV1Alpha2(gateway *gatewayv1alpha2.Gateway) (map[string]*types.ListenerConf, error)
	// TranslateGatewayHTTPRouteV1Alpha2 translates Gateway API HTTPRoute to APISIX resources
	TranslateGatewayHTTPRouteV1Alpha2(httpRoute *gatewayv1alpha2.HTTPRoute) (*translation.TranslateContext, error)
	// TranslateGatewayTLSRouteV1Alpha2 translates Gateway API TLSRoute to APISIX resources
	TranslateGatewayTLSRouteV1Alpha2(tlsRoute *gatewayv1alpha2.TLSRoute) (*translation.TranslateContext, error)
	// TranslateGatewayUDPRouteV1Alpha2 translates Gateway API UDPRoute to APISIX resources
	TranslateGatewayUDPRouteV1Alpha2(udpRoute *gatewayv1alpha2.UDPRoute) (*translation.TranslateContext, error)
}

// NewTranslator initializes a APISIX CRD resources Translator.
func NewTranslator(opts *TranslatorOptions) Translator {
	return &translator{
		TranslatorOptions: opts,
	}
}
