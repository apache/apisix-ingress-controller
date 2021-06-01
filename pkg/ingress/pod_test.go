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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/types"
)

func TestPodOnAdd(t *testing.T) {
	ctl := &podController{
		controller: &Controller{
			watchingNamespace: map[string]struct{}{
				"default": {},
			},
			podCache: types.NewPodCache(),
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "nginx",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.5.12",
		},
	}
	ctl.onAdd(pod)
	name, err := ctl.controller.podCache.GetNameByIP("10.0.5.12")
	assert.Nil(t, err)
	assert.Equal(t, name, "nginx")

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "public",
			Name:      "abc",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.5.13",
		},
	}
	ctl.onAdd(pod2)
	name, err = ctl.controller.podCache.GetNameByIP("10.0.5.13")
	assert.Empty(t, name)
	assert.Equal(t, err, types.ErrPodNotFound)
}

func TestPodOnDelete(t *testing.T) {
	ctl := &podController{
		controller: &Controller{
			watchingNamespace: map[string]struct{}{
				"default": {},
			},
			podCache: types.NewPodCache(),
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "nginx",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.5.12",
		},
	}
	assert.Nil(t, ctl.controller.podCache.Add(pod), "adding pod")

	ctl.onDelete(pod)
	name, err := ctl.controller.podCache.GetNameByIP("10.0.5.12")
	assert.Empty(t, name)
	assert.Equal(t, err, types.ErrPodNotFound)

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "public",
			Name:      "abc",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.5.13",
		},
	}
	assert.Nil(t, ctl.controller.podCache.Add(pod2), "adding pod")
	ctl.onDelete(pod2)
	name, err = ctl.controller.podCache.GetNameByIP("10.0.5.13")
	assert.Equal(t, name, "abc")
	assert.Nil(t, err)
}

func TestPodOnUpdate(t *testing.T) {
	ctl := &podController{
		controller: &Controller{
			watchingNamespace: map[string]struct{}{
				"default": {},
			},
			podCache: types.NewPodCache(),
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "nginx",
			DeletionTimestamp: &metav1.Time{
				Time: time.Now(),
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.5.12",
		},
	}
	assert.Nil(t, ctl.controller.podCache.Add(pod), "adding pod")

	ctl.onUpdate(nil, pod)
	name, err := ctl.controller.podCache.GetNameByIP("10.0.5.12")
	assert.Equal(t, name, "nginx")
	assert.Equal(t, err, nil)

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "public",
			Name:      "abc",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.5.13",
		},
	}
	assert.Nil(t, ctl.controller.podCache.Add(pod2), "adding pod")
	ctl.onUpdate(nil, pod2)
	name, err = ctl.controller.podCache.GetNameByIP("10.0.5.13")
	assert.Equal(t, name, "abc")
	assert.Nil(t, err)
}
