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
	"fmt"
	"maps"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func (t *Translator) translateApisixUpstream(tctx *provider.TranslateContext, au *apiv2.ApisixUpstream) (*adc.Upstream, error) {
	return t.translateApisixUpstreamForPort(tctx, au, nil)
}

func (t *Translator) translateApisixUpstreamForPort(tctx *provider.TranslateContext, au *apiv2.ApisixUpstream, port *int32) (*adc.Upstream, error) {
	t.Log.V(1).Info("translating ApisixUpstream", "apisixupstream", au, "port", port)

	ups := adc.NewDefaultUpstream()
	ups.Name = composeExternalUpstreamName(au)
	maps.Copy(ups.Labels, au.Labels)

	// translateApisixUpstreamConfig translates the core upstream configuration fields
	// from au.Spec.ApisixUpstreamConfig into the ADC upstream.
	//
	// Note: ExternalNodes is not part of ApisixUpstreamConfig but a separate field
	// on ApisixUpstreamSpec, so it is handled separately in translateApisixUpstreamExternalNodes.
	if err := translateApisixUpstreamConfig(tctx, &au.Spec.ApisixUpstreamConfig, ups); err != nil {
		return nil, err
	}
	if err := translateApisixUpstreamExternalNodes(tctx, au, ups); err != nil {
		return nil, err
	}

	// If PortLevelSettings is configured and a specific port is provided,
	// apply the ApisixUpstreamConfig for the matching port to the upstream.
	if len(au.Spec.PortLevelSettings) > 0 && port != nil {
		for _, pls := range au.Spec.PortLevelSettings {
			if pls.Port != *port {
				continue
			}
			if err := translateApisixUpstreamConfig(tctx, &pls.ApisixUpstreamConfig, ups); err != nil {
				return nil, err
			}
		}
	}

	t.Log.V(1).Info("translated ApisixUpstream", "upstream", ups)

	return ups, nil
}

func translateApisixUpstreamConfig(tctx *provider.TranslateContext, config *apiv2.ApisixUpstreamConfig, ups *adc.Upstream) (err error) {
	for _, f := range []func(*apiv2.ApisixUpstreamConfig, *adc.Upstream) error{
		translateApisixUpstreamScheme,
		translateApisixUpstreamLoadBalancer,
		translateApisixUpstreamRetriesAndTimeout,
		translateApisixUpstreamPassHost,
		translateUpstreamHealthCheck,
		translateUpstreamDiscovery,
	} {
		if err = f(config, ups); err != nil {
			return
		}
	}
	for _, f := range []func(*provider.TranslateContext, *apiv2.ApisixUpstreamConfig, *adc.Upstream) error{
		translateApisixUpstreamClientTLS,
	} {
		if err = f(tctx, config, ups); err != nil {
			return
		}
	}

	return
}

func translateApisixUpstreamScheme(config *apiv2.ApisixUpstreamConfig, ups *adc.Upstream) error {
	ups.Scheme = cmp.Or(config.Scheme, apiv2.SchemeHTTP)
	return nil
}

func translateApisixUpstreamLoadBalancer(config *apiv2.ApisixUpstreamConfig, ups *adc.Upstream) error {
	lb := config.LoadBalancer
	if lb == nil || lb.Type == "" {
		ups.Type = apiv2.LbRoundRobin
		return nil
	}
	switch lb.Type {
	case apiv2.LbRoundRobin, apiv2.LbLeastConn, apiv2.LbEwma:
		ups.Type = adc.UpstreamType(lb.Type)
	case apiv2.LbConsistentHash:
		ups.Type = adc.UpstreamType(lb.Type)
		ups.Key = lb.Key
		switch lb.HashOn {
		case apiv2.HashOnVars:
			fallthrough
		case apiv2.HashOnHeader:
			fallthrough
		case apiv2.HashOnCookie:
			fallthrough
		case apiv2.HashOnConsumer:
			fallthrough
		case apiv2.HashOnVarsCombination:
			ups.HashOn = lb.HashOn
		default:
			return errors.New("invalid hashOn")
		}
	default:
		return errors.New("invalid loadBalancer type")
	}
	return nil
}

func translateApisixUpstreamRetriesAndTimeout(config *apiv2.ApisixUpstreamConfig, ups *adc.Upstream) error {
	retries := config.Retries
	timeout := config.Timeout

	if retries != nil && *retries < 0 {
		return errors.New("invalid value retries")
	}
	ups.Retries = retries

	if timeout == nil {
		return nil
	}
	if timeout.Connect.Duration < 0 {
		return errors.New("invalid value timeout.connect")
	}
	if timeout.Read.Duration < 0 {
		return errors.New("invalid value timeout.read")
	}
	if timeout.Send.Duration < 0 {
		return errors.New("invalid value timeout.send")
	}

	// Since the schema of timeout doesn't allow only configuring
	// one or two items. Here we assign the default value first.
	connTimeout := cmp.Or(timeout.Connect.Duration, apiv2.DefaultUpstreamTimeout)
	readTimeout := cmp.Or(timeout.Read.Duration, apiv2.DefaultUpstreamTimeout)
	sendTimeout := cmp.Or(timeout.Send.Duration, apiv2.DefaultUpstreamTimeout)

	ups.Timeout = &adc.Timeout{
		Connect: int(connTimeout.Seconds()),
		Read:    int(readTimeout.Seconds()),
		Send:    int(sendTimeout.Seconds()),
	}

	return nil
}

func translateApisixUpstreamClientTLS(tctx *provider.TranslateContext, config *apiv2.ApisixUpstreamConfig, ups *adc.Upstream) error {
	if config.TLSSecret == nil {
		return nil
	}

	var (
		secretNN = types.NamespacedName{
			Namespace: config.TLSSecret.Namespace,
			Name:      config.TLSSecret.Name,
		}
	)
	secret, ok := tctx.Secrets[secretNN]
	if !ok {
		return errors.Errorf("sercret %s not found", secretNN)
	}

	cert, key, err := extractKeyPair(secret, true)
	if err != nil {
		return err
	}

	ups.TLS = &adc.ClientTLS{
		Cert: string(cert),
		Key:  string(key),
	}

	return nil
}

func translateApisixUpstreamPassHost(config *apiv2.ApisixUpstreamConfig, ups *adc.Upstream) error {
	ups.PassHost = config.PassHost
	ups.UpstreamHost = config.UpstreamHost

	return nil
}

func composeExternalUpstreamName(au *apiv2.ApisixUpstream) string {
	return au.GetGenerateName() + "_" + au.GetName()
}

func translateApisixUpstreamExternalNodes(tctx *provider.TranslateContext, au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	for _, node := range au.Spec.ExternalNodes {
		switch node.Type {
		case apiv2.ExternalTypeDomain:
			if err := translateApisixUpstreamExternalNodesDomain(au, ups, node); err != nil {
				return err
			}
		default: // apiv2.ExternalTypeService or default
			if err := translateApisixUpstreamExternalNodesService(tctx, au, ups, node); err != nil {
				return err
			}
		}
	}

	return nil
}
func translateApisixUpstreamExternalNodesDomain(au *apiv2.ApisixUpstream, ups *adc.Upstream, node apiv2.ApisixUpstreamExternalNode) error {
	weight := apiv2.DefaultWeight
	if node.Weight != nil {
		weight = *node.Weight
	}

	if !utils.MatchHostDef(node.Name) {
		return fmt.Errorf("ApisixUpstream %s/%s ExternalNodes[]'s name %s as Domain must match lowercase RFC 1123 subdomain.  "+
			"a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character",
			au.Namespace, au.Name, node.Name)
	}

	n := adc.UpstreamNode{
		Host:   node.Name,
		Weight: weight,
	}

	if node.Port != nil {
		n.Port = *node.Port
	} else {
		n.Port = apiv2.SchemeToPort(au.Spec.Scheme)
	}

	ups.Nodes = append(ups.Nodes, n)

	return nil
}

func translateApisixUpstreamExternalNodesService(tctx *provider.TranslateContext, au *apiv2.ApisixUpstream, ups *adc.Upstream, node apiv2.ApisixUpstreamExternalNode) error {
	serviceNN := types.NamespacedName{Namespace: au.GetNamespace(), Name: node.Name}
	svc, ok := tctx.Services[serviceNN]
	if !ok {
		return errors.Errorf("service not found, service: %s", serviceNN)
	}

	if svc.Spec.Type != corev1.ServiceTypeExternalName {
		return errors.Errorf("ApisixUpstream %s ExternalNodes[] must refers to a ExternalName service: %s", utils.NamespacedName(au), node.Name)
	}

	weight := apiv2.DefaultWeight
	if node.Weight != nil {
		weight = *node.Weight
	}
	n := adc.UpstreamNode{
		Host:   svc.Spec.ExternalName,
		Weight: weight,
	}

	if node.Port != nil {
		n.Port = *node.Port
	} else {
		n.Port = apiv2.SchemeToPort(au.Spec.Scheme)
	}

	ups.Nodes = append(ups.Nodes, n)

	return nil
}

func translateUpstreamHealthCheck(config *apiv2.ApisixUpstreamConfig, ups *adc.Upstream) error {
	healcheck := config.HealthCheck
	if healcheck == nil || (healcheck.Passive == nil && healcheck.Active == nil) {
		return nil
	}
	var hc adc.UpstreamHealthCheck
	if healcheck.Passive != nil {
		hc.Passive = translateUpstreamPassiveHealthCheck(healcheck.Passive)
	}

	if healcheck.Active != nil {
		active, err := translateUpstreamActiveHealthCheck(healcheck.Active)
		if err != nil {
			return err
		}
		hc.Active = active
	}

	ups.Checks = &hc
	return nil
}

func translateUpstreamActiveHealthCheck(config *apiv2.ActiveHealthCheck) (*adc.UpstreamActiveHealthCheck, error) {
	var active adc.UpstreamActiveHealthCheck
	if config.Type == "" {
		config.Type = apiv2.HealthCheckHTTP
	}

	active.Timeout = int(config.Timeout.Seconds())
	active.Port = config.Port
	active.Concurrency = config.Concurrency
	active.Host = config.Host
	active.HTTPPath = config.HTTPPath
	active.HTTPRequestHeaders = config.RequestHeaders

	if config.StrictTLS == nil || *config.StrictTLS {
		active.HTTPSVerifyCert = true
	}

	if config.Healthy != nil {
		active.Healthy.Successes = config.Healthy.Successes
		active.Healthy.HTTPStatuses = config.Healthy.HTTPCodes

		if config.Healthy.Interval.Duration < apiv2.ActiveHealthCheckMinInterval {
			return nil, fmt.Errorf(`"healthCheck.active.healthy.interval" has invalid value`)
		}
		active.Healthy.Interval = int(config.Healthy.Interval.Seconds())
	}

	if config.Unhealthy != nil {
		active.Unhealthy.HTTPFailures = config.Unhealthy.HTTPFailures
		active.Unhealthy.TCPFailures = config.Unhealthy.TCPFailures
		active.Unhealthy.Timeouts = config.Unhealthy.Timeouts
		active.Unhealthy.HTTPStatuses = config.Unhealthy.HTTPCodes

		if config.Unhealthy.Interval.Duration < apiv2.ActiveHealthCheckMinInterval {
			return nil, fmt.Errorf(`"healthCheck.active.unhealthy.interval" has invalid value`)
		}
		active.Unhealthy.Interval = int(config.Unhealthy.Interval.Seconds())
	}

	return &active, nil
}

func translateUpstreamPassiveHealthCheck(config *apiv2.PassiveHealthCheck) *adc.UpstreamPassiveHealthCheck {
	var passive adc.UpstreamPassiveHealthCheck
	if config.Type == "" {
		config.Type = apiv2.HealthCheckHTTP
	}

	if config.Healthy != nil {
		passive.Healthy.Successes = config.Healthy.Successes
		passive.Healthy.HTTPStatuses = config.Healthy.HTTPCodes
	}

	if config.Unhealthy != nil {
		passive.Unhealthy.HTTPFailures = config.Unhealthy.HTTPFailures
		passive.Unhealthy.TCPFailures = config.Unhealthy.TCPFailures
		passive.Unhealthy.Timeouts = config.Unhealthy.Timeouts
		passive.Unhealthy.HTTPStatuses = config.Unhealthy.HTTPCodes
	}
	return &passive
}

func translateUpstreamDiscovery(config *apiv2.ApisixUpstreamConfig, ups *adc.Upstream) error {
	discovery := config.Discovery
	if discovery == nil {
		return nil
	}
	ups.ServiceName = discovery.ServiceName
	ups.DiscoveryType = discovery.Type
	ups.DiscoveryArgs = discovery.Args
	return nil
}
