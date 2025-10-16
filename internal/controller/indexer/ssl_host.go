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

package indexer

import (
	"sort"

	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	sslutil "github.com/apache/apisix-ingress-controller/internal/ssl"
)

var (
	tlsHostIndexLogger = ctrl.Log.WithName("tls-host-indexer")
	// Empty host is used to match the resource which does not specify any explicit host.
	emptyHost = ""
)

// GatewayTLSHostIndexFunc indexes Gateways by their TLS SNI hosts.
func GatewayTLSHostIndexFunc(rawObj client.Object) []string {
	gateway, ok := rawObj.(*gatewayv1.Gateway)
	if !ok {
		return nil
	}
	if len(gateway.Spec.Listeners) == 0 {
		return nil
	}

	hosts := make(map[string]struct{})

	for _, listener := range gateway.Spec.Listeners {
		if listener.TLS == nil || len(listener.TLS.CertificateRefs) == 0 {
			continue
		}

		hasExplicitHost := false
		if listener.Hostname != nil {
			candidates := sslutil.NormalizeHosts([]string{string(*listener.Hostname)})
			for _, host := range candidates {
				if host == "" {
					continue
				}
				hasExplicitHost = true
				hosts[host] = struct{}{}
			}
		}

		if !hasExplicitHost {
			hosts[emptyHost] = struct{}{}
		}
	}

	tlsHostIndexLogger.Info("GatewayTLSHostIndexFunc", "hosts", hostSetToSlice(hosts), "len", len(hostSetToSlice(hosts)))

	return hostSetToSlice(hosts)
}

// IngressTLSHostIndexFunc indexes Ingresses by their TLS SNI hosts.
func IngressTLSHostIndexFunc(rawObj client.Object) []string {
	ingress, ok := rawObj.(*networkingv1.Ingress)
	if !ok {
		return nil
	}
	if len(ingress.Spec.TLS) == 0 {
		return nil
	}

	hosts := make(map[string]struct{})
	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}

		hasExplicitHost := false
		candidates := sslutil.NormalizeHosts(tls.Hosts)
		for _, host := range candidates {
			if host == "" {
				continue
			}
			hasExplicitHost = true
			hosts[host] = struct{}{}
		}

		if !hasExplicitHost {
			hosts[emptyHost] = struct{}{}
		}
	}

	return hostSetToSlice(hosts)
}

// ApisixTlsHostIndexFunc indexes ApisixTls resources by their declared TLS hosts.
func ApisixTlsHostIndexFunc(rawObj client.Object) []string {
	tls, ok := rawObj.(*apiv2.ApisixTls)
	if !ok {
		return nil
	}
	if len(tls.Spec.Hosts) == 0 {
		return nil
	}

	hostSet := make(map[string]struct{}, len(tls.Spec.Hosts))
	for _, host := range tls.Spec.Hosts {
		for _, normalized := range sslutil.NormalizeHosts([]string{string(host)}) {
			if normalized == "" {
				continue
			}
			hostSet[normalized] = struct{}{}
		}
	}
	return hostSetToSlice(hostSet)
}

func hostSetToSlice(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	result := make([]string, 0, len(set))
	for host := range set {
		result = append(result, host)
	}
	sort.Strings(result)
	return result
}
