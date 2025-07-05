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

package translator

import (
	"cmp"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/api7/gopkg/pkg/log"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	pkgutils "github.com/apache/apisix-ingress-controller/pkg/utils"
)

func (t *Translator) TranslateApisixRoute(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute) (result *TranslateResult, err error) {
	result = &TranslateResult{}
	for ruleIndex, rule := range ar.Spec.HTTP {
		service, err := t.translateHTTPRule(tctx, ar, rule, ruleIndex)
		if err != nil {
			return nil, err
		}
		result.Services = append(result.Services, service)
	}
	return result, nil
}

func (t *Translator) translateHTTPRule(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP, ruleIndex int) (*adc.Service, error) {
	timeout := t.buildTimeout(rule)
	plugins := t.buildPlugins(tctx, ar, rule)

	vars, err := rule.Match.NginxVars.ToVars()
	if err != nil {
		return nil, err
	}

	service := t.buildService(ar, rule, ruleIndex)
	t.buildRoute(ar, service, rule, plugins, timeout, vars)
	t.buildUpstream(tctx, service, ar, rule)

	return service, nil
}

func (t *Translator) buildTimeout(rule apiv2.ApisixRouteHTTP) *adc.Timeout {
	if rule.Timeout == nil {
		return nil
	}
	defaultTimeout := metav1.Duration{Duration: apiv2.DefaultUpstreamTimeout}
	return &adc.Timeout{
		Connect: cmp.Or(int(rule.Timeout.Connect.Seconds()), int(defaultTimeout.Seconds())),
		Read:    cmp.Or(int(rule.Timeout.Read.Seconds()), int(defaultTimeout.Seconds())),
		Send:    cmp.Or(int(rule.Timeout.Send.Seconds()), int(defaultTimeout.Seconds())),
	}
}

func (t *Translator) buildPlugins(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP) adc.Plugins {
	plugins := make(adc.Plugins)

	// Load plugins from referenced PluginConfig
	t.loadPluginConfigPlugins(tctx, ar, rule, plugins)

	// Apply plugins from the route itself
	t.loadRoutePlugins(tctx, ar, rule, plugins)

	// Add authentication plugins
	t.addAuthenticationPlugins(rule, plugins)

	return plugins
}

func (t *Translator) loadPluginConfigPlugins(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP, plugins adc.Plugins) {
	if rule.PluginConfigName == "" {
		return
	}

	pcNamespace := ar.Namespace
	if rule.PluginConfigNamespace != "" {
		pcNamespace = rule.PluginConfigNamespace
	}

	pcKey := types.NamespacedName{Namespace: pcNamespace, Name: rule.PluginConfigName}
	pc, ok := tctx.ApisixPluginConfigs[pcKey]
	if !ok || pc == nil {
		return
	}

	for _, plugin := range pc.Spec.Plugins {
		if !plugin.Enable {
			continue
		}
		config := t.buildPluginConfig(plugin, pc.Namespace, tctx.Secrets)
		plugins[plugin.Name] = config
	}
}

func (t *Translator) loadRoutePlugins(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP, plugins adc.Plugins) {
	for _, plugin := range rule.Plugins {
		if !plugin.Enable {
			continue
		}
		config := t.buildPluginConfig(plugin, ar.Namespace, tctx.Secrets)
		plugins[plugin.Name] = config
	}
}

func (t *Translator) buildPluginConfig(plugin apiv2.ApisixRoutePlugin, namespace string, secrets map[types.NamespacedName]*v1.Secret) map[string]any {
	config := make(map[string]any)
	if len(plugin.Config.Raw) > 0 {
		if err := json.Unmarshal(plugin.Config.Raw, &config); err != nil {
			t.Log.Error(err, "failed to unmarshal plugin config")
		}
	}
	if plugin.SecretRef != "" {
		if secret, ok := secrets[types.NamespacedName{Namespace: namespace, Name: plugin.SecretRef}]; ok {
			for key, value := range secret.Data {
				pkgutils.InsertKeyInMap(key, string(value), config)
			}
		}
	}
	return config
}

func (t *Translator) addAuthenticationPlugins(rule apiv2.ApisixRouteHTTP, plugins adc.Plugins) {
	if !rule.Authentication.Enable {
		return
	}

	switch rule.Authentication.Type {
	case "keyAuth":
		plugins["key-auth"] = rule.Authentication.KeyAuth
	case "basicAuth":
		plugins["basic-auth"] = make(map[string]any)
	case "wolfRBAC":
		plugins["wolf-rbac"] = make(map[string]any)
	case "jwtAuth":
		plugins["jwt-auth"] = rule.Authentication.JwtAuth
	case "hmacAuth":
		plugins["hmac-auth"] = make(map[string]any)
	case "ldapAuth":
		plugins["ldap-auth"] = rule.Authentication.LDAPAuth
	default:
		plugins["basic-auth"] = make(map[string]any)
	}
}

func (t *Translator) buildRoute(ar *apiv2.ApisixRoute, service *adc.Service, rule apiv2.ApisixRouteHTTP, plugins adc.Plugins, timeout *adc.Timeout, vars adc.Vars) {
	route := adc.NewDefaultRoute()
	route.Name = adc.ComposeRouteName(ar.Namespace, ar.Name, rule.Name)
	route.ID = id.GenID(route.Name)
	route.Desc = "Created by apisix-ingress-controller, DO NOT modify it manually"
	route.Labels = label.GenLabel(ar)
	route.EnableWebsocket = ptr.To(true)
	route.FilterFunc = rule.Match.FilterFunc
	route.Hosts = rule.Match.Hosts
	route.Methods = rule.Match.Methods
	route.Plugins = plugins
	route.Priority = ptr.To(int64(rule.Priority))
	route.RemoteAddrs = rule.Match.RemoteAddrs
	route.Timeout = timeout
	route.Uris = rule.Match.Paths
	route.Vars = vars
	for key, value := range ar.GetObjectMeta().GetLabels() {
		route.Labels[key] = value
	}

	service.Routes = []*adc.Route{route}
}

func (t *Translator) buildUpstream(tctx *provider.TranslateContext, service *adc.Service, ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP) {
	var (
		upstreams         = make([]*adc.Upstream, 0)
		weightedUpstreams = make([]adc.TrafficSplitConfigRuleWeightedUpstream, 0)
		backendErr        error
	)

	for _, backend := range rule.Backends {
		upstream := adc.NewDefaultUpstream()
		// try to get the apisixupstream with the same name as the backend service to be upstream config.
		// err is ignored because it does not care about the externalNodes of the apisixupstream.
		auNN := types.NamespacedName{Namespace: ar.GetNamespace(), Name: backend.ServiceName}
		if au, ok := tctx.Upstreams[auNN]; ok {
			upstream, _ = t.translateApisixUpstream(tctx, au)
		}

		if backend.ResolveGranularity == "service" {
			upstream.Nodes, backendErr = t.translateApisixRouteBackendResolveGranularityService(tctx, utils.NamespacedName(ar), backend)
			if backendErr != nil {
				t.Log.Error(backendErr, "failed to translate ApisixRoute backend with ResolveGranularity Service")
				continue
			}
		} else {
			upstream.Nodes, backendErr = t.translateApisixRouteBackendResolveGranularityEndpoint(tctx, utils.NamespacedName(ar), backend)
			if backendErr != nil {
				t.Log.Error(backendErr, "failed to translate ApisixRoute backend with ResolveGranularity Endpoint")
				continue
			}
		}

		upstreams = append(upstreams, upstream)
	}

	for _, upstreamRef := range rule.Upstreams {
		upsNN := types.NamespacedName{
			Namespace: ar.GetNamespace(),
			Name:      upstreamRef.Name,
		}
		au, ok := tctx.Upstreams[upsNN]
		if !ok {
			log.Debugf("failed to retrieve ApisixUpstream from tctx, ApisixUpstream: %s", upsNN)
			continue
		}
		upstream, err := t.translateApisixUpstream(tctx, au)
		if err != nil {
			t.Log.Error(err, "failed to translate ApisixUpstream", "ApisixUpstream", utils.NamespacedName(au))
			continue
		}
		if upstreamRef.Weight != nil {
			upstream.Labels["meta_weight"] = strconv.FormatInt(int64(*upstreamRef.Weight), 10)
		}

		upstreams = append(upstreams, upstream)
	}

	// no valid upstream
	if backendErr != nil || len(upstreams) == 0 || len(upstreams[0].Nodes) == 0 {
		return
	}

	// the first valid upstream is used as service.upstream;
	// the others are configured in the traffic-split plugin
	service.Upstream = upstreams[0]
	upstreams = upstreams[1:]

	// set weight in traffic-split for the default upstream
	if len(upstreams) > 0 {
		weight, err := strconv.Atoi(service.Upstream.Labels["meta_weight"])
		if err != nil {
			weight = apiv2.DefaultWeight
		}
		weightedUpstreams = append(weightedUpstreams, adc.TrafficSplitConfigRuleWeightedUpstream{
			Weight: weight,
		})
	}

	// set others upstreams in traffic-split
	for _, item := range upstreams {
		weight, err := strconv.Atoi(item.Labels["meta_weight"])
		if err != nil {
			weight = apiv2.DefaultWeight
		}
		weightedUpstreams = append(weightedUpstreams, adc.TrafficSplitConfigRuleWeightedUpstream{
			Upstream: item,
			Weight:   weight,
		})
	}

	if len(weightedUpstreams) > 0 {
		service.Plugins["traffic-split"] = &adc.TrafficSplitConfig{
			Rules: []adc.TrafficSplitConfigRule{
				{
					WeightedUpstreams: weightedUpstreams,
				},
			},
		}
	}
}

func (t *Translator) buildService(ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP, ruleIndex int) *adc.Service {
	service := adc.NewDefaultService()
	service.Name = adc.ComposeServiceNameWithRule(ar.Namespace, ar.Name, fmt.Sprintf("%d", ruleIndex))
	service.ID = id.GenID(service.Name)
	service.Labels = label.GenLabel(ar)
	service.Hosts = rule.Match.Hosts
	service.Upstream = adc.NewDefaultUpstream()
	return service
}

func (t *Translator) translateApisixRouteBackendResolveGranularityService(tctx *provider.TranslateContext, arNN types.NamespacedName, backend apiv2.ApisixRouteHTTPBackend) (adc.UpstreamNodes, error) {
	serviceNN := types.NamespacedName{
		Namespace: arNN.Namespace,
		Name:      backend.ServiceName,
	}
	svc, ok := tctx.Services[serviceNN]
	if !ok {
		return nil, errors.Errorf("service not found, ApisixRoute: %s, Service: %s", arNN, serviceNN)
	}
	if svc.Spec.ClusterIP == "" {
		return nil, errors.Errorf("conflict headless service and backend resolve granularity, ApisixRoute: %s, Service: %s", arNN, serviceNN)
	}
	return adc.UpstreamNodes{
		{
			Host:   svc.Spec.ClusterIP,
			Port:   backend.ServicePort.IntValue(),
			Weight: *cmp.Or(backend.Weight, ptr.To(apiv2.DefaultWeight)),
		},
	}, nil
}

func (t *Translator) translateApisixRouteBackendResolveGranularityEndpoint(tctx *provider.TranslateContext, arNN types.NamespacedName, backend apiv2.ApisixRouteHTTPBackend) (adc.UpstreamNodes, error) {
	weight := int32(*cmp.Or(backend.Weight, ptr.To(apiv2.DefaultWeight)))
	backendRef := gatewayv1.BackendRef{
		BackendObjectReference: gatewayv1.BackendObjectReference{
			Group:     (*gatewayv1.Group)(&apiv2.GroupVersion.Group),
			Kind:      (*gatewayv1.Kind)(ptr.To("Service")),
			Name:      gatewayv1.ObjectName(backend.ServiceName),
			Namespace: (*gatewayv1.Namespace)(&arNN.Namespace),
			Port:      (*gatewayv1.PortNumber)(&backend.ServicePort.IntVal),
		},
		Weight: &weight,
	}
	return t.translateBackendRef(tctx, backendRef, DefaultEndpointFilter)
}
