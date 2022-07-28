package types

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
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
	Run(ctx context.Context)
}

// TODO: rename to BaseController
type CommonConfig struct {
	*config.Config

	MetricsCollector metrics.Collector
	Recorder         record.EventRecorder
	APISIX           apisix.APISIX
	KubeClient       *kube.KubeClient
}

// RecordEvent recorder events for resources
func (c *CommonConfig) RecordEvent(object runtime.Object, eventtype, reason string, err error) {
	if err != nil {
		message := fmt.Sprintf(utils.MessageResourceFailed, utils.Component, err.Error())
		c.Recorder.Event(object, eventtype, reason, message)
	} else {
		message := fmt.Sprintf(utils.MessageResourceSynced, utils.Component)
		c.Recorder.Event(object, eventtype, reason, message)
	}
}

// RecordEventS recorder events for resources
func (c *CommonConfig) RecordEventS(object runtime.Object, eventtype, reason string, msg string) {
	c.Recorder.Event(object, eventtype, reason, msg)
}

// === TODO ===
// Move sync utils to apisix.APISIX interface

func (c *CommonConfig) SyncManifests(ctx context.Context, added, updated, deleted *utils.Manifest) error {
	return utils.SyncManifests(ctx, c.APISIX, c.Config.APISIX.DefaultClusterName, added, updated, deleted)
}

func (c *CommonConfig) SyncSSL(ctx context.Context, ssl *apisixv1.Ssl, event types.EventType) error {
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

func (c *CommonConfig) SyncConsumer(ctx context.Context, consumer *apisixv1.Consumer, event types.EventType) (err error) {
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

func (c *CommonConfig) SyncUpstreamNodesChangeToCluster(ctx context.Context, cluster apisix.Cluster, nodes apisixv1.UpstreamNodes, upsName string) error {
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
