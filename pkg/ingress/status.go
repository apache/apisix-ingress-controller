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

package ingress

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	configv2beta1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta1"
	configv2beta2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta2"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

const (
	_conditionType            = "ResourcesAvailable"
	_commonSuccessMessage     = "Sync Successfully"
	_gatewayLBNotReadyMessage = "The LoadBalancer used by the APISIX gateway is not yet ready"
)

// verifyGeneration verify generation to decide whether to update status
func (c *Controller) verifyGeneration(conditions *[]metav1.Condition, newCondition metav1.Condition) bool {
	existingCondition := meta.FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition != nil && existingCondition.ObservedGeneration >= newCondition.ObservedGeneration {
		return false
	}
	return true
}

// recordStatus record resources status
func (c *Controller) recordStatus(at interface{}, reason string, err error, status v1.ConditionStatus, generation int64) {
	// build condition
	message := _commonSuccessMessage
	if err != nil {
		message = err.Error()
	}
	condition := metav1.Condition{
		Type:               _conditionType,
		Reason:             reason,
		Status:             status,
		Message:            message,
		ObservedGeneration: generation,
	}
	client := c.kubeClient.APISIXClient
	kubeClient := c.kubeClient.Client

	switch v := at.(type) {
	case *configv1.ApisixTls:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = &conditions
		}
		if c.verifyGeneration(v.Status.Conditions, condition) {
			meta.SetStatusCondition(v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV1().ApisixTlses(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixTls",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv1.ApisixUpstream:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = &conditions
		}
		if c.verifyGeneration(v.Status.Conditions, condition) {
			meta.SetStatusCondition(v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV1().ApisixUpstreams(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixUpstream",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2alpha1.ApisixRoute:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = &conditions
		}
		if c.verifyGeneration(v.Status.Conditions, condition) {
			meta.SetStatusCondition(v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2alpha1().ApisixRoutes(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixRoute",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2beta1.ApisixRoute:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta1().ApisixRoutes(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixRoute",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2beta2.ApisixRoute:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta2().ApisixRoutes(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixRoute",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2alpha1.ApisixConsumer:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = &conditions
		}
		if c.verifyGeneration(v.Status.Conditions, condition) {
			meta.SetStatusCondition(v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2alpha1().ApisixConsumers(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixConsumer",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *networkingv1.Ingress:
		// set to status
		lbips, err := c.ingressLBStatusIPs()
		if err != nil {
			log.Errorw("failed to get APISIX gateway external IPs",
				zap.Error(err),
			)

		}

		v.Status.LoadBalancer.Ingress = lbips
		if _, errRecord := kubeClient.NetworkingV1().Ingresses(v.Namespace).UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
			log.Errorw("failed to record status change for IngressV1",
				zap.Error(errRecord),
				zap.String("name", v.Name),
				zap.String("namespace", v.Namespace),
			)
		}

	case *networkingv1beta1.Ingress:
		// set to status
		lbips, err := c.ingressLBStatusIPs()
		if err != nil {
			log.Errorw("failed to get APISIX gateway external IPs",
				zap.Error(err),
			)

		}

		v.Status.LoadBalancer.Ingress = lbips
		if _, errRecord := kubeClient.NetworkingV1beta1().Ingresses(v.Namespace).UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
			log.Errorw("failed to record status change for IngressV1",
				zap.Error(errRecord),
				zap.String("name", v.Name),
				zap.String("namespace", v.Namespace),
			)
		}
	case *extensionsv1beta1.Ingress:
		// set to status
		lbips, err := c.ingressLBStatusIPs()
		if err != nil {
			log.Errorw("failed to get APISIX gateway external IPs",
				zap.Error(err),
			)

		}

		v.Status.LoadBalancer.Ingress = lbips
		if _, errRecord := kubeClient.ExtensionsV1beta1().Ingresses(v.Namespace).UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
			log.Errorw("failed to record status change for IngressV1",
				zap.Error(errRecord),
				zap.String("name", v.Name),
				zap.String("namespace", v.Namespace),
			)
		}
	default:
		// This should not be executed
		log.Errorf("unsupported resource record: %s", v)
	}
}

// ingressPublishAddresses get addressed used to expose Ingress
func (c *Controller) ingressPublishAddresses() ([]string, error) {
	ingressPublishService := c.cfg.IngressPublishService
	ingressStatusAddress := c.cfg.IngressStatusAddress
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

	kubeClient := c.kubeClient.Client
	svc, err := kubeClient.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	switch svc.Spec.Type {
	case apiv1.ServiceTypeLoadBalancer:
		if len(svc.Status.LoadBalancer.Ingress) < 1 {
			return addrs, fmt.Errorf(_gatewayLBNotReadyMessage)
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

// ingressLBStatusIPs organizes the available addresses
func (c *Controller) ingressLBStatusIPs() ([]apiv1.LoadBalancerIngress, error) {
	lbips := []apiv1.LoadBalancerIngress{}
	var ips []string

	for {
		var err error
		ips, err = c.ingressPublishAddresses()
		if err != nil {
			if err.Error() == _gatewayLBNotReadyMessage {
				log.Warnf("%s. Provided service: %s", _gatewayLBNotReadyMessage, c.cfg.IngressPublishService)
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
