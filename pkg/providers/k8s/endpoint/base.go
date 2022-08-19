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
package endpoint

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	listerscorev1 "k8s.io/client-go/listers/core/v1"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type baseEndpointController struct {
	*providertypes.Common
	translator translation.Translator

	apisixUpstreamLister kube.ApisixUpstreamLister
	svcLister            listerscorev1.ServiceLister
}

func (c *baseEndpointController) syncEndpoint(ctx context.Context, ep kube.Endpoint) error {
	log.Debugw("endpoint controller syncing endpoint",
		zap.Any("endpoint", ep),
	)

	namespace, err := ep.Namespace()
	if err != nil {
		return err
	}
	svcName := ep.ServiceName()
	svc, err := c.svcLister.Services(namespace).Get(svcName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return c.syncEmptyEndpoint(ctx, ep)
		}
		log.Errorf("failed to get service %s/%s: %s", namespace, svcName, err)
		return err
	}

	switch c.Kubernetes.APIVersion {
	case config.ApisixV2beta3:
		var subsets []configv2beta3.ApisixUpstreamSubset
		subsets = append(subsets, configv2beta3.ApisixUpstreamSubset{})
		auKube, err := c.apisixUpstreamLister.V2beta3(namespace, svcName)
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				log.Errorf("failed to get ApisixUpstream %s/%s: %s", namespace, svcName, err)
				return err
			}
		} else if auKube.V2beta3().Spec != nil && len(auKube.V2beta3().Spec.Subsets) > 0 {
			subsets = append(subsets, auKube.V2beta3().Spec.Subsets...)
		}
		clusters := c.APISIX.ListClusters()
		for _, port := range svc.Spec.Ports {
			for _, subset := range subsets {
				nodes, err := c.translator.TranslateEndpoint(ep, port.Port, subset.Labels)
				if err != nil {
					log.Errorw("failed to translate upstream nodes",
						zap.Error(err),
						zap.Any("endpoints", ep),
						zap.Int32("port", port.Port),
					)
				}
				name := apisixv1.ComposeUpstreamName(namespace, svcName, subset.Name, port.Port, types.ResolveGranularity.Endpoint)
				for _, cluster := range clusters {
					if err := c.SyncUpstreamNodesChangeToCluster(ctx, cluster, nodes, name); err != nil {
						return err
					}
				}
			}
		}
	case config.ApisixV2:
		var subsets []configv2.ApisixUpstreamSubset
		subsets = append(subsets, configv2.ApisixUpstreamSubset{})
		auKube, err := c.apisixUpstreamLister.V2(namespace, svcName)
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				log.Errorf("failed to get ApisixUpstream %s/%s: %s", namespace, svcName, err)
				return err
			}
		} else if auKube.V2().Spec != nil && len(auKube.V2().Spec.Subsets) > 0 {
			subsets = append(subsets, auKube.V2().Spec.Subsets...)
		}
		clusters := c.APISIX.ListClusters()
		for _, port := range svc.Spec.Ports {
			for _, subset := range subsets {
				nodes, err := c.translator.TranslateEndpoint(ep, port.Port, subset.Labels)
				if err != nil {
					log.Errorw("failed to translate upstream nodes",
						zap.Error(err),
						zap.Any("endpoints", ep),
						zap.Int32("port", port.Port),
					)
				}
				name := apisixv1.ComposeUpstreamName(namespace, svcName, subset.Name, port.Port, types.ResolveGranularity.Endpoint)
				for _, cluster := range clusters {
					if err := c.SyncUpstreamNodesChangeToCluster(ctx, cluster, nodes, name); err != nil {
						return err
					}
				}
			}
		}
	default:
		panic(fmt.Errorf("unsupported ApisixUpstream version %v", c.Kubernetes.APIVersion))
	}
	return nil
}

func (c *baseEndpointController) syncEmptyEndpoint(ctx context.Context, ep kube.Endpoint) error {
	namespace, err := ep.Namespace()
	if err != nil {
		return err
	}
	svcName := ep.ServiceName()
	log.Infow("The syncEndpoint, service has been deleted, try to update upstream",
		zap.String("Namespace", namespace),
		zap.String("ServiceName", svcName),
	)
	clusterName := c.Config.APISIX.DefaultClusterName
	err = c.APISIX.Cluster(clusterName).UpstreamServiceRelation().Delete(ctx, namespace+"_"+svcName)
	if err != nil {
		log.Errorw("try to update upstream failed!",
			zap.Error(err),
		)
	}
	return nil
}
