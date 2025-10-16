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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/id"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	sslutils "github.com/apache/apisix-ingress-controller/internal/ssl"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func (t *Translator) translateIngressTLS(namespace, name string, tlsIndex int, ingressTLS *networkingv1.IngressTLS, secret *corev1.Secret, labels map[string]string) (*adctypes.SSL, error) {
	// extract the key pair from the secret
	cert, key, err := sslutils.ExtractKeyPair(secret, true)
	if err != nil {
		return nil, err
	}

	hosts := ingressTLS.Hosts
	if len(hosts) == 0 {
		certHosts, err := sslutils.ExtractHostsFromCertificate(cert)
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
	ssl.ID = id.GenID(fmt.Sprintf("%s_%d", adctypes.ComposeSSLName(internaltypes.KindIngress, namespace, name), tlsIndex))

	return ssl, nil
}

func (t *Translator) TranslateIngress(
	tctx *provider.TranslateContext,
	obj *networkingv1.Ingress,
) (*TranslateResult, error) {
	result := &TranslateResult{}

	labels := label.GenLabel(obj)

	// handle TLS configuration, convert to SSL objects
	if err := t.translateIngressTLSSection(tctx, obj, result, labels); err != nil {
		return nil, err
	}

	// process Ingress rules, convert to Service and Route objects
	for i, rule := range obj.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		hosts := []string{}
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}

		for j, path := range rule.HTTP.Paths {
			if svc := t.buildServiceFromIngressPath(tctx, obj, &path, i, j, hosts, labels); svc != nil {
				result.Services = append(result.Services, svc)
			}
		}
	}

	return result, nil
}

func (t *Translator) translateIngressTLSSection(
	tctx *provider.TranslateContext,
	obj *networkingv1.Ingress,
	result *TranslateResult,
	labels map[string]string,
) error {
	for tlsIndex, tls := range obj.Spec.TLS {
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
		ssl, err := t.translateIngressTLS(obj.Namespace, obj.Name, tlsIndex, &tls, secret, labels)
		if err != nil {
			return err
		}
		result.SSL = append(result.SSL, ssl)
	}
	return nil
}

func (t *Translator) buildServiceFromIngressPath(
	tctx *provider.TranslateContext,
	obj *networkingv1.Ingress,
	path *networkingv1.HTTPIngressPath,
	ruleIndex, pathIndex int,
	hosts []string,
	labels map[string]string,
) *adctypes.Service {
	if path.Backend.Service == nil {
		return nil
	}

	service := adctypes.NewDefaultService()
	service.Labels = labels
	service.Name = adctypes.ComposeServiceNameWithRule(obj.Namespace, obj.Name, fmt.Sprintf("%d-%d", ruleIndex, pathIndex))
	service.ID = id.GenID(service.Name)
	service.Hosts = hosts

	upstream := adctypes.NewDefaultUpstream()
	protocol := t.resolveIngressUpstream(tctx, obj, path.Backend.Service, upstream)
	service.Upstream = upstream

	route := buildRouteFromIngressPath(obj, path, ruleIndex, pathIndex, labels)
	if protocol == internaltypes.AppProtocolWS || protocol == internaltypes.AppProtocolWSS {
		route.EnableWebsocket = ptr.To(true)
	}
	service.Routes = []*adctypes.Route{route}

	t.fillHTTPRoutePoliciesForIngress(tctx, service.Routes)
	return service
}

func (t *Translator) resolveIngressUpstream(
	tctx *provider.TranslateContext,
	obj *networkingv1.Ingress,
	backendService *networkingv1.IngressServiceBackend,
	upstream *adctypes.Upstream,
) string {
	backendRef := convertBackendRef(obj.Namespace, backendService.Name, internaltypes.KindService)
	t.AttachBackendTrafficPolicyToUpstream(backendRef, tctx.BackendTrafficPolicies, upstream)
	// determine service port/port name
	var protocol string
	var port intstr.IntOrString
	if backendService.Port.Number != 0 {
		port = intstr.FromInt32(backendService.Port.Number)
	} else if backendService.Port.Name != "" {
		port = intstr.FromString(backendService.Port.Name)
	}

	getService := tctx.Services[types.NamespacedName{
		Namespace: obj.Namespace,
		Name:      backendService.Name,
	}]
	if getService == nil {
		return protocol
	}
	getServicePort, _ := findMatchingServicePort(getService, port)
	if getServicePort != nil && getServicePort.AppProtocol != nil {
		protocol = *getServicePort.AppProtocol
		if upstream.Scheme == "" {
			upstream.Scheme = appProtocolToUpstreamScheme(*getServicePort.AppProtocol)
		}
	}
	if getService.Spec.Type == corev1.ServiceTypeExternalName {
		servicePort := 80
		if getServicePort != nil {
			servicePort = int(getServicePort.Port)
		}
		upstream.Nodes = adctypes.UpstreamNodes{
			{
				Host:   getService.Spec.ExternalName,
				Port:   servicePort,
				Weight: 1,
			},
		}
		return protocol
	}

	endpointSlices := tctx.EndpointSlices[types.NamespacedName{
		Namespace: obj.Namespace,
		Name:      backendService.Name,
	}]
	if len(endpointSlices) > 0 {
		upstream.Nodes = t.translateEndpointSliceForIngress(1, endpointSlices, getServicePort)
	}

	return protocol
}

func buildRouteFromIngressPath(
	obj *networkingv1.Ingress,
	path *networkingv1.HTTPIngressPath,
	ruleIndex, pathIndex int,
	labels map[string]string,
) *adctypes.Route {
	route := adctypes.NewDefaultRoute()
	route.Name = adctypes.ComposeRouteName(obj.Namespace, obj.Name, fmt.Sprintf("%d-%d", ruleIndex, pathIndex))
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
			prefix := strings.TrimSuffix(path.Path, "/") + "/*"
			uris = append(uris, prefix)
		case networkingv1.PathTypeImplementationSpecific:
			uris = []string{"/*"}
		}
	}
	route.Uris = uris
	return route
}

// translateEndpointSliceForIngress create upstream nodes from EndpointSlice
func (t *Translator) translateEndpointSliceForIngress(weight int, endpointSlices []discoveryv1.EndpointSlice, servicePort *corev1.ServicePort) adctypes.UpstreamNodes {
	nodes := adctypes.UpstreamNodes{}
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
				if !DefaultEndpointFilter(&endpoint) {
					continue
				}
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
