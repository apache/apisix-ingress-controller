package types

import (
	"context"
	"fmt"
	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/metrics"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
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

func (c *CommonConfig) SyncManifests(ctx context.Context, added, updated, deleted *utils.Manifest) error {
	return utils.SyncManifests(ctx, c.APISIX, c.Config.APISIX.DefaultClusterName, added, updated, deleted)
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
