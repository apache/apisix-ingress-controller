package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/seven/state"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type secretController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newSecretController() *secretController {
	ctl := &secretController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "Secrets"),
		workers:    1,
	}

	ctl.controller.secretInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)

	return ctl
}

func (c *secretController) run(ctx context.Context) {
	log.Info("secret controller started")
	defer log.Info("secret controller exited")

	if ok := cache.WaitForCacheSync(ctx.Done(), c.controller.secretInformer.HasSynced); !ok {
		log.Error("informers sync failed")
		return
	}

	handler := func() {
		for {
			obj, shutdown := c.workqueue.Get()
			if shutdown {
				return
			}
			err := func(obj interface{}) error {
				defer c.workqueue.Done(obj)
				var key string
				event := obj.(*types.Event)
				if secret, ok := event.Object.(*corev1.Secret); !ok {
					c.workqueue.Forget(obj)
					return fmt.Errorf("expected Secret in workqueue but got %#v", obj)
				} else {
					if err := c.sync(ctx, obj.(*types.Event)); err != nil {
						c.workqueue.AddRateLimited(obj)
						log.Errorf("sync secret with ssl %s failed", secret.Namespace+"_"+secret.Name)
						return fmt.Errorf("error syncing '%s': %s", key, err.Error())
					}
					c.workqueue.Forget(obj)
					return nil
				}
			}(obj)
			if err != nil {
				runtime.HandleError(err)
			}
		}
	}

	for i := 0; i < c.workers; i++ {
		go handler()
	}

	<-ctx.Done()
	c.workqueue.ShutDown()
}

func (c *secretController) sync(ctx context.Context, ev *types.Event) error {
	obj := ev.Object.(*corev1.Secret)
	sec, err := c.controller.secretLister.Secrets(obj.Namespace).Get(obj.Name)

	key := obj.Namespace + "_" + obj.Name
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get Secret",
				zap.String("version", obj.ResourceVersion),
				zap.String("key", key),
				zap.Error(err),
			)
			return err
		}

		if ev.Type != types.EventDelete {
			log.Warnw("Secret was deleted before it can be delivered",
				zap.String("key", key),
				zap.String("version", obj.ResourceVersion),
			)
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if sec != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnw("discard the stale secret delete event since the resource still exists",
				zap.String("key", key),
			)
			return nil
		}
		sec = ev.Tombstone.(*corev1.Secret)
	}
	// sync SSL in APISIX which is store in secretSSLMap
	// FixMe Need to update the status of CRD ApisixTls
	ssls, ok := secretSSLMap.Load(key)
	if ok {
		sslMap := ssls.(sync.Map)
		sslMap.Range(func(_, v interface{}) bool {
			ssl := v.(*apisixv1.Ssl)
			err = state.SyncSsl(ssl, ev.Type.String())
			if err != nil {
				return false
			}
			return true
		})
	}
	return err
}

func (c *secretController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found secret object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}

	c.workqueue.AddRateLimited(&types.Event{
		Type:   types.EventAdd,
		Object: obj,
	})
}

func (c *secretController) onUpdate(prev, curr interface{}) {
	prevSec := prev.(*corev1.Secret)
	currSec := curr.(*corev1.Secret)

	if prevSec.GetResourceVersion() == currSec.GetResourceVersion() {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(currSec)
	if err != nil {
		log.Errorf("found secrets object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	c.workqueue.AddRateLimited(&types.Event{
		Type:   types.EventUpdate,
		Object: curr,
	})
}

func (c *secretController) onDelete(obj interface{}) {
	sec, ok := obj.(*corev1.Secret)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Errorf("found secrets: %+v in bad tombstone state", obj)
			return
		}
		sec = tombstone.Obj.(*corev1.Secret)
	}

	// FIXME Refactor Controller.namespaceWatching to just use
	// namespace after all controllers use the same way to fetch
	// the object.
	if !c.controller.namespaceWatching(sec.Namespace + "/" + sec.Name) {
		return
	}
	c.workqueue.AddRateLimited(&types.Event{
		Type:      types.EventDelete,
		Object:    sec,
		Tombstone: sec,
	})
}
