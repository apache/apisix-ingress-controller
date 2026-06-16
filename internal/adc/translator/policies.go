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
	"encoding/json"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

func convertBackendRef(namespace, name, kind string) gatewayv1.BackendRef {
	backendRef := gatewayv1.BackendRef{}
	backendRef.Name = gatewayv1.ObjectName(name)
	backendRef.Namespace = ptr.To(gatewayv1.Namespace(namespace))
	backendRef.Kind = ptr.To(gatewayv1.Kind(kind))
	return backendRef
}

func (t *Translator) AttachBackendTrafficPolicyToUpstream(ref gatewayv1.BackendRef, policies map[types.NamespacedName]*v1alpha1.BackendTrafficPolicy, upstream *adctypes.Upstream) {
	if len(policies) == 0 {
		return
	}
	var policy *v1alpha1.BackendTrafficPolicy
	for _, po := range policies {
		if ref.Namespace != nil && string(*ref.Namespace) != po.Namespace {
			continue
		}
		for _, targetRef := range po.Spec.TargetRefs {
			if ref.Name == targetRef.Name {
				policy = po
				break
			}
		}
	}
	if policy == nil {
		return
	}
	t.attachBackendTrafficPolicyToUpstream(policy, upstream)
}

func (t *Translator) attachBackendTrafficPolicyToUpstream(policy *v1alpha1.BackendTrafficPolicy, upstream *adctypes.Upstream) {
	if policy == nil {
		return
	}
	upstream.PassHost = policy.Spec.PassHost
	upstream.UpstreamHost = string(policy.Spec.Host)
	upstream.Scheme = policy.Spec.Scheme
	if policy.Spec.Retries != nil {
		upstream.Retries = new(int64)
		*upstream.Retries = int64(*policy.Spec.Retries)
	}
	if policy.Spec.Timeout != nil {
		upstream.Timeout = &adctypes.Timeout{
			Connect: int(policy.Spec.Timeout.Connect.Seconds()),
			Read:    int(policy.Spec.Timeout.Read.Seconds()),
			Send:    int(policy.Spec.Timeout.Send.Seconds()),
		}
	}
	if policy.Spec.LoadBalancer != nil {
		upstream.Type = adctypes.UpstreamType(policy.Spec.LoadBalancer.Type)
		upstream.HashOn = policy.Spec.LoadBalancer.HashOn
		upstream.Key = policy.Spec.LoadBalancer.Key
	}
	if policy.Spec.HealthCheck != nil {
		upstream.Checks = translateBTPHealthCheck(policy.Spec.HealthCheck)
	}
}

func translateBTPHealthCheck(hc *v1alpha1.HealthCheck) *adctypes.UpstreamHealthCheck {
	if hc == nil || (hc.Active == nil && hc.Passive == nil) {
		return nil
	}
	result := &adctypes.UpstreamHealthCheck{}
	if hc.Active != nil {
		result.Active = translateBTPActiveHealthCheck(hc.Active)
	}
	if hc.Passive != nil {
		result.Passive = translateBTPPassiveHealthCheck(hc.Passive)
	}
	return result
}

func translateBTPActiveHealthCheck(config *v1alpha1.ActiveHealthCheck) *adctypes.UpstreamActiveHealthCheck {
	t := config.Type
	if t == "" {
		t = "http"
	}
	active := &adctypes.UpstreamActiveHealthCheck{
		Type:                   t,
		Timeout:                int(config.Timeout.Seconds()),
		Concurrency:            config.Concurrency,
		Host:                   config.Host,
		Port:                   config.Port,
		HTTPPath:               config.HTTPPath,
		HTTPSVerifyCertificate: config.StrictTLS == nil || *config.StrictTLS,
		HTTPRequestHeaders:     config.RequestHeaders,
	}
	if config.Healthy != nil {
		interval := config.Healthy.Interval.Duration
		if interval < apiv2.ActiveHealthCheckMinInterval {
			interval = apiv2.ActiveHealthCheckMinInterval
		}
		active.Healthy = adctypes.UpstreamActiveHealthCheckHealthy{
			Interval: int(interval.Seconds()),
			UpstreamPassiveHealthCheckHealthy: adctypes.UpstreamPassiveHealthCheckHealthy{
				HTTPStatuses: config.Healthy.HTTPCodes,
				Successes:    config.Healthy.Successes,
			},
		}
	}
	if config.Unhealthy != nil {
		interval := config.Unhealthy.Interval.Duration
		if interval < apiv2.ActiveHealthCheckMinInterval {
			interval = apiv2.ActiveHealthCheckMinInterval
		}
		active.Unhealthy = adctypes.UpstreamActiveHealthCheckUnhealthy{
			Interval: int(interval.Seconds()),
			UpstreamPassiveHealthCheckUnhealthy: adctypes.UpstreamPassiveHealthCheckUnhealthy{
				HTTPStatuses: config.Unhealthy.HTTPCodes,
				HTTPFailures: config.Unhealthy.HTTPFailures,
				TCPFailures:  config.Unhealthy.TCPFailures,
				Timeouts:     config.Unhealthy.Timeouts,
			},
		}
	}
	return active
}

func translateBTPPassiveHealthCheck(config *v1alpha1.PassiveHealthCheck) *adctypes.UpstreamPassiveHealthCheck {
	t := config.Type
	if t == "" {
		t = "http"
	}
	passive := &adctypes.UpstreamPassiveHealthCheck{
		Type: t,
	}
	if config.Healthy != nil {
		passive.Healthy = adctypes.UpstreamPassiveHealthCheckHealthy{
			HTTPStatuses: config.Healthy.HTTPCodes,
			Successes:    config.Healthy.Successes,
		}
	}
	if config.Unhealthy != nil {
		passive.Unhealthy = adctypes.UpstreamPassiveHealthCheckUnhealthy{
			HTTPStatuses: config.Unhealthy.HTTPCodes,
			HTTPFailures: config.Unhealthy.HTTPFailures,
			TCPFailures:  config.Unhealthy.TCPFailures,
			Timeouts:     config.Unhealthy.Timeouts,
		}
	}
	return passive
}

// AttachL4RoutePolicyPlugins merges plugins from the matching L4RoutePolicy (if any) into the
// provided plugins map. It looks up policies targeting the route identified by routeNamespace,
// routeName, and routeKind.
func (t *Translator) AttachL4RoutePolicyPlugins(
	policies map[types.NamespacedName]*v1alpha1.L4RoutePolicy,
	routeNamespace, routeName, routeKind string,
	plugins adctypes.Plugins,
) {
	if len(policies) == 0 {
		return
	}
	for _, policy := range policies {
		if policy.Namespace != routeNamespace {
			continue
		}
		for _, ref := range policy.Spec.TargetRefs {
			if string(ref.Group) != gatewayv1alpha2.GroupName {
				continue
			}
			if string(ref.Kind) != routeKind {
				continue
			}
			if string(ref.Name) != routeName {
				continue
			}
			// sectionName targeting is not supported for L4 routes; skip such refs
			// so plugins are not attached for an attachment that cannot be honored.
			if ref.SectionName != nil && *ref.SectionName != "" {
				continue
			}
			t.mergeL4PolicyPlugins(policy, plugins)
			return
		}
	}
}

func (t *Translator) mergeL4PolicyPlugins(policy *v1alpha1.L4RoutePolicy, plugins adctypes.Plugins) {
	for _, plugin := range policy.Spec.Plugins {
		cfg := make(map[string]any)
		if len(plugin.Config.Raw) > 0 {
			if err := json.Unmarshal(plugin.Config.Raw, &cfg); err != nil {
				t.Log.Error(err, "failed to unmarshal L4RoutePolicy plugin config", "plugin", plugin.Name, "policy", policy.Name)
				continue
			}
		}
		// A literal `config: null` unmarshals to a nil map, which serializes back to
		// null and is rejected by most APISIX plugins; normalize it to an empty object.
		if cfg == nil {
			cfg = map[string]any{}
		}
		plugins[plugin.Name] = cfg
	}
}
