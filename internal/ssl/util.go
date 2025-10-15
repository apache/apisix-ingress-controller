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
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

var (
	// ErrUnknownSecretFormat indicates the secret does not contain supported TLS data keys.
	ErrUnknownSecretFormat = errors.New("unknown secret format")
	// ErrMissingCert indicates the secret is missing the certificate part.
	ErrMissingCert = errors.New("missing cert field")
	// ErrMissingKey indicates the secret is missing the private key part when it is required.
	ErrMissingKey = errors.New("missing key field")
	// ErrInvalidPEM is returned when the provided certificate is not valid PEM encoded data.
	ErrInvalidPEM = errors.New("certificate is not valid PEM data")
)

// ExtractKeyPair extracts the certificate and, optionally, the private key from a Secret.
//
// Supported formats:
//  1. APISIX style: data keys `cert` and `key`
//  2. Kubernetes TLS secret: data keys `tls.crt` and `tls.key`
//  3. Kubernetes CA secret: data key `ca.crt` (without private key)
func ExtractKeyPair(secret *corev1.Secret, includePrivateKey bool) ([]byte, []byte, error) {
	if secret == nil {
		return nil, nil, ErrMissingCert
	}

	if cert, ok := secret.Data["cert"]; ok {
		if includePrivateKey {
			key, ok := secret.Data["key"]
			if !ok {
				return nil, nil, ErrMissingKey
			}
			return cert, key, nil
		}
		return cert, nil, nil
	}

	if cert, ok := secret.Data[corev1.TLSCertKey]; ok {
		if includePrivateKey {
			key, ok := secret.Data[corev1.TLSPrivateKeyKey]
			if !ok {
				return nil, nil, ErrMissingKey
			}
			return cert, key, nil
		}
		return cert, nil, nil
	}

	if cert, ok := secret.Data[corev1.ServiceAccountRootCAKey]; ok && !includePrivateKey {
		return cert, nil, nil
	}

	return nil, nil, ErrUnknownSecretFormat
}

// ExtractCertificate extracts only the certificate data from a Secret.
func ExtractCertificate(secret *corev1.Secret) ([]byte, error) {
	cert, _, err := ExtractKeyPair(secret, false)
	return cert, err
}

// ExtractHostsFromCertificate parses the certificate PEM block and returns the DNS names.
func ExtractHostsFromCertificate(certPEM []byte) ([]string, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, ErrInvalidPEM
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	hosts := make([]string, 0, len(cert.DNSNames))
	for _, dnsName := range cert.DNSNames {
		if dnsName != "*" {
			hosts = append(hosts, dnsName)
		}
	}
	return hosts, nil
}

// NormalizeHosts removes duplicate entries
func NormalizeHosts(hosts []string) []string {
	if len(hosts) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(hosts))
	seen := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		candidate := strings.ToLower(strings.TrimSpace(host))
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		normalized = append(normalized, candidate)
	}
	return normalized
}

// CertificateHash returns the SHA-256 hash of the leaf certificate contained in the PEM data.
// The hash is calculated from the DER-encoded bytes so that formatting differences (whitespace,
// line endings, certificate ordering) do not affect the result.
func CertificateHash(certPEM []byte) (string, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return "", ErrInvalidPEM
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:]), nil
}
