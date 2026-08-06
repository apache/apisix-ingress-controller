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

	"k8s.io/apimachinery/pkg/types"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/id"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	sslutils "github.com/apache/apisix-ingress-controller/internal/ssl"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func (t *Translator) TranslateApisixTls(tctx *provider.TranslateContext, tls *apiv2.ApisixTls) (*TranslateResult, error) {
	result := &TranslateResult{}

	// Get the secret from the context
	secretKey := types.NamespacedName{
		Namespace: tls.Spec.Secret.Namespace,
		Name:      tls.Spec.Secret.Name,
	}
	secret, ok := tctx.Secrets[secretKey]
	if !ok || secret == nil {
		return nil, fmt.Errorf("secret %s not found", secretKey.String())
	}

	// Extract cert and key from secret
	cert, key, err := sslutils.ExtractKeyPair(secret, true)
	if err != nil {
		return nil, err
	}

	// APISIX serves the cert regardless of SAN, so this is advisory: warn when a
	// declared host isn't covered by the cert SANs (clients may reject it).
	if uncovered, sans := uncoveredSNIHosts(cert, tls.Spec.Hosts); len(uncovered) > 0 {
		t.Log.Info("ApisixTls certificate does not cover all declared SNI hosts",
			"namespace", tls.Namespace, "name", tls.Name,
			"uncoveredHosts", uncovered, "certificateSANs", sans)
	}

	// Convert hosts to strings
	snis := make([]string, len(tls.Spec.Hosts))
	for i, host := range tls.Spec.Hosts {
		snis[i] = string(host)
	}

	// Create SSL object
	ssl := &adctypes.SSL{
		Metadata: adctypes.Metadata{
			ID:     id.GenID(adctypes.ComposeSSLName(internaltypes.KindApisixTls, tls.Namespace, tls.Name)),
			Labels: label.GenLabel(tls),
		},
		Certificates: []adctypes.Certificate{
			{
				Certificate: string(cert),
				Key:         string(key),
			},
		},
		Snis: snis,
	}

	// Handle mutual TLS client configuration if present
	if tls.Spec.Client != nil {
		caSecretKey := types.NamespacedName{
			Namespace: tls.Spec.Client.CASecret.Namespace,
			Name:      tls.Spec.Client.CASecret.Name,
		}
		caSecret, ok := tctx.Secrets[caSecretKey]
		if !ok || caSecret == nil {
			return nil, fmt.Errorf("client CA secret %s not found", caSecretKey.String())
		}

		ca, _, err := sslutils.ExtractKeyPair(caSecret, false)
		if err != nil {
			return nil, err
		}
		depth := int64(tls.Spec.Client.Depth)
		ssl.Client = &adctypes.ClientClass{
			CA:               string(ca),
			Depth:            &depth,
			SkipMtlsURIRegex: tls.Spec.Client.SkipMTLSUriRegex,
		}
	}

	result.SSL = append(result.SSL, ssl)
	return result, nil
}

// uncoveredSNIHosts returns the declared hosts not covered by any DNS SAN in the
// certificate (wildcard-aware), plus the cert SANs for logging. Returns nothing
// when the cert declares no DNS SANs or can't be parsed, since coverage can't be
// judged there.
func uncoveredSNIHosts(cert []byte, hosts []apiv2.HostType) (uncovered []string, sans []string) {
	sans, err := sslutils.ExtractHostsFromCertificate(cert)
	if err != nil || len(sans) == 0 {
		return nil, nil
	}
	for _, h := range hosts {
		host := string(h)
		covered := false
		for _, san := range sans {
			if sslutils.HostCoveredBy(host, san) {
				covered = true
				break
			}
		}
		if !covered {
			uncovered = append(uncovered, host)
		}
	}
	return uncovered, sans
}
