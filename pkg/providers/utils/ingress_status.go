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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

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

func lessNetworkingV1LB(addrs []networkingv1.IngressLoadBalancerIngress) func(int, int) bool {
	return func(a, b int) bool {
		switch strings.Compare(addrs[a].Hostname, addrs[b].Hostname) {
		case -1:
			return true
		case 1:
			return false
		}
		return addrs[a].IP < addrs[b].IP
	}
}

func lessNetworkingV1beta1LB(addrs []networkingv1beta1.IngressLoadBalancerIngress) func(int, int) bool {
	return func(a, b int) bool {
		switch strings.Compare(addrs[a].Hostname, addrs[b].Hostname) {
		case -1:
			return true
		case 1:
			return false
		}
		return addrs[a].IP < addrs[b].IP
	}
}

func lessExtensionsV1beta1LB(addrs []extensionsv1beta1.IngressLoadBalancerIngress) func(int, int) bool {
	return func(a, b int) bool {
		switch strings.Compare(addrs[a].Hostname, addrs[b].Hostname) {
		case -1:
			return true
		case 1:
			return false
		}
		return addrs[a].IP < addrs[b].IP
	}
}

func CompareNetworkingV1LBEqual(lb1 []networkingv1.IngressLoadBalancerIngress, lb2 []networkingv1.IngressLoadBalancerIngress) bool {
	if len(lb1) != len(lb2) {
		return false
	}
	sort.SliceStable(lb1, lessNetworkingV1LB(lb1))
	sort.SliceStable(lb2, lessNetworkingV1LB(lb2))
	size := len(lb1)
	for i := 0; i < size; i++ {
		if lb1[i].IP != lb2[i].IP {
			return false
		}
		if lb1[i].Hostname != lb2[i].Hostname {
			return false
		}
	}
	return true
}

func CompareNetworkingV1beta1LBEqual(lb1 []networkingv1beta1.IngressLoadBalancerIngress, lb2 []networkingv1beta1.IngressLoadBalancerIngress) bool {
	if len(lb1) != len(lb2) {
		return false
	}
	sort.SliceStable(lb1, lessNetworkingV1beta1LB(lb1))
	sort.SliceStable(lb2, lessNetworkingV1beta1LB(lb2))
	size := len(lb1)
	for i := 0; i < size; i++ {
		if lb1[i].IP != lb2[i].IP {
			return false
		}
		if lb1[i].Hostname != lb2[i].Hostname {
			return false
		}
	}
	return true
}

func CompareExtensionsV1beta1LBEqual(lb1 []extensionsv1beta1.IngressLoadBalancerIngress, lb2 []extensionsv1beta1.IngressLoadBalancerIngress) bool {
	if len(lb1) != len(lb2) {
		return false
	}
	sort.SliceStable(lb1, lessExtensionsV1beta1LB(lb1))
	sort.SliceStable(lb2, lessExtensionsV1beta1LB(lb2))
	size := len(lb1)
	for i := 0; i < size; i++ {
		if lb1[i].IP != lb2[i].IP {
			return false
		}
		if lb1[i].Hostname != lb2[i].Hostname {
			return false
		}
	}
	return true
}

// CoreV1ToNetworkV1LB convert []corev1.LoadBalancerIngress to []networkingv1.IngressLoadBalancerIngress
func CoreV1ToNetworkV1LB(lbips []corev1.LoadBalancerIngress) []networkingv1.IngressLoadBalancerIngress {
	t := make([]networkingv1.IngressLoadBalancerIngress, 0, len(lbips))
	for _, lbip := range lbips {
		t = append(t, networkingv1.IngressLoadBalancerIngress{
			Hostname: lbip.Hostname,
			IP:       lbip.IP,
		})
	}
	return t
}

// CoreV1ToNetworkV1beta1LB convert []corev1.LoadBalancerIngress to []networkingv1beta1.IngressLoadBalancerIngress
func CoreV1ToNetworkV1beta1LB(lbips []corev1.LoadBalancerIngress) []networkingv1beta1.IngressLoadBalancerIngress {
	t := make([]networkingv1beta1.IngressLoadBalancerIngress, 0, len(lbips))
	for _, lbip := range lbips {
		t = append(t, networkingv1beta1.IngressLoadBalancerIngress{
			Hostname: lbip.Hostname,
			IP:       lbip.IP,
		})
	}
	return t
}

// CoreV1ToExtensionsV1beta1LB convert []corev1.LoadBalancerIngress to []extensionsv1beta1.IngressLoadBalancerIngress
func CoreV1ToExtensionsV1beta1LB(lbips []corev1.LoadBalancerIngress) []extensionsv1beta1.IngressLoadBalancerIngress {
	t := make([]extensionsv1beta1.IngressLoadBalancerIngress, 0, len(lbips))
	for _, lbip := range lbips {
		t = append(t, extensionsv1beta1.IngressLoadBalancerIngress{
			Hostname: lbip.Hostname,
			IP:       lbip.IP,
		})
	}
	return t
}

func CoreV1ToGatewayV1beta1Addr(lbips []corev1.LoadBalancerIngress) []gatewayv1beta1.GatewayAddress {
	t := make([]gatewayv1beta1.GatewayAddress, 0, len(lbips))

	// In the definition, there is also an address type called NamedAddress,
	// which we currently do not implement
	HostnameAddressType := gatewayv1beta1.HostnameAddressType
	IPAddressType := gatewayv1beta1.IPAddressType

	for _, lbip := range lbips {
		if v := lbip.Hostname; v != "" {
			t = append(t, gatewayv1beta1.GatewayAddress{
				Type:  &HostnameAddressType,
				Value: v,
			})
		}

		if v := lbip.IP; v != "" {
			t = append(t, gatewayv1beta1.GatewayAddress{
				Type:  &IPAddressType,
				Value: v,
			})
		}
	}
	return t
}
