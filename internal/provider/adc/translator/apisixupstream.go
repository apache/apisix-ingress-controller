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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func (t *Translator) translateApisixUpstream(tctx *provider.TranslateContext, au *apiv2.ApisixUpstream) (ups *adc.Upstream, err error) {
	ups = adc.NewDefaultUpstream()
	for _, f := range []func(*apiv2.ApisixUpstream, *adc.Upstream) error{
		patchApisixUpstreamBasics,
		translateApisixUpstreamScheme,
		translateApisixUpstreamLoadBalancer,
		translateApisixUpstreamRetriesAndTimeout,
		translateApisixUpstreamPassHost,
	} {
		if err = f(au, ups); err != nil {
			return
		}
	}
	for _, f := range []func(*provider.TranslateContext, *apiv2.ApisixUpstream, *adc.Upstream) error{
		translateApisixUpstreamClientTLS,
		translateApisixUpstreamExternalNodes,
	} {
		if err = f(tctx, au, ups); err != nil {
			return
		}
	}

	return
}

func patchApisixUpstreamBasics(au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	ups.Name = composeExternalUpstreamName(au)
	for k, v := range au.Labels {
		ups.Labels[k] = v
	}
	return nil
}

func translateApisixUpstreamScheme(au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	ups.Scheme = cmp.Or(au.Spec.Scheme, apiv2.SchemeHTTP)
	return nil
}

func translateApisixUpstreamLoadBalancer(au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	lb := au.Spec.LoadBalancer
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

func translateApisixUpstreamRetriesAndTimeout(au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	retries := au.Spec.Retries
	timeout := au.Spec.Timeout

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

func translateApisixUpstreamClientTLS(tctx *provider.TranslateContext, au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	if au.Spec.TLSSecret == nil {
		return nil
	}

	var (
		secretNN = types.NamespacedName{
			Namespace: au.Spec.TLSSecret.Namespace,
			Name:      au.Spec.TLSSecret.Name,
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

func translateApisixUpstreamPassHost(au *apiv2.ApisixUpstream, ups *adc.Upstream) error {
	ups.PassHost = au.Spec.PassHost
	ups.UpstreamHost = au.Spec.UpstreamHost

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
