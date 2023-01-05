// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package translation

import (
	"encoding/json"
	"errors"
	"strconv"

	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

var (
	_errKeyNotFoundOrInvalid      = errors.New("key \"key\" not found or invalid in secret")
	_errUsernameNotFoundOrInvalid = errors.New("key \"username\" not found or invalid in secret")
	_errPasswordNotFoundOrInvalid = errors.New("key \"password\" not found or invalid in secret")

	_jwtAuthExpDefaultValue = int64(868400)

	_hmacAuthAlgorithmDefaultValue           = "hmac-sha256"
	_hmacAuthClockSkewDefaultValue           = int64(0)
	_hmacAuthKeepHeadersDefaultValue         = false
	_hmacAuthEncodeURIParamsDefaultValue     = true
	_hmacAuthValidateRequestBodyDefaultValue = false
	_hmacAuthMaxReqBodyDefaultValue          = int64(524288)

	_openidConnectScopeDefaultValue                            = "openid"
	_openidConnectRealmDefaultValue                            = "apisix"
	_openidConnectBearerOnlyDefaultValue                       = false
	_openidConnectLogoutPathDefaultValue                       = "/logout"
	_openidConnectRedirectURIDefaultValue                      = "ngx.var.request_uri"
	_openidConnectTimeoutDefaultValue                          = int64(3)
	_openidConnectSslVerifyDefaultValue                        = false
	_openidConnectIntrospectionEndpointAuthMethodDefaultValue  = "client_secret_basic"
	_openidConnectUseJwksDefaultValue                          = false
	_openidConnectUsePkceDefaultValue                          = false
	_openidConnectSetAccessTokenHeaderDefaultValue             = true
	_openidConnectAccessTokenInAuthorizationHeaderDefaultValue = false
	_openidConnectSetIDTokenHeaderDefaultValue                 = true
	_openidConnectSetUserInfoHeaderDefaultValue                = true
	_openidConnectSetRefreshTokenHeaderDefaultValue            = false
)

func (t *translator) translateTrafficSplitPlugin(ctx *translation.TranslateContext, ns string, defaultBackendWeight int,
	backends []configv2.ApisixRouteHTTPBackend) (*apisixv1.TrafficSplitConfig, error) {
	var (
		wups []apisixv1.TrafficSplitConfigRuleWeightedUpstream
	)

	for _, backend := range backends {
		svcClusterIP, svcPort, err := t.GetServiceClusterIPAndPort(&backend, ns)
		if err != nil {
			return nil, err
		}
		ups, err := t.translateService(ns, backend.ServiceName, backend.Subset, backend.ResolveGranularity, svcClusterIP, svcPort)
		if err != nil {
			return nil, err
		}
		ctx.AddUpstream(ups)

		weight := translation.DefaultWeight
		if backend.Weight != nil {
			weight = *backend.Weight
		}
		wups = append(wups, apisixv1.TrafficSplitConfigRuleWeightedUpstream{
			UpstreamID: ups.ID,
			Weight:     weight,
		})
	}

	// append the default upstream in the route.
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

func (t *translator) translateConsumerKeyAuthPluginV2beta3(consumerNamespace string, cfg *configv2beta3.ApisixConsumerKeyAuth) (*apisixv1.KeyAuthConsumerConfig, error) {
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

func (t *translator) translateConsumerBasicAuthPluginV2beta3(consumerNamespace string, cfg *configv2beta3.ApisixConsumerBasicAuth) (*apisixv1.BasicAuthConsumerConfig, error) {
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

func (t *translator) translateConsumerKeyAuthPluginV2(consumerNamespace string, cfg *configv2.ApisixConsumerKeyAuth) (*apisixv1.KeyAuthConsumerConfig, error) {
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

func (t *translator) translateConsumerBasicAuthPluginV2(consumerNamespace string, cfg *configv2.ApisixConsumerBasicAuth) (*apisixv1.BasicAuthConsumerConfig, error) {
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

func (t *translator) translateConsumerWolfRBACPluginV2beta3(consumerNamespace string, cfg *configv2beta3.ApisixConsumerWolfRBAC) (*apisixv1.WolfRBACConsumerConfig, error) {
	if cfg.Value != nil {
		return &apisixv1.WolfRBACConsumerConfig{
			Server:       cfg.Value.Server,
			Appid:        cfg.Value.Appid,
			HeaderPrefix: cfg.Value.HeaderPrefix,
		}, nil
	}
	sec, err := t.SecretLister.Secrets(consumerNamespace).Get(cfg.SecretRef.Name)
	if err != nil {
		return nil, err
	}
	raw1 := sec.Data["server"]
	raw2 := sec.Data["appid"]
	raw3 := sec.Data["header_prefix"]
	return &apisixv1.WolfRBACConsumerConfig{
		Server:       string(raw1),
		Appid:        string(raw2),
		HeaderPrefix: string(raw3),
	}, nil
}

func (t *translator) translateConsumerWolfRBACPluginV2(consumerNamespace string, cfg *configv2.ApisixConsumerWolfRBAC) (*apisixv1.WolfRBACConsumerConfig, error) {
	if cfg.Value != nil {
		return &apisixv1.WolfRBACConsumerConfig{
			Server:       cfg.Value.Server,
			Appid:        cfg.Value.Appid,
			HeaderPrefix: cfg.Value.HeaderPrefix,
		}, nil
	}
	sec, err := t.SecretLister.Secrets(consumerNamespace).Get(cfg.SecretRef.Name)
	if err != nil {
		return nil, err
	}
	raw1 := sec.Data["server"]
	raw2 := sec.Data["appid"]
	raw3 := sec.Data["header_prefix"]
	return &apisixv1.WolfRBACConsumerConfig{
		Server:       string(raw1),
		Appid:        string(raw2),
		HeaderPrefix: string(raw3),
	}, nil
}

func (t *translator) translateConsumerJwtAuthPluginV2beta3(consumerNamespace string, cfg *configv2beta3.ApisixConsumerJwtAuth) (*apisixv1.JwtAuthConsumerConfig, error) {
	if cfg.Value != nil {
		// The field exp must be a positive integer, default value 86400.
		if cfg.Value.Exp < 1 {
			cfg.Value.Exp = _jwtAuthExpDefaultValue
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
	// The field exp must be a positive integer, default value 86400.
	if exp < 1 {
		exp = _jwtAuthExpDefaultValue
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

func (t *translator) translateConsumerJwtAuthPluginV2(consumerNamespace string, cfg *configv2.ApisixConsumerJwtAuth) (*apisixv1.JwtAuthConsumerConfig, error) {
	if cfg.Value != nil {
		// The field exp must be a positive integer, default value 86400.
		if cfg.Value.Exp < 1 {
			cfg.Value.Exp = _jwtAuthExpDefaultValue
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
	// The field exp must be a positive integer, default value 86400.
	if exp < 1 {
		exp = _jwtAuthExpDefaultValue
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

func (t *translator) translateConsumerHMACAuthPluginV2beta3(consumerNamespace string, cfg *configv2beta3.ApisixConsumerHMACAuth) (*apisixv1.HMACAuthConsumerConfig, error) {
	if cfg.Value != nil {
		return &apisixv1.HMACAuthConsumerConfig{
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

	return &apisixv1.HMACAuthConsumerConfig{
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

func (t *translator) translateConsumerHMACAuthPluginV2(consumerNamespace string, cfg *configv2.ApisixConsumerHMACAuth) (*apisixv1.HMACAuthConsumerConfig, error) {
	if cfg.Value != nil {
		return &apisixv1.HMACAuthConsumerConfig{
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

	return &apisixv1.HMACAuthConsumerConfig{
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

func (t *translator) translateConsumerOpenIDConnectPluginV2(consumerNamespace string, cfg *configv2.ApisixConsumerOpenIDConnect) (*apisixv1.OpenIDConnectConsumerConfig, error) {
	if cfg.Value != nil {
		openIDConnectConfig := &apisixv1.OpenIDConnectConsumerConfig{
			ClientID:                         cfg.Value.ClientID,
			ClientSecret:                     cfg.Value.ClientSecret,
			Discovery:                        cfg.Value.Discovery,
			Scope:                            cfg.Value.Scope,
			Realm:                            cfg.Value.Realm,
			BearerOnly:                       cfg.Value.BearerOnly,
			LogoutPath:                       cfg.Value.LogoutPath,
			PostLogoutRedirectURI:            cfg.Value.PostLogoutRedirectURI,
			RedirectURI:                      cfg.Value.RedirectURI,
			Timeout:                          cfg.Value.Timeout,
			SslVerify:                        cfg.Value.SslVerify,
			IntrospectionEndpoint:            cfg.Value.IntrospectionEndpoint,
			IntrospectionEndpointAuthMethod:  cfg.Value.IntrospectionEndpointAuthMethod,
			TokenEndpointAuthMethod:          cfg.Value.TokenEndpointAuthMethod,
			PublicKey:                        cfg.Value.PublicKey,
			UseJwks:                          cfg.Value.UseJwks,
			UsePkce:                          cfg.Value.UsePkce,
			TokenSigningAlgValuesExpected:    cfg.Value.TokenSigningAlgValuesExpected,
			SetAccessTokenHeader:             cfg.Value.SetAccessTokenHeader,
			AccessTokeninAuthorizationHeader: cfg.Value.AccessTokeninAuthorizationHeader,
			SetIdTokenHeader:                 cfg.Value.SetIdTokenHeader,
			SetUserinfoHeader:                cfg.Value.SetUserinfoHeader,
			SetRefreshTokenHeader:            cfg.Value.SetRefreshTokenHeader,
		}

		if cfg.Value.Scope == "" {
			openIDConnectConfig.Scope = _openidConnectScopeDefaultValue
		}

		if cfg.Value.Realm == "" {
			openIDConnectConfig.Realm = _openidConnectRealmDefaultValue
		}

		if cfg.Value.LogoutPath == "" {
			openIDConnectConfig.LogoutPath = _openidConnectLogoutPathDefaultValue
		}

		if cfg.Value.RedirectURI == "" {
			openIDConnectConfig.RedirectURI = _openidConnectRedirectURIDefaultValue
		}

		if cfg.Value.Timeout <= int64(0) {
			openIDConnectConfig.Timeout = _openidConnectTimeoutDefaultValue
		}

		if cfg.Value.IntrospectionEndpointAuthMethod == "" {
			openIDConnectConfig.IntrospectionEndpointAuthMethod = _openidConnectIntrospectionEndpointAuthMethodDefaultValue
		}

		if cfg.Value.Session != nil {
			openIDConnectConfig.Session = &apisixv1.OpenIDConnectSession{Secret: cfg.Value.Session.Secret}
		}
		return openIDConnectConfig, nil
	}

	sec, err := t.SecretLister.Secrets(consumerNamespace).Get(cfg.SecretRef.Name)
	if err != nil {
		return nil, err
	}

	clientID, ok := sec.Data["client_id"]
	if !ok || len(clientID) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}

	clientSecret, ok := sec.Data["client_secret"]
	if !ok || len(clientSecret) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}

	discovery, ok := sec.Data["discovery"]
	if !ok || len(discovery) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}

	scopeRaw, ok := sec.Data["scope"]
	var scope string
	if !ok {
		scope = _openidConnectScopeDefaultValue
	} else {
		scope = string(scopeRaw)
	}

	realmRaw, ok := sec.Data["realm"]
	var realm string
	if !ok {
		realm = _openidConnectRealmDefaultValue
	} else {
		realm = string(realmRaw)
	}

	var bearerOnly bool
	bearerOnlyRaw, ok := sec.Data["bearer_only"]
	if !ok {
		bearerOnly = _openidConnectBearerOnlyDefaultValue
	} else {
		if string(bearerOnlyRaw) == "true" {
			bearerOnly = true
		} else {
			bearerOnly = false
		}
	}

	logoutPathRaw, ok := sec.Data["logout_path"]
	var logoutPath string
	if !ok {
		logoutPath = _openidConnectLogoutPathDefaultValue
	} else {
		logoutPath = string(logoutPathRaw)
	}

	postLogoutRedirectURIRaw := sec.Data["post_logout_redirect_uri"]

	redirectURIRaw, ok := sec.Data["redirect_uri"]
	var redirectURI string
	if !ok {
		redirectURI = _openidConnectRedirectURIDefaultValue
	} else {
		redirectURI = string(redirectURIRaw)
	}

	timeoutRaw := sec.Data["timeout"]
	timeout, _ := strconv.ParseInt(string(timeoutRaw), 10, 64)
	if timeout <= int64(0) {
		timeout = _openidConnectTimeoutDefaultValue
	}

	var sslVerify bool
	sslVerifyRaw, ok := sec.Data["ssl_verify"]
	if !ok {
		sslVerify = _openidConnectSslVerifyDefaultValue
	} else {
		if string(sslVerifyRaw) == "true" {
			sslVerify = true
		} else {
			sslVerify = false
		}
	}

	introspectionEndpointRaw := sec.Data["introspection_endpoint"]

	introspectionEndpointAuthMethodRaw, ok := sec.Data["introspection_endpoint_auth_method"]
	var introspectionEndpointAuthMethod string
	if !ok {
		introspectionEndpointAuthMethod = _openidConnectIntrospectionEndpointAuthMethodDefaultValue
	} else {
		introspectionEndpointAuthMethod = string(introspectionEndpointAuthMethodRaw)
	}

	tokenEndpointAuthMethodRaw := sec.Data["token_endpoint_auth_method"]
	publicKeyRaw := sec.Data["public_key"]

	var useJwks bool
	useJwksRaw, ok := sec.Data["use_jwks"]
	if !ok {
		useJwks = _openidConnectUseJwksDefaultValue
	} else {
		if string(useJwksRaw) == "true" {
			useJwks = true
		} else {
			useJwks = false
		}
	}

	var usePkce bool
	usePkceRaw, ok := sec.Data["use_pkce"]
	if !ok {
		usePkce = _openidConnectUsePkceDefaultValue
	} else {
		if string(usePkceRaw) == "true" {
			usePkce = true
		} else {
			usePkce = false
		}
	}

	tokenSigningAlgValuesExpectedRaw := sec.Data["token_signing_alg_values_expected"]

	var setAccessTokenHeader bool
	setAccessTokenHeaderRaw, ok := sec.Data["set_access_token_header"]
	if !ok {
		setAccessTokenHeader = _openidConnectSetAccessTokenHeaderDefaultValue
	} else {
		if string(setAccessTokenHeaderRaw) == "true" {
			setAccessTokenHeader = true
		} else {
			setAccessTokenHeader = false
		}
	}

	var accessTokenInAuthorizationHeader bool
	accessTokenInAuthorizationHeaderRaw, ok := sec.Data["access_token_in_authorization_header"]
	if !ok {
		accessTokenInAuthorizationHeader = _openidConnectAccessTokenInAuthorizationHeaderDefaultValue
	} else {
		if string(accessTokenInAuthorizationHeaderRaw) == "true" {
			accessTokenInAuthorizationHeader = true
		} else {
			accessTokenInAuthorizationHeader = false
		}
	}

	var setIDTokenHeader bool
	setIDTokenHeaderRaw, ok := sec.Data["set_id_token_header"]
	if !ok {
		setIDTokenHeader = _openidConnectSetIDTokenHeaderDefaultValue
	} else {
		if string(setIDTokenHeaderRaw) == "true" {
			setIDTokenHeader = true
		} else {
			setIDTokenHeader = false
		}
	}

	var setUserinfoHeader bool
	setUserinfoHeaderRaw, ok := sec.Data["set_userinfo_header"]
	if !ok {
		setUserinfoHeader = _openidConnectSetUserInfoHeaderDefaultValue
	} else {
		if string(setUserinfoHeaderRaw) == "true" {
			setUserinfoHeader = true
		} else {
			setUserinfoHeader = false
		}
	}

	var setRefreshTokenHeader bool
	setRefreshTokenHeaderRaw, ok := sec.Data["set_refresh_token_header"]
	if !ok {
		setRefreshTokenHeader = _openidConnectSetRefreshTokenHeaderDefaultValue
	} else {
		if string(setRefreshTokenHeaderRaw) == "true" {
			setRefreshTokenHeader = true
		} else {
			setRefreshTokenHeader = false
		}
	}

	var session *apisixv1.OpenIDConnectSession
	if sessionRaw, ok := sec.Data["session"]; ok {
		if err = json.Unmarshal(sessionRaw, session); err != nil {
			log.Errorf("json.Unmarshal(sessionRaw, session) got error: %v", err)
		}
	}

	return &apisixv1.OpenIDConnectConsumerConfig{
		ClientID:                         string(clientID),
		ClientSecret:                     string(clientSecret),
		Discovery:                        string(discovery),
		Scope:                            scope,
		Realm:                            realm,
		BearerOnly:                       bearerOnly,
		LogoutPath:                       logoutPath,
		PostLogoutRedirectURI:            string(postLogoutRedirectURIRaw),
		RedirectURI:                      redirectURI,
		Timeout:                          timeout,
		SslVerify:                        sslVerify,
		IntrospectionEndpoint:            string(introspectionEndpointRaw),
		IntrospectionEndpointAuthMethod:  introspectionEndpointAuthMethod,
		TokenEndpointAuthMethod:          string(tokenEndpointAuthMethodRaw),
		PublicKey:                        string(publicKeyRaw),
		UseJwks:                          useJwks,
		UsePkce:                          usePkce,
		TokenSigningAlgValuesExpected:    string(tokenSigningAlgValuesExpectedRaw),
		SetAccessTokenHeader:             setAccessTokenHeader,
		AccessTokeninAuthorizationHeader: accessTokenInAuthorizationHeader,
		SetIdTokenHeader:                 setIDTokenHeader,
		SetUserinfoHeader:                setUserinfoHeader,
		SetRefreshTokenHeader:            setRefreshTokenHeader,
		Session:                          session,
	}, nil
}

func (t *translator) translateConsumerOpenIDConnectPluginV2beta3(consumerNamespace string, cfg *configv2beta3.ApisixConsumerOpenIDConnect) (*apisixv1.OpenIDConnectConsumerConfig, error) {
	if cfg.Value != nil {
		openIDConnectConfig := &apisixv1.OpenIDConnectConsumerConfig{
			ClientID:                         cfg.Value.ClientID,
			ClientSecret:                     cfg.Value.ClientSecret,
			Discovery:                        cfg.Value.Discovery,
			Scope:                            cfg.Value.Scope,
			Realm:                            cfg.Value.Realm,
			BearerOnly:                       cfg.Value.BearerOnly,
			LogoutPath:                       cfg.Value.LogoutPath,
			PostLogoutRedirectURI:            cfg.Value.PostLogoutRedirectURI,
			RedirectURI:                      cfg.Value.RedirectURI,
			Timeout:                          cfg.Value.Timeout,
			SslVerify:                        cfg.Value.SslVerify,
			IntrospectionEndpoint:            cfg.Value.IntrospectionEndpoint,
			IntrospectionEndpointAuthMethod:  cfg.Value.IntrospectionEndpointAuthMethod,
			TokenEndpointAuthMethod:          cfg.Value.TokenEndpointAuthMethod,
			PublicKey:                        cfg.Value.PublicKey,
			UseJwks:                          cfg.Value.UseJwks,
			UsePkce:                          cfg.Value.UsePkce,
			TokenSigningAlgValuesExpected:    cfg.Value.TokenSigningAlgValuesExpected,
			SetAccessTokenHeader:             cfg.Value.SetAccessTokenHeader,
			AccessTokeninAuthorizationHeader: cfg.Value.AccessTokeninAuthorizationHeader,
			SetIdTokenHeader:                 cfg.Value.SetIdTokenHeader,
			SetUserinfoHeader:                cfg.Value.SetUserinfoHeader,
			SetRefreshTokenHeader:            cfg.Value.SetRefreshTokenHeader,
		}

		if cfg.Value.Scope == "" {
			openIDConnectConfig.Scope = _openidConnectScopeDefaultValue
		}

		if cfg.Value.Realm == "" {
			openIDConnectConfig.Realm = _openidConnectRealmDefaultValue
		}

		if cfg.Value.LogoutPath == "" {
			openIDConnectConfig.LogoutPath = _openidConnectLogoutPathDefaultValue
		}

		if cfg.Value.RedirectURI == "" {
			openIDConnectConfig.RedirectURI = _openidConnectRedirectURIDefaultValue
		}

		if cfg.Value.Timeout <= int64(0) {
			openIDConnectConfig.Timeout = _openidConnectTimeoutDefaultValue
		}

		if cfg.Value.IntrospectionEndpointAuthMethod == "" {
			openIDConnectConfig.IntrospectionEndpointAuthMethod = _openidConnectIntrospectionEndpointAuthMethodDefaultValue
		}

		if cfg.Value.Session != nil {
			openIDConnectConfig.Session = &apisixv1.OpenIDConnectSession{Secret: cfg.Value.Session.Secret}
		}
		return openIDConnectConfig, nil
	}

	sec, err := t.SecretLister.Secrets(consumerNamespace).Get(cfg.SecretRef.Name)
	if err != nil {
		return nil, err
	}

	clientID, ok := sec.Data["client_id"]
	if !ok || len(clientID) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}

	clientSecret, ok := sec.Data["client_secret"]
	if !ok || len(clientSecret) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}

	discovery, ok := sec.Data["discovery"]
	if !ok || len(discovery) == 0 {
		return nil, _errKeyNotFoundOrInvalid
	}

	scopeRaw, ok := sec.Data["scope"]
	var scope string
	if !ok {
		scope = _openidConnectScopeDefaultValue
	} else {
		scope = string(scopeRaw)
	}

	realmRaw, ok := sec.Data["realm"]
	var realm string
	if !ok {
		realm = _openidConnectRealmDefaultValue
	} else {
		realm = string(realmRaw)
	}

	var bearerOnly bool
	bearerOnlyRaw, ok := sec.Data["bearer_only"]
	if !ok {
		bearerOnly = _openidConnectBearerOnlyDefaultValue
	} else {
		if string(bearerOnlyRaw) == "true" {
			bearerOnly = true
		} else {
			bearerOnly = false
		}
	}

	logoutPathRaw, ok := sec.Data["logout_path"]
	var logoutPath string
	if !ok {
		logoutPath = _openidConnectLogoutPathDefaultValue
	} else {
		logoutPath = string(logoutPathRaw)
	}

	postLogoutRedirectURIRaw := sec.Data["post_logout_redirect_uri"]

	redirectURIRaw, ok := sec.Data["redirect_uri"]
	var redirectURI string
	if !ok {
		redirectURI = _openidConnectRedirectURIDefaultValue
	} else {
		redirectURI = string(redirectURIRaw)
	}

	timeoutRaw := sec.Data["timeout"]
	timeout, _ := strconv.ParseInt(string(timeoutRaw), 10, 64)
	if timeout <= int64(0) {
		timeout = _openidConnectTimeoutDefaultValue
	}

	var sslVerify bool
	sslVerifyRaw, ok := sec.Data["ssl_verify"]
	if !ok {
		sslVerify = _openidConnectSslVerifyDefaultValue
	} else {
		if string(sslVerifyRaw) == "true" {
			sslVerify = true
		} else {
			sslVerify = false
		}
	}

	introspectionEndpointRaw := sec.Data["introspection_endpoint"]

	introspectionEndpointAuthMethodRaw, ok := sec.Data["introspection_endpoint_auth_method"]
	var introspectionEndpointAuthMethod string
	if !ok {
		introspectionEndpointAuthMethod = _openidConnectIntrospectionEndpointAuthMethodDefaultValue
	} else {
		introspectionEndpointAuthMethod = string(introspectionEndpointAuthMethodRaw)
	}

	tokenEndpointAuthMethodRaw := sec.Data["token_endpoint_auth_method"]
	publicKeyRaw := sec.Data["public_key"]

	var useJwks bool
	useJwksRaw, ok := sec.Data["use_jwks"]
	if !ok {
		useJwks = _openidConnectUseJwksDefaultValue
	} else {
		if string(useJwksRaw) == "true" {
			useJwks = true
		} else {
			useJwks = false
		}
	}

	var usePkce bool
	usePkceRaw, ok := sec.Data["use_pkce"]
	if !ok {
		usePkce = _openidConnectUsePkceDefaultValue
	} else {
		if string(usePkceRaw) == "true" {
			usePkce = true
		} else {
			usePkce = false
		}
	}

	tokenSigningAlgValuesExpectedRaw := sec.Data["token_signing_alg_values_expected"]

	var setAccessTokenHeader bool
	setAccessTokenHeaderRaw, ok := sec.Data["set_access_token_header"]
	if !ok {
		setAccessTokenHeader = _openidConnectSetAccessTokenHeaderDefaultValue
	} else {
		if string(setAccessTokenHeaderRaw) == "true" {
			setAccessTokenHeader = true
		} else {
			setAccessTokenHeader = false
		}
	}

	var accessTokenInAuthorizationHeader bool
	accessTokenInAuthorizationHeaderRaw, ok := sec.Data["access_token_in_authorization_header"]
	if !ok {
		accessTokenInAuthorizationHeader = _openidConnectAccessTokenInAuthorizationHeaderDefaultValue
	} else {
		if string(accessTokenInAuthorizationHeaderRaw) == "true" {
			accessTokenInAuthorizationHeader = true
		} else {
			accessTokenInAuthorizationHeader = false
		}
	}

	var setIDTokenHeader bool
	setIDTokenHeaderRaw, ok := sec.Data["set_id_token_header"]
	if !ok {
		setIDTokenHeader = _openidConnectSetIDTokenHeaderDefaultValue
	} else {
		if string(setIDTokenHeaderRaw) == "true" {
			setIDTokenHeader = true
		} else {
			setIDTokenHeader = false
		}
	}

	var setUserinfoHeader bool
	setUserinfoHeaderRaw, ok := sec.Data["set_userinfo_header"]
	if !ok {
		setUserinfoHeader = _openidConnectSetUserInfoHeaderDefaultValue
	} else {
		if string(setUserinfoHeaderRaw) == "true" {
			setUserinfoHeader = true
		} else {
			setUserinfoHeader = false
		}
	}

	var setRefreshTokenHeader bool
	setRefreshTokenHeaderRaw, ok := sec.Data["set_refresh_token_header"]
	if !ok {
		setRefreshTokenHeader = _openidConnectSetRefreshTokenHeaderDefaultValue
	} else {
		if string(setRefreshTokenHeaderRaw) == "true" {
			setRefreshTokenHeader = true
		} else {
			setRefreshTokenHeader = false
		}
	}

	var session *apisixv1.OpenIDConnectSession
	if sessionRaw, ok := sec.Data["session"]; ok {
		if err = json.Unmarshal(sessionRaw, session); err != nil {
			log.Errorf("json.Unmarshal(sessionRaw, session) got error: %v", err)
		}
	}

	return &apisixv1.OpenIDConnectConsumerConfig{
		ClientID:                         string(clientID),
		ClientSecret:                     string(clientSecret),
		Discovery:                        string(discovery),
		Scope:                            scope,
		Realm:                            realm,
		BearerOnly:                       bearerOnly,
		LogoutPath:                       logoutPath,
		PostLogoutRedirectURI:            string(postLogoutRedirectURIRaw),
		RedirectURI:                      redirectURI,
		Timeout:                          timeout,
		SslVerify:                        sslVerify,
		IntrospectionEndpoint:            string(introspectionEndpointRaw),
		IntrospectionEndpointAuthMethod:  introspectionEndpointAuthMethod,
		TokenEndpointAuthMethod:          string(tokenEndpointAuthMethodRaw),
		PublicKey:                        string(publicKeyRaw),
		UseJwks:                          useJwks,
		UsePkce:                          usePkce,
		TokenSigningAlgValuesExpected:    string(tokenSigningAlgValuesExpectedRaw),
		SetAccessTokenHeader:             setAccessTokenHeader,
		AccessTokeninAuthorizationHeader: accessTokenInAuthorizationHeader,
		SetIdTokenHeader:                 setIDTokenHeader,
		SetUserinfoHeader:                setUserinfoHeader,
		SetRefreshTokenHeader:            setRefreshTokenHeader,
		Session:                          session,
	}, nil
}
