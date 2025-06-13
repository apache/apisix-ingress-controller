// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package adc

import (
	"errors"
	"fmt"
	"slices"

	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	v1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	types "github.com/apache/apisix-ingress-controller/internal/types"
)

func (d *adcClient) getConfigsForGatewayProxy(tctx *provider.TranslateContext, gatewayProxy *v1alpha1.GatewayProxy) (*adcConfig, error) {
	if gatewayProxy == nil || gatewayProxy.Spec.Provider == nil {
		return nil, nil
	}

	provider := gatewayProxy.Spec.Provider
	if provider.Type != v1alpha1.ProviderTypeControlPlane || provider.ControlPlane == nil {
		return nil, nil
	}

	config := adcConfig{
		Name: k8stypes.NamespacedName{Namespace: gatewayProxy.Namespace, Name: gatewayProxy.Name}.String(),
	}

	if provider.ControlPlane.TlsVerify != nil {
		config.TlsVerify = *provider.ControlPlane.TlsVerify
	}

	if provider.ControlPlane.Auth.Type == v1alpha1.AuthTypeAdminKey && provider.ControlPlane.Auth.AdminKey != nil {
		if provider.ControlPlane.Auth.AdminKey.ValueFrom != nil && provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef != nil {
			secretRef := provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef
			secret, ok := tctx.Secrets[k8stypes.NamespacedName{
				// we should use gateway proxy namespace
				Namespace: gatewayProxy.GetNamespace(),
				Name:      secretRef.Name,
			}]
			if ok {
				if token, ok := secret.Data[secretRef.Key]; ok {
					config.Token = string(token)
				}
			}
		} else if provider.ControlPlane.Auth.AdminKey.Value != "" {
			config.Token = provider.ControlPlane.Auth.AdminKey.Value
		}
	}

	if config.Token == "" {
		return nil, errors.New("no token found")
	}

	endpoints := provider.ControlPlane.Endpoints
	if len(endpoints) > 0 {
		config.ServerAddrs = endpoints
		return &config, nil
	}

	if provider.ControlPlane.Service != nil {
		namespacedName := k8stypes.NamespacedName{
			Namespace: gatewayProxy.Namespace,
			Name:      provider.ControlPlane.Service.Name,
		}
		_, ok := tctx.Services[namespacedName]
		if !ok {
			return nil, errors.New("no service found for service reference")
		}
		endpoint := tctx.EndpointSlices[namespacedName]
		if endpoint == nil {
			return nil, nil
		}
		upstreamNodes, err := d.translator.TranslateBackendRef(tctx, v1.BackendRef{
			BackendObjectReference: v1.BackendObjectReference{
				Name: v1.ObjectName(provider.ControlPlane.Service.Name),
				Port: ptr.To(v1.PortNumber(provider.ControlPlane.Service.Port)),
			},
		})
		if err != nil {
			return nil, err
		}
		for _, node := range upstreamNodes {
			config.ServerAddrs = append(config.ServerAddrs, fmt.Sprintf("http://%s:%d", node.Host, node.Port))
		}
	}

	return &config, nil
}

func (d *adcClient) deleteConfigs(rk types.NamespacedNameKind) {
	d.Lock()
	defer d.Unlock()
	delete(d.configs, rk)
	delete(d.parentRefs, rk)
}

func (d *adcClient) getParentRefs(rk types.NamespacedNameKind) []types.NamespacedNameKind {
	d.Lock()
	defer d.Unlock()
	return d.parentRefs[rk]
}

func (d *adcClient) getConfigs(rk types.NamespacedNameKind) []adcConfig {
	d.Lock()
	defer d.Unlock()
	parentRefs := d.parentRefs[rk]
	configs := make([]adcConfig, 0, len(parentRefs))
	for _, parentRef := range parentRefs {
		if config, ok := d.configs[parentRef]; ok {
			configs = append(configs, config)
		}
	}
	return configs
}

func (d *adcClient) updateConfigs(rk types.NamespacedNameKind, tctx *provider.TranslateContext) error {
	d.Lock()
	defer d.Unlock()

	// set parent refs
	d.parentRefs[rk] = tctx.ResourceParentRefs[rk]
	parentRefs := d.parentRefs[rk]

	for _, parentRef := range parentRefs {
		gatewayProxy, ok := tctx.GatewayProxies[parentRef]
		if !ok {
			log.Debugw("no gateway proxy found for parent ref", zap.Any("parentRef", parentRef))
			continue
		}
		config, err := d.getConfigsForGatewayProxy(tctx, &gatewayProxy)
		if err != nil {
			return err
		}
		if config == nil {
			log.Debugw("no config found for gateway proxy", zap.Any("parentRef", parentRef))
			continue
		}
		d.configs[parentRef] = *config
	}

	return nil
}

func (d *adcClient) findConfigsToDelete(oldParentRefs, newParentRefs []types.NamespacedNameKind) []adcConfig {
	var deleteConfigs []adcConfig
	for _, parentRef := range oldParentRefs {
		if !slices.ContainsFunc(newParentRefs, func(rk types.NamespacedNameKind) bool {
			return rk.Kind == parentRef.Kind && rk.Namespace == parentRef.Namespace && rk.Name == parentRef.Name
		}) {
			deleteConfigs = append(deleteConfigs, d.configs[parentRef])
		}
	}
	return deleteConfigs
}
