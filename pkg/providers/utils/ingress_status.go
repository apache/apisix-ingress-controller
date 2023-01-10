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
package utils

import (
	"fmt"
	"net"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/log"
)

const (
	_gatewayLBNotReadyMessage = "The LoadBalancer used by the APISIX gateway is not yet ready"
)

// IngressPublishAddresses get addressed used to expose Ingress
func IngressPublishAddresses(ingressPublishService string, ingressStatusAddress []string, svcLister listerscorev1.ServiceLister) ([]string, error) {
	addrs := []string{}

	// if ingressStatusAddress is specified, it will be used first
	if len(ingressStatusAddress) > 0 {
		addrs = append(addrs, ingressStatusAddress...)
		return addrs, nil
	}

	namespace, name, err := cache.SplitMetaNamespaceKey(ingressPublishService)
	if err != nil {
		log.Errorf("invalid ingressPublishService %s: %s", ingressPublishService, err)
		return nil, err
	}

	svc, err := svcLister.Services(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	switch svc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		if len(svc.Status.LoadBalancer.Ingress) < 1 {
			return addrs, fmt.Errorf("%s", _gatewayLBNotReadyMessage)
		}

		for _, ip := range svc.Status.LoadBalancer.Ingress {
			if ip.IP == "" {
				// typically AWS load-balancers
				addrs = append(addrs, ip.Hostname)
			} else {
				addrs = append(addrs, ip.IP)
			}
		}

		addrs = append(addrs, svc.Spec.ExternalIPs...)
		return addrs, nil
	default:
		return addrs, nil
	}

}

// IngressLBStatusIPs organizes the available addresses
func IngressLBStatusIPs(ingressPublishService string, ingressStatusAddress []string, svcLister listerscorev1.ServiceLister) ([]corev1.LoadBalancerIngress, error) {
	lbips := []corev1.LoadBalancerIngress{}
	var ips []string

	for {
		var err error
		ips, err = IngressPublishAddresses(ingressPublishService, ingressStatusAddress, svcLister)
		if err != nil {
			if err.Error() == _gatewayLBNotReadyMessage {
				log.Warnf("%s. Provided service: %s", _gatewayLBNotReadyMessage, ingressPublishService)
				time.Sleep(time.Second)
				continue
			}

			return nil, err
		}
		break
	}

	for _, ip := range ips {
		if net.ParseIP(ip) == nil {
			lbips = append(lbips, corev1.LoadBalancerIngress{Hostname: ip})
		} else {
			lbips = append(lbips, corev1.LoadBalancerIngress{IP: ip})
		}

	}

	return lbips, nil
}

func CompareLoadBalancerIngressEqual(lb1 []corev1.LoadBalancerIngress, lb2 []corev1.LoadBalancerIngress) bool {
	if len(lb1) != len(lb2) {
		return false
	}
	addrs := []string{}
	addrs2 := []string{}
	for _, lb := range lb1 {
		if lb.IP != "" {
			addrs = append(addrs, lb.IP)
		}
		if lb.Hostname != "" {
			addrs = append(addrs, lb.Hostname)
		}
	}
	for _, lb := range lb2 {
		if lb.IP != "" {
			addrs2 = append(addrs2, lb.IP)
		}
		if lb.Hostname != "" {
			addrs2 = append(addrs2, lb.Hostname)
		}
	}
	if len(addrs) != len(addrs2) {
		return false
	}
	sort.Strings(addrs)
	sort.Strings(addrs2)
	for i := 0; i < len(addrs); i++ {
		if addrs[i] != addrs2[i] {
			return false
		}
	}
	return true
}
