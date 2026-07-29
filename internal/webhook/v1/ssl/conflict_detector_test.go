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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

const (
	testNamespace    = "default"
	testIngressClass = "example-class"
)

func TestConflictDetectorDetectsGatewayConflict(t *testing.T) {
	scheme := buildScheme(t)
	secretA := newTLSSecret(t, "cert-a", []string{"example.com"})
	secretB := newTLSSecret(t, "cert-b", []string{"example.com"})

	gatewayProxy := &v1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-gp",
			Namespace: testNamespace,
			UID:       "gatewayproxy-uid",
		},
	}

	modeTerminate := gatewayv1.TLSModeTerminate
	hostname := gatewayv1.Hostname("example.com")
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-gateway",
			Namespace: testNamespace,
			UID:       "gateway-uid",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName("demo-gc"),
			Listeners: []gatewayv1.Listener{
				{
					Name:     "tls",
					Protocol: gatewayv1.HTTPSProtocolType,
					Port:     443,
					Hostname: &hostname,
					TLS: &gatewayv1.GatewayTLSConfig{
						Mode: &modeTerminate,
						CertificateRefs: []gatewayv1.SecretObjectReference{
							{Name: gatewayv1.ObjectName(secretA.Name)},
						},
					},
				},
			},
		},
	}
	gateway.Spec.Infrastructure = &gatewayv1.GatewayInfrastructure{
		ParametersRef: &gatewayv1.LocalParametersReference{
			Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
			Kind:  gatewayv1.Kind(internaltypes.KindGatewayProxy),
			Name:  gatewayProxy.Name,
		},
	}

	ingressClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: testIngressClass,
		},
		Spec: networkingv1.IngressClassSpec{
			Controller: config.ControllerConfig.ControllerName,
			Parameters: &networkingv1.IngressClassParametersReference{
				APIGroup:  ptr.To(v1alpha1.GroupVersion.Group),
				Kind:      internaltypes.KindGatewayProxy,
				Name:      gatewayProxy.Name,
				Namespace: ptr.To(testNamespace),
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithIndex(&gatewayv1.Gateway{}, indexer.ParametersRef, indexer.GatewayParametersRefIndexFunc).
		WithIndex(&gatewayv1.Gateway{}, indexer.TLSHostIndexRef, indexer.GatewayTLSHostIndexFunc).
		WithIndex(&networkingv1.IngressClass{}, indexer.IngressClassParametersRef, indexer.IngressClassParametersRefIndexFunc).
		WithIndex(&networkingv1.Ingress{}, indexer.IngressClassRef, indexer.IngressClassRefIndexFunc).
		WithIndex(&networkingv1.Ingress{}, indexer.TLSHostIndexRef, indexer.IngressTLSHostIndexFunc).
		WithIndex(&apiv2.ApisixTls{}, indexer.IngressClassRef, indexer.ApisixTlsIngressClassIndexFunc).
		WithIndex(&apiv2.ApisixTls{}, indexer.TLSHostIndexRef, indexer.ApisixTlsHostIndexFunc).
		WithObjects(secretA, secretB, gatewayProxy, gateway, ingressClass).
		Build()

	detector := NewConflictDetector(fakeClient)
	ctx := context.Background()

	newTls := &apiv2.ApisixTls{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "incoming",
			Namespace: testNamespace,
			UID:       "apisixtls-uid",
		},
		Spec: apiv2.ApisixTlsSpec{
			IngressClassName: testIngressClass,
			Hosts:            []apiv2.HostType{"example.com"},
			Secret: apiv2.ApisixSecret{
				Name:      secretB.Name,
				Namespace: secretB.Namespace,
			},
		},
	}

	conflicts := detector.DetectConflicts(ctx, newTls)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	conflict := conflicts[0]
	if conflict.Host != "example.com" {
		t.Fatalf("unexpected host: %s", conflict.Host)
	}
	expectedRef := fmt.Sprintf("Gateway/%s/%s", gateway.Namespace, gateway.Name)
	if conflict.ConflictingResource != expectedRef {
		t.Fatalf("unexpected conflicting resource: %s", conflict.ConflictingResource)
	}
}

func TestConflictDetectorAllowedWhenCertificateMatches(t *testing.T) {
	scheme := buildScheme(t)
	secret := newTLSSecret(t, "shared-cert", []string{"shared.example.com"})

	gatewayProxy := &v1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gp",
			Namespace: testNamespace,
			UID:       "gatewayproxy-uid-2",
		},
	}
	modeTerminate := gatewayv1.TLSModeTerminate
	listenerHostname := gatewayv1.Hostname("shared.example.com")
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gw",
			Namespace: testNamespace,
			UID:       "gateway-uid-2",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName("gc"),
			Listeners: []gatewayv1.Listener{
				{
					Name:     "tls",
					Protocol: gatewayv1.HTTPSProtocolType,
					Port:     443,
					Hostname: &listenerHostname,
					TLS: &gatewayv1.GatewayTLSConfig{
						Mode:            &modeTerminate,
						CertificateRefs: []gatewayv1.SecretObjectReference{{Name: gatewayv1.ObjectName(secret.Name)}},
					},
				},
			},
		},
	}
	gateway.Spec.Infrastructure = &gatewayv1.GatewayInfrastructure{
		ParametersRef: &gatewayv1.LocalParametersReference{
			Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
			Kind:  gatewayv1.Kind(internaltypes.KindGatewayProxy),
			Name:  gatewayProxy.Name,
		},
	}

	ingressClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: testIngressClass},
		Spec: networkingv1.IngressClassSpec{
			Controller: config.ControllerConfig.ControllerName,
			Parameters: &networkingv1.IngressClassParametersReference{
				APIGroup:  ptr.To(v1alpha1.GroupVersion.Group),
				Kind:      internaltypes.KindGatewayProxy,
				Name:      gatewayProxy.Name,
				Namespace: ptr.To(testNamespace),
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithIndex(&gatewayv1.Gateway{}, indexer.ParametersRef, indexer.GatewayParametersRefIndexFunc).
		WithIndex(&gatewayv1.Gateway{}, indexer.TLSHostIndexRef, indexer.GatewayTLSHostIndexFunc).
		WithIndex(&networkingv1.IngressClass{}, indexer.IngressClassParametersRef, indexer.IngressClassParametersRefIndexFunc).
		WithIndex(&networkingv1.Ingress{}, indexer.IngressClassRef, indexer.IngressClassRefIndexFunc).
		WithIndex(&networkingv1.Ingress{}, indexer.TLSHostIndexRef, indexer.IngressTLSHostIndexFunc).
		WithIndex(&apiv2.ApisixTls{}, indexer.IngressClassRef, indexer.ApisixTlsIngressClassIndexFunc).
		WithIndex(&apiv2.ApisixTls{}, indexer.TLSHostIndexRef, indexer.ApisixTlsHostIndexFunc).
		WithObjects(secret, gatewayProxy, gateway, ingressClass).
		Build()

	detector := NewConflictDetector(client)
	ctx := context.Background()

	newTls := &apiv2.ApisixTls{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allowed",
			Namespace: testNamespace,
			UID:       "apisixtls-uid-2",
		},
		Spec: apiv2.ApisixTlsSpec{
			IngressClassName: testIngressClass,
			Hosts:            []apiv2.HostType{"shared.example.com"},
			Secret:           apiv2.ApisixSecret{Name: secret.Name, Namespace: secret.Namespace},
		},
	}

	conflicts := detector.DetectConflicts(ctx, newTls)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
}

func buildScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{
		corev1.AddToScheme,
		networkingv1.AddToScheme,
		gatewayv1.Install,
		apiv2.AddToScheme,
		v1alpha1.AddToScheme,
	} {
		if err := add(scheme); err != nil {
			t.Fatalf("failed to add to scheme: %v", err)
		}
	}
	return scheme
}

func newTLSSecret(t *testing.T, name string, hosts []string) *corev1.Secret {
	cert, key := generateCertificate(t, hosts)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       cert,
			corev1.TLSPrivateKeyKey: key,
		},
	}
}

func TestConflictDetectorDetectsSelfConflict(t *testing.T) {
	scheme := buildScheme(t)
	secretA := newTLSSecret(t, "cert-a", []string{"example.com"})
	secretB := newTLSSecret(t, "cert-b", []string{"example.com"})

	gatewayProxy := &v1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-gp",
			Namespace: testNamespace,
			UID:       "gatewayproxy-uid-3",
		},
	}

	modeTerminate := gatewayv1.TLSModeTerminate
	hostname := gatewayv1.Hostname("example.com")
	// Create a Gateway with TWO listeners using DIFFERENT certificates for the SAME host
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-gateway",
			Namespace: testNamespace,
			UID:       "gateway-uid-3",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName("demo-gc"),
			Listeners: []gatewayv1.Listener{
				{
					Name:     "tls-1",
					Protocol: gatewayv1.HTTPSProtocolType,
					Port:     443,
					Hostname: &hostname,
					TLS: &gatewayv1.GatewayTLSConfig{
						Mode: &modeTerminate,
						CertificateRefs: []gatewayv1.SecretObjectReference{
							{Name: gatewayv1.ObjectName(secretA.Name)},
						},
					},
				},
				{
					Name:     "tls-2",
					Protocol: gatewayv1.HTTPSProtocolType,
					Port:     8443,
					Hostname: &hostname,
					TLS: &gatewayv1.GatewayTLSConfig{
						Mode: &modeTerminate,
						CertificateRefs: []gatewayv1.SecretObjectReference{
							{Name: gatewayv1.ObjectName(secretB.Name)},
						},
					},
				},
			},
		},
	}
	gateway.Spec.Infrastructure = &gatewayv1.GatewayInfrastructure{
		ParametersRef: &gatewayv1.LocalParametersReference{
			Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
			Kind:  gatewayv1.Kind(internaltypes.KindGatewayProxy),
			Name:  gatewayProxy.Name,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithIndex(&gatewayv1.Gateway{}, indexer.ParametersRef, indexer.GatewayParametersRefIndexFunc).
		WithIndex(&gatewayv1.Gateway{}, indexer.TLSHostIndexRef, indexer.GatewayTLSHostIndexFunc).
		WithIndex(&networkingv1.IngressClass{}, indexer.IngressClassParametersRef, indexer.IngressClassParametersRefIndexFunc).
		WithIndex(&networkingv1.Ingress{}, indexer.IngressClassRef, indexer.IngressClassRefIndexFunc).
		WithIndex(&networkingv1.Ingress{}, indexer.TLSHostIndexRef, indexer.IngressTLSHostIndexFunc).
		WithIndex(&apiv2.ApisixTls{}, indexer.IngressClassRef, indexer.ApisixTlsIngressClassIndexFunc).
		WithIndex(&apiv2.ApisixTls{}, indexer.TLSHostIndexRef, indexer.ApisixTlsHostIndexFunc).
		WithObjects(secretA, secretB, gatewayProxy, gateway).
		Build()

	detector := NewConflictDetector(fakeClient)
	ctx := context.Background()

	// Build mappings for this Gateway - should have 2 mappings for same host with different certs
	mappings := detector.BuildGatewayMappings(ctx, gateway)
	if len(mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(mappings))
	}

	// Both mappings should be for the same host
	if mappings[0].Host != mappings[1].Host {
		t.Fatalf("expected same host, got %s and %s", mappings[0].Host, mappings[1].Host)
	}

	// But with different certificate hashes
	if mappings[0].CertificateHash == mappings[1].CertificateHash {
		t.Fatalf("expected different certificate hashes, but they are the same: %s", mappings[0].CertificateHash)
	}

	// DetectConflicts should detect this self-conflict
	conflicts := detector.DetectConflicts(ctx, gateway)

	// Should detect 1 conflict (the resource conflicts with itself)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 self-conflict, got %d", len(conflicts))
	}

	conflict := conflicts[0]
	if conflict.Host != "example.com" {
		t.Fatalf("unexpected host: %s", conflict.Host)
	}

	// The conflicting resource should point to itself
	expectedRef := fmt.Sprintf("Gateway/%s/%s", gateway.Namespace, gateway.Name)
	if conflict.ConflictingResource != expectedRef {
		t.Fatalf("unexpected conflicting resource: %s, expected %s", conflict.ConflictingResource, expectedRef)
	}
}

// newApisixTls builds an ApisixTls bound to the shared testIngressClass.
func newApisixTls(name, uid string, hosts []string, secret *corev1.Secret) *apiv2.ApisixTls {
	hostTypes := make([]apiv2.HostType, 0, len(hosts))
	for _, h := range hosts {
		hostTypes = append(hostTypes, apiv2.HostType(h))
	}
	return &apiv2.ApisixTls{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			UID:       types.UID(uid),
		},
		Spec: apiv2.ApisixTlsSpec{
			IngressClassName: testIngressClass,
			Hosts:            hostTypes,
			Secret: apiv2.ApisixSecret{
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
		},
	}
}

func buildConflictClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()
	scheme := buildScheme(t)
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithIndex(&gatewayv1.Gateway{}, indexer.ParametersRef, indexer.GatewayParametersRefIndexFunc).
		WithIndex(&gatewayv1.Gateway{}, indexer.TLSHostIndexRef, indexer.GatewayTLSHostIndexFunc).
		WithIndex(&networkingv1.IngressClass{}, indexer.IngressClassParametersRef, indexer.IngressClassParametersRefIndexFunc).
		WithIndex(&networkingv1.Ingress{}, indexer.IngressClassRef, indexer.IngressClassRefIndexFunc).
		WithIndex(&networkingv1.Ingress{}, indexer.TLSHostIndexRef, indexer.IngressTLSHostIndexFunc).
		WithIndex(&apiv2.ApisixTls{}, indexer.IngressClassRef, indexer.ApisixTlsIngressClassIndexFunc).
		WithIndex(&apiv2.ApisixTls{}, indexer.TLSHostIndexRef, indexer.ApisixTlsHostIndexFunc).
		WithObjects(objs...).
		Build()
}

func sharedIngressClassFixtures() (*v1alpha1.GatewayProxy, *networkingv1.IngressClass) {
	gatewayProxy := &v1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-gp",
			Namespace: testNamespace,
			UID:       "gatewayproxy-uid-wildcard",
		},
	}
	ingressClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: testIngressClass},
		Spec: networkingv1.IngressClassSpec{
			Controller: config.ControllerConfig.ControllerName,
			Parameters: &networkingv1.IngressClassParametersReference{
				APIGroup:  ptr.To(v1alpha1.GroupVersion.Group),
				Kind:      internaltypes.KindGatewayProxy,
				Name:      gatewayProxy.Name,
				Namespace: ptr.To(testNamespace),
			},
		},
	}
	return gatewayProxy, ingressClass
}

// TestConflictDetectorDetectsExactOverlappingWildcard verifies an incoming
// exact host conflicts with an existing covering wildcard, even though the two
// are indexed under different keys.
func TestConflictDetectorDetectsExactOverlappingWildcard(t *testing.T) {
	existingSecret := newTLSSecret(t, "existing-cert", []string{"*.example.com"})
	incomingSecret := newTLSSecret(t, "incoming-cert", []string{"app.example.com"})
	gatewayProxy, ingressClass := sharedIngressClassFixtures()

	existing := newApisixTls("existing", "existing-uid", []string{"*.example.com"}, existingSecret)
	incoming := newApisixTls("incoming", "incoming-uid", []string{"app.example.com"}, incomingSecret)

	c := buildConflictClient(t, existingSecret, incomingSecret, gatewayProxy, ingressClass, existing)
	detector := NewConflictDetector(c)

	conflicts := detector.DetectConflicts(context.Background(), incoming)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %+v", len(conflicts), conflicts)
	}
	if conflicts[0].Host != "app.example.com" {
		t.Fatalf("unexpected host: %s", conflicts[0].Host)
	}
	expectedRef := fmt.Sprintf("ApisixTls/%s/%s", existing.Namespace, existing.Name)
	if conflicts[0].ConflictingResource != expectedRef {
		t.Fatalf("unexpected conflicting resource: %s", conflicts[0].ConflictingResource)
	}
}

// TestConflictDetectorDetectsWildcardOverlappingExact verifies the inverse: an
// incoming wildcard conflicts with an existing exact host it covers.
func TestConflictDetectorDetectsWildcardOverlappingExact(t *testing.T) {
	existingSecret := newTLSSecret(t, "existing-cert", []string{"app.example.com"})
	incomingSecret := newTLSSecret(t, "incoming-cert", []string{"*.example.com"})
	gatewayProxy, ingressClass := sharedIngressClassFixtures()

	existing := newApisixTls("existing", "existing-uid", []string{"app.example.com"}, existingSecret)
	incoming := newApisixTls("incoming", "incoming-uid", []string{"*.example.com"}, incomingSecret)

	c := buildConflictClient(t, existingSecret, incomingSecret, gatewayProxy, ingressClass, existing)
	detector := NewConflictDetector(c)

	conflicts := detector.DetectConflicts(context.Background(), incoming)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %+v", len(conflicts), conflicts)
	}
	if conflicts[0].Host != "*.example.com" {
		t.Fatalf("unexpected host: %s", conflicts[0].Host)
	}
}

// TestConflictDetectorAllowsNonOverlappingWildcard guards against false
// positives: a wildcard and an unrelated exact host must not conflict.
func TestConflictDetectorAllowsNonOverlappingWildcard(t *testing.T) {
	existingSecret := newTLSSecret(t, "existing-cert", []string{"a.b.example.com"})
	incomingSecret := newTLSSecret(t, "incoming-cert", []string{"*.example.com"})
	gatewayProxy, ingressClass := sharedIngressClassFixtures()

	existing := newApisixTls("existing", "existing-uid", []string{"a.b.example.com"}, existingSecret)
	incoming := newApisixTls("incoming", "incoming-uid", []string{"*.example.com"}, incomingSecret)

	c := buildConflictClient(t, existingSecret, incomingSecret, gatewayProxy, ingressClass, existing)
	detector := NewConflictDetector(c)

	conflicts := detector.DetectConflicts(context.Background(), incoming)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflict (multi-label host not covered by wildcard), got %+v", conflicts)
	}
}

// newApisixTlsMTLS builds an ApisixTls with an optional mTLS client config.
func newApisixTlsMTLS(name, uid string, hosts []string, secret *corev1.Secret, client *apiv2.ApisixMutualTlsClientConfig) *apiv2.ApisixTls {
	tls := newApisixTls(name, uid, hosts, secret)
	tls.Spec.Client = client
	return tls
}

// TestConflictDetectorDetectsDifferingMTLS verifies an incoming resource with
// the same host and server cert but no mTLS conflicts with an existing resource
// that enforces mTLS.
func TestConflictDetectorDetectsDifferingMTLS(t *testing.T) {
	sharedSecret := newTLSSecret(t, "shared-cert", []string{"api.example.com"})
	gatewayProxy, ingressClass := sharedIngressClassFixtures()

	mtls := &apiv2.ApisixMutualTlsClientConfig{
		CASecret: apiv2.ApisixSecret{Name: "client-ca", Namespace: testNamespace},
		Depth:    1,
	}
	existing := newApisixTlsMTLS("existing", "existing-uid", []string{"api.example.com"}, sharedSecret, mtls)
	incoming := newApisixTlsMTLS("incoming", "incoming-uid", []string{"api.example.com"}, sharedSecret, nil)

	c := buildConflictClient(t, sharedSecret, gatewayProxy, ingressClass, existing)
	detector := NewConflictDetector(c)

	conflicts := detector.DetectConflicts(context.Background(), incoming)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict (differing mTLS config), got %d: %+v", len(conflicts), conflicts)
	}
	if conflicts[0].Host != "api.example.com" {
		t.Fatalf("unexpected host: %s", conflicts[0].Host)
	}
}

// TestConflictDetectorAllowsIdenticalMTLS guards against false positives: same
// host, same server cert, same mTLS client config must not conflict.
func TestConflictDetectorAllowsIdenticalMTLS(t *testing.T) {
	sharedSecret := newTLSSecret(t, "shared-cert", []string{"api.example.com"})
	gatewayProxy, ingressClass := sharedIngressClassFixtures()

	mtls := func() *apiv2.ApisixMutualTlsClientConfig {
		return &apiv2.ApisixMutualTlsClientConfig{
			CASecret: apiv2.ApisixSecret{Name: "client-ca", Namespace: testNamespace},
			Depth:    1,
		}
	}
	existing := newApisixTlsMTLS("existing", "existing-uid", []string{"api.example.com"}, sharedSecret, mtls())
	incoming := newApisixTlsMTLS("incoming", "incoming-uid", []string{"api.example.com"}, sharedSecret, mtls())

	c := buildConflictClient(t, sharedSecret, gatewayProxy, ingressClass, existing)
	detector := NewConflictDetector(c)

	conflicts := detector.DetectConflicts(context.Background(), incoming)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflict (identical mTLS config), got %+v", conflicts)
	}
}

// TestClientConfigHashInjectiveOnCommaRegex guards the skip-regex encoding:
// {"a,b"} and {"a","b"} are different configs and must not hash alike.
func TestClientConfigHashInjectiveOnCommaRegex(t *testing.T) {
	ca := apiv2.ApisixSecret{Name: "client-ca", Namespace: testNamespace}
	single := clientConfigHash(&apiv2.ApisixMutualTlsClientConfig{
		CASecret: ca, SkipMTLSUriRegex: []string{"a,b"},
	})
	split := clientConfigHash(&apiv2.ApisixMutualTlsClientConfig{
		CASecret: ca, SkipMTLSUriRegex: []string{"a", "b"},
	})
	if single == split {
		t.Fatalf("distinct skip-regex configs collided: %s", single)
	}
}

func generateCertificate(t *testing.T, hosts []string) ([]byte, []byte) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatalf("failed to generate serial: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: hosts[0],
		},
		DNSNames:              hosts,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return certPEM, keyPEM
}
