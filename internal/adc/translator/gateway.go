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
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/id"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func (t *Translator) TranslateGateway(tctx *provider.TranslateContext, obj *gatewayv1.Gateway) (*TranslateResult, error) {
	result := &TranslateResult{}
	for _, listener := range obj.Spec.Listeners {
		if listener.TLS != nil {
			tctx.GatewayTLSConfig = append(tctx.GatewayTLSConfig, *listener.TLS)
			ssl, err := t.translateSecret(tctx, listener, obj)
			if err != nil {
				return nil, fmt.Errorf("failed to translate secret: %w", err)
			}
			result.SSL = append(result.SSL, ssl...)
		}
	}
	// No longer need to merge SSLs since each SNI now has a unique ID based on namespace+secretName+sni

	rk := utils.NamespacedNameKind(obj)
	gatewayProxy, ok := tctx.GatewayProxies[rk]
	if !ok {
		t.Log.V(1).Info("no GatewayProxy found for Gateway", "gateway", obj.Name)
		return result, nil
	}

	globalRules := make(adctypes.GlobalRule)
	pluginMetadata := make(adctypes.PluginMetadata)
	// apply plugins from GatewayProxy to global rules
	t.fillPluginsFromGatewayProxy(globalRules, &gatewayProxy)
	t.fillPluginMetadataFromGatewayProxy(pluginMetadata, &gatewayProxy)
	result.GlobalRules = globalRules
	result.PluginMetadata = pluginMetadata

	return result, nil
}

func (t *Translator) translateSecret(tctx *provider.TranslateContext, listener gatewayv1.Listener, obj *gatewayv1.Gateway) ([]*adctypes.SSL, error) {
	if tctx.Secrets == nil {
		return nil, nil
	}
	if listener.TLS.CertificateRefs == nil {
		return nil, fmt.Errorf("no certificateRefs found in listener %s", listener.Name)
	}
	sslObjs := make([]*adctypes.SSL, 0)
	switch *listener.TLS.Mode {
	case gatewayv1.TLSModeTerminate:
		for _, ref := range listener.TLS.CertificateRefs {
			ns := obj.GetNamespace()
			if ref.Namespace != nil {
				ns = string(*ref.Namespace)
			}
			if listener.TLS.CertificateRefs[0].Kind != nil && *listener.TLS.CertificateRefs[0].Kind == internaltypes.KindSecret {
				name := listener.TLS.CertificateRefs[0].Name
				secretNN := types.NamespacedName{Namespace: ns, Name: string(ref.Name)}
				secret := tctx.Secrets[secretNN]
				if secret == nil {
					continue
				}
				if secret.Data == nil {
					t.Log.Error(errors.New("secret data is nil"), "failed to get secret data", "secret", secretNN)
					return nil, fmt.Errorf("no secret data found for %s/%s", ns, name)
				}
				cert, key, err := extractKeyPair(secret, true)
				if err != nil {
					t.Log.Error(err, "extract key pair", "secret", secretNN)
					return nil, err
				}

				// Collect SNIs from listener hostname or certificate
				var snis []string
				if listener.Hostname != nil && *listener.Hostname != "" {
					snis = []string{string(*listener.Hostname)}
				} else {
					hosts, err := extractHost(cert)
					if err != nil {
						return nil, err
					}
					if len(hosts) == 0 {
						t.Log.Info("no valid hostname found in certificate", "secret", secretNN.String())
						continue
					}
					snis = hosts
				}

				// Create one SSL object per SNI to avoid conflicts when the same certificate
				// is used across multiple Gateway listeners with different SNIs.
				// Using namespace + secretName + sni as the ID ensures:
				// 1. Different Gateways with same cert+sni will share the same SSL (expected behavior)
				// 2. Same Gateway with same cert but different SNIs will have separate SSL objects
				// 3. No overwrites in MemDB due to ID collision
				for _, sni := range snis {
					sslObj := &adctypes.SSL{
						Metadata: adctypes.Metadata{
							Labels: label.GenLabel(obj),
							// Generate unique ID based on namespace, secret name, and SNI
							// This allows the same wildcard certificate to be used for multiple SNIs
							ID: id.GenID(secretNN.Namespace + "_" + secretNN.Name + "_" + sni),
						},
						Certificates: []adctypes.Certificate{
							{
								Certificate: string(cert),
								Key:         string(key),
							},
						},
						Snis: []string{sni},
					}
					t.Log.V(1).Info("generated ssl id", "ssl id", sslObj.ID, "secret", secretNN.String(), "sni", sni)
					sslObjs = append(sslObjs, sslObj)
				}
			}

		}
	// Only supported on TLSRoute. The certificateRefs field is ignored in this mode.
	case gatewayv1.TLSModePassthrough:
		return sslObjs, nil
	default:
		return nil, fmt.Errorf("unknown TLS mode %s", *listener.TLS.Mode)
	}

	return sslObjs, nil
}

func extractHost(cert []byte) ([]string, error) {
	block, _ := pem.Decode(cert)
	if block == nil {
		return nil, errors.New("parse certificate: not in PEM format")
	}
	der, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "parse certificate")
	}
	hosts := make([]string, 0, len(der.DNSNames))
	for _, dnsName := range der.DNSNames {
		if dnsName != "*" {
			hosts = append(hosts, dnsName)
		}
	}
	return hosts, nil
}

func extractKeyPair(s *corev1.Secret, hasPrivateKey bool) ([]byte, []byte, error) {
	if _, ok := s.Data["cert"]; ok {
		return extractApisixSecretKeyPair(s, hasPrivateKey)
	} else if _, ok := s.Data[corev1.TLSCertKey]; ok {
		return extractKubeSecretKeyPair(s, hasPrivateKey)
	} else if ca, ok := s.Data[corev1.ServiceAccountRootCAKey]; ok && !hasPrivateKey {
		return ca, nil, nil
	} else {
		return nil, nil, errors.New("unknown secret format")
	}
}

func extractApisixSecretKeyPair(s *corev1.Secret, hasPrivateKey bool) (cert []byte, key []byte, err error) {
	var ok bool
	cert, ok = s.Data["cert"]
	if !ok {
		return nil, nil, errors.New("missing cert field")
	}

	if hasPrivateKey {
		key, ok = s.Data["key"]
		if !ok {
			return nil, nil, errors.New("missing key field")
		}
	}
	return
}

func extractKubeSecretKeyPair(s *corev1.Secret, hasPrivateKey bool) (cert []byte, key []byte, err error) {
	var ok bool
	cert, ok = s.Data[corev1.TLSCertKey]
	if !ok {
		return nil, nil, errors.New("missing cert field")
	}

	if hasPrivateKey {
		key, ok = s.Data[corev1.TLSPrivateKeyKey]
		if !ok {
			return nil, nil, errors.New("missing key field")
		}
	}
	return
}

// fillPluginsFromGatewayProxy fill plugins from GatewayProxy to given plugins
func (t *Translator) fillPluginsFromGatewayProxy(plugins adctypes.GlobalRule, gatewayProxy *v1alpha1.GatewayProxy) {
	if gatewayProxy == nil {
		return
	}

	for _, plugin := range gatewayProxy.Spec.Plugins {
		// only apply enabled plugins
		if !plugin.Enabled {
			continue
		}

		pluginName := plugin.Name
		pluginConfig := map[string]any{}
		if len(plugin.Config.Raw) > 0 {
			if err := json.Unmarshal(plugin.Config.Raw, &pluginConfig); err != nil {
				t.Log.Error(err, "gateway proxy plugin config unmarshal failed", "plugin", pluginName)
				continue
			}
		}
		plugins[pluginName] = pluginConfig
	}
	t.Log.V(1).Info("fill plugins for gateway proxy", "plugins", plugins)
}

func (t *Translator) fillPluginMetadataFromGatewayProxy(pluginMetadata adctypes.PluginMetadata, gatewayProxy *v1alpha1.GatewayProxy) {
	if gatewayProxy == nil {
		return
	}
	for pluginName, plugin := range gatewayProxy.Spec.PluginMetadata {
		var pluginConfig map[string]any
		if err := json.Unmarshal(plugin.Raw, &pluginConfig); err != nil {
			t.Log.Error(err, "gateway proxy plugin_metadata unmarshal failed", "plugin", pluginName, "config", string(plugin.Raw))
			continue
		}
		t.Log.V(1).Info("fill plugin_metadata for gateway proxy", "plugin", pluginName, "config", pluginConfig)
		pluginMetadata[pluginName] = pluginConfig
	}
}
