// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package apisix

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/go-playground/pool.v3"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type apisixUpstreamController struct {
	*apisixCommon

	workqueue    workqueue.RateLimitingInterface
	svcWorkqueue workqueue.RateLimitingInterface
	workers      int
	pool         pool.Pool

	externalSvcLock sync.RWMutex
	// external name service name -> apisix upstream name
	externalServiceMap map[string]map[string]struct{}

	// ApisixRouteController don't know how service change affect ApisixUpstream
	// So we need to notify it here
	notifyApisixUpstreamChange func(string)
}

func newApisixUpstreamController(common *apisixCommon, notifyApisixUpstreamChange func(string)) *apisixUpstreamController {
	c := &apisixUpstreamController{
		apisixCommon: common,
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixUpstream"),
		svcWorkqueue: workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixUpstreamService"),
		workers:      1,
		pool:         pool.NewLimited(2),

		externalServiceMap:         make(map[string]map[string]struct{}),
		notifyApisixUpstreamChange: notifyApisixUpstreamChange,
	}

	c.ApisixUpstreamInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		},
	)
	c.SvcInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onSvcAdd,
			UpdateFunc: c.onSvcUpdate,
			DeleteFunc: c.onSvcDelete,
		},
	)
	return c
}

func (c *apisixUpstreamController) run(ctx context.Context) {
	log.Info("ApisixUpstream controller started")
	defer log.Info("ApisixUpstream controller exited")
	defer c.workqueue.ShutDown()
	defer c.svcWorkqueue.ShutDown()
	defer c.pool.Close()

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
		go c.runSvcWorker(ctx)
	}

	<-ctx.Done()
}

func (c *apisixUpstreamController) runWorker(ctx context.Context) {
	for {
		obj, quit := c.workqueue.Get()
		if quit {
			return
		}
		err := c.sync(ctx, obj.(*types.Event))
		c.workqueue.Done(obj)
		c.handleSyncErr(obj, err)
	}
}

func (c *apisixUpstreamController) runSvcWorker(ctx context.Context) {
	for {
		obj, quit := c.svcWorkqueue.Get()
		if quit {
			return
		}
		key := obj.(string)
		err := c.handleSvcChange(ctx, key)
		c.svcWorkqueue.Done(obj)
		c.handleSvcErr(key, err)
	}
}

// sync Used to synchronize ApisixUpstream resources, because upstream alone exists in APISIX and will not be affected,
// the synchronization logic only includes upstream's unique configuration management
// So when ApisixUpstream was deleted, only the scheme / load balancer / healthcheck / retry / timeout
// on ApisixUpstream was cleaned up
func (c *apisixUpstreamController) sync(ctx context.Context, ev *types.Event) error {
	event := ev.Object.(kube.ApisixUpstreamEvent)
	key := event.Key
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("found ApisixUpstream resource with invalid meta namespace key %s: %s", key, err)
		return err
	}

	var multiVersioned kube.ApisixUpstream
	switch event.GroupVersion {
	case config.ApisixV2:
		multiVersioned, err = c.ApisixUpstreamLister.V2(namespace, name)
	default:
		return fmt.Errorf("unsupported ApisixUpstream group version %s", event.GroupVersion)
	}

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get ApisixUpstream",
				zap.Error(err),
				zap.String("key", key),
				zap.String("version", event.GroupVersion),
			)
			return err
		}
		if ev.Type == types.EventSync {
			// ignore not found error in delay sync
			return nil
		}
		if ev.Type != types.EventDelete {
			log.Warnw("ApisixUpstream was deleted before it can be delivered",
				zap.String("key", key),
				zap.String("version", event.GroupVersion),
			)
			// Don't need to retry.
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if multiVersioned != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnf("discard the stale ApisixUpstream delete event since the %s exists", key)
			return nil
		}
		multiVersioned = ev.Tombstone.(kube.ApisixUpstream)
	}

	c.syncRelationship(ev, key, multiVersioned)

	var errRecord error
	switch event.GroupVersion {
	case config.ApisixV2:
		au := multiVersioned.V2()
		if au.Spec == nil {
			return nil
		}

		// We will prioritize ExternalNodes and Discovery.
		if len(au.Spec.ExternalNodes) != 0 || au.Spec.Discovery != nil {
			var newUps *apisixv1.Upstream
			if ev.Type != types.EventDelete {
				cfg := &au.Spec.ApisixUpstreamConfig
				newUps, err = c.translator.TranslateUpstreamConfigV2(cfg)
				if err != nil {
					log.Errorw("failed to translate upstream config",
						zap.Any("object", au),
						zap.Error(err),
					)
					errRecord = err
					goto updateStatus
				}
			}

			if len(au.Spec.ExternalNodes) != 0 {
				errRecord = c.updateExternalNodes(ctx, au, nil, newUps, au.Namespace, au.Name, ev.Type.IsSyncEvent())
				goto updateStatus
			}

			// for service discovery related configuration
			if au.Spec.Discovery.ServiceName == "" || au.Spec.Discovery.Type == "" {
				log.Error("If you setup Discovery for ApisixUpstream, you need to specify the ServiceName and Type fields.")
				errRecord = fmt.Errorf("No ServiceName or Type fields found")
				goto updateStatus
			}
			// updateUpstream for real
			upsName := apisixv1.ComposeExternalUpstreamName(au.Namespace, au.Name)
			errRecord = c.updateUpstream(ctx, upsName, &au.Spec.ApisixUpstreamConfig, ev.Type.IsSyncEvent())
			goto updateStatus
		}

		var portLevelSettings map[int32]configv2.ApisixUpstreamConfig
		if len(au.Spec.PortLevelSettings) > 0 {
			portLevelSettings = make(map[int32]configv2.ApisixUpstreamConfig, len(au.Spec.PortLevelSettings))
			for _, port := range au.Spec.PortLevelSettings {
				portLevelSettings[port.Port] = port.ApisixUpstreamConfig
			}
		}

		svc, err := c.SvcLister.Services(namespace).Get(name)
		if err != nil {
			log.Errorf("failed to get service %s: %s", key, err)
			errRecord = err
			goto updateStatus
		}

		var subsets []configv2.ApisixUpstreamSubset
		subsets = append(subsets, configv2.ApisixUpstreamSubset{})
		if len(au.Spec.Subsets) > 0 {
			subsets = append(subsets, au.Spec.Subsets...)
		}
		for _, port := range svc.Spec.Ports {
			for _, subset := range subsets {
				var cfg configv2.ApisixUpstreamConfig
				if ev.Type != types.EventDelete {
					var ok bool
					cfg, ok = portLevelSettings[port.Port]
					if !ok {
						cfg = au.Spec.ApisixUpstreamConfig
					}
				}

				err := c.updateUpstream(ctx, apisixv1.ComposeUpstreamName(namespace, name, subset.Name, port.Port, types.ResolveGranularity.Endpoint), &cfg, ev.Type.IsSyncEvent())
				if err != nil {
					errRecord = err
					goto updateStatus
				}
				err = c.updateUpstream(ctx, apisixv1.ComposeUpstreamName(namespace, name, subset.Name, port.Port, types.ResolveGranularity.Service), &cfg, ev.Type.IsSyncEvent())
				if err != nil {
					errRecord = err
					goto updateStatus
				}
			}
		}
	}
updateStatus:
	c.pool.Queue(func(wu pool.WorkUnit) (interface{}, error) {
		if wu.IsCancelled() {
			return nil, nil
		}
		c.updateStatus(multiVersioned, errRecord)
		return true, nil
	})
	return errRecord
}

func (c *apisixUpstreamController) updateStatus(obj kube.ApisixUpstream, statusErr error) {
	if obj == nil || c.Kubernetes.DisableStatusUpdates || !c.Elector.IsLeader() {
		return
	}
	var (
		au        kube.ApisixUpstream
		err       error
		namespace = obj.GetNamespace()
		name      = obj.GetName()
	)

	switch obj.GroupVersion() {
	case config.ApisixV2:
		au, err = c.ApisixUpstreamLister.V2(namespace, name)
	}
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Warnw("Failed to update status, unable to get ApisixUpstream",
				zap.Error(err),
				zap.String("name", name),
				zap.String("namespace", namespace),
			)
		}
		return
	}
	if au.ResourceVersion() != obj.ResourceVersion() {
		return
	}
	var (
		reason    = utils.ResourceSynced
		condition = metav1.ConditionTrue
		eventType = corev1.EventTypeNormal
	)
	if statusErr != nil {
		reason = utils.ResourceSyncAborted
		condition = metav1.ConditionFalse
		eventType = corev1.EventTypeWarning
	}
	switch obj.GroupVersion() {
	case config.ApisixV2:
		c.RecordEvent(obj.V2(), eventType, reason, statusErr)
		c.recordStatus(obj.V2(), reason, statusErr, condition, au.GetGeneration())
	}
}

func (c *apisixUpstreamController) updateUpstream(ctx context.Context, upsName string, cfg *configv2.ApisixUpstreamConfig, shouldCompare bool) error {
	// TODO: multi cluster
	clusterName := c.Config.APISIX.DefaultClusterName

	ups, err := c.APISIX.Cluster(clusterName).Upstream().Get(ctx, upsName)
	if err != nil {
		log.Errorf("failed to get upstream %s: %s", upsName, err)
		return err
	}
	var newUps *apisixv1.Upstream
	if cfg != nil {
		newUps, err = c.translator.TranslateUpstreamConfigV2(cfg)
		if err != nil {
			log.Errorw("ApisixUpstream conversion cannot be completed, or the format is incorrect",
				zap.String("ApisixUpstream name", upsName),
				zap.Error(err),
			)
			return err
		}
	} else {
		newUps = apisixv1.NewDefaultUpstream()
	}

	newUps.Metadata = ups.Metadata
	newUps.Nodes = ups.Nodes
	log.Debugw("updating upstream since ApisixUpstream changed",
		zap.Any("upstream", newUps),
		zap.String("ApisixUpstream name", upsName),
	)
	if _, err := c.APISIX.Cluster(clusterName).Upstream().Update(ctx, newUps, shouldCompare); err != nil {
		log.Errorw("failed to update upstream",
			zap.Error(err),
			zap.Any("upstream", newUps),
			zap.String("ApisixUpstream name", upsName),
			zap.String("cluster", clusterName),
		)
		return err
	}
	return nil
}

func (c *apisixUpstreamController) updateExternalNodes(ctx context.Context, au *configv2.ApisixUpstream, old *configv2.ApisixUpstream, newUps *apisixv1.Upstream, ns, name string, shouldCompare bool) error {
	clusterName := c.Config.APISIX.DefaultClusterName

	// TODO: if old is not nil, diff the external nodes change first

	upsName := apisixv1.ComposeExternalUpstreamName(ns, name)
	ups, err := c.APISIX.Cluster(clusterName).Upstream().Get(ctx, upsName)
	if err != nil {
		log.Errorf("failed to get upstream %s: %s", upsName, err)
		c.RecordEvent(au, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
		c.recordStatus(au, utils.ResourceSyncAborted, err, metav1.ConditionFalse, au.GetGeneration())
		return err
	} else if ups != nil {
		nodes, err := c.translator.TranslateApisixUpstreamExternalNodes(au)
		if err != nil {
			log.Errorf("failed to translate upstream external nodes %s: %s", upsName, err)
			c.RecordEvent(au, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
			c.recordStatus(au, utils.ResourceSyncAborted, err, metav1.ConditionFalse, au.GetGeneration())
			return err
		}
		if newUps != nil {
			newUps.Metadata = ups.Metadata
			ups = newUps
		}

		ups.Nodes = nodes
		if _, err := c.APISIX.Cluster(clusterName).Upstream().Update(ctx, ups, shouldCompare); err != nil {
			log.Errorw("failed to update external nodes upstream",
				zap.Error(err),
				zap.Any("upstream", ups),
				zap.Any("ApisixUpstream", au),
				zap.String("cluster", clusterName),
			)
			c.RecordEvent(au, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
			c.recordStatus(au, utils.ResourceSyncAborted, err, metav1.ConditionFalse, au.GetGeneration())
			return err
		}
	}
	return nil
}

func (c *apisixUpstreamController) syncRelationship(ev *types.Event, auKey string, au kube.ApisixUpstream) {
	obj := ev.Object.(kube.ApisixUpstreamEvent)

	if obj.GroupVersion != config.ApisixV2 {
		return
	}

	var (
		old    *configv2.ApisixUpstream
		newObj *configv2.ApisixUpstream
	)

	if ev.Type == types.EventUpdate {
		old = obj.OldObject.V2()
	} else if ev.Type == types.EventDelete {
		old = ev.Tombstone.(kube.ApisixUpstream).V2()
	}

	if ev.Type != types.EventDelete {
		newObj = au.V2()
	}

	var (
		//oldExternalDomains  []string
		//newExternalDomains  []string
		oldExternalServices []string
		newExternalServices []string
	)
	if old != nil && old.Spec != nil {
		for _, node := range old.Spec.ExternalNodes {
			if node.Type == configv2.ExternalTypeDomain {
				//oldExternalDomains = append(oldExternalDomains, node.Name)
			} else if node.Type == configv2.ExternalTypeService {
				oldExternalServices = append(oldExternalServices, old.Namespace+"/"+node.Name)
			}
		}
	}
	if newObj != nil && newObj.Spec != nil {
		for _, node := range newObj.Spec.ExternalNodes {
			if node.Type == configv2.ExternalTypeDomain {
				//newExternalDomains = append(newExternalDomains, node.Name)
			} else if node.Type == configv2.ExternalTypeService {
				newExternalServices = append(newExternalServices, newObj.Namespace+"/"+node.Name)
			}
		}
	}

	c.externalSvcLock.Lock()
	defer c.externalSvcLock.Unlock()

	toDelete := utils.Difference(oldExternalServices, newExternalServices)
	toAdd := utils.Difference(newExternalServices, oldExternalServices)
	for _, svc := range toDelete {
		delete(c.externalServiceMap[svc], auKey)
	}

	for _, svc := range toAdd {
		if _, ok := c.externalServiceMap[svc]; !ok {
			c.externalServiceMap[svc] = make(map[string]struct{})
		}
		c.externalServiceMap[svc][auKey] = struct{}{}
	}
}

func (c *apisixUpstreamController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.MetricsCollector.IncrSyncOperation("upstream", "success")
		return
	}

	event := obj.(*types.Event)
	if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
		log.Infow("sync ApisixUpstream but not found, ignore",
			zap.String("event_type", event.Type.String()),
			zap.Any("ApisixUpstream", event.Object.(kube.ApisixUpstreamEvent)),
		)
		c.workqueue.Forget(event)
		return
	}
	log.Warnw("sync ApisixUpstream failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
	c.MetricsCollector.IncrSyncOperation("upstream", "failure")
}

func (c *apisixUpstreamController) onAdd(obj interface{}) {
	au, err := kube.NewApisixUpstream(obj)
	if err != nil {
		log.Errorw("found ApisixUpstream resource with bad type", zap.Error(err))
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixUpstream resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	if !c.isEffective(au) {
		return
	}
	log.Debugw("ApisixUpstream add event arrived",
		zap.Any("object", obj))

	c.workqueue.Add(&types.Event{
		Type: types.EventAdd,
		Object: kube.ApisixUpstreamEvent{
			Key:          key,
			GroupVersion: au.GroupVersion(),
		},
	})

	c.MetricsCollector.IncrEvents("upstream", "add")
}

func (c *apisixUpstreamController) onUpdate(oldObj, newObj interface{}) {
	prev, err := kube.NewApisixUpstream(oldObj)
	if err != nil {
		log.Errorw("found ApisixUpstream resource with bad type", zap.Error(err))
		return
	}
	curr, err := kube.NewApisixUpstream(newObj)
	if err != nil {
		log.Errorw("found ApisixUpstream resource with bad type", zap.Error(err))
		return
	}
	oldRV, _ := strconv.ParseInt(prev.ResourceVersion(), 0, 64)
	newRV, _ := strconv.ParseInt(curr.ResourceVersion(), 0, 64)
	if oldRV >= newRV {
		return
	}
	// Updates triggered by status are ignored.
	if prev.GetGeneration() == curr.GetGeneration() && prev.GetUID() == curr.GetUID() {
		switch curr.GroupVersion() {
		case config.ApisixV2:
			if reflect.DeepEqual(prev.V2().Spec, curr.V2().Spec) && !reflect.DeepEqual(prev.V2().Status, curr.V2().Status) {
				return
			}
		}
	}
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ApisixUpstream resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	if !c.isEffective(curr) {
		return
	}
	log.Debugw("ApisixUpstream update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)

	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		Object: kube.ApisixUpstreamEvent{
			Key:          key,
			OldObject:    prev,
			GroupVersion: curr.GroupVersion(),
		},
	})

	c.MetricsCollector.IncrEvents("upstream", "update")
}

func (c *apisixUpstreamController) onDelete(obj interface{}) {
	au, err := kube.NewApisixUpstream(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		au, err = kube.NewApisixUpstream(tombstone.Obj)
		if err != nil {
			log.Errorw("found ApisixUpstream resource with bad type", zap.Error(err))
			return
		}
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixUpstream resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	if !c.isEffective(au) {
		return
	}
	log.Debugw("ApisixUpstream delete event arrived",
		zap.Any("final state", au),
	)

	c.workqueue.Add(&types.Event{
		Type: types.EventDelete,
		Object: kube.ApisixUpstreamEvent{
			Key:          key,
			GroupVersion: au.GroupVersion(),
		},
		Tombstone: au,
	})

	c.MetricsCollector.IncrEvents("upstream", "delete")
}

// ResourceSync syncs ApisixUpstream resources within namespace to workqueue.
// If namespace is "", it syncs all namespaces ApisixUpstream resources.
func (c *apisixUpstreamController) ResourceSync(interval time.Duration, namespace string) {
	objs := c.ApisixUpstreamInformer.GetIndexer().List()
	delay := GetSyncDelay(interval, len(objs))
	for i, obj := range objs {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			log.Errorw("ApisixUpstream sync failed, found ApisixUpstream resource with bad meta namespace key", zap.String("error", err.Error()))
			continue
		}
		if !c.namespaceProvider.IsWatchingNamespace(key) {
			continue
		}
		ns, _, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			log.Errorw("split ApisixRoute meta key failed",
				zap.Error(err),
				zap.String("key", key),
			)
			continue
		}
		if namespace != "" && ns != namespace {
			continue
		}
		au, err := kube.NewApisixUpstream(obj)
		if err != nil {
			log.Errorw("ApisixUpstream sync failed, found ApisixUpstream resource with bad type", zap.Error(err))
			continue
		}
		if !c.isEffective(au) {
			continue
		}
		log.Debugw("ResourceSync",
			zap.String("resource", "ApisixUpstream"),
			zap.String("key", key),
			zap.Duration("calc_delay", delay),
			zap.Int("i", i),
			zap.Duration("delay", delay*time.Duration(i)),
		)
		c.workqueue.AddAfter(&types.Event{
			Type: types.EventSync,
			Object: kube.ApisixUpstreamEvent{
				Key:          key,
				GroupVersion: au.GroupVersion(),
			},
		}, delay*time.Duration(i))
	}
}

func (c *apisixUpstreamController) onSvcAdd(obj interface{}) {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		log.Errorw("got service add event, but it is not a Service",
			zap.Any("obj", obj),
		)
	}

	log.Debugw("Service add event arrived",
		zap.Any("object", obj),
	)

	if svc.Spec.Type != corev1.ServiceTypeExternalName {
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorw("found Service with bad meta key",
			zap.Error(err),
			zap.Any("obj", obj),
		)
		return
	}
	c.svcWorkqueue.Add(key)
}

func (c *apisixUpstreamController) onSvcUpdate(old, new interface{}) {
	oldSvc, ok := old.(*corev1.Service)
	if !ok {
		log.Errorw("got service update event, but old one is not a Service",
			zap.Any("old", old),
		)
	}
	newSvc, ok := new.(*corev1.Service)
	if !ok {
		log.Errorw("got service update event, but new one is not a Service",
			zap.Any("new", new),
		)
	}

	if newSvc.Spec.Type != corev1.ServiceTypeExternalName {
		return
	}

	if newSvc.Spec.ExternalName != oldSvc.Spec.ExternalName {
		key, err := cache.MetaNamespaceKeyFunc(newSvc)
		if err != nil {
			log.Errorw("found Service with bad meta key",
				zap.Error(err),
				zap.Any("obj", newSvc),
			)
			return
		}
		c.svcWorkqueue.Add(key)
	}
}

func (c *apisixUpstreamController) onSvcDelete(obj interface{}) {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		svc, ok = tombstone.Obj.(*corev1.Service)
		if !ok {
			log.Errorw("got service delete event, but it is not a Service",
				zap.Any("obj", obj),
			)
			return
		}
	}
	if svc.Spec.Type != corev1.ServiceTypeExternalName {
		return
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorw("found Service with bad meta key",
			zap.Error(err),
			zap.Any("obj", obj),
		)
		return
	}
	c.svcWorkqueue.Add(key)
}

func (c *apisixUpstreamController) handleSvcChange(ctx context.Context, key string) error {
	var toUpdateUpstreams []string

	c.externalSvcLock.RLock()
	if ups, ok := c.externalServiceMap[key]; ok {
		for upKey := range ups {
			toUpdateUpstreams = append(toUpdateUpstreams, upKey)
		}
	}
	c.externalSvcLock.RUnlock()

	//log.Debugw("handleSvcChange",
	//	zap.Any("service map", c.externalServiceMap),
	//	zap.Strings("affectedUpstreams", toUpdateUpstreams),
	//)

	for _, upKey := range toUpdateUpstreams {
		log.Debugw("Service change event trigger ApisixUpstream sync",
			zap.Any("service", key),
			zap.Any("ApisixUpstream", upKey),
		)
		c.notifyApisixUpstreamChange(upKey)
		ns, name, err := cache.SplitMetaNamespaceKey(upKey)
		if err != nil {
			return err
		}
		au, err := c.ApisixUpstreamLister.V2(ns, name)
		if err != nil {
			return err
		}
		err = c.updateExternalNodes(ctx, au.V2(), nil, nil, ns, name, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *apisixUpstreamController) handleSvcErr(key string, errOrigin error) {
	if errOrigin == nil {
		c.workqueue.Forget(key)
		return
	}

	log.Warnw("sync Service failed, will retry",
		zap.Any("key", key),
		zap.Error(errOrigin),
	)
	c.svcWorkqueue.AddRateLimited(key)
}

// recordStatus record resources status
func (c *apisixUpstreamController) recordStatus(at interface{}, reason string, err error, status metav1.ConditionStatus, generation int64) {
	if c.Kubernetes.DisableStatusUpdates {
		return
	}
	// build condition
	message := utils.CommonSuccessMessage
	if err != nil {
		message = err.Error()
	}
	condition := metav1.Condition{
		Type:               utils.ConditionType,
		Reason:             reason,
		Status:             status,
		Message:            message,
		ObservedGeneration: generation,
	}
	apisixClient := c.KubeClient.APISIXClient

	if kubeObj, ok := at.(runtime.Object); ok {
		at = kubeObj.DeepCopyObject()
	}

	switch v := at.(type) {
	case *configv2.ApisixUpstream:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if utils.VerifyConditions(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			log.Errorw("update status", zap.Any("status", v.Status))
			if _, errRecord := apisixClient.ApisixV2().ApisixUpstreams(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixUpstream",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	default:
		// This should not be executed
		log.Errorf("unsupported resource record: %s", v)
	}
}

func (c *apisixUpstreamController) isEffective(au kube.ApisixUpstream) bool {
	if au.GroupVersion() == config.ApisixV2 {
		var ingClassName string
		if au.V2().Spec != nil {
			ingClassName = au.V2().Spec.IngressClassName
		}
		return utils.MatchCRDsIngressClass(ingClassName, c.Kubernetes.IngressClass)
	}
	// Compatible with legacy versions
	return true
}
