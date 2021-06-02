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
package types

import (
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodCacheBadCases(t *testing.T) {
	pc := NewPodCache()

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "pod1",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}
	assert.Equal(t, pc.Add(pod1), ErrPodNoAssignedIP, "adding pod")
	assert.Equal(t, pc.Delete(pod1), ErrPodNoAssignedIP, "deleting pod")
}

func TestPodCache(t *testing.T) {
	pc := NewPodCache()

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "pod1",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.5.11",
		},
	}
	assert.Equal(t, pc.Add(pod1), nil, "adding pod")
	name, err := pc.GetNameByIP("10.0.5.11")
	assert.Nil(t, err)
	assert.Equal(t, name, "pod1")

	name, err = pc.GetNameByIP("10.0.5.12")
	assert.Empty(t, name)
	assert.Equal(t, err, ErrPodNotFound)

	assert.Nil(t, pc.Delete(pod1), nil, "deleting pod")

	name, err = pc.GetNameByIP("10.0.5.11")
	assert.Empty(t, name)
	assert.Equal(t, err, ErrPodNotFound)
}
