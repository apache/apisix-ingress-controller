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
	"net"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	configv2beta1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta1"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

var (
	_errInvalidAddress = errors.New("address is neither IP or CIDR")
)

func (t *translator) getServiceClusterIPAndPort(backend *configv2alpha1.ApisixRouteHTTPBackend, ns string) (string, int32, error) {
	svc, err := t.ServiceLister.Services(ns).Get(backend.ServiceName)
	if err != nil {
		return "", 0, err
	}
	svcPort := int32(-1)
	if backend.ResolveGranularity == "service" && svc.Spec.ClusterIP == "" {
		log.Errorw("ApisixRoute refers to a headless service but want to use the service level resolve granularity",
			zap.Any("namespace", ns),
			zap.Any("service", svc),
		)
		return "", 0, errors.New("conflict headless service and backend resolve granularity")
	}
loop:
	for _, port := range svc.Spec.Ports {
		switch backend.ServicePort.Type {
		case intstr.Int:
			if backend.ServicePort.IntVal == port.Port {
				svcPort = port.Port
				break loop
			}
		case intstr.String:
			if backend.ServicePort.StrVal == port.Name {
				svcPort = port.Port
				break loop
			}
		}
	}
	if svcPort == -1 {
		log.Errorw("ApisixRoute refers to non-existent Service port",
			zap.String("namespace", ns),
			zap.String("port", backend.ServicePort.String()),
		)
		return "", 0, err
	}

	return svc.Spec.ClusterIP, svcPort, nil
}

func (t *translator) getTCPServiceClusterIPAndPort(backend *configv2alpha1.ApisixRouteTCPBackend, ar *configv2alpha1.ApisixRoute) (string, int32, error) {
	svc, err := t.ServiceLister.Services(ar.Namespace).Get(backend.ServiceName)
	if err != nil {
		return "", 0, err
	}
	svcPort := int32(-1)
	if backend.ResolveGranularity == "service" && svc.Spec.ClusterIP == "" {
		log.Errorw("ApisixRoute refers to a headless service but want to use the service level resolve granularity",
			zap.Any("ApisixRoute", ar),
			zap.Any("service", svc),
		)
		return "", 0, errors.New("conflict headless service and backend resolve granularity")
	}
loop:
	for _, port := range svc.Spec.Ports {
		switch backend.ServicePort.Type {
		case intstr.Int:
			if backend.ServicePort.IntVal == port.Port {
				svcPort = port.Port
				break loop
			}
		case intstr.String:
			if backend.ServicePort.StrVal == port.Name {
				svcPort = port.Port
				break loop
			}
		}
	}
	if svcPort == -1 {
		log.Errorw("ApisixRoute refers to non-existent Service port",
			zap.Any("ApisixRoute", ar),
			zap.String("port", backend.ServicePort.String()),
		)
		return "", 0, err
	}

	return svc.Spec.ClusterIP, svcPort, nil
}

// getStreamServiceClusterIPAndPort is for v2beta1 streamRoute
func (t *translator) getStreamServiceClusterIPAndPort(backend configv2beta1.ApisixRouteStreamBackend, ns string) (string, int32, error) {
	svc, err := t.ServiceLister.Services(ns).Get(backend.ServiceName)
	if err != nil {
		return "", 0, err
	}
	svcPort := int32(-1)
	if backend.ResolveGranularity == "service" && svc.Spec.ClusterIP == "" {
		log.Errorw("ApisixRoute refers to a headless service but want to use the service level resolve granularity",
			zap.String("ApisixRoute namespace", ns),
			zap.Any("service", svc),
		)
		return "", 0, errors.New("conflict headless service and backend resolve granularity")
	}
loop:
	for _, port := range svc.Spec.Ports {
		switch backend.ServicePort.Type {
		case intstr.Int:
			if backend.ServicePort.IntVal == port.Port {
				svcPort = port.Port
				break loop
			}
		case intstr.String:
			if backend.ServicePort.StrVal == port.Name {
				svcPort = port.Port
				break loop
			}
		}
	}
	if svcPort == -1 {
		log.Errorw("ApisixRoute refers to non-existent Service port",
			zap.String("ApisixRoute namespace", ns),
			zap.String("port", backend.ServicePort.String()),
		)
		return "", 0, err
	}

	return svc.Spec.ClusterIP, svcPort, nil
}

// translateUpstreamNotStrictly translates Upstream nodes with a loose way, only generate ID and Name for delete Event.
func (t *translator) translateUpstreamNotStrictly(namespace, svcName, subset string, svcPort int32) (*apisixv1.Upstream, error) {
	ups := &apisixv1.Upstream{}
	ups.Name = apisixv1.ComposeUpstreamName(namespace, svcName, subset, svcPort)
	ups.ID = id.GenID(ups.Name)
	return ups, nil
}

func (t *translator) translateUpstream(namespace, svcName, subset, svcResolveGranularity, svcClusterIP string, svcPort int32) (*apisixv1.Upstream, error) {
	ups, err := t.TranslateUpstream(namespace, svcName, subset, svcPort)
	if err != nil {
		return nil, err
	}
	if svcResolveGranularity == "service" {
		ups.Nodes = apisixv1.UpstreamNodes{
			{
				Host:   svcClusterIP,
				Port:   int(svcPort),
				Weight: _defaultWeight,
			},
		}
	}
	ups.Name = apisixv1.ComposeUpstreamName(namespace, svcName, subset, svcPort)
	ups.ID = id.GenID(ups.Name)
	return ups, nil
}

func (t *translator) filterNodesByLabels(nodes apisixv1.UpstreamNodes, labels types.Labels, namespace string) apisixv1.UpstreamNodes {
	if labels == nil {
		return nodes
	}

	filteredNodes := make(apisixv1.UpstreamNodes, 0)
	for _, node := range nodes {
		podName, err := t.PodCache.GetNameByIP(node.Host)
		if err != nil {
			log.Errorw("failed to find pod name by ip, ignore it",
				zap.Error(err),
				zap.String("pod_ip", node.Host),
			)
			continue
		}
		pod, err := t.PodLister.Pods(namespace).Get(podName)
		if err != nil {
			log.Errorw("failed to find pod, ignore it",
				zap.Error(err),
				zap.String("pod_name", podName),
			)
			continue
		}
		if labels.IsSubsetOf(pod.Labels) {
			filteredNodes = append(filteredNodes, node)
		}
	}
	return filteredNodes
}

func validateRemoteAddrs(remoteAddrs []string) error {
	for _, addr := range remoteAddrs {
		if ip := net.ParseIP(addr); ip == nil {
			// addr is not an IP address, try to parse it as a CIDR.
			if _, _, err := net.ParseCIDR(addr); err != nil {
				return _errInvalidAddress
			}
		}
	}
	return nil
}
