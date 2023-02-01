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
package ingress

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/go-playground/pool.v3"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_ingressKey = "kubernetes.io/ingress.class"
)

type ingressController struct {
	*ingressCommon

	workqueue workqueue.RateLimitingInterface
	workers   int

	pool pool.Pool

	// secretSSLMap stores reference from K8s secret to Ingress
	// type: Map<SecretKey, Map<IngressVersionKey, SSL in APISIX>>
	// SecretKey -> IngressVersionKey -> []string
	// Secret key is kube-style meta key: `namespace/name`
	// Ingress Version Key is: `namespace/name_groupVersion`
	secretSSLMap *sync.Map
}

func newIngressController(common *ingressCommon) *ingressController {
	c := &ingressController{
		ingressCommon: common,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ingress"),
		workers:   1,
		pool:      pool.NewLimited(2),

		secretSSLMap: new(sync.Map),
	}

	c.IngressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.OnDelete,
	})
	return c
}

func (c *ingressController) run(ctx context.Context) {
	log.Info("ingress controller started")
	defer log.Infof("ingress controller exited")
	defer c.workqueue.ShutDown()
	defer c.pool.Close()

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *ingressController) runWorker(ctx context.Context) {
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

func (c *ingressController) sync(ctx context.Context, ev *types.Event) error {
	ingEv := ev.Object.(kube.IngressEvent)
	namespace, name, err := cache.SplitMetaNamespaceKey(ingEv.Key)
	if err != nil {
		log.Errorf("found ingress resource with invalid meta namespace key %s: %s", ingEv.Key, err)
		return err
	}

	var (
		ing  kube.Ingress
		tctx *translation.TranslateContext
	)
	switch ingEv.GroupVersion {
	case kube.IngressV1:
		ing, err = c.IngressLister.V1(namespace, name)
	case kube.IngressV1beta1:
		ing, err = c.IngressLister.V1beta1(namespace, name)
	case kube.IngressExtensionsV1beta1:
		ing, err = c.IngressLister.ExtensionsV1beta1(namespace, name)
	default:
		err = fmt.Errorf("unsupported group version %s, one of (%s/%s/%s) is expected", ingEv.GroupVersion,
			kube.IngressV1, kube.IngressV1beta1, kube.IngressExtensionsV1beta1)
	}

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorf("failed to get ingress %s (group version: %s): %s", ingEv.Key, ingEv.GroupVersion, err)
			return err
		}

		if ev.Type != types.EventDelete {
			log.Warnf("ingress %s (group version: %s) was deleted before it can be delivered", ingEv.Key, ingEv.GroupVersion)
			// Don't need to retry.
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if ing != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnf("discard the stale ingress delete event since the %s exists", ingEv.Key)
			return nil
		}
		ing = ev.Tombstone.(kube.Ingress)
	}
	{
		tctx, err = c.translator.TranslateIngress(ing)
		if err != nil {
			log.Errorw("failed to translate ingress",
				zap.Error(err),
				zap.Any("ingress", ing),
			)
			goto updateStatus
		}

		for _, ssl := range tctx.SSL {
			ns, ok1 := ssl.Labels[translation.MetaSecretNamespace]
			sec, ok2 := ssl.Labels[translation.MetaSecretName]
			if ok1 && ok2 {
				// We don't support annotation in Ingress
				// 	_caAnnotation = "nginx.ingress.kubernetes.io/auth-tls-secret"
				c.storeSecretReference(ns+"/"+sec, ingEv.Key, ev.Type, ssl)
			}
		}

		log.Debugw("translated ingress resource to a couple of routes, upstreams and pluginConfigs",
			zap.Any("ingress", ing),
			zap.Any("routes", tctx.Routes),
			zap.Any("upstreams", tctx.Upstreams),
			zap.Any("ssl", tctx.SSL),
			zap.Any("pluginConfigs", tctx.PluginConfigs),
		)
	}
	{
		m := &utils.Manifest{
			SSLs:          tctx.SSL,
			Routes:        tctx.Routes,
			Upstreams:     tctx.Upstreams,
			PluginConfigs: tctx.PluginConfigs,
		}

		var (
			added   *utils.Manifest
			updated *utils.Manifest
			deleted *utils.Manifest
		)

		if ev.Type == types.EventDelete {
			deleted = m
		} else if ev.Type == types.EventAdd {
			added = m
		} else {
			oldCtx, _ := c.translator.TranslateOldIngress(ingEv.OldObject)
			om := &utils.Manifest{
				Routes:        oldCtx.Routes,
				Upstreams:     oldCtx.Upstreams,
				SSLs:          oldCtx.SSL,
				PluginConfigs: oldCtx.PluginConfigs,
			}
			added, updated, deleted = m.Diff(om)
		}
		if err = c.SyncManifests(ctx, added, updated, deleted); err != nil {
			log.Errorw("failed to sync Ingress to apisix",
				zap.Error(err),
			)
			goto updateStatus
		}
	}
updateStatus:
	c.pool.Queue(func(wu pool.WorkUnit) (interface{}, error) {
		if wu.IsCancelled() {
			return nil, nil
		}
		c.UpdateStatus(ing)
		return true, nil
	})
	return err
}

func (c *ingressController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.MetricsCollector.IncrSyncOperation("ingress", "success")
		return
	}
	ev := obj.(*types.Event)
	event := ev.Object.(kube.IngressEvent)
	if k8serrors.IsNotFound(err) && ev.Type != types.EventDelete {
		log.Infow("sync ingress but not found, ignore",
			zap.String("event_type", ev.Type.String()),
			zap.String("ingress", event.Key),
		)
		c.workqueue.Forget(obj)
		return
	}
	log.Warnw("sync ingress failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
	c.MetricsCollector.IncrSyncOperation("ingress", "failure")
}

func (c *ingressController) UpdateStatus(obj kube.Ingress) {
	if obj == nil {
		return
	}
	var (
		namespace = obj.GetNamespace()
		name      = obj.GetName()
		ing       kube.Ingress
		err       error
	)

	switch obj.GroupVersion() {
	case kube.IngressV1:
		ing, err = c.IngressLister.V1(namespace, name)
	case kube.IngressV1beta1:
		ing, err = c.IngressLister.V1beta1(namespace, name)
	case kube.IngressExtensionsV1beta1:
		ing, err = c.IngressLister.ExtensionsV1beta1(namespace, name)
	}
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Warnw("failed to update status, unable to get Ingress",
				zap.Error(err),
				zap.String("name", name),
				zap.String("namespace", namespace),
			)
		}
		return
	}
	if ing.ResourceVersion() != obj.ResourceVersion() {
		return
	}
	switch obj.GroupVersion() {
	case kube.IngressV1:
		c.recordStatus(obj.V1())
	case kube.IngressV1beta1:
		c.recordStatus(obj.V1beta1())
	case kube.IngressExtensionsV1beta1:
		c.recordStatus(obj.ExtensionsV1beta1())
	}
}

func (c *ingressController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ingress resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}

	ing := kube.MustNewIngress(obj)
	valid := c.isIngressEffective(ing)
	if valid {
		log.Debugw("ingress add event arrived",
			zap.Any("object", obj),
		)
	} else {
		log.Debugw("ignore noneffective ingress add event",
			zap.Any("object", obj),
		)
		return
	}

	c.workqueue.Add(&types.Event{
		Type: types.EventAdd,
		Object: kube.IngressEvent{
			Key:          key,
			GroupVersion: ing.GroupVersion(),
		},
	})

	c.MetricsCollector.IncrEvents("ingress", "add")
}

func (c *ingressController) onUpdate(oldObj, newObj interface{}) {
	prev := kube.MustNewIngress(oldObj)
	curr := kube.MustNewIngress(newObj)
	if prev.ResourceVersion() >= curr.ResourceVersion() {
		return
	}
	// Updates triggered by status are ignored.
	if prev.GetGeneration() == curr.GetGeneration() && prev.GetUID() == curr.GetUID() {
		switch curr.GroupVersion() {
		case kube.IngressV1:
			if reflect.DeepEqual(prev.V1().Spec, curr.V1().Spec) &&
				!reflect.DeepEqual(prev.V1().Status, curr.V1().Status) {
				return
			}
		case kube.IngressV1beta1:
			if reflect.DeepEqual(prev.V1beta1().Spec, curr.V1beta1().Spec) &&
				!reflect.DeepEqual(prev.V1beta1().Status, curr.V1beta1().Status) {
				return
			}
		case kube.IngressExtensionsV1beta1:
			if reflect.DeepEqual(prev.ExtensionsV1beta1().Spec, curr.ExtensionsV1beta1().Spec) &&
				!reflect.DeepEqual(prev.ExtensionsV1beta1().Status, curr.ExtensionsV1beta1().Status) {
				return
			}
		}
	}

	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ingress resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	valid := c.isIngressEffective(curr)
	if valid {
		log.Debugw("ingress update event arrived",
			zap.Any("new object", newObj),
			zap.Any("old object", oldObj),
		)
	} else {
		log.Debugw("ignore noneffective ingress update event",
			zap.Any("new object", oldObj),
			zap.Any("old object", newObj),
		)
		return
	}

	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		Object: kube.IngressEvent{
			Key:          key,
			GroupVersion: curr.GroupVersion(),
			OldObject:    prev,
		},
	})

	c.MetricsCollector.IncrEvents("ingress", "update")
}

func (c *ingressController) OnDelete(obj interface{}) {
	ing, err := kube.NewIngress(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		ing = kube.MustNewIngress(tombstone)
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ingress resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	valid := c.isIngressEffective(ing)
	if valid {
		log.Debugw("ingress delete event arrived",
			zap.Any("final state", ing),
		)
	} else {
		log.Debugw("ignore noneffective ingress delete event",
			zap.Any("object", ing),
		)
		return
	}
	c.workqueue.Add(&types.Event{
		Type: types.EventDelete,
		Object: kube.IngressEvent{
			Key:          key,
			GroupVersion: ing.GroupVersion(),
		},
		Tombstone: ing,
	})

	c.MetricsCollector.IncrEvents("ingress", "delete")
}

func (c *ingressController) isIngressEffective(ing kube.Ingress) bool {
	var (
		ic  *string
		ica string
	)
	if ing.GroupVersion() == kube.IngressV1 {
		ic = ing.V1().Spec.IngressClassName
		ica = ing.V1().GetAnnotations()[_ingressKey]
	} else if ing.GroupVersion() == kube.IngressV1beta1 {
		ic = ing.V1beta1().Spec.IngressClassName
		ica = ing.V1beta1().GetAnnotations()[_ingressKey]
	} else {
		ic = ing.ExtensionsV1beta1().Spec.IngressClassName
		ica = ing.ExtensionsV1beta1().GetAnnotations()[_ingressKey]
	}

	// kubernetes.io/ingress.class takes the precedence.
	if ica != "" {
		return ica == c.Kubernetes.IngressClass
	}
	if ic != nil {
		return *ic == c.Kubernetes.IngressClass
	}
	return false
}

func (c *ingressController) ResourceSync() {
	objs := c.IngressInformer.GetIndexer().List()
	for _, obj := range objs {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			log.Errorw("found Ingress resource with bad meta namespace key", zap.String("error", err.Error()))
			continue
		}
		if !c.namespaceProvider.IsWatchingNamespace(key) {
			continue
		}
		ing := kube.MustNewIngress(obj)
		if !c.isIngressEffective(ing) {
			return
		}
		log.Debugw("ingress add event arrived",
			zap.Any("object", obj),
		)
		c.workqueue.Add(&types.Event{
			Type: types.EventAdd,
			Object: kube.IngressEvent{
				Key:          key,
				GroupVersion: ing.GroupVersion(),
			},
		})
	}
}

// recordStatus record resources status
func (c *ingressController) recordStatus(at runtime.Object) {
	client := c.KubeClient.Client

	at = at.DeepCopyObject()

	switch v := at.(type) {
	case *networkingv1.Ingress:
		// set to status
		lbips, err := c.ingressLBStatusIPs()
		if err != nil {
			log.Errorw("failed to get APISIX gateway external IPs",
				zap.Error(err),
			)

		}

		v.Status.LoadBalancer.Ingress = utils.CoreV1ToNetworkV1LB(lbips)
		if _, errRecord := client.NetworkingV1().Ingresses(v.Namespace).UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
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

		v.Status.LoadBalancer.Ingress = utils.CoreV1ToNetworkV1beta1LB(lbips)
		if _, errRecord := client.NetworkingV1beta1().Ingresses(v.Namespace).UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
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

		v.Status.LoadBalancer.Ingress = utils.CoreV1ToExtensionsV1beta1LB(lbips)
		if _, errRecord := client.ExtensionsV1beta1().Ingresses(v.Namespace).UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
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

// ingressLBStatusIPs organizes the available addresses
func (c *ingressController) ingressLBStatusIPs() ([]corev1.LoadBalancerIngress, error) {
	return utils.IngressLBStatusIPs(c.IngressPublishService, c.IngressStatusAddress, c.KubeClient.Client)
}

func (c *ingressController) storeSecretReference(secretKey string, ingressKey string, evType types.EventType, ssl *v1.Ssl) {
	if ssls, ok := c.secretSSLMap.Load(secretKey); ok {
		sslMap := ssls.(*sync.Map)
		switch evType {
		case types.EventDelete:
			sslMap.Delete(ingressKey)
			c.secretSSLMap.Store(secretKey, sslMap)
		default:
			sslMap.Store(ingressKey, ssl)
			c.secretSSLMap.Store(secretKey, sslMap)
		}
	} else if evType != types.EventDelete {
		sslMap := new(sync.Map)
		sslMap.Store(ingressKey, ssl)
		c.secretSSLMap.Store(secretKey, sslMap)
	}
}

func (c *ingressController) SyncSecretChange(ctx context.Context, ev *types.Event, secret *corev1.Secret, secretKey string) {
	ssls, ok := c.secretSSLMap.Load(secretKey)
	if !ok {
		return
	}

	sslMap, ok := ssls.(*sync.Map) // ingress version key -> SSL
	if !ok {
		return
	}

	sslMap.Range(func(k, v interface{}) bool {
		ingressVersionKey := k.(string)
		ssl := v.(*v1.Ssl)
		return c.syncSSLs(ctx, ev.Type, secret, secretKey, ingressVersionKey, ssl)
	})
}

func (c *ingressController) syncSSLs(ctx context.Context, evType types.EventType, secret *corev1.Secret, secretKey, ingressVersionKey string, ssl *v1.Ssl) bool {
	vals := strings.Split(ingressVersionKey, "_")
	if len(vals) != 2 {
		log.Errorw("cache recorded invalid ingress version key",
			zap.String("key", ingressVersionKey),
		)
	}
	ingressKey := vals[0]
	ingressVersion := vals[1]

	ingressNamespace, ingressName, err := cache.SplitMetaNamespaceKey(ingressKey)
	if err != nil {
		log.Errorf("invalid cached ApisixTls key: %s", ingressKey)
		return true
	}

	var (
		obj metav1.Object
		ing kube.Ingress
	)
	switch ingressVersion {
	case kube.IngressV1:
		ing, err = c.IngressLister.V1(ingressNamespace, ingressName)
		obj = ing.V1()
	case kube.IngressV1beta1:
		ing, err = c.IngressLister.V1(ingressNamespace, ingressName)
		obj = ing.V1beta1()
	case kube.IngressExtensionsV1beta1:
		ing, err = c.IngressLister.V1(ingressNamespace, ingressName)
		obj = ing.ExtensionsV1beta1()
	}
	if err != nil {
		log.Warnw("secret related ingress resource not found, skip",
			zap.String("ingress", ingressKey),
		)
		return true
	}

	cert, pkey, err := translation.ExtractKeyPair(secret, true)
	if err != nil {
		log.Errorw("secret required by Ingress invalid",
			zap.String("ingress", ingressKey),
			zap.String("secret", secretKey),
			zap.Error(err),
		)
		go func(obj metav1.Object) {
			runtimeObj := obj.(runtime.Object)
			c.RecordEventS(runtimeObj, corev1.EventTypeWarning, utils.ResourceSyncAborted,
				fmt.Sprintf("sync from secret %s changes failed, error: %s", secretKey, err.Error()))
			c.recordStatus(runtimeObj)
		}(obj)
		return true
	}

	// update ssl
	ssl.Cert = string(cert)
	ssl.Key = string(pkey)

	go func(ssl *v1.Ssl, obj metav1.Object) {
		runtimeObj := obj.(runtime.Object)

		err := c.SyncSSL(ctx, ssl, evType)
		if err != nil {
			log.Errorw("failed to sync ssl to APISIX",
				zap.Error(err),
				zap.Any("ssl", ssl),
				zap.Any("secret", secret),
			)
			c.RecordEventS(runtimeObj, corev1.EventTypeWarning, utils.ResourceSyncAborted,
				fmt.Sprintf("sync from secret %s changes failed, error: %s", secretKey, err.Error()))
			c.recordStatus(runtimeObj)
		} else {
			c.RecordEventS(runtimeObj, corev1.EventTypeNormal, utils.ResourceSynced,
				fmt.Sprintf("sync from secret %s changes", secretKey))
			c.recordStatus(runtimeObj)
		}
	}(ssl, obj)
	return true
}
