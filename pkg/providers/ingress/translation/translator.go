// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package translation

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	listerscorev1 "k8s.io/client-go/listers/core/v1"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	kubev2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	apisixconst "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/const"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisixtranslation "github.com/apache/apisix-ingress-controller/pkg/providers/apisix/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type TranslatorOptions struct {
	Apisix      apisix.APISIX
	ClusterName string

	ServiceLister listerscorev1.ServiceLister
}

type translator struct {
	*TranslatorOptions
	translation.Translator
	ApisixTranslator apisixtranslation.ApisixTranslator
}

type IngressTranslator interface {
	// TranslateIngress composes a couple of APISIX Routes and upstreams according
	// to the given Ingress resource.
	// For old objects, you cannot use TranslateIngress to build. Because it needs to parse the latest service, which will cause data inconsistency.
	TranslateIngress(ing kube.Ingress) (*translation.TranslateContext, error)
	// TranslateOldIngress get route objects from cache
	// Build upstream and plugin_config through route
	TranslateOldIngress(kube.Ingress) (*translation.TranslateContext, error)
	// TranslateSSLV2 translate networkingv1.IngressTLS to APISIX SSL
	TranslateIngressTLS(namespace, ingName, secretName string, hosts []string) (*apisixv1.Ssl, error)
}

func NewIngressTranslator(opts *TranslatorOptions,
	commonTranslator translation.Translator, apisixTranslator apisixtranslation.ApisixTranslator) IngressTranslator {
	t := &translator{
		TranslatorOptions: opts,
		Translator:        commonTranslator,
		ApisixTranslator:  apisixTranslator,
	}

	return t
}

func (t *translator) TranslateIngressTLS(namespace, ingName, secretName string, hosts []string) (*apisixv1.Ssl, error) {
	apisixTls := kubev2.ApisixTls{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApisixTls",
			APIVersion: "apisix.apache.org/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", ingName, "tls"),
			Namespace: namespace,
		},
		Spec: &kubev2.ApisixTlsSpec{
			Secret: kubev2.ApisixSecret{
				Name:      secretName,
				Namespace: namespace,
			},
		},
	}
	for _, host := range hosts {
		apisixTls.Spec.Hosts = append(apisixTls.Spec.Hosts, kubev2.HostType(host))
	}

	return t.ApisixTranslator.TranslateSSLV2(&apisixTls)
}

func (t *translator) TranslateIngress(ing kube.Ingress) (*translation.TranslateContext, error) {
	switch ing.GroupVersion() {
	case kube.IngressV1:
		return t.translateIngressV1(ing.V1())
	case kube.IngressV1beta1:
		return t.translateIngressV1beta1(ing.V1beta1())
	case kube.IngressExtensionsV1beta1:
		return t.translateIngressExtensionsV1beta1(ing.ExtensionsV1beta1())
	default:
		return nil, fmt.Errorf("translator: source group version not supported: %s", ing.GroupVersion())
	}
}

const (
	_regexPriority = 100
)

func (t *translator) translateIngressV1(ing *networkingv1.Ingress) (*translation.TranslateContext, error) {
	ctx := translation.DefaultEmptyTranslateContext()
	ingress := t.TranslateAnnotations(ing.Annotations)

	// add https
	for _, tls := range ing.Spec.TLS {
		ssl, err := t.TranslateIngressTLS(ing.Namespace, ing.Name, tls.SecretName, tls.Hosts)
		if err != nil {
			log.Errorw("failed to translate ingress tls to apisix tls",
				zap.Error(err),
				zap.Any("ingress", ing),
			)
			return nil, err
		}
		ctx.AddSSL(ssl)
	}
	for _, rule := range ing.Spec.Rules {
		for _, pathRule := range rule.HTTP.Paths {
			var (
				ups *apisixv1.Upstream
				err error
			)
			if pathRule.Backend.Service != nil {
				var port intstr.IntOrString
				if pathRule.Backend.Service.Port.Name != "" {
					port = intstr.FromString(pathRule.Backend.Service.Port.Name)
				} else {
					port = intstr.FromInt(int(pathRule.Backend.Service.Port.Number))
				}
				ups, err = t.TranslateUpstream(ing.Namespace, pathRule.Backend.Service.Name, "", "", port)
				if err != nil {
					log.Errorw("failed to translate ingress backend to upstream",
						zap.Error(err),
						zap.Any("ingress", ing),
					)
					return nil, err
				}
				ctx.AddUpstream(ups)
			}
			uris := []string{pathRule.Path}
			var nginxVars []kubev2.ApisixRouteHTTPMatchExpr
			if pathRule.PathType != nil {
				if *pathRule.PathType == networkingv1.PathTypePrefix {
					// As per the specification of Ingress path matching rule:
					// if the last element of the path is a substring of the
					// last element in request path, it is not a match, e.g. /foo/bar
					// matches /foo/bar/baz, but does not match /foo/barbaz.
					// While in APISIX, /foo/bar matches both /foo/bar/baz and
					// /foo/barbaz.
					// In order to be conformant with Ingress specification, here
					// we create two paths here, the first is the path itself
					// (exact match), the other is path + "/*" (prefix match).
					prefix := pathRule.Path
					if strings.HasSuffix(prefix, "/") {
						prefix += "*"
					} else {
						prefix += "/*"
					}
					uris = append(uris, prefix)
				} else if *pathRule.PathType == networkingv1.PathTypeImplementationSpecific && ingress.UseRegex {
					nginxVars = append(nginxVars, kubev2.ApisixRouteHTTPMatchExpr{
						Subject: kubev2.ApisixRouteHTTPMatchExprSubject{
							Scope: apisixconst.ScopePath,
						},
						Op:    apisixconst.OpRegexMatch,
						Value: &pathRule.Path,
					})
					uris = []string{"/*"}
				}
			}
			route := apisixv1.NewDefaultRoute()
			route.Name = composeIngressRouteName(ing.Namespace, ing.Name, rule.Host, pathRule.Path)
			route.ID = id.GenID(route.Name)
			route.Host = rule.Host
			route.Uris = uris
			route.EnableWebsocket = ingress.EnableWebSocket
			if len(nginxVars) > 0 {
				routeVars, err := t.ApisixTranslator.TranslateRouteMatchExprs(nginxVars)
				if err != nil {
					return nil, err
				}
				route.Vars = routeVars
				route.Priority = _regexPriority
			}
			if len(ingress.Plugins) > 0 {
				route.Plugins = *(ingress.Plugins.DeepCopy())
			}

			if ingress.PluginConfigName != "" {
				route.PluginConfigId = id.GenID(apisixv1.ComposePluginConfigName(ing.Namespace, ingress.PluginConfigName))
			}
			if ups != nil {
				route.UpstreamId = ups.ID
			}
			ctx.AddRoute(route)
		}
	}
	return ctx, nil
}

func (t *translator) translateIngressV1beta1(ing *networkingv1beta1.Ingress) (*translation.TranslateContext, error) {
	ctx := translation.DefaultEmptyTranslateContext()
	ingress := t.TranslateAnnotations(ing.Annotations)

	// add https
	for _, tls := range ing.Spec.TLS {
		ssl, err := t.TranslateIngressTLS(ing.Namespace, ing.Name, tls.SecretName, tls.Hosts)
		if err != nil {
			log.Errorw("failed to translate ingress tls to apisix tls",
				zap.Error(err),
				zap.Any("ingress", ing),
			)
			return nil, err
		}
		ctx.AddSSL(ssl)
	}
	for _, rule := range ing.Spec.Rules {
		for _, pathRule := range rule.HTTP.Paths {
			var (
				ups *apisixv1.Upstream
				err error
			)
			if pathRule.Backend.ServiceName != "" {
				ups, err = t.TranslateUpstream(ing.Namespace, pathRule.Backend.ServiceName, "", "", pathRule.Backend.ServicePort)
				if err != nil {
					log.Errorw("failed to translate ingress backend to upstream",
						zap.Error(err),
						zap.Any("ingress", ing),
					)
					return nil, err
				}
				ctx.AddUpstream(ups)
			}
			uris := []string{pathRule.Path}
			var nginxVars []kubev2.ApisixRouteHTTPMatchExpr
			if pathRule.PathType != nil {
				if *pathRule.PathType == networkingv1beta1.PathTypePrefix {
					// As per the specification of Ingress path matching rule:
					// if the last element of the path is a substring of the
					// last element in request path, it is not a match, e.g. /foo/bar
					// matches /foo/bar/baz, but does not match /foo/barbaz.
					// While in APISIX, /foo/bar matches both /foo/bar/baz and
					// /foo/barbaz.
					// In order to be conformant with Ingress specification, here
					// we create two paths here, the first is the path itself
					// (exact match), the other is path + "/*" (prefix match).
					prefix := pathRule.Path
					if strings.HasSuffix(prefix, "/") {
						prefix += "*"
					} else {
						prefix += "/*"
					}
					uris = append(uris, prefix)
				} else if *pathRule.PathType == networkingv1beta1.PathTypeImplementationSpecific && ingress.UseRegex {
					nginxVars = append(nginxVars, kubev2.ApisixRouteHTTPMatchExpr{
						Subject: kubev2.ApisixRouteHTTPMatchExprSubject{
							Scope: apisixconst.ScopePath,
						},
						Op:    apisixconst.OpRegexMatch,
						Value: &pathRule.Path,
					})
					uris = []string{"/*"}
				}
			}
			route := apisixv1.NewDefaultRoute()
			route.Name = composeIngressRouteName(ing.Namespace, ing.Name, rule.Host, pathRule.Path)
			route.ID = id.GenID(route.Name)
			route.Host = rule.Host
			route.Uris = uris
			route.EnableWebsocket = ingress.EnableWebSocket
			if len(nginxVars) > 0 {
				routeVars, err := t.ApisixTranslator.TranslateRouteMatchExprs(nginxVars)
				if err != nil {
					return nil, err
				}
				route.Vars = routeVars
				route.Priority = _regexPriority
			}
			if len(ingress.Plugins) > 0 {
				route.Plugins = *(ingress.Plugins.DeepCopy())
			}

			if ingress.PluginConfigName != "" {
				route.PluginConfigId = id.GenID(apisixv1.ComposePluginConfigName(ing.Namespace, ingress.PluginConfigName))
			}
			if ups != nil {
				route.UpstreamId = ups.ID
			}
			ctx.AddRoute(route)
		}
	}
	return ctx, nil
}

func (t *translator) translateIngressExtensionsV1beta1(ing *extensionsv1beta1.Ingress) (*translation.TranslateContext, error) {
	ctx := translation.DefaultEmptyTranslateContext()
	ingress := t.TranslateAnnotations(ing.Annotations)

	for _, rule := range ing.Spec.Rules {
		for _, pathRule := range rule.HTTP.Paths {
			var (
				ups *apisixv1.Upstream
				err error
			)
			if pathRule.Backend.ServiceName != "" {
				// Structure here is same to ingress.extensions/v1beta1, so just use this method.
				ups, err = t.TranslateUpstream(ing.Namespace, pathRule.Backend.ServiceName, "", "", pathRule.Backend.ServicePort)
				if err != nil {
					log.Errorw("failed to translate ingress backend to upstream",
						zap.Error(err),
						zap.Any("ingress", ing),
					)
					return nil, err
				}
				ctx.AddUpstream(ups)
			}
			uris := []string{pathRule.Path}
			var nginxVars []kubev2.ApisixRouteHTTPMatchExpr
			if pathRule.PathType != nil {
				if *pathRule.PathType == extensionsv1beta1.PathTypePrefix {
					// As per the specification of Ingress path matching rule:
					// if the last element of the path is a substring of the
					// last element in request path, it is not a match, e.g. /foo/bar
					// matches /foo/bar/baz, but does not match /foo/barbaz.
					// While in APISIX, /foo/bar matches both /foo/bar/baz and
					// /foo/barbaz.
					// In order to be conformant with Ingress specification, here
					// we create two paths here, the first is the path itself
					// (exact match), the other is path + "/*" (prefix match).
					prefix := pathRule.Path
					if strings.HasSuffix(prefix, "/") {
						prefix += "*"
					} else {
						prefix += "/*"
					}
					uris = append(uris, prefix)
				} else if *pathRule.PathType == extensionsv1beta1.PathTypeImplementationSpecific && ingress.UseRegex {
					nginxVars = append(nginxVars, kubev2.ApisixRouteHTTPMatchExpr{
						Subject: kubev2.ApisixRouteHTTPMatchExprSubject{
							Scope: apisixconst.ScopePath,
						},
						Op:    apisixconst.OpRegexMatch,
						Value: &pathRule.Path,
					})
					uris = []string{"/*"}
				}
			}
			route := apisixv1.NewDefaultRoute()
			route.Name = composeIngressRouteName(ing.Namespace, ing.Name, rule.Host, pathRule.Path)
			route.ID = id.GenID(route.Name)
			route.Host = rule.Host
			route.Uris = uris
			route.EnableWebsocket = ingress.EnableWebSocket
			if len(nginxVars) > 0 {
				routeVars, err := t.ApisixTranslator.TranslateRouteMatchExprs(nginxVars)
				if err != nil {
					return nil, err
				}
				route.Vars = routeVars
				route.Priority = _regexPriority
			}
			if len(ingress.Plugins) > 0 {
				route.Plugins = *(ingress.Plugins.DeepCopy())
			}

			if ingress.PluginConfigName != "" {
				route.PluginConfigId = id.GenID(apisixv1.ComposePluginConfigName(ing.Namespace, ingress.PluginConfigName))
			}

			if ups != nil {
				route.UpstreamId = ups.ID
			}
			ctx.AddRoute(route)
		}
	}
	return ctx, nil
}

func (t *translator) TranslateOldIngress(ing kube.Ingress) (*translation.TranslateContext, error) {
	switch ing.GroupVersion() {
	case kube.IngressV1:
		return t.translateOldIngressV1(ing.V1())
	case kube.IngressV1beta1:
		return t.translateOldIngressV1beta1(ing.V1beta1())
	case kube.IngressExtensionsV1beta1:
		return t.translateOldIngressExtensionsv1beta1(ing.ExtensionsV1beta1())
	default:
		return nil, fmt.Errorf("translator: source group version not supported: %s", ing.GroupVersion())
	}
}

func (t *translator) translateOldIngressV1(ing *networkingv1.Ingress) (*translation.TranslateContext, error) {
	oldCtx := translation.DefaultEmptyTranslateContext()

	for _, tls := range ing.Spec.TLS {
		ssl, err := t.TranslateIngressTLS(ing.Namespace, ing.Name, tls.SecretName, tls.Hosts)
		if err != nil {
			log.Debugw("failed to translate ingress tls to apisix tls",
				zap.Error(err),
				zap.Any("ingress", ing),
			)
			continue
		}
		oldCtx.AddSSL(ssl)
	}
	for _, rule := range ing.Spec.Rules {
		for _, pathRule := range rule.HTTP.Paths {
			name := composeIngressRouteName(ing.Namespace, ing.Name, rule.Host, pathRule.Path)
			r, err := t.Apisix.Cluster(t.ClusterName).Route().Get(context.Background(), name)
			if err != nil {
				continue
			}
			if r.UpstreamId != "" {
				ups := apisixv1.NewDefaultUpstream()
				ups.ID = r.UpstreamId
				oldCtx.AddUpstream(ups)
			}
			if r.PluginConfigId != "" {
				pc := apisixv1.NewDefaultPluginConfig()
				pc.ID = r.PluginConfigId
				oldCtx.AddPluginConfig(pc)
			}
			oldCtx.AddRoute(r)
		}
	}
	return oldCtx, nil
}

func (t *translator) translateOldIngressV1beta1(ing *networkingv1beta1.Ingress) (*translation.TranslateContext, error) {
	oldCtx := translation.DefaultEmptyTranslateContext()

	for _, tls := range ing.Spec.TLS {
		ssl, err := t.TranslateIngressTLS(ing.Namespace, ing.Name, tls.SecretName, tls.Hosts)
		if err != nil {
			continue
		}
		oldCtx.AddSSL(ssl)
	}
	for _, rule := range ing.Spec.Rules {
		for _, pathRule := range rule.HTTP.Paths {
			name := composeIngressRouteName(ing.Namespace, ing.Name, rule.Host, pathRule.Path)
			r, err := t.Apisix.Cluster(t.ClusterName).Route().Get(context.Background(), name)
			if err != nil {
				continue
			}
			if r.UpstreamId != "" {
				ups := apisixv1.NewDefaultUpstream()
				ups.ID = r.UpstreamId
				oldCtx.AddUpstream(ups)
			}
			if r.PluginConfigId != "" {
				pc := apisixv1.NewDefaultPluginConfig()
				pc.ID = r.PluginConfigId
				oldCtx.AddPluginConfig(pc)
			}
			oldCtx.AddRoute(r)
		}
	}
	return oldCtx, nil
}

func (t *translator) translateOldIngressExtensionsv1beta1(ing *extensionsv1beta1.Ingress) (*translation.TranslateContext, error) {
	oldCtx := translation.DefaultEmptyTranslateContext()

	for _, tls := range ing.Spec.TLS {
		ssl, err := t.TranslateIngressTLS(ing.Namespace, ing.Name, tls.SecretName, tls.Hosts)
		if err != nil {
			continue
		}
		oldCtx.AddSSL(ssl)
	}
	for _, rule := range ing.Spec.Rules {
		for _, pathRule := range rule.HTTP.Paths {
			name := composeIngressRouteName(ing.Namespace, ing.Name, rule.Host, pathRule.Path)
			r, err := t.Apisix.Cluster(t.ClusterName).Route().Get(context.Background(), name)
			if err != nil {
				continue
			}
			if r.UpstreamId != "" {
				ups := apisixv1.NewDefaultUpstream()
				ups.ID = r.UpstreamId
				oldCtx.AddUpstream(ups)
			}
			if r.PluginConfigId != "" {
				pc := apisixv1.NewDefaultPluginConfig()
				pc.ID = r.PluginConfigId
				oldCtx.AddPluginConfig(pc)
			}
			oldCtx.AddRoute(r)
		}
	}
	return oldCtx, nil
}

// In the past, we used host + path directly to form its route name for readability,
// but this method can cause problems in some scenarios.
// For example, the generated name is too long.
// The current APISIX limit its maximum length to 100.
// ref: https://github.com/apache/apisix-ingress-controller/issues/781
// We will construct the following structure for easy reading and debugging.
// ing_namespace_ingressName_id
func composeIngressRouteName(namespace, name, host, path string) string {
	pID := id.GenID(host + path)
	p := make([]byte, 0, len(namespace)+len(name)+len("ing")+len(pID)+3)
	buf := bytes.NewBuffer(p)

	buf.WriteString("ing")
	buf.WriteByte('_')
	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)
	buf.WriteByte('_')
	buf.WriteString(pID)

	return buf.String()
}
