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
	"fmt"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/id"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	sslutils "github.com/apache/apisix-ingress-controller/internal/ssl"
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
	sslObjs := make([]*adctypes.SSL, 0)

	// TLS is terminated at the gateway unless the listener explicitly asks for
	// passthrough, in which case certificateRefs are not needed (and are ignored
	// per the Gateway API spec).
	mode := gatewayv1.TLSModeTerminate
	if listener.TLS.Mode != nil {
		mode = *listener.TLS.Mode
	}
	if mode == gatewayv1.TLSModePassthrough {
		return sslObjs, nil
	}

	if listener.TLS.CertificateRefs == nil {
		return nil, fmt.Errorf("no certificateRefs found in listener %s", listener.Name)
	}
	switch mode {
	case gatewayv1.TLSModeTerminate:
		// frontendValidation configures downstream mTLS: clients must present a
		// certificate signed by one of the referenced CAs during the TLS handshake.
		client, err := t.translateFrontendValidation(tctx, listener, obj)
		if err != nil {
			return nil, err
		}
		for refIndex, ref := range listener.TLS.CertificateRefs {
			ns := obj.GetNamespace()
			if ref.Namespace != nil {
				ns = string(*ref.Namespace)
			}
			if listener.TLS.CertificateRefs[0].Kind != nil && *listener.TLS.CertificateRefs[0].Kind == internaltypes.KindSecret {
				sslObj := &adctypes.SSL{
					Snis: []string{},
				}
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
				cert, key, err := sslutils.ExtractKeyPair(secret, true)
				if err != nil {
					t.Log.Error(err, "extract key pair", "secret", secretNN)
					return nil, err
				}
				sslObj.Certificates = append(sslObj.Certificates, adctypes.Certificate{
					Certificate: string(cert),
					Key:         string(key),
				})
				// we doesn't allow wildcard hostname
				if listener.Hostname != nil && *listener.Hostname != "" {
					sslObj.Snis = append(sslObj.Snis, string(*listener.Hostname))
				} else {
					hosts, err := sslutils.ExtractHostsFromCertificate(cert)
					if err != nil {
						return nil, err
					}
					if len(hosts) == 0 {
						t.Log.Info("no valid hostname found in certificate", "secret", secretNN.String())
						continue
					}
					sslObj.Snis = append(sslObj.Snis, hosts...)
				}
				sslObj.Client = client
				sslObj.ID = id.GenID(fmt.Sprintf("%s_%s_%d", adctypes.ComposeSSLName(internaltypes.KindGateway, obj.Namespace, obj.Name), listener.Name, refIndex))
				t.Log.V(1).Info("generated ssl id", "ssl id", sslObj.ID, "secret", secretNN.String())
				sslObj.Labels = label.GenLabel(obj)
				sslObjs = append(sslObjs, sslObj)
			}

		}
	default:
		return nil, fmt.Errorf("unknown TLS mode %s", mode)
	}

	return sslObjs, nil
}

// frontendTLSValidation resolves the Gateway-level frontend TLS client-cert
// validation that applies to the given HTTPS listener. In Gateway API v1.6,
// frontendValidation moved from the per-listener TLS config to the Gateway-level
// spec.tls.frontend: Default applies to all HTTPS listeners, and a PerPort entry
// overrides it for listeners on the matching port.
func frontendTLSValidation(obj *gatewayv1.Gateway, listener gatewayv1.Listener) *gatewayv1.FrontendTLSValidation {
	// In Gateway API v1.6 spec.tls.frontend applies only to HTTPS listeners, so
	// never resolve downstream mTLS for any other protocol (including TLS).
	if listener.Protocol != gatewayv1.HTTPSProtocolType {
		return nil
	}
	if obj.Spec.TLS == nil || obj.Spec.TLS.Frontend == nil {
		return nil
	}
	frontend := obj.Spec.TLS.Frontend
	for i := range frontend.PerPort {
		if frontend.PerPort[i].Port == listener.Port {
			return frontend.PerPort[i].TLS.Validation
		}
	}
	return frontend.Default.Validation
}

// translateFrontendValidation builds the downstream mTLS client configuration from the
// Gateway's frontendValidation that applies to the listener. The referenced CA
// certificates (ConfigMap, key `ca.crt`) are bundled into a single trust anchor used
// to validate client certificates.
func (t *Translator) translateFrontendValidation(tctx *provider.TranslateContext, listener gatewayv1.Listener, obj *gatewayv1.Gateway) (*adctypes.ClientClass, error) {
	validation := frontendTLSValidation(obj, listener)
	if validation == nil || len(validation.CACertificateRefs) == 0 {
		return nil, nil
	}
	// APISIX can only enforce strict client-certificate verification (setting ca
	// turns on ssl_verify_client). AllowInsecureFallback ("accept even if the
	// client cert is missing or fails verification") cannot be expressed, so
	// reject it instead of silently programming the opposite, strict behaviour.
	if validation.Mode == gatewayv1.AllowInsecureFallback {
		return nil, fmt.Errorf("unsupported frontendValidation mode %q in listener %s: APISIX cannot make client certificate verification optional", validation.Mode, listener.Name)
	}

	cas := make([]string, 0, len(validation.CACertificateRefs))
	for _, ref := range validation.CACertificateRefs {
		// caCertificateRefs must be in the core API group. ConfigMap is the
		// Gateway API Core support; Secret is an implementation-specific extension.
		if ref.Group != "" && string(ref.Group) != corev1.GroupName {
			return nil, fmt.Errorf("unsupported frontendValidation caCertificateRef group %q in listener %s, only the core group is supported", ref.Group, listener.Name)
		}
		ns := obj.GetNamespace()
		if ref.Namespace != nil {
			ns = string(*ref.Namespace)
		}
		nn := types.NamespacedName{Namespace: ns, Name: string(ref.Name)}

		kind := internaltypes.KindConfigMap
		if ref.Kind != "" {
			kind = string(ref.Kind)
		}
		var (
			ca  []byte
			err error
		)
		switch kind {
		case internaltypes.KindConfigMap:
			cm := tctx.ConfigMaps[nn]
			if cm == nil {
				return nil, fmt.Errorf("frontendValidation CA ConfigMap %s not found", nn.String())
			}
			if ca, err = sslutils.ExtractCAFromConfigMap(cm); err != nil {
				t.Log.Error(err, "failed to extract CA from configmap", "configmap", nn.String())
				return nil, fmt.Errorf("failed to extract CA from ConfigMap %s: %w", nn.String(), err)
			}
		case internaltypes.KindSecret:
			secret := tctx.Secrets[nn]
			if secret == nil {
				return nil, fmt.Errorf("frontendValidation CA Secret %s not found", nn.String())
			}
			if ca, err = sslutils.ExtractCAFromSecret(secret); err != nil {
				t.Log.Error(err, "failed to extract CA from secret", "secret", nn.String())
				return nil, fmt.Errorf("failed to extract CA from Secret %s: %w", nn.String(), err)
			}
		default:
			return nil, fmt.Errorf("unsupported frontendValidation caCertificateRef kind %q in listener %s, only ConfigMap and Secret are supported", ref.Kind, listener.Name)
		}
		cas = append(cas, strings.TrimSpace(string(ca)))
	}

	return &adctypes.ClientClass{
		CA: strings.Join(cas, "\n"),
	}, nil
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
