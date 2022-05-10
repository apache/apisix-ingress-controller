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
	"strconv"

	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

var (
	_errKeyNotFoundOrInvalid      = errors.New("key \"key\" not found or invalid in secret")
	_errUsernameNotFoundOrInvalid = errors.New("key \"username\" not found or invalid in secret")
	_errPasswordNotFoundOrInvalid = errors.New("key \"password\" not found or invalid in secret")
	_errExpNotPositiveInteger     = errors.New("key \"exp\" not a positive integer")
)

func (t *translator) translateTrafficSplitPlugin(ctx *TranslateContext, ns string, defaultBackendWeight int,
	backends []configv2.ApisixRouteHTTPBackend) (*apisixv1.TrafficSplitConfig, error) {
	var (
		wups []apisixv1.TrafficSplitConfigRuleWeightedUpstream
	)

	for _, backend := range backends {
		svcClusterIP, svcPort, err := t.getServiceClusterIPAndPort(&backend, ns)
		if err != nil {
			return nil, err
		}
		ups, err := t.translateUpstream(ns, backend.ServiceName, backend.Subset, backend.ResolveGranularity, svcClusterIP, svcPort)
		if err != nil {
			return nil, err
		}
		ctx.addUpstream(ups)

		weight := _defaultWeight
		if backend.Weight != nil {
			weight = *backend.Weight
		}
		wups = append(wups, apisixv1.TrafficSplitConfigRuleWeightedUpstream{
			UpstreamID: ups.ID,
			Weight:     weight,
		})
	}

	// Finally append the default upstream in the route.
	wups = append(wups, apisixv1.TrafficSplitConfigRuleWeightedUpstream{
		Weight: defaultBackendWeight,
	})

	tsCfg := &apisixv1.TrafficSplitConfig{
		Rules: []apisixv1.TrafficSplitConfigRule{
			{
				WeightedUpstreams: wups,
			},
		},
	}
	return tsCfg, nil
}

func (t *translator) translateConsumerKeyAuthPlugin(consumerNamespace string, cfg *configv2beta3.ApisixConsumerKeyAuth) (*apisixv1.KeyAuthConsumerConfig, error) {
	if cfg.Value != nil {
		return &apisixv1.KeyAuthConsumerConfig{Key: cfg.Value.Key}, nil
	}

	sec, err := t.SecretLister.Secrets(consumerNamespace).Get(cfg.SecretRef.Name)
	if err != nil {
		return nil, err
	}
	raw, ok := sec.Data["key"]
	if !ok || len(raw) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}
	return &apisixv1.KeyAuthConsumerConfig{Key: string(raw)}, nil
}

func (t *translator) translateConsumerBasicAuthPlugin(consumerNamespace string, cfg *configv2beta3.ApisixConsumerBasicAuth) (*apisixv1.BasicAuthConsumerConfig, error) {
	if cfg.Value != nil {
		return &apisixv1.BasicAuthConsumerConfig{
			Username: cfg.Value.Username,
			Password: cfg.Value.Password,
		}, nil
	}

	sec, err := t.SecretLister.Secrets(consumerNamespace).Get(cfg.SecretRef.Name)
	if err != nil {
		return nil, err
	}
	raw1, ok := sec.Data["username"]
	if !ok || len(raw1) == 0 {
		return nil, _errUsernameNotFoundOrInvalid
	}
	raw2, ok := sec.Data["password"]
	if !ok || len(raw2) == 0 {
		return nil, _errPasswordNotFoundOrInvalid
	}
	return &apisixv1.BasicAuthConsumerConfig{
		Username: string(raw1),
		Password: string(raw2),
	}, nil
}

func (t *translator) translateConsumerJwtAuthPlugin(consumerNamespace string, cfg *configv2beta3.ApisixConsumerJwtAuth) (*apisixv1.JwtAuthConsumerConfig, error) {
	if cfg.Value != nil {
		if cfg.Value.Exp < 1 {
			return nil, _errExpNotPositiveInteger
		}
		return &apisixv1.JwtAuthConsumerConfig{
			Key:          cfg.Value.Key,
			Secret:       cfg.Value.Secret,
			PublicKey:    cfg.Value.PublicKey,
			PrivateKey:   cfg.Value.PrivateKey,
			Algorithm:    cfg.Value.Algorithm,
			Exp:          cfg.Value.Exp,
			Base64Secret: cfg.Value.Base64Secret,
		}, nil
	}

	sec, err := t.SecretLister.Secrets(consumerNamespace).Get(cfg.SecretRef.Name)
	if err != nil {
		return nil, err
	}
	keyRaw, ok := sec.Data["key"]
	if !ok || len(keyRaw) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}
	base64SecretRaw := sec.Data["base64_secret"]
	var base64Secret bool
	if string(base64SecretRaw) == "true" {
		base64Secret = true
	}
	expRaw := sec.Data["exp"]
	exp, _ := strconv.ParseInt(string(expRaw), 10, 64)
	if exp < 1 {
		return nil, _errExpNotPositiveInteger
	}
	secretRaw := sec.Data["secret"]
	publicKeyRaw := sec.Data["public_key"]
	privateKeyRaw := sec.Data["private_key"]
	algorithmRaw := sec.Data["algorithm"]
	return &apisixv1.JwtAuthConsumerConfig{
		Key:          string(keyRaw),
		Secret:       string(secretRaw),
		PublicKey:    string(publicKeyRaw),
		PrivateKey:   string(privateKeyRaw),
		Algorithm:    string(algorithmRaw),
		Exp:          exp,
		Base64Secret: base64Secret,
	}, nil
}
