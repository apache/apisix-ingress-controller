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
package kube

import (
	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) translateUpstreamScheme(scheme string, ups *apisixv1.Upstream) error {
	if scheme == "" {
		ups.Scheme = apisixv1.SchemeHTTP
		return nil
	}
	switch scheme {
	case apisixv1.SchemeHTTP, apisixv1.SchemeGRPC:
		ups.Scheme = scheme
		return nil
	default:
		return &translateError{field: "scheme", reason: "invalid value"}
	}
}

func (t *translator) translateUpstreamLoadBalancer(lb *configv1.LoadBalancer, ups *apisixv1.Upstream) error {
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
			return &translateError{field: "loadbalancer.hashOn", reason: "invalid value"}
		}
	default:
		return &translateError{
			field:  "loadbalancer.type",
			reason: "invalid value",
		}
	}
	return nil
}

func (t *translator) translateUpstreamHealthCheck(config *configv1.HealthCheck, ups *apisixv1.Upstream) error {
	if config == nil || (config.Passive == nil && config.Active == nil) {
		return nil
	}
	var hc apisixv1.UpstreamHealthCheck
	if config.Passive != nil {
		passive, err := t.translateUpstreamPassiveHealthCheck(config.Passive)
		if err != nil {
			return err
		}
		hc.Passive = passive
	}

	if config.Active != nil {
		active, err := t.translateUpstreamActiveHealthCheck(config.Active)
		if err != nil {
			return err
		}
		hc.Active = active
	} else {
		return &translateError{
			field:  "healthCheck.active",
			reason: "not exist",
		}
	}

	ups.Checks = &hc
	return nil
}

func (t *translator) translateUpstreamActiveHealthCheck(config *configv1.ActiveHealthCheck) (*apisixv1.UpstreamActiveHealthCheck, error) {
	var active apisixv1.UpstreamActiveHealthCheck
	switch config.Type {
	case apisixv1.HealthCheckHTTP, apisixv1.HealthCheckHTTPS, apisixv1.HealthCheckTCP:
		active.Type = config.Type
	default:
		return nil, &translateError{
			field:  "healthCheck.active.Type",
			reason: "invalid value",
		}
	}

	active.Timeout = int(config.Timeout.Seconds())
	if config.Port < 0 || config.Port > 65535 {
		return nil, &translateError{
			field:  "healthCheck.active.port",
			reason: "invalid value",
		}
	} else {
		active.Port = config.Port
	}
	if config.Concurrency < 0 {
		return nil, &translateError{
			field:  "healthCheck.active.concurrency",
			reason: "invalid value",
		}
	} else {
		active.Concurrency = config.Concurrency
	}
	active.Host = config.Host
	active.HTTPPath = config.HTTPPath
	active.HTTPRequestHeaders = config.RequestHeaders

	if config.StrictTLS == nil || *config.StrictTLS == true {
		active.HTTPSVerifyCert = true
	}

	if config.Healthy != nil {
		if config.Healthy.Successes < 0 || config.Healthy.Successes > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &translateError{
				field:  "healthCheck.active.healthy.successes",
				reason: "invalid value",
			}
		}
		active.Healthy.Successes = config.Healthy.Successes
		if config.Healthy.HTTPCodes != nil && len(config.Healthy.HTTPCodes) < 1 {
			return nil, &translateError{
				field:  "healthCheck.active.healthy.httpCodes",
				reason: "empty",
			}
		}
		active.Healthy.HTTPStatuses = config.Healthy.HTTPCodes

		if config.Healthy.Interval != 0 && config.Healthy.Interval < apisixv1.ActiveHealthCheckMinInterval {
			return nil, &translateError{
				field:  "healthCheck.active.healthy.interval",
				reason: "invalid value",
			}
		}
		active.Healthy.Interval = int(config.Healthy.Interval.Seconds())
	}

	if config.Unhealthy != nil {
		if config.Unhealthy.HTTPFailures < 0 || config.Unhealthy.HTTPFailures > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &translateError{
				field:  "healthCheck.active.unhealthy.httpFailures",
				reason: "invalid value",
			}
		}
		active.Unhealthy.HTTPFailures = config.Unhealthy.HTTPFailures

		if config.Unhealthy.TCPFailures < 0 || config.Unhealthy.TCPFailures > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &translateError{
				field:  "healthCheck.active.unhealthy.tcpFailures",
				reason: "invalid value",
			}
		}
		active.Unhealthy.TCPFailures = config.Unhealthy.TCPFailures
		active.Unhealthy.Timeouts = config.Unhealthy.Timeout.Seconds()

		if config.Unhealthy.HTTPCodes != nil && len(config.Unhealthy.HTTPCodes) < 1 {
			return nil, &translateError{
				field:  "healthCheck.active.unhealthy.httpCodes",
				reason: "empty",
			}
		}
		active.Unhealthy.HTTPStatuses = config.Unhealthy.HTTPCodes

		if config.Unhealthy.Interval != 0 && config.Unhealthy.Interval < apisixv1.ActiveHealthCheckMinInterval {
			return nil, &translateError{
				field:  "healthCheck.active.unhealthy.interval",
				reason: "invalid value",
			}
		}
		active.Unhealthy.Interval = int(config.Unhealthy.Interval.Seconds())
	}

	return &active, nil
}

func (t *translator) translateUpstreamPassiveHealthCheck(config *configv1.PassiveHealthCheck) (*apisixv1.UpstreamPassiveHealthCheck, error) {
	var passive apisixv1.UpstreamPassiveHealthCheck
	switch config.Type {
	case apisixv1.HealthCheckHTTP, apisixv1.HealthCheckHTTPS, apisixv1.HealthCheckTCP:
		passive.Type = config.Type
	default:
		return nil, &translateError{
			field:  "healthCheck.passive.Type",
			reason: "invalid value",
		}
	}
	if config.Healthy != nil {
		// zero means use the default value.
		if config.Healthy.Successes < 0 || config.Healthy.Successes > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &translateError{
				field:  "healthCheck.passive.healthy.successes",
				reason: "invalid value",
			}
		}
		passive.Healthy.Successes = config.Healthy.Successes
		if config.Healthy.HTTPCodes != nil && len(config.Healthy.HTTPCodes) < 1 {
			return nil, &translateError{
				field:  "healthCheck.passive.healthy.httpCodes",
				reason: "empty",
			}
		}
		passive.Healthy.HTTPStatuses = config.Healthy.HTTPCodes
	}

	if config.Unhealthy != nil {
		if config.Unhealthy.HTTPFailures < 0 || config.Unhealthy.HTTPFailures > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &translateError{
				field:  "healthCheck.passive.unhealthy.httpFailures",
				reason: "invalid value",
			}
		}
		passive.Unhealthy.HTTPFailures = config.Unhealthy.HTTPFailures

		if config.Unhealthy.TCPFailures < 0 || config.Unhealthy.TCPFailures > apisixv1.HealthCheckMaxConsecutiveNumber {
			return nil, &translateError{
				field:  "healthCheck.passive.unhealthy.tcpFailures",
				reason: "invalid value",
			}
		}
		passive.Unhealthy.TCPFailures = config.Unhealthy.TCPFailures
		passive.Unhealthy.Timeouts = config.Unhealthy.Timeout.Seconds()

		if config.Unhealthy.HTTPCodes != nil && len(config.Unhealthy.HTTPCodes) < 1 {
			return nil, &translateError{
				field:  "healthCheck.passive.unhealthy.httpCodes",
				reason: "empty",
			}
		}
		passive.Unhealthy.HTTPStatuses = config.Unhealthy.HTTPCodes
	}
	return &passive, nil
}
