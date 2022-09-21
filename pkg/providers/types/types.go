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
package types

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	apisixcache "github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/metrics"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type Provider interface {
	// Init() // TODO: should all provider implement this?
	Run(ctx context.Context)
}

type ListerInformer struct {
	EpLister   kube.EndpointLister
	EpInformer cache.SharedIndexInformer

	SvcLister   listerscorev1.ServiceLister
	SvcInformer cache.SharedIndexInformer

	SecretLister   listerscorev1.SecretLister
	SecretInformer cache.SharedIndexInformer

	PodLister   listerscorev1.PodLister
	PodInformer cache.SharedIndexInformer

	ApisixUpstreamLister   kube.ApisixUpstreamLister
	ApisixUpstreamInformer cache.SharedIndexInformer
}

func (c *ListerInformer) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		c.EpInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.SvcInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.SecretInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.PodInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.ApisixUpstreamInformer.Run(ctx.Done())
	})

	e.Wait()
}

type Common struct {
	*config.Config
	*ListerInformer

	APISIX           apisix.APISIX
	KubeClient       *kube.KubeClient
	MetricsCollector metrics.Collector
	Recorder         record.EventRecorder
}

// RecordEvent recorder events for resources
func (c *Common) RecordEvent(object runtime.Object, eventtype, reason string, err error) {
	if err != nil {
		message := fmt.Sprintf(utils.MessageResourceFailed, utils.Component, err.Error())
		c.Recorder.Event(object, eventtype, reason, message)
	} else {
		message := fmt.Sprintf(utils.MessageResourceSynced, utils.Component)
		c.Recorder.Event(object, eventtype, reason, message)
	}
}

// RecordEventS recorder events for resources
func (c *Common) RecordEventS(object runtime.Object, eventtype, reason string, msg string) {
	c.Recorder.Event(object, eventtype, reason, msg)
}

// TODO: Move sync utils to apisix.APISIX interface?
func (c *Common) SyncManifests(ctx context.Context, added, updated, deleted *utils.Manifest) error {
	return utils.SyncManifests(ctx, c.APISIX, c.Config.APISIX.DefaultClusterName, added, updated, deleted)
}

func (c *Common) SyncSSL(ctx context.Context, ssl *apisixv1.Ssl, event types.EventType) error {
	var (
		err error
	)
	clusterName := c.Config.APISIX.DefaultClusterName
	if event == types.EventDelete {
		err = c.APISIX.Cluster(clusterName).SSL().Delete(ctx, ssl)
	} else if event == types.EventUpdate {
		_, err = c.APISIX.Cluster(clusterName).SSL().Update(ctx, ssl)
	} else {
		_, err = c.APISIX.Cluster(clusterName).SSL().Create(ctx, ssl)
	}
	return err
}

func (c *Common) SyncConsumer(ctx context.Context, consumer *apisixv1.Consumer, event types.EventType) (err error) {
	clusterName := c.Config.APISIX.DefaultClusterName
	if event == types.EventDelete {
		err = c.APISIX.Cluster(clusterName).Consumer().Delete(ctx, consumer)
	} else if event == types.EventUpdate {
		_, err = c.APISIX.Cluster(clusterName).Consumer().Update(ctx, consumer)
	} else {
		_, err = c.APISIX.Cluster(clusterName).Consumer().Create(ctx, consumer)
	}
	return
}

func (c *Common) SyncUpstreamNodesChangeToCluster(ctx context.Context, cluster apisix.Cluster, nodes apisixv1.UpstreamNodes, upsName string) error {
	log.Debugw("sync upstream nodes change",
		zap.String("cluster", cluster.String()),
		zap.String("upstream_name", upsName),
		zap.Any("nodes", nodes),
	)
	upstream, err := cluster.Upstream().Get(ctx, upsName)
	if err != nil {
		if err == apisixcache.ErrNotFound {
			log.Warnw("upstream is not referenced",
				zap.String("cluster", cluster.String()),
				zap.String("upstream", upsName),
			)
			return nil
		} else {
			log.Errorw("failed to get upstream",
				zap.String("upstream", upsName),
				zap.String("cluster", cluster.String()),
				zap.Error(err),
			)
			return err
		}
	}

	upstream.Nodes = nodes

	log.Debugw("upstream binds new nodes",
		zap.Any("upstream", upstream),
		zap.String("cluster", cluster.String()),
	)

	updated := &utils.Manifest{
		Upstreams: []*apisixv1.Upstream{upstream},
	}
	return c.SyncManifests(ctx, nil, updated, nil)
}
