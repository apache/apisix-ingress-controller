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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
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
func (c *Controller) recordStatus(at interface{}, reason string, err error, status metav1.ConditionStatus, generation int64) {
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

	if kubeObj, ok := at.(runtime.Object); ok {
		at = kubeObj.DeepCopyObject()
	}

	switch v := at.(type) {
	case *configv2beta3.ApisixTls:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta3().ApisixTlses(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixTls",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2.ApisixTls:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2().ApisixTlses(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixTls",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2beta3.ApisixUpstream:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta3().ApisixUpstreams(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixUpstream",
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
	case *configv2beta3.ApisixRoute:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta3().ApisixRoutes(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixRoute",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2beta3.ApisixConsumer:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta3().ApisixConsumers(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixConsumer",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2.ApisixConsumer:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2().ApisixConsumers(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixConsumer",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2beta3.ApisixPluginConfig:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta3().ApisixPluginConfigs(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixPluginConfig",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2beta3.ApisixClusterConfig:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta3().ApisixClusterConfigs().
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixClusterConfig",
					zap.Error(errRecord),
					zap.String("name", v.Name),
				)
			}
		}
	case *configv2.ApisixClusterConfig:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2().ApisixClusterConfigs().
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixClusterConfig",
					zap.Error(errRecord),
					zap.String("name", v.Name),
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

		v.ObjectMeta.Generation = generation
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

		v.ObjectMeta.Generation = generation
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

		v.ObjectMeta.Generation = generation
		v.Status.LoadBalancer.Ingress = lbips
		if _, errRecord := kubeClient.ExtensionsV1beta1().Ingresses(v.Namespace).UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
			log.Errorw("failed to record status change for IngressV1",
				zap.Error(errRecord),
				zap.String("name", v.Name),
				zap.String("namespace", v.Namespace),
			)
		}
	case *gatewayv1alpha2.Gateway:
		gatewayCondition := metav1.Condition{
			Type:               string(gatewayv1alpha2.ListenerConditionReady),
			Reason:             reason,
			Status:             status,
			Message:            "Gateway's status has been successfully updated",
			ObservedGeneration: generation,
		}

		gatewayKubeClient := c.kubeClient.GatewayClient

		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		} else {
			meta.SetStatusCondition(&v.Status.Conditions, gatewayCondition)
		}

		lbips, err := c.ingressLBStatusIPs()
		if err != nil {
			log.Errorw("failed to get APISIX gateway external IPs",
				zap.Error(err),
			)
		}

		v.Status.Addresses = convLBIPToGatewayAddr(lbips)
		if _, errRecord := gatewayKubeClient.GatewayV1alpha2().Gateways(v.Namespace).UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
			log.Errorw("failed to record status change for Gateway resource",
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

// convLBIPToGatewayAddr convert LoadBalancerIngress to GatewayAddress format
func convLBIPToGatewayAddr(lbips []apiv1.LoadBalancerIngress) []gatewayv1alpha2.GatewayAddress {
	gas := []gatewayv1alpha2.GatewayAddress{}

	// In the definition, there is also an address type called NamedAddress,
	// which we currently do not implement
	HostnameAddressType := gatewayv1alpha2.HostnameAddressType
	IPAddressType := gatewayv1alpha2.IPAddressType

	for _, lbip := range lbips {
		if v := lbip.Hostname; v != "" {
			gas = append(gas, gatewayv1alpha2.GatewayAddress{
				Type:  &HostnameAddressType,
				Value: v,
			})
		}

		if v := lbip.IP; v != "" {
			gas = append(gas, gatewayv1alpha2.GatewayAddress{
				Type:  &IPAddressType,
				Value: v,
			})
		}
	}

	return gas
}
