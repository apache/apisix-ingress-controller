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
	"errors"

	"go.uber.org/zap"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/gateway/types"
)

const (
	kindUDPRoute  gatewayv1beta1.Kind = "UDPRoute"
	kindTCPRoute  gatewayv1beta1.Kind = "TCPRoute"
	kindTLSRoute  gatewayv1beta1.Kind = "TLSRoute"
	kindHTTPRoute gatewayv1beta1.Kind = "HTTPRoute"
)

func (t *translator) TranslateGatewayV1beta1(gateway *gatewayv1beta1.Gateway) (map[string]*types.ListenerConf, error) {
	listeners := make(map[string]*types.ListenerConf)

	for i, listener := range gateway.Spec.Listeners {
		allowedKinds, err := getAllowedKinds(listener)
		if err != nil {
			return nil, err
		}
		if len(allowedKinds) == 0 {
			log.Warnw("listener allowed kinds is empty",
				zap.String("gateway", gateway.Name),
				zap.String("namespace", gateway.Namespace),
				zap.Int("listener_index", i),
			)
			continue
		}

		err = validateListenerConfigurations(gateway, i, allowedKinds, listener)
		if err != nil {
			// TODO: Update CRD status
			log.Warnw("invalid listener conf",
				zap.Error(err),
				zap.String("gateway", gateway.Name),
				zap.String("namespace", gateway.Namespace),
				zap.Int("listener_index", i),
			)
			continue
		}

		conf := &types.ListenerConf{
			Namespace:      gateway.Namespace,
			Name:           gateway.Name,
			SectionName:    string(listener.Name),
			Protocol:       listener.Protocol,
			Port:           listener.Port,
			RouteNamespace: nil,
			AllowedKinds:   allowedKinds,
		}

		if listener.AllowedRoutes.Namespaces != nil {
			conf.RouteNamespace = listener.AllowedRoutes.Namespaces
		}

		listeners[conf.SectionName] = conf
	}

	return listeners, nil
}

func validateListenerConfigurations(gateway *gatewayv1beta1.Gateway, idx int, allowedKinds []gatewayv1beta1.RouteGroupKind,
	listener gatewayv1beta1.Listener) error {
	// Check protocols and allowedKinds
	protocol := listener.Protocol
	if protocol == gatewayv1beta1.HTTPProtocolType || protocol == gatewayv1beta1.TCPProtocolType || protocol == gatewayv1beta1.UDPProtocolType {
		// Non-TLS
		if listener.TLS != nil {
			return errors.New("non-empty TLS conf for protocol " + string(protocol))
		}
		if protocol == gatewayv1beta1.HTTPProtocolType {
			if len(allowedKinds) != 1 || allowedKinds[0].Kind != kindHTTPRoute {
				return errors.New("HTTP protocol must allow route type HTTPRoute")
			}
		} else if protocol == gatewayv1beta1.TCPProtocolType {
			if len(allowedKinds) != 1 || allowedKinds[0].Kind != kindTCPRoute {
				return errors.New("TCP protocol must allow route type TCPRoute")
			}
		} else if protocol == gatewayv1beta1.UDPProtocolType {
			if len(allowedKinds) != 1 || allowedKinds[0].Kind != kindUDPRoute {
				return errors.New("UDP protocol must allow route type UDPRoute")
			}
		}

	} else if protocol == gatewayv1beta1.HTTPSProtocolType || protocol == gatewayv1beta1.TLSProtocolType {
		// TLS
		if listener.TLS == nil {
			return errors.New("empty TLS conf for protocol " + string(protocol))
		}

		if *listener.TLS.Mode == gatewayv1beta1.TLSModeTerminate {
			if len(listener.TLS.CertificateRefs) == 0 {
				return errors.New("TLS mode Terminate requires CertificateRefs")
			}

			if len(listener.TLS.CertificateRefs) > 1 {
				log.Warnw("only the first CertificateRefs take effect",
					zap.String("gateway", gateway.Name),
					zap.String("namespace", gateway.Namespace),
					zap.Int("listener_index", idx),
				)
			}
		} else {
			if len(listener.TLS.CertificateRefs) != 0 {
				log.Warnw("no CertificateRefs will take effect in non-terminate TLS mode",
					zap.String("gateway", gateway.Name),
					zap.String("namespace", gateway.Namespace),
					zap.Int("listener_index", idx),
				)
			}
		}

		if protocol == gatewayv1beta1.HTTPSProtocolType {
			if *listener.TLS.Mode != gatewayv1beta1.TLSModeTerminate {
				return errors.New("TLS mode for HTTPS protocol must be Terminate")
			}
			if len(allowedKinds) != 1 || allowedKinds[0].Kind != kindHTTPRoute {
				return errors.New("HTTP protocol must allow route type HTTPRoute")
			}
		} else if protocol == gatewayv1beta1.TLSProtocolType {
			for _, kind := range allowedKinds {
				if kind.Kind != kindTLSRoute && kind.Kind != kindTCPRoute {
					return errors.New("TLS protocol only support route type TLSRoute and TCPRoute")
				}
			}
		}
	}

	return nil
}

func getAllowedKinds(listener gatewayv1beta1.Listener) ([]gatewayv1beta1.RouteGroupKind, error) {
	var expectedKinds []gatewayv1beta1.RouteGroupKind
	group := gatewayv1beta1.Group(gatewayv1beta1.GroupName)

	var kind gatewayv1beta1.Kind
	switch listener.Protocol {
	case gatewayv1beta1.HTTPProtocolType, gatewayv1beta1.HTTPSProtocolType:
		kind = kindHTTPRoute
	case gatewayv1beta1.TLSProtocolType:
		kind = kindTLSRoute
	case gatewayv1beta1.TCPProtocolType:
		kind = kindTCPRoute
	case gatewayv1beta1.UDPProtocolType:
		kind = kindUDPRoute
	default:
		return nil, errors.New("unknown protocol " + string(listener.Protocol))
	}

	expectedKinds = []gatewayv1beta1.RouteGroupKind{
		{
			Group: &group,
			Kind:  kind,
		},
	}

	if listener.AllowedRoutes == nil || len(listener.AllowedRoutes.Kinds) == 0 {
		return expectedKinds, nil
	}

	uniqueAllowedKinds := make(map[gatewayv1beta1.Kind]struct{})
	var allowedKinds []gatewayv1beta1.RouteGroupKind

	for _, kind := range listener.AllowedRoutes.Kinds {
		expected := false
		for _, expectedKind := range expectedKinds {
			if kind.Kind == expectedKind.Kind &&
				kind.Group != nil && *kind.Group == *expectedKind.Group {
				expected = true
				break
			}
		}
		if expected {
			if _, ok := uniqueAllowedKinds[kind.Kind]; !ok {
				uniqueAllowedKinds[kind.Kind] = struct{}{}
				allowedKinds = append(allowedKinds, kind)
			}
		}
	}

	return allowedKinds, nil
}
