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
package translation

import (
	"errors"

	v1 "k8s.io/api/core/v1"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

var (
	// ErrUnknownSecretFormat means the secret doesn't contain required fields
	ErrUnknownSecretFormat = errors.New("unknown secret format")
	// ErrEmptyCert means the cert field in Kubernetes Secret is not found.
	ErrEmptyCert = errors.New("missing cert field")
	// ErrEmptyPrivKey means the key field in Kubernetes Secret is not found.
	ErrEmptyPrivKey = errors.New("missing key field")
)

func (t *translator) TranslateSSLV2Beta3(tls *configv2beta3.ApisixTls) (*apisixv1.Ssl, error) {
	s, err := t.SecretLister.Secrets(tls.Spec.Secret.Namespace).Get(tls.Spec.Secret.Name)
	if err != nil {
		return nil, err
	}
	cert, key, err := t.ExtractKeyPair(s, true)
	if err != nil {
		return nil, err
	}

	var snis []string
	for _, host := range tls.Spec.Hosts {
		snis = append(snis, string(host))
	}
	ssl := &apisixv1.Ssl{
		ID:     id.GenID(tls.Namespace + "_" + tls.Name),
		Snis:   snis,
		Cert:   string(cert),
		Key:    string(key),
		Status: 1,
		Labels: map[string]string{
			"managed-by": "apisix-ingress-controller",
		},
	}
	if tls.Spec.Client != nil {
		caSecret, err := t.SecretLister.Secrets(tls.Spec.Client.CASecret.Namespace).Get(tls.Spec.Client.CASecret.Name)
		if err != nil {
			return nil, err
		}
		ca, _, err := t.ExtractKeyPair(caSecret, false)
		if err != nil {
			return nil, err
		}
		ssl.Client = &apisixv1.MutualTLSClientConfig{
			CA:    string(ca),
			Depth: tls.Spec.Client.Depth,
		}
	}

	return ssl, nil
}

func (t *translator) TranslateSSLV2(tls *configv2.ApisixTls) (*apisixv1.Ssl, error) {
	s, err := t.SecretLister.Secrets(tls.Spec.Secret.Namespace).Get(tls.Spec.Secret.Name)
	if err != nil {
		return nil, err
	}
	cert, key, err := t.ExtractKeyPair(s, true)
	if err != nil {
		return nil, err
	}

	var snis []string
	for _, host := range tls.Spec.Hosts {
		snis = append(snis, string(host))
	}
	ssl := &apisixv1.Ssl{
		ID:     id.GenID(tls.Namespace + "_" + tls.Name),
		Snis:   snis,
		Cert:   string(cert),
		Key:    string(key),
		Status: 1,
		Labels: map[string]string{
			"managed-by": "apisix-ingress-controller",
		},
	}
	if tls.Spec.Client != nil {
		caSecret, err := t.SecretLister.Secrets(tls.Spec.Client.CASecret.Namespace).Get(tls.Spec.Client.CASecret.Name)
		if err != nil {
			return nil, err
		}
		ca, _, err := t.ExtractKeyPair(caSecret, false)
		if err != nil {
			return nil, err
		}
		ssl.Client = &apisixv1.MutualTLSClientConfig{
			CA:    string(ca),
			Depth: tls.Spec.Client.Depth,
		}
	}

	return ssl, nil
}

func (t *translator) ExtractKeyPair(s *v1.Secret, hasPrivateKey bool) ([]byte, []byte, error) {
	if _, ok := s.Data["cert"]; ok {
		return t.extractApisixSecretKeyPair(s, hasPrivateKey)
	} else if _, ok := s.Data[v1.TLSCertKey]; ok {
		return t.extractKubeSecretKeyPair(s, hasPrivateKey)
	} else {
		return nil, nil, ErrUnknownSecretFormat
	}
}

func (t *translator) extractApisixSecretKeyPair(s *v1.Secret, hasPrivateKey bool) (cert []byte, key []byte, err error) {
	var ok bool
	cert, ok = s.Data["cert"]
	if !ok {
		return nil, nil, ErrEmptyCert
	}

	if hasPrivateKey {
		key, ok = s.Data["key"]
		if !ok {
			return nil, nil, ErrEmptyPrivKey
		}
	}
	return
}

func (t *translator) extractKubeSecretKeyPair(s *v1.Secret, hasPrivateKey bool) (cert []byte, key []byte, err error) {
	var ok bool
	cert, ok = s.Data[v1.TLSCertKey]
	if !ok {
		return nil, nil, ErrEmptyCert
	}

	if hasPrivateKey {
		key, ok = s.Data[v1.TLSPrivateKeyKey]
		if !ok {
			return nil, nil, ErrEmptyPrivKey
		}
	}
	return
}
