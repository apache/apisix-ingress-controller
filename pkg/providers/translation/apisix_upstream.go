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
	"fmt"

	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslateUpstreamConfigV2beta3(au *configv2beta3.ApisixUpstreamConfig) (*apisixv1.Upstream, error) {
	ups := apisixv1.NewDefaultUpstream()
	if err := t.translateUpstreamScheme(au.Scheme, ups); err != nil {
		return nil, err
	}
	if err := t.translateUpstreamLoadBalancerV2beta3(au.LoadBalancer, ups); err != nil {
		return nil, err
	}
	if err := t.translateUpstreamHealthCheckV2beta3(au.HealthCheck, ups); err != nil {
		return nil, err
	}
	if err := t.translateUpstreamRetriesAndTimeoutV2beta3(au.Retries, au.Timeout, ups); err != nil {
		return nil, err
	}
	if err := t.translateClientTLSV2beta3(au.TLSSecret, ups); err != nil {
		return nil, err
	}
	return ups, nil
}

func (t *translator) TranslateUpstreamConfigV2(au *configv2.ApisixUpstreamConfig, ups *apisixv1.Upstream) error {
	if err := t.translateUpstreamScheme(au.Scheme, ups); err != nil {
		return err
	}
	if err := t.translateUpstreamLoadBalancerV2(au.LoadBalancer, ups); err != nil {
		return err
	}
	if err := t.translateUpstreamHealthCheckV2(au.HealthCheck, ups); err != nil {
		return err
	}
	if err := t.translateUpstreamRetriesAndTimeoutV2(au.Retries, au.Timeout, ups); err != nil {
		return err
	}
	if err := t.translateClientTLSV2(au.TLSSecret, ups); err != nil {
		return err
	}
	return nil
}

func (t *translator) translateUpstreamRetriesAndTimeoutV2beta3(retries *int, timeout *configv2beta3.UpstreamTimeout, ups *apisixv1.Upstream) error {
	if retries != nil && *retries < 0 {
		return &TranslateError{
			Field:  "retries",
			Reason: "invalid value",
		}
	}
	ups.Retries = retries
	if timeout == nil {
		return nil
	}

	// Since the schema of timeout doesn't allow only configuring
	// one or two items. Here we assign the default value first.
	connTimeout := apisixv1.DefaultUpstreamTimeout
	readTimeout := apisixv1.DefaultUpstreamTimeout
	sendTimeout := apisixv1.DefaultUpstreamTimeout
	if timeout.Connect.Duration < 0 {
		return &TranslateError{
			Field:  "timeout.connect",
			Reason: "invalid value",
		}
	} else if timeout.Connect.Duration > 0 {
		connTimeout = int(timeout.Connect.Seconds())
	}
	if timeout.Read.Duration < 0 {
		return &TranslateError{
			Field:  "timeout.read",
			Reason: "invalid value",
		}
	} else if timeout.Read.Duration > 0 {
		readTimeout = int(timeout.Read.Seconds())
	}
	if timeout.Send.Duration < 0 {
		return &TranslateError{
			Field:  "timeout.send",
			Reason: "invalid value",
		}
	} else if timeout.Send.Duration > 0 {
		sendTimeout = int(timeout.Send.Seconds())
	}
	ups.Timeout = &apisixv1.UpstreamTimeout{
		Connect: connTimeout,
		Send:    sendTimeout,
		Read:    readTimeout,
	}
	return nil
}

func (t *translator) translateUpstreamScheme(scheme string, ups *apisixv1.Upstream) error {
	if scheme == "" {
		ups.Scheme = apisixv1.SchemeHTTP
		return nil
	}
	switch scheme {
	case apisixv1.SchemeHTTP, apisixv1.SchemeGRPC, apisixv1.SchemeHTTPS, apisixv1.SchemeGRPCS:
		ups.Scheme = scheme
		return nil
	default:
		return &TranslateError{Field: "scheme", Reason: "invalid value"}
	}
}

func (t *translator) translateUpstreamLoadBalancerV2beta3(lb *configv2beta3.LoadBalancer, ups *apisixv1.Upstream) error {
	if lb == nil || lb.Type == "" {
		ups.Type = apisixv1.LbRoundRobin
		return nil
	}
	switch lb.Type {
	case apisixv1.LbRoundRobin, apisixv1.LbLeastConn, apisixv1.LbEwma:
		ups.Type = lb.Type
	case apisixv1.LbConsistentHash:
		ups.Type = lb.Type
		ups.Key = lb.Key
		switch lb.HashOn {
		case apisixv1.HashOnVars:
			fallthrough
		case apisixv1.HashOnHeader:
			fallthrough
		case apisixv1.HashOnCookie:
			fallthrough
		case apisixv1.HashOnConsumer:
			fallthrough
		case apisixv1.HashOnVarsCombination:
			ups.HashOn = lb.HashOn
		default:
			return &TranslateError{Field: "loadbalancer.hashOn", Reason: "invalid value"}
		}
	default:
		return &TranslateError{
			Field:  "loadbalancer.type",
			Reason: "invalid value",
		}
	}
	return nil
}

func (t *translator) translateUpstreamHealthCheckV2beta3(config *configv2beta3.HealthCheck, ups *apisixv1.Upstream) error {
	if config == nil || (config.Passive == nil && config.Active == nil) {
		return nil
	}
	var hc apisixv1.UpstreamHealthCheck
	if config.Passive != nil {
		passive, err := t.translateUpstreamPassiveHealthCheckV2beta3(config.Passive)
		if err != nil {
			return err
		}
		hc.Passive = passive
	}

	if config.Active != nil {
		active, err := t.translateUpstreamActiveHealthCheckV2beta3(config.Active)
		if err != nil {
			return err
		}
		hc.Active = active
	} else {
		return &TranslateError{
			Field:  "healthCheck.active",
			Reason: "not exist",
		}
	}

	ups.Checks = &hc
	return nil
}

func (t translator) translateClientTLSV2beta3(config *configv2beta3.ApisixSecret, ups *apisixv1.Upstream) error {
	if config == nil {
		return nil
	}
	s, err := t.SecretLister.Secrets(config.Namespace).Get(config.Name)
	if err != nil {
		return &TranslateError{
			Field:  "tlsSecret",
			Reason: fmt.Sprintf("get secret failed, %v", err),
		}
	}
	cert, key, err := ExtractKeyPair(s, true)
	if err != nil {
		return &TranslateError{
			Field:  "tlsSecret",
			Reason: fmt.Sprintf("extract cert and key from secret failed, %v", err),
		}
	}
	ups.TLS = &apisixv1.ClientTLS{
		Cert: string(cert),
		Key:  string(key),
	}
	return nil
}

func (t *translator) translateUpstreamActiveHealthCheckV2beta3(config *configv2beta3.ActiveHealthCheck) (*apisixv1.UpstreamActiveHealthCheck, error) {
	var active apisixv1.UpstreamActiveHealthCheck
	switch config.Type {
	case apisixv1.HealthCheckHTTP, apisixv1.HealthCheckHTTPS, apisixv1.HealthCheckTCP:
		active.Type = config.Type
	case "":
		active.Type = apisixv1.HealthCheckHTTP
	default:
		return nil, &TranslateError{
			Field:  "healthCheck.active.Type",
			Reason: "invalid value",
		}
	}

	active.Timeout = int(config.Timeout.Seconds())
	if config.Port < 0 || config.Port > 65535 {
		return nil, &TranslateError{
			Field:  "healthCheck.active.port",
			Reason: "invalid value",
		}
	} else {
		active.Port = config.Port
	}
	if config.Concurrency < 0 {
		return nil, &TranslateError{
			Field:  "healthCheck.active.concurrency",
			Reason: "invalid value",
		}
	} else {
		active.Concurrency = config.Concurrency
	}
	active.Host = config.Host
	active.HTTPPath = config.HTTPPath
	active.HTTPRequestHeaders = config.RequestHeaders

	if config.StrictTLS == nil || *config.StrictTLS {
		active.HTTPSVerifyCert = true
	}

	if config.Healthy != nil {
		if config.Healthy.Successes < 0 || config.Healthy.Successes > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &TranslateError{
				Field:  "healthCheck.active.healthy.successes",
				Reason: "invalid value",
			}
		}
		active.Healthy.Successes = config.Healthy.Successes
		if config.Healthy.HTTPCodes != nil && len(config.Healthy.HTTPCodes) < 1 {
			return nil, &TranslateError{
				Field:  "healthCheck.active.healthy.httpCodes",
				Reason: "empty",
			}
		}
		active.Healthy.HTTPStatuses = config.Healthy.HTTPCodes

		if config.Healthy.Interval.Duration < apisixv1.ActiveHealthCheckMinInterval {
			return nil, &TranslateError{
				Field:  "healthCheck.active.healthy.interval",
				Reason: "invalid value",
			}
		}
		active.Healthy.Interval = int(config.Healthy.Interval.Seconds())
	}

	if config.Unhealthy != nil {
		if config.Unhealthy.HTTPFailures < 0 || config.Unhealthy.HTTPFailures > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &TranslateError{
				Field:  "healthCheck.active.unhealthy.httpFailures",
				Reason: "invalid value",
			}
		}
		active.Unhealthy.HTTPFailures = config.Unhealthy.HTTPFailures

		if config.Unhealthy.TCPFailures < 0 || config.Unhealthy.TCPFailures > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &TranslateError{
				Field:  "healthCheck.active.unhealthy.tcpFailures",
				Reason: "invalid value",
			}
		}
		active.Unhealthy.TCPFailures = config.Unhealthy.TCPFailures
		active.Unhealthy.Timeouts = config.Unhealthy.Timeouts

		if config.Unhealthy.HTTPCodes != nil && len(config.Unhealthy.HTTPCodes) < 1 {
			return nil, &TranslateError{
				Field:  "healthCheck.active.unhealthy.httpCodes",
				Reason: "empty",
			}
		}
		active.Unhealthy.HTTPStatuses = config.Unhealthy.HTTPCodes

		if config.Unhealthy.Interval.Duration < apisixv1.ActiveHealthCheckMinInterval {
			return nil, &TranslateError{
				Field:  "healthCheck.active.unhealthy.interval",
				Reason: "invalid value",
			}
		}
		active.Unhealthy.Interval = int(config.Unhealthy.Interval.Seconds())
	}

	return &active, nil
}

func (t *translator) translateUpstreamPassiveHealthCheckV2beta3(config *configv2beta3.PassiveHealthCheck) (*apisixv1.UpstreamPassiveHealthCheck, error) {
	var passive apisixv1.UpstreamPassiveHealthCheck
	switch config.Type {
	case apisixv1.HealthCheckHTTP, apisixv1.HealthCheckHTTPS, apisixv1.HealthCheckTCP:
		passive.Type = config.Type
	case "":
		passive.Type = apisixv1.HealthCheckHTTP
	default:
		return nil, &TranslateError{
			Field:  "healthCheck.passive.Type",
			Reason: "invalid value",
		}
	}
	if config.Healthy != nil {
		// zero means use the default value.
		if config.Healthy.Successes < 0 || config.Healthy.Successes > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &TranslateError{
				Field:  "healthCheck.passive.healthy.successes",
				Reason: "invalid value",
			}
		}
		passive.Healthy.Successes = config.Healthy.Successes
		if config.Healthy.HTTPCodes != nil && len(config.Healthy.HTTPCodes) < 1 {
			return nil, &TranslateError{
				Field:  "healthCheck.passive.healthy.httpCodes",
				Reason: "empty",
			}
		}
		passive.Healthy.HTTPStatuses = config.Healthy.HTTPCodes
	}

	if config.Unhealthy != nil {
		if config.Unhealthy.HTTPFailures < 0 || config.Unhealthy.HTTPFailures > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &TranslateError{
				Field:  "healthCheck.passive.unhealthy.httpFailures",
				Reason: "invalid value",
			}
		}
		passive.Unhealthy.HTTPFailures = config.Unhealthy.HTTPFailures

		if config.Unhealthy.TCPFailures < 0 || config.Unhealthy.TCPFailures > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &TranslateError{
				Field:  "healthCheck.passive.unhealthy.tcpFailures",
				Reason: "invalid value",
			}
		}
		passive.Unhealthy.TCPFailures = config.Unhealthy.TCPFailures
		passive.Unhealthy.Timeouts = config.Unhealthy.Timeouts

		if config.Unhealthy.HTTPCodes != nil && len(config.Unhealthy.HTTPCodes) < 1 {
			return nil, &TranslateError{
				Field:  "healthCheck.passive.unhealthy.httpCodes",
				Reason: "empty",
			}
		}
		passive.Unhealthy.HTTPStatuses = config.Unhealthy.HTTPCodes
	}
	return &passive, nil
}

func (t *translator) translateUpstreamRetriesAndTimeoutV2(retries *int, timeout *configv2.UpstreamTimeout, ups *apisixv1.Upstream) error {
	if retries != nil && *retries < 0 {
		return &TranslateError{
			Field:  "retries",
			Reason: "invalid value",
		}
	}
	ups.Retries = retries
	if timeout == nil {
		return nil
	}

	// Since the schema of timeout doesn't allow only configuring
	// one or two items. Here we assign the default value first.
	connTimeout := apisixv1.DefaultUpstreamTimeout
	readTimeout := apisixv1.DefaultUpstreamTimeout
	sendTimeout := apisixv1.DefaultUpstreamTimeout
	if timeout.Connect.Duration < 0 {
		return &TranslateError{
			Field:  "timeout.connect",
			Reason: "invalid value",
		}
	} else if timeout.Connect.Duration > 0 {
		connTimeout = int(timeout.Connect.Seconds())
	}
	if timeout.Read.Duration < 0 {
		return &TranslateError{
			Field:  "timeout.read",
			Reason: "invalid value",
		}
	} else if timeout.Read.Duration > 0 {
		readTimeout = int(timeout.Read.Seconds())
	}
	if timeout.Send.Duration < 0 {
		return &TranslateError{
			Field:  "timeout.send",
			Reason: "invalid value",
		}
	} else if timeout.Send.Duration > 0 {
		sendTimeout = int(timeout.Send.Seconds())
	}
	ups.Timeout = &apisixv1.UpstreamTimeout{
		Connect: connTimeout,
		Send:    sendTimeout,
		Read:    readTimeout,
	}
	return nil
}

func (t *translator) translateUpstreamLoadBalancerV2(lb *configv2.LoadBalancer, ups *apisixv1.Upstream) error {
	if lb == nil || lb.Type == "" {
		ups.Type = apisixv1.LbRoundRobin
		return nil
	}
	switch lb.Type {
	case apisixv1.LbRoundRobin, apisixv1.LbLeastConn, apisixv1.LbEwma:
		ups.Type = lb.Type
	case apisixv1.LbConsistentHash:
		ups.Type = lb.Type
		ups.Key = lb.Key
		switch lb.HashOn {
		case apisixv1.HashOnVars:
			fallthrough
		case apisixv1.HashOnHeader:
			fallthrough
		case apisixv1.HashOnCookie:
			fallthrough
		case apisixv1.HashOnConsumer:
			fallthrough
		case apisixv1.HashOnVarsCombination:
			ups.HashOn = lb.HashOn
		default:
			return &TranslateError{Field: "loadbalancer.hashOn", Reason: "invalid value"}
		}
	default:
		return &TranslateError{
			Field:  "loadbalancer.type",
			Reason: "invalid value",
		}
	}
	return nil
}

func (t *translator) translateUpstreamHealthCheckV2(config *configv2.HealthCheck, ups *apisixv1.Upstream) error {
	if config == nil || (config.Passive == nil && config.Active == nil) {
		return nil
	}
	var hc apisixv1.UpstreamHealthCheck
	if config.Passive != nil {
		passive, err := t.translateUpstreamPassiveHealthCheckV2(config.Passive)
		if err != nil {
			return err
		}
		hc.Passive = passive
	}

	if config.Active != nil {
		active, err := t.translateUpstreamActiveHealthCheckV2(config.Active)
		if err != nil {
			return err
		}
		hc.Active = active
	} else {
		return &TranslateError{
			Field:  "healthCheck.active",
			Reason: "not exist",
		}
	}

	ups.Checks = &hc
	return nil
}

func (t *translator) translateClientTLSV2(config *configv2.ApisixSecret, ups *apisixv1.Upstream) error {
	if config == nil {
		return nil
	}
	s, err := t.SecretLister.Secrets(config.Namespace).Get(config.Name)
	if err != nil {
		return &TranslateError{
			Field:  "tlsSecret",
			Reason: fmt.Sprintf("get secret failed, %v", err),
		}
	}
	cert, key, err := ExtractKeyPair(s, true)
	if err != nil {
		return &TranslateError{
			Field:  "tlsSecret",
			Reason: fmt.Sprintf("extract cert and key from secret failed, %v", err),
		}
	}
	ups.TLS = &apisixv1.ClientTLS{
		Cert: string(cert),
		Key:  string(key),
	}
	return nil
}

func (t *translator) translateUpstreamActiveHealthCheckV2(config *configv2.ActiveHealthCheck) (*apisixv1.UpstreamActiveHealthCheck, error) {
	var active apisixv1.UpstreamActiveHealthCheck
	switch config.Type {
	case apisixv1.HealthCheckHTTP, apisixv1.HealthCheckHTTPS, apisixv1.HealthCheckTCP:
		active.Type = config.Type
	case "":
		active.Type = apisixv1.HealthCheckHTTP
	default:
		return nil, &TranslateError{
			Field:  "healthCheck.active.Type",
			Reason: "invalid value",
		}
	}

	active.Timeout = int(config.Timeout.Seconds())
	if config.Port < 0 || config.Port > 65535 {
		return nil, &TranslateError{
			Field:  "healthCheck.active.port",
			Reason: "invalid value",
		}
	} else {
		active.Port = config.Port
	}
	if config.Concurrency < 0 {
		return nil, &TranslateError{
			Field:  "healthCheck.active.concurrency",
			Reason: "invalid value",
		}
	} else {
		active.Concurrency = config.Concurrency
	}
	active.Host = config.Host
	active.HTTPPath = config.HTTPPath
	active.HTTPRequestHeaders = config.RequestHeaders

	if config.StrictTLS == nil || *config.StrictTLS {
		active.HTTPSVerifyCert = true
	}

	if config.Healthy != nil {
		if config.Healthy.Successes < 0 || config.Healthy.Successes > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &TranslateError{
				Field:  "healthCheck.active.healthy.successes",
				Reason: "invalid value",
			}
		}
		active.Healthy.Successes = config.Healthy.Successes
		if config.Healthy.HTTPCodes != nil && len(config.Healthy.HTTPCodes) < 1 {
			return nil, &TranslateError{
				Field:  "healthCheck.active.healthy.httpCodes",
				Reason: "empty",
			}
		}
		active.Healthy.HTTPStatuses = config.Healthy.HTTPCodes

		if config.Healthy.Interval.Duration < apisixv1.ActiveHealthCheckMinInterval {
			return nil, &TranslateError{
				Field:  "healthCheck.active.healthy.interval",
				Reason: "invalid value",
			}
		}
		active.Healthy.Interval = int(config.Healthy.Interval.Seconds())
	}

	if config.Unhealthy != nil {
		if config.Unhealthy.HTTPFailures < 0 || config.Unhealthy.HTTPFailures > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &TranslateError{
				Field:  "healthCheck.active.unhealthy.httpFailures",
				Reason: "invalid value",
			}
		}
		active.Unhealthy.HTTPFailures = config.Unhealthy.HTTPFailures

		if config.Unhealthy.TCPFailures < 0 || config.Unhealthy.TCPFailures > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &TranslateError{
				Field:  "healthCheck.active.unhealthy.tcpFailures",
				Reason: "invalid value",
			}
		}
		active.Unhealthy.TCPFailures = config.Unhealthy.TCPFailures
		active.Unhealthy.Timeouts = config.Unhealthy.Timeouts

		if config.Unhealthy.HTTPCodes != nil && len(config.Unhealthy.HTTPCodes) < 1 {
			return nil, &TranslateError{
				Field:  "healthCheck.active.unhealthy.httpCodes",
				Reason: "empty",
			}
		}
		active.Unhealthy.HTTPStatuses = config.Unhealthy.HTTPCodes

		if config.Unhealthy.Interval.Duration < apisixv1.ActiveHealthCheckMinInterval {
			return nil, &TranslateError{
				Field:  "healthCheck.active.unhealthy.interval",
				Reason: "invalid value",
			}
		}
		active.Unhealthy.Interval = int(config.Unhealthy.Interval.Seconds())
	}

	return &active, nil
}

func (t *translator) translateUpstreamPassiveHealthCheckV2(config *configv2.PassiveHealthCheck) (*apisixv1.UpstreamPassiveHealthCheck, error) {
	var passive apisixv1.UpstreamPassiveHealthCheck
	switch config.Type {
	case apisixv1.HealthCheckHTTP, apisixv1.HealthCheckHTTPS, apisixv1.HealthCheckTCP:
		passive.Type = config.Type
	case "":
		passive.Type = apisixv1.HealthCheckHTTP
	default:
		return nil, &TranslateError{
			Field:  "healthCheck.passive.Type",
			Reason: "invalid value",
		}
	}
	if config.Healthy != nil {
		// zero means use the default value.
		if config.Healthy.Successes < 0 || config.Healthy.Successes > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &TranslateError{
				Field:  "healthCheck.passive.healthy.successes",
				Reason: "invalid value",
			}
		}
		passive.Healthy.Successes = config.Healthy.Successes
		if config.Healthy.HTTPCodes != nil && len(config.Healthy.HTTPCodes) < 1 {
			return nil, &TranslateError{
				Field:  "healthCheck.passive.healthy.httpCodes",
				Reason: "empty",
			}
		}
		passive.Healthy.HTTPStatuses = config.Healthy.HTTPCodes
	}

	if config.Unhealthy != nil {
		if config.Unhealthy.HTTPFailures < 0 || config.Unhealthy.HTTPFailures > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &TranslateError{
				Field:  "healthCheck.passive.unhealthy.httpFailures",
				Reason: "invalid value",
			}
		}
		passive.Unhealthy.HTTPFailures = config.Unhealthy.HTTPFailures

		if config.Unhealthy.TCPFailures < 0 || config.Unhealthy.TCPFailures > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &TranslateError{
				Field:  "healthCheck.passive.unhealthy.tcpFailures",
				Reason: "invalid value",
			}
		}
		passive.Unhealthy.TCPFailures = config.Unhealthy.TCPFailures
		passive.Unhealthy.Timeouts = config.Unhealthy.Timeouts

		if config.Unhealthy.HTTPCodes != nil && len(config.Unhealthy.HTTPCodes) < 1 {
			return nil, &TranslateError{
				Field:  "healthCheck.passive.unhealthy.httpCodes",
				Reason: "empty",
			}
		}
		passive.Unhealthy.HTTPStatuses = config.Unhealthy.HTTPCodes
	}
	return &passive, nil
}
