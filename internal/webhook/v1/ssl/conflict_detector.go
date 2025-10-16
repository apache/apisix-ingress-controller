// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ssl

import (
	"context"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	sslutil "github.com/apache/apisix-ingress-controller/internal/ssl"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

var logger = log.Log.WithName("ssl-conflict-detector")

// HostCertMapping represents the relationship between a host and its certificate hash.
type HostCertMapping struct {
	Host            string
	CertificateHash string
	ResourceRef     string
}

// SSLConflict exposes the conflict details to the admission webhook for reporting.
type SSLConflict struct {
	Host                string
	ConflictingResource string
	CertificateHash     string
}

// ConflictDetector detects SSL conflicts among Gateway, Ingress, and ApisixTls resources.
type ConflictDetector struct {
	client      client.Client
	secretCache map[types.NamespacedName]*secretInfo
}

type secretInfo struct {
	hash  string
	hosts []string
}

// NewConflictDetector creates a detector backed by the provided client.
func NewConflictDetector(c client.Client) *ConflictDetector {
	return &ConflictDetector{
		client:      c,
		secretCache: make(map[types.NamespacedName]*secretInfo),
	}
}

// DetectConflicts returns the list of conflicts between the new resource and
// existing resources that are associated with the same GatewayProxy. Best-effort:
// failures while enumerating existing resources or reading Secrets will be logged
// and result in no conflicts instead of blocking the admission.
func (d *ConflictDetector) DetectConflicts(ctx context.Context, obj client.Object) []SSLConflict {
	newMappings := d.buildMappingsForObject(ctx, obj)
	if len(newMappings) == 0 {
		return nil
	}
	gatewayProxy, err := d.resolveGatewayProxy(ctx, obj)
	if err != nil {
		logger.Error(err, "failed to resolve GatewayProxy", "object", objectKey(obj))
		return nil
	}
	if gatewayProxy == nil {
		return nil
	}

	conflicts := make([]SSLConflict, 0)

	// First, check for conflicts within the new resource itself.
	seen := make(map[string]string, len(newMappings))
	for _, mapping := range newMappings {
		if mapping.Host == "" || mapping.CertificateHash == "" {
			continue
		}
		if prev, ok := seen[mapping.Host]; ok {
			if prev != mapping.CertificateHash {
				conflicts = append(conflicts, SSLConflict{
					Host:                mapping.Host,
					ConflictingResource: mapping.ResourceRef,
					CertificateHash:     prev,
				})
			}
			continue
		}
		seen[mapping.Host] = mapping.CertificateHash
	}

	if len(conflicts) > 0 {
		return conflicts
	}

	externalConflicts, err := d.findExternalConflicts(ctx, obj, gatewayProxy, seen)
	if err != nil {
		logger.Error(err, "failed to evaluate existing TLS host mappings", "gatewayProxy", objectKey(gatewayProxy))
		return conflicts
	}

	conflicts = append(conflicts, externalConflicts...)
	return conflicts
}

// FormatConflicts renders a human-readable error message for multiple conflicts.
func FormatConflicts(conflicts []SSLConflict) string {
	if len(conflicts) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("SSL configuration conflicts detected:")
	for _, conflict := range conflicts {
		sb.WriteString(fmt.Sprintf("\n- Host '%s' is already configured with a different certificate in %s", conflict.Host, conflict.ConflictingResource))
	}
	return sb.String()
}

// BuildGatewayMappings calculates host-to-certificate mappings for a Gateway.
func (d *ConflictDetector) BuildGatewayMappings(ctx context.Context, gateway *gatewayv1.Gateway) []HostCertMapping {
	mappings := make([]HostCertMapping, 0)

	if gateway == nil {
		return mappings
	}

	for _, listener := range gateway.Spec.Listeners {
		if listener.TLS == nil || listener.TLS.CertificateRefs == nil {
			continue
		}
		for _, ref := range listener.TLS.CertificateRefs {
			if ref.Kind != nil && *ref.Kind != internaltypes.KindSecret {
				continue
			}
			if ref.Group != nil && string(*ref.Group) != corev1.GroupName {
				continue
			}
			secretNN := types.NamespacedName{
				Namespace: gateway.Namespace,
				Name:      string(ref.Name),
			}
			if ref.Namespace != nil && *ref.Namespace != "" {
				secretNN.Namespace = string(*ref.Namespace)
			}

			info, err := d.getSecretInfo(ctx, secretNN)
			if err != nil {
				logger.Error(err, "failed to read secret for Gateway", "gateway", objectKey(gateway), "secret", secretNN)
				continue
			}

			hosts := make([]string, 0, 1)
			if listener.Hostname != nil && *listener.Hostname != "" {
				hosts = append(hosts, string(*listener.Hostname))
			}
			hosts = sslutil.NormalizeHosts(hosts)
			if len(hosts) == 0 {
				hosts = info.hosts
			}
			for _, host := range hosts {
				mappings = append(mappings, HostCertMapping{
					Host:            host,
					CertificateHash: info.hash,
					ResourceRef:     fmt.Sprintf("%s/%s/%s", internaltypes.KindGateway, gateway.Namespace, gateway.Name),
				})
			}
		}
	}

	return mappings
}

// BuildIngressMappings calculates host-to-certificate mappings for an Ingress.
func (d *ConflictDetector) BuildIngressMappings(ctx context.Context, ingress *networkingv1.Ingress) []HostCertMapping {
	mappings := make([]HostCertMapping, 0)
	if ingress == nil {
		return mappings
	}

	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}
		secretNN := types.NamespacedName{Namespace: ingress.Namespace, Name: tls.SecretName}
		info, err := d.getSecretInfo(ctx, secretNN)
		if err != nil {
			logger.Error(err, "failed to read secret for Ingress", "ingress", objectKey(ingress), "secret", secretNN)
			continue
		}

		hosts := sslutil.NormalizeHosts(tls.Hosts)
		if len(hosts) == 0 {
			hosts = info.hosts
		}
		for _, host := range hosts {
			mappings = append(mappings, HostCertMapping{
				Host:            host,
				CertificateHash: info.hash,
				ResourceRef:     fmt.Sprintf("%s/%s/%s", internaltypes.KindIngress, ingress.Namespace, ingress.Name),
			})
		}
	}

	return mappings
}

// BuildApisixTlsMappings calculates host-to-certificate mappings for an ApisixTls resource.
func (d *ConflictDetector) BuildApisixTlsMappings(ctx context.Context, tls *apiv2.ApisixTls) []HostCertMapping {
	mappings := make([]HostCertMapping, 0)
	if tls == nil {
		return mappings
	}

	secretNN := types.NamespacedName{
		Namespace: tls.Spec.Secret.Namespace,
		Name:      tls.Spec.Secret.Name,
	}
	info, err := d.getSecretInfo(ctx, secretNN)
	if err != nil {
		logger.Error(err, "failed to read secret for ApisixTls", "apisixtls", objectKey(tls), "secret", secretNN)
		return mappings
	}

	hosts := make([]string, 0, len(tls.Spec.Hosts))
	for _, host := range tls.Spec.Hosts {
		hosts = append(hosts, string(host))
	}
	hosts = sslutil.NormalizeHosts(hosts)
	// NOTICE: hosts is required by the CRD, so this should never happen
	// if len(hosts) == 0 {
	// 	hosts = info.hosts
	// }
	for _, host := range hosts {
		mappings = append(mappings, HostCertMapping{
			Host:            host,
			CertificateHash: info.hash,
			ResourceRef:     fmt.Sprintf("%s/%s/%s", internaltypes.KindApisixTls, tls.Namespace, tls.Name),
		})
	}

	return mappings
}

func (d *ConflictDetector) getSecretInfo(ctx context.Context, nn types.NamespacedName) (*secretInfo, error) {
	if nn.Name == "" || nn.Namespace == "" {
		return nil, fmt.Errorf("secret namespaced name is incomplete: %s", nn)
	}
	if info, ok := d.secretCache[nn]; ok {
		return info, nil
	}

	var secret corev1.Secret
	if err := d.client.Get(ctx, nn, &secret); err != nil {
		return nil, err
	}

	cert, err := sslutil.ExtractCertificate(&secret)
	if err != nil {
		return nil, err
	}

	hash, err := sslutil.CertificateHash(cert)
	if err != nil {
		return nil, err
	}
	hosts, err := sslutil.ExtractHostsFromCertificate(cert)
	if err != nil {
		logger.Error(err, "failed to extract hosts from certificate", "secret", nn)
		hosts = nil
	}
	info := &secretInfo{
		hash:  hash,
		hosts: sslutil.NormalizeHosts(hosts),
	}
	d.secretCache[nn] = info
	return info, nil
}

func (d *ConflictDetector) resolveGatewayProxy(ctx context.Context, obj client.Object) (*v1alpha1.GatewayProxy, error) {
	switch resource := obj.(type) {
	case *gatewayv1.Gateway:
		return controller.GetGatewayProxyByGateway(ctx, d.client, resource)
	case *networkingv1.Ingress:
		ingressClass, err := controller.FindMatchingIngressClass(ctx, d.client, logger, resource)
		if err != nil {
			return nil, err
		}
		if ingressClass == nil {
			return nil, nil
		}
		return controller.GetGatewayProxyByIngressClass(ctx, d.client, ingressClass)
	case *apiv2.ApisixTls:
		ingressClass, err := controller.FindMatchingIngressClass(ctx, d.client, logger, resource)
		if err != nil {
			return nil, err
		}
		if ingressClass == nil {
			return nil, nil
		}
		return controller.GetGatewayProxyByIngressClass(ctx, d.client, ingressClass)
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}
}

func (d *ConflictDetector) findExternalConflicts(ctx context.Context, obj client.Object, gatewayProxy *v1alpha1.GatewayProxy, hosts map[string]string) ([]SSLConflict, error) {
	excludeUID := obj.GetUID()
	hostValues := make([]string, 0, len(hosts))
	for host := range hosts {
		hostValues = append(hostValues, host)
	}
	sort.Strings(hostValues)

	conflictSet := make(map[string]SSLConflict)
	proxyCache := make(map[types.UID]*v1alpha1.GatewayProxy)
	mappingCache := make(map[types.UID][]HostCertMapping)

	var noHostCandidates []client.Object
	noHostFetched := false

	for _, host := range hostValues {
		candidates, err := d.listResourcesByHost(ctx, host)
		if err != nil {
			logger.Error(err, "failed to list resources by host", "host", host)
			return nil, err
		}
		if host != "" {
			if !noHostFetched {
				// List resources with empty host.
				noHostCandidates, err = d.listResourcesByHost(ctx, "")
				if err != nil {
					logger.Error(err, "failed to list resources by host", "host", "", "object", objectKey(obj))
					return nil, err
				}
				noHostFetched = true
			}
			candidates = mergeCandidateObjects(candidates, noHostCandidates)
		}
		for _, candidate := range candidates {
			if candidate.GetUID() == excludeUID {
				continue
			}

			resolvedProxy, err := d.resolveGatewayProxyWithCache(ctx, candidate, proxyCache)
			if err != nil {
				logger.Error(err, "failed to resolve GatewayProxy for indexed resource", "resource", objectKey(candidate), "host", host)
				continue
			}
			// we only check if the resolved proxy is the same as the gateway proxy,
			if resolvedProxy == nil || !gatewayProxiesEqual(resolvedProxy, gatewayProxy) {
				continue
			}

			mapping, ok := d.mappingForHostWithCache(ctx, candidate, host, mappingCache)
			if !ok {
				continue
			}
			// same cert hash, no conflict
			if mapping.CertificateHash == hosts[host] {
				continue
			}

			key := fmt.Sprintf("%s|%s|%s", host, mapping.ResourceRef, mapping.CertificateHash)
			if _, exists := conflictSet[key]; exists {
				continue
			}
			conflictSet[key] = SSLConflict{
				Host:                host,
				ConflictingResource: mapping.ResourceRef,
				CertificateHash:     mapping.CertificateHash,
			}
		}
	}

	if len(conflictSet) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(conflictSet))
	for key := range conflictSet {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	results := make([]SSLConflict, 0, len(keys))
	for _, key := range keys {
		results = append(results, conflictSet[key])
	}
	return results, nil
}

func (d *ConflictDetector) listResourcesByHost(ctx context.Context, host string) ([]client.Object, error) {
	results := make([]client.Object, 0)

	var gatewayList gatewayv1.GatewayList
	if err := d.client.List(ctx, &gatewayList, client.MatchingFields{indexer.TLSHostIndexRef: host}); err != nil {
		return nil, err
	}
	for i := range gatewayList.Items {
		results = append(results, gatewayList.Items[i].DeepCopy())
	}

	var ingressList networkingv1.IngressList
	if err := d.client.List(ctx, &ingressList, client.MatchingFields{indexer.TLSHostIndexRef: host}); err != nil {
		return nil, err
	}
	for i := range ingressList.Items {
		results = append(results, ingressList.Items[i].DeepCopy())
	}

	var tlsList apiv2.ApisixTlsList
	if err := d.client.List(ctx, &tlsList, client.MatchingFields{indexer.TLSHostIndexRef: host}); err != nil {
		return nil, err
	}
	for i := range tlsList.Items {
		results = append(results, tlsList.Items[i].DeepCopy())
	}

	return results, nil
}

func mergeCandidateObjects(primary, additional []client.Object) []client.Object {
	if len(additional) == 0 {
		return primary
	}
	seen := make(map[types.UID]struct{}, len(primary))
	for _, obj := range primary {
		seen[obj.GetUID()] = struct{}{}
	}
	for _, obj := range additional {
		if _, exists := seen[obj.GetUID()]; exists {
			continue
		}
		primary = append(primary, obj)
		seen[obj.GetUID()] = struct{}{}
	}
	return primary
}

func (d *ConflictDetector) resolveGatewayProxyWithCache(ctx context.Context, obj client.Object, cache map[types.UID]*v1alpha1.GatewayProxy) (*v1alpha1.GatewayProxy, error) {
	if proxy, ok := cache[obj.GetUID()]; ok {
		return proxy, nil
	}
	proxy, err := d.resolveGatewayProxy(ctx, obj)
	if err != nil {
		return nil, err
	}
	cache[obj.GetUID()] = proxy
	return proxy, nil
}

func (d *ConflictDetector) mappingForHostWithCache(ctx context.Context, obj client.Object, host string, cache map[types.UID][]HostCertMapping) (HostCertMapping, bool) {
	mappings, ok := cache[obj.GetUID()]
	if !ok {
		mappings = d.buildMappingsForObject(ctx, obj)
		cache[obj.GetUID()] = mappings
	}

	for _, mapping := range mappings {
		if mapping.Host == host {
			return mapping, true
		}
	}
	return HostCertMapping{}, false
}

func (d *ConflictDetector) buildMappingsForObject(ctx context.Context, obj client.Object) []HostCertMapping {
	switch resource := obj.(type) {
	case *gatewayv1.Gateway:
		return d.BuildGatewayMappings(ctx, resource)
	case *networkingv1.Ingress:
		return d.BuildIngressMappings(ctx, resource)
	case *apiv2.ApisixTls:
		return d.BuildApisixTlsMappings(ctx, resource)
	default:
		return nil
	}
}

func gatewayProxiesEqual(a, b *v1alpha1.GatewayProxy) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Namespace == b.Namespace && a.Name == b.Name
}

func objectKey(obj client.Object) types.NamespacedName {
	if obj == nil {
		return types.NamespacedName{}
	}
	return types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
}
