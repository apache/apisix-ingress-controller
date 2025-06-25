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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/id"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func (t *Translator) translateIngressTLS(ingressTLS *networkingv1.IngressTLS, secret *corev1.Secret, labels map[string]string) (*adctypes.SSL, error) {
	// extract the key pair from the secret
	cert, key, err := extractKeyPair(secret, true)
	if err != nil {
		return nil, err
	}

	hosts := ingressTLS.Hosts
	if len(hosts) == 0 {
		certHosts, err := extractHost(cert)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, certHosts...)
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts found in ingress TLS")
	}

	ssl := &adctypes.SSL{
		Metadata: adctypes.Metadata{
			Labels: labels,
		},
		Certificates: []adctypes.Certificate{
			{
				Certificate: string(cert),
				Key:         string(key),
			},
		},
		Snis: hosts,
	}
	ssl.ID = id.GenID(string(cert))

	return ssl, nil
}

func (t *Translator) TranslateIngress(tctx *provider.TranslateContext, obj *networkingv1.Ingress) (*TranslateResult, error) {
	result := &TranslateResult{}

	labels := label.GenLabel(obj)

	// handle TLS configuration, convert to SSL objects
	for _, tls := range obj.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}
		secret := tctx.Secrets[types.NamespacedName{
			Namespace: obj.Namespace,
			Name:      tls.SecretName,
		}]
		if secret == nil {
			continue
		}
		ssl, err := t.translateIngressTLS(&tls, secret, labels)
		if err != nil {
			return nil, err
		}

		result.SSL = append(result.SSL, ssl)
	}

	// process Ingress rules, convert to Service and Route objects
	for i, rule := range obj.Spec.Rules {
		// extract hostnames
		var hosts []string
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
		// if there is no HTTP path, skip
		if rule.HTTP == nil {
			continue
		}

		// create a service for each path
		for j, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				continue
			}

			service := adctypes.NewDefaultService()
			service.Labels = labels
			service.Name = adctypes.ComposeServiceNameWithRule(obj.Namespace, obj.Name, fmt.Sprintf("%d-%d", i, j))
			service.ID = id.GenID(service.Name)
			service.Hosts = hosts

			// create an upstream
			upstream := adctypes.NewDefaultUpstream()

			// get the EndpointSlice of the backend service
			backendService := path.Backend.Service
			if backendService != nil {
				backendRef := convertBackendRef(obj.Namespace, backendService.Name, "Service")
				t.AttachBackendTrafficPolicyToUpstream(backendRef, tctx.BackendTrafficPolicies, upstream)
			}

			// get the service port configuration
			var servicePort int32 = 0
			var servicePortName string
			if backendService.Port.Number != 0 {
				servicePort = backendService.Port.Number
			} else if backendService.Port.Name != "" {
				servicePortName = backendService.Port.Name
			}

			getService := tctx.Services[types.NamespacedName{
				Namespace: obj.Namespace,
				Name:      backendService.Name,
			}]
			if getService == nil {
				continue
			}
			if getService.Spec.Type == corev1.ServiceTypeExternalName {
				defaultServicePort := 80
				if servicePort > 0 {
					defaultServicePort = int(servicePort)
				}
				upstream.Nodes = adctypes.UpstreamNodes{
					{
						Host:   getService.Spec.ExternalName,
						Port:   defaultServicePort,
						Weight: 1,
					},
				}
			} else {
				var getServicePort *corev1.ServicePort
				for _, port := range getService.Spec.Ports {
					port := port
					if servicePort > 0 && port.Port == servicePort {
						getServicePort = &port
						break
					}
					if servicePortName != "" && port.Name == servicePortName {
						getServicePort = &port
						break
					}
				}
				endpointSlices := tctx.EndpointSlices[types.NamespacedName{
					Namespace: obj.Namespace,
					Name:      backendService.Name,
				}]
				// convert the EndpointSlice to upstream nodes
				if len(endpointSlices) > 0 {
					upstream.Nodes = t.translateEndpointSliceForIngress(1, endpointSlices, getServicePort)
				}
			}

			service.Upstream = upstream

			// create a route
			route := adctypes.NewDefaultRoute()
			route.Name = adctypes.ComposeRouteName(obj.Namespace, obj.Name, fmt.Sprintf("%d-%d", i, j))
			route.ID = id.GenID(route.Name)
			route.Labels = labels

			uris := []string{path.Path}
			if path.PathType != nil {
				switch *path.PathType {
				case networkingv1.PathTypePrefix:
					// As per the specification of Ingress path matching rule:
					// if the last element of the path is a substring of the
					// last element in request path, it is not a match, e.g. /foo/bar
					// matches /foo/bar/baz, but does not match /foo/barbaz.
					// While in APISIX, /foo/bar matches both /foo/bar/baz and
					// /foo/barbaz.
					// In order to be conformant with Ingress specification, here
					// we create two paths here, the first is the path itself
					// (exact match), the other is path + "/*" (prefix match).
					prefix := path.Path
					if strings.HasSuffix(prefix, "/") {
						prefix += "*"
					} else {
						prefix += "/*"
					}
					uris = append(uris, prefix)
				case networkingv1.PathTypeImplementationSpecific:
					uris = []string{"/*"}
				}
			}
			route.Uris = uris
			service.Routes = []*adctypes.Route{route}
			t.fillHTTPRoutePoliciesForIngress(tctx, service.Routes)
			result.Services = append(result.Services, service)
		}
	}

	return result, nil
}

// translateEndpointSliceForIngress create upstream nodes from EndpointSlice
func (t *Translator) translateEndpointSliceForIngress(weight int, endpointSlices []discoveryv1.EndpointSlice, servicePort *corev1.ServicePort) adctypes.UpstreamNodes {
	var nodes adctypes.UpstreamNodes
	if len(endpointSlices) == 0 {
		return nodes
	}

	for _, endpointSlice := range endpointSlices {
		for _, port := range endpointSlice.Ports {
			// if the port number is specified, only use the matching port
			if servicePort != nil && port.Name != nil && *port.Name != servicePort.Name {
				continue
			}
			for _, endpoint := range endpointSlice.Endpoints {
				for _, addr := range endpoint.Addresses {
					node := adctypes.UpstreamNode{
						Host:   addr,
						Port:   int(*port.Port),
						Weight: weight,
					}
					nodes = append(nodes, node)
				}
			}
		}
	}

	return nodes
}
