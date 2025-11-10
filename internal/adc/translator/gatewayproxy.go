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

package translator

import (
	"fmt"
	"net"
	"strconv"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	types "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func (t *Translator) TranslateGatewayProxyToConfig(tctx *provider.TranslateContext, gatewayProxy *v1alpha1.GatewayProxy, resolveEndpoints bool) (*types.Config, error) {
	if gatewayProxy == nil || gatewayProxy.Spec.Provider == nil {
		return nil, nil
	}

	provider := gatewayProxy.Spec.Provider
	if provider.Type != v1alpha1.ProviderTypeControlPlane || provider.ControlPlane == nil {
		return nil, nil
	}
	cp := provider.ControlPlane

	cfg := types.Config{
		Name:        utils.NamespacedNameKind(gatewayProxy).String(),
		BackendType: cp.Mode,
	}

	if cp.TlsVerify != nil {
		cfg.TlsVerify = *cp.TlsVerify
	}

	if cp.Auth.Type == v1alpha1.AuthTypeAdminKey && cp.Auth.AdminKey != nil {
		if cp.Auth.AdminKey.ValueFrom != nil && cp.Auth.AdminKey.ValueFrom.SecretKeyRef != nil {
			secretRef := cp.Auth.AdminKey.ValueFrom.SecretKeyRef
			secret, ok := tctx.Secrets[k8stypes.NamespacedName{
				// we should use gateway proxy namespace
				Namespace: gatewayProxy.GetNamespace(),
				Name:      secretRef.Name,
			}]
			if ok {
				if token, ok := secret.Data[secretRef.Key]; ok {
					cfg.Token = string(token)
				}
			}
		} else if cp.Auth.AdminKey.Value != "" {
			cfg.Token = cp.Auth.AdminKey.Value
		}
	}

	if cfg.Token == "" {
		return nil, errors.New("no token found")
	}

	endpoints := cp.Endpoints
	if len(endpoints) > 0 {
		cfg.ServerAddrs = endpoints
		return &cfg, nil
	}

	// If Mode is empty, use the default static configuration.
	// If Mode is set, resolve endpoints only when the ControlPlane is in standalone mode.
	if cp.Mode != "" {
		resolveEndpoints = cp.Mode == string(config.ProviderTypeStandalone)
	}

	if cp.Service != nil {
		namespacedName := k8stypes.NamespacedName{
			Namespace: gatewayProxy.Namespace,
			Name:      cp.Service.Name,
		}
		svc, ok := tctx.Services[namespacedName]
		if !ok {
			return nil, fmt.Errorf("no service found for service reference: %s", namespacedName)
		}

		// APISIXStandalone, configurations need to be sent to each data plane instance;
		// In other cases, the service is directly accessed as the adc backend server address.
		if resolveEndpoints {
			endpoint := tctx.EndpointSlices[namespacedName]
			if endpoint == nil {
				return nil, nil
			}
			upstreamNodes, _, err := t.TranslateBackendRefWithFilter(tctx, gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Name:      gatewayv1.ObjectName(cp.Service.Name),
					Namespace: (*gatewayv1.Namespace)(&gatewayProxy.Namespace),
					Port:      ptr.To(gatewayv1.PortNumber(cp.Service.Port)),
				},
			}, func(endpoint *discoveryv1.Endpoint) bool {
				if endpoint.Conditions.Terminating != nil && *endpoint.Conditions.Terminating {
					t.Log.V(1).Info("skip terminating endpoint", "endpoint", endpoint)
					return false
				}
				return true
			})
			if err != nil {
				return nil, err
			}
			for _, node := range upstreamNodes {
				cfg.ServerAddrs = append(cfg.ServerAddrs, "http://"+net.JoinHostPort(node.Host, strconv.Itoa(node.Port)))
			}
		} else {
			refPort := cp.Service.Port
			var serverAddr string
			if svc.Spec.Type == corev1.ServiceTypeExternalName {
				serverAddr = fmt.Sprintf("http://%s:%d", svc.Spec.ExternalName, refPort)
			} else {
				serverAddr = fmt.Sprintf("http://%s.%s.svc:%d", cp.Service.Name, gatewayProxy.Namespace, refPort)
			}
			cfg.ServerAddrs = []string{serverAddr}
		}

		t.Log.V(1).Info("add server address to config.ServiceAddrs", "config.ServerAddrs", cfg.ServerAddrs)
	}

	return &cfg, nil
}
