// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package status

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const UpdateChannelBufferSize = 1000

type Update struct {
	NamespacedName types.NamespacedName
	Resource       client.Object
	Mutator        Mutator
}

type Mutator interface {
	Mutate(obj client.Object) client.Object
}

type MutatorFunc func(client.Object) client.Object

func (m MutatorFunc) Mutate(obj client.Object) client.Object {
	if m == nil {
		return nil
	}

	return m(obj)
}

type UpdateHandler struct {
	log           logr.Logger
	client        client.Client
	updateChannel chan Update
	wg            *sync.WaitGroup
}

func NewStatusUpdateHandler(log logr.Logger, client client.Client) *UpdateHandler {
	u := &UpdateHandler{
		log:           log,
		client:        client,
		updateChannel: make(chan Update, UpdateChannelBufferSize),
		wg:            new(sync.WaitGroup),
	}

	u.wg.Add(1)
	return u
}

func (u *UpdateHandler) apply(ctx context.Context, update Update) {
	if err := retry.OnError(retry.DefaultBackoff, func(err error) bool {
		return k8serrors.IsConflict(err)
	}, func() error {
		return u.updateStatus(ctx, update)
	}); err != nil {
		u.log.Error(err, "unable to update status", "name", update.NamespacedName.Name,
			"namespace", update.NamespacedName.Namespace)
	}
}

func (u *UpdateHandler) updateStatus(ctx context.Context, update Update) error {
	var obj = update.Resource
	if err := u.client.Get(ctx, update.NamespacedName, obj); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	newObj := update.Mutator.Mutate(obj)
	if newObj == nil {
		return nil
	}

	newObj.SetUID(obj.GetUID())

	return u.client.Status().Update(ctx, newObj)
}

func (u *UpdateHandler) Start(ctx context.Context) error {
	u.log.Info("started status update handler")
	defer u.log.Info("stopped status update handler")

	u.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-u.updateChannel:
			u.log.Info("received a status update", "namespace", update.NamespacedName.Namespace,
				"name", update.NamespacedName.Name)

			u.apply(ctx, update)
		}
	}
}

func (u *UpdateHandler) Writer() Updater {
	return &UpdateWriter{
		updateChannel: u.updateChannel,
		wg:            u.wg,
	}
}

type Updater interface {
	Update(u Update)
}

type UpdateWriter struct {
	updateChannel chan<- Update
	wg            *sync.WaitGroup
}

func (u *UpdateWriter) Update(update Update) {
	u.wg.Wait()
	u.updateChannel <- update
}
