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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"slices"
	"testing"
	"time"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

func genCertWithSANs(t *testing.T, sans []string) []byte {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatalf("failed to generate serial: %v", err)
	}
	cn := "test"
	if len(sans) > 0 {
		cn = sans[0]
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: cn},
		DNSNames:              sans,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create cert: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func hostTypes(hosts ...string) []apiv2.HostType {
	out := make([]apiv2.HostType, 0, len(hosts))
	for _, h := range hosts {
		out = append(out, apiv2.HostType(h))
	}
	return out
}

func TestUncoveredSNIHosts(t *testing.T) {
	cases := []struct {
		name          string
		sans          []string
		hosts         []string
		wantUncovered []string
	}{
		{"exact match", []string{"shop.example.com"}, []string{"shop.example.com"}, nil},
		{"wildcard covers subdomain", []string{"*.example.com"}, []string{"shop.example.com"}, nil},
		{"declared host not in SANs", []string{"internal.corp.local"}, []string{"shop.example.com"}, []string{"shop.example.com"}},
		{"one of many uncovered", []string{"a.example.com"}, []string{"a.example.com", "b.example.com"}, []string{"b.example.com"}},
		{"no DNS SANs is lenient", nil, []string{"shop.example.com"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			uncovered, _ := uncoveredSNIHosts(genCertWithSANs(t, c.sans), hostTypes(c.hosts...))
			if !slices.Equal(uncovered, c.wantUncovered) {
				t.Fatalf("uncoveredSNIHosts() = %v, want %v", uncovered, c.wantUncovered)
			}
		})
	}
}
