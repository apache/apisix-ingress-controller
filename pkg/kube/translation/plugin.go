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

	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

var (
	_errKeyNotFoundOrInvalid      = errors.New("key \"key\" not found or invalid in secret")
	_errUsernameNotFoundOrInvalid = errors.New("key \"username\" not found or invalid in secret")
	_errPasswordNotFoundOrInvalid = errors.New("key \"password\" not found or invalid in secret")

	_hmacAuthAlgorithmDefaultValue           = "hmac-sha256"
	_hmacAuthClockSkewDefaultValue           = int64(0)
	_hmacAuthKeepHeadersDefaultValue         = false
	_hmacAuthEncodeURIParamsDefaultValue     = true
	_hmacAuthValidateRequestBodyDefaultValue = false
	_hmacAuthMaxReqBodyDefaultValue          = int64(524288)
)

func (t *translator) translateTrafficSplitPlugin(ctx *TranslateContext, ns string, defaultBackendWeight int,
	backends []configv2beta3.ApisixRouteHTTPBackend) (*apisixv1.TrafficSplitConfig, error) {
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

func (t *translator) translateConsumerHMacAuthPlugin(consumerNamespace string, cfg *configv2beta3.ApisixConsumerHMacAuth) (*apisixv1.HMacAuthConsumerConfig, error) {
	if cfg.Value != nil {
		return &apisixv1.HMacAuthConsumerConfig{
			AccessKey:           cfg.Value.AccessKey,
			SecretKey:           cfg.Value.SecretKey,
			Algorithm:           cfg.Value.Algorithm,
			ClockSkew:           cfg.Value.ClockSkew,
			SignedHeaders:       cfg.Value.SignedHeaders,
			KeepHeaders:         cfg.Value.KeepHeaders,
			EncodeURIParams:     cfg.Value.EncodeURIParams,
			ValidateRequestBody: cfg.Value.ValidateRequestBody,
			MaxReqBody:          cfg.Value.MaxReqBody,
		}, nil
	}

	sec, err := t.SecretLister.Secrets(consumerNamespace).Get(cfg.SecretRef.Name)
	if err != nil {
		return nil, err
	}

	accessKeyRaw, ok := sec.Data["access_key"]
	if !ok || len(accessKeyRaw) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}

	secretKeyRaw, ok := sec.Data["secret_key"]
	if !ok || len(secretKeyRaw) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}

	algorithmRaw, ok := sec.Data["algorithm"]
	var algorithm string
	if !ok {
		algorithm = _hmacAuthAlgorithmDefaultValue
	} else {
		algorithm = string(algorithmRaw)
	}

	clockSkewRaw := sec.Data["clock_skew"]
	clockSkew, _ := strconv.ParseInt(string(clockSkewRaw), 10, 64)
	if clockSkew < 0 {
		clockSkew = _hmacAuthClockSkewDefaultValue
	}

	var signedHeaders []string
	signedHeadersRaw := sec.Data["signed_headers"]
	for _, b := range signedHeadersRaw {
		signedHeaders = append(signedHeaders, string(b))
	}

	var keepHeader bool
	keepHeaderRaw, ok := sec.Data["keep_headers"]
	if !ok {
		keepHeader = _hmacAuthKeepHeadersDefaultValue
	} else {
		if string(keepHeaderRaw) == "true" {
			keepHeader = true
		} else {
			keepHeader = false
		}
	}

	var encodeURIParams bool
	encodeURIParamsRaw, ok := sec.Data["encode_uri_params"]
	if !ok {
		encodeURIParams = _hmacAuthEncodeURIParamsDefaultValue
	} else {
		if string(encodeURIParamsRaw) == "true" {
			encodeURIParams = true
		} else {
			encodeURIParams = false
		}
	}

	var validateRequestBody bool
	validateRequestBodyRaw, ok := sec.Data["validate_request_body"]
	if !ok {
		validateRequestBody = _hmacAuthValidateRequestBodyDefaultValue
	} else {
		if string(validateRequestBodyRaw) == "true" {
			validateRequestBody = true
		} else {
			validateRequestBody = false
		}
	}

	maxReqBodyRaw := sec.Data["max_req_body"]
	maxReqBody, _ := strconv.ParseInt(string(maxReqBodyRaw), 10, 64)
	if maxReqBody < 0 {
		maxReqBody = _hmacAuthMaxReqBodyDefaultValue
	}

	return &apisixv1.HMacAuthConsumerConfig{
		AccessKey:           string(accessKeyRaw),
		SecretKey:           string(secretKeyRaw),
		Algorithm:           algorithm,
		ClockSkew:           clockSkew,
		SignedHeaders:       signedHeaders,
		KeepHeaders:         keepHeader,
		EncodeURIParams:     encodeURIParams,
		ValidateRequestBody: validateRequestBody,
		MaxReqBody:          maxReqBody,
	}, nil
}
