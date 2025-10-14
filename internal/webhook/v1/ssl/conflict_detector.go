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
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
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

// DetectConflicts returns the list of conflicts between the provided mappings and
// existing resources that are associated with the same GatewayProxy. Best-effort:
// failures while enumerating existing resources or reading Secrets will be logged
// and result in no conflicts instead of blocking the admission.
func (d *ConflictDetector) DetectConflicts(ctx context.Context, obj client.Object, newMappings []HostCertMapping) ([]SSLConflict, error) {
	gatewayProxy, err := d.resolveGatewayProxy(ctx, obj)
	if err != nil {
		logger.Error(err, "failed to resolve GatewayProxy", "object", objectKey(obj))
		return nil, nil
	}
	if gatewayProxy == nil {
		return nil, nil
	}

	existingMappings, err := d.collectExistingMappings(ctx, gatewayProxy, obj.GetUID())
	if err != nil {
		logger.Error(err, "failed to collect existing SSL mappings", "gatewayProxy", objectKey(gatewayProxy))
		return nil, nil
	}

	conflicts := make([]SSLConflict, 0)
	byHost := make(map[string]HostCertMapping, len(existingMappings))
	for _, mapping := range existingMappings {
		if mapping.Host == "" || mapping.CertificateHash == "" {
			continue
		}
		if existing, ok := byHost[mapping.Host]; ok {
			if existing.CertificateHash == mapping.CertificateHash {
				continue
			}
			// keep the first encountered mapping to surface a deterministic conflict reference
			continue
		}
		byHost[mapping.Host] = mapping
	}

	seen := make(map[string]string, len(newMappings))
	// TODO: need to check with self-referencing mappings
	for _, mapping := range newMappings {
		if mapping.Host == "" || mapping.CertificateHash == "" {
			continue
		}
		if prev, ok := seen[mapping.Host]; ok {
			// prefer the first hash when duplicates appear inside the same object
			if prev != mapping.CertificateHash {
				seen[mapping.Host] = mapping.CertificateHash
			}
			continue
		}
		seen[mapping.Host] = mapping.CertificateHash
	}

	for host, hash := range seen {
		existing, ok := byHost[host]
		if !ok {
			continue
		}
		if existing.CertificateHash == hash {
			continue
		}
		conflicts = append(conflicts, SSLConflict{
			Host:                host,
			ConflictingResource: existing.ResourceRef,
			CertificateHash:     existing.CertificateHash,
		})
	}

	return conflicts, nil
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
func (d *ConflictDetector) BuildGatewayMappings(ctx context.Context, gateway *gatewayv1.Gateway) ([]HostCertMapping, []string) {
	mappings := make([]HostCertMapping, 0)
	warnings := make([]string, 0)

	if gateway == nil {
		return mappings, warnings
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
				warnings = append(warnings, fmt.Sprintf("failed to read Secret %s for Gateway %s/%s: %v", secretNN, gateway.Namespace, gateway.Name, err))
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
					ResourceRef:     fmt.Sprintf("Gateway/%s/%s", gateway.Namespace, gateway.Name),
				})
			}
		}
	}

	return mappings, warnings
}

// BuildIngressMappings calculates host-to-certificate mappings for an Ingress.
func (d *ConflictDetector) BuildIngressMappings(ctx context.Context, ingress *networkingv1.Ingress) ([]HostCertMapping, []string) {
	mappings := make([]HostCertMapping, 0)
	warnings := make([]string, 0)
	if ingress == nil {
		return mappings, warnings
	}

	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}
		secretNN := types.NamespacedName{Namespace: ingress.Namespace, Name: tls.SecretName}
		info, err := d.getSecretInfo(ctx, secretNN)
		if err != nil {
			logger.Error(err, "failed to read secret for Ingress", "ingress", objectKey(ingress), "secret", secretNN)
			warnings = append(warnings, fmt.Sprintf("failed to read Secret %s for Ingress %s/%s: %v", secretNN, ingress.Namespace, ingress.Name, err))
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
				ResourceRef:     fmt.Sprintf("Ingress/%s/%s", ingress.Namespace, ingress.Name),
			})
		}
	}

	return mappings, warnings
}

// BuildApisixTlsMappings calculates host-to-certificate mappings for an ApisixTls resource.
func (d *ConflictDetector) BuildApisixTlsMappings(ctx context.Context, tls *apiv2.ApisixTls) ([]HostCertMapping, []string) {
	mappings := make([]HostCertMapping, 0)
	warnings := make([]string, 0)
	if tls == nil {
		return mappings, warnings
	}

	secretNN := types.NamespacedName{
		Namespace: tls.Spec.Secret.Namespace,
		Name:      tls.Spec.Secret.Name,
	}
	info, err := d.getSecretInfo(ctx, secretNN)
	if err != nil {
		logger.Error(err, "failed to read secret for ApisixTls", "apisixtls", objectKey(tls), "secret", secretNN)
		warnings = append(warnings, fmt.Sprintf("failed to read Secret %s for ApisixTls %s/%s: %v", secretNN, tls.Namespace, tls.Name, err))
		return mappings, warnings
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
			ResourceRef:     fmt.Sprintf("ApisixTls/%s/%s", tls.Namespace, tls.Name),
		})
	}

	return mappings, warnings
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

	hash := sslutil.CertificateHash(cert)
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

func (d *ConflictDetector) collectExistingMappings(ctx context.Context, gatewayProxy *v1alpha1.GatewayProxy, excludeUID types.UID) ([]HostCertMapping, error) {
	mappings := make([]HostCertMapping, 0)

	if gatewayProxy == nil {
		return mappings, nil
	}

	indexKey := indexer.GenIndexKey(gatewayProxy.Namespace, gatewayProxy.Name)

	processedGateways := make(map[types.UID]struct{})
	var gatewayList gatewayv1.GatewayList
	if err := d.client.List(ctx, &gatewayList, client.MatchingFields{indexer.ParametersRef: indexKey}); err != nil {
		return nil, err
	}
	for i := range gatewayList.Items {
		gateway := &gatewayList.Items[i]
		if gateway.GetUID() == excludeUID {
			continue
		}
		if _, ok := processedGateways[gateway.GetUID()]; ok {
			continue
		}
		gatewayMappings, _ := d.BuildGatewayMappings(ctx, gateway)
		mappings = append(mappings, gatewayMappings...)
		processedGateways[gateway.GetUID()] = struct{}{}
	}

	processedIngress := make(map[types.UID]struct{})
	processedTls := make(map[types.UID]struct{})
	defaultIngressClasses := make(map[string]struct{})

	var ingressClassList networkingv1.IngressClassList
	if err := d.client.List(ctx, &ingressClassList, client.MatchingFields{indexer.IngressClassParametersRef: indexKey}); err != nil {
		return nil, err
	}
	for i := range ingressClassList.Items {
		ingressClass := &ingressClassList.Items[i]
		if controller.IsDefaultIngressClass(ingressClass) && ingressClass.Spec.Controller == config.ControllerConfig.ControllerName {
			defaultIngressClasses[ingressClass.Name] = struct{}{}
		}

		var ingressList networkingv1.IngressList
		if err := d.client.List(ctx, &ingressList, client.MatchingFields{indexer.IngressClassRef: ingressClass.Name}); err != nil {
			return nil, err
		}
		for j := range ingressList.Items {
			ingress := &ingressList.Items[j]
			if ingress.GetUID() == excludeUID {
				continue
			}
			if _, ok := processedIngress[ingress.GetUID()]; ok {
				continue
			}
			ingressMappings, _ := d.BuildIngressMappings(ctx, ingress)
			mappings = append(mappings, ingressMappings...)
			processedIngress[ingress.GetUID()] = struct{}{}
		}

		var tlsList apiv2.ApisixTlsList
		if err := d.client.List(ctx, &tlsList, client.MatchingFields{indexer.IngressClassRef: ingressClass.Name}); err != nil {
			return nil, err
		}
		for j := range tlsList.Items {
			tls := &tlsList.Items[j]
			if tls.GetUID() == excludeUID {
				continue
			}
			if _, ok := processedTls[tls.GetUID()]; ok {
				continue
			}
			tlsMappings, _ := d.BuildApisixTlsMappings(ctx, tls)
			mappings = append(mappings, tlsMappings...)
			processedTls[tls.GetUID()] = struct{}{}
		}
	}

	if len(defaultIngressClasses) > 0 {
		var allIngress networkingv1.IngressList
		if err := d.client.List(ctx, &allIngress); err != nil {
			return nil, err
		}
		for i := range allIngress.Items {
			ingress := &allIngress.Items[i]
			if ingress.Spec.IngressClassName != nil {
				continue
			}
			if ingress.GetUID() == excludeUID {
				continue
			}
			if _, ok := processedIngress[ingress.GetUID()]; ok {
				continue
			}
			ingressMappings, _ := d.BuildIngressMappings(ctx, ingress)
			mappings = append(mappings, ingressMappings...)
			processedIngress[ingress.GetUID()] = struct{}{}
		}

		var allTls apiv2.ApisixTlsList
		if err := d.client.List(ctx, &allTls); err != nil {
			return nil, err
		}
		for i := range allTls.Items {
			tls := &allTls.Items[i]
			if tls.Spec.IngressClassName != "" {
				continue
			}
			if tls.GetUID() == excludeUID {
				continue
			}
			if _, ok := processedTls[tls.GetUID()]; ok {
				continue
			}
			tlsMappings, _ := d.BuildApisixTlsMappings(ctx, tls)
			mappings = append(mappings, tlsMappings...)
			processedTls[tls.GetUID()] = struct{}{}
		}
	}

	return mappings, nil
}

func objectKey(obj client.Object) types.NamespacedName {
	if obj == nil {
		return types.NamespacedName{}
	}
	return types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
}
