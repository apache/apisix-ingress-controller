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
//
package utils

import (
	"context"
	"fmt"
	"net"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/log"
)

const (
	_gatewayLBNotReadyMessage = "The LoadBalancer used by the APISIX gateway is not yet ready"
)

// IngressPublishAddresses get addressed used to expose Ingress
func IngressPublishAddresses(ingressPublishService string, ingressStatusAddress []string, kubeClient kubernetes.Interface) ([]string, error) {
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

	svc, err := kubeClient.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	switch svc.Spec.Type {
	case apiv1.ServiceTypeLoadBalancer:
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
func IngressLBStatusIPs(ingressPublishService string, ingressStatusAddress []string, kubeClient kubernetes.Interface) ([]apiv1.LoadBalancerIngress, error) {
	lbips := []apiv1.LoadBalancerIngress{}
	var ips []string

	for {
		var err error
		ips, err = IngressPublishAddresses(ingressPublishService, ingressStatusAddress, kubeClient)
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
			lbips = append(lbips, apiv1.LoadBalancerIngress{Hostname: ip})
		} else {
			lbips = append(lbips, apiv1.LoadBalancerIngress{IP: ip})
		}

	}

	return lbips, nil
}
