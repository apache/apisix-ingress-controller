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
	"errors"
	"sync"

	corev1 "k8s.io/api/core/v1"
)

var (
	// ErrPodNoAssignedIP represents that PodCache operation is failed due to the
	// target Pod is not in Running phase.
	ErrPodNoAssignedIP = errors.New("pod not running")
	// ErrPodNotFound represents that the target pod not found from the PodCache.
	ErrPodNotFound = errors.New("pod not found")
)

// PodCache caches pod. Currently it doesn't cache the pod object but only its
// name.
type PodCache interface {
	// Add adds a pod to cache, only pod which state is RUNNING will be
	// accepted.
	Add(*corev1.Pod) error
	// Delete deletes a pod from the cache
	Delete(*corev1.Pod) error
	// GetNameByIP returns the pod name according to the given pod IP.
	GetNameByIP(string) (string, error)
}

type podCache struct {
	sync.RWMutex

	nameByIP map[string]string
}

// NewPodCache creates a PodCache object.
func NewPodCache() PodCache {
	return &podCache{
		nameByIP: make(map[string]string),
	}
}

func (p *podCache) Add(pod *corev1.Pod) error {
	ip := pod.Status.PodIP
	if len(ip) == 0 {
		return ErrPodNoAssignedIP
	}
	p.Lock()
	defer p.Unlock()
	p.nameByIP[ip] = pod.Name
	return nil
}

func (p *podCache) Delete(pod *corev1.Pod) error {
	ip := pod.Status.PodIP
	if len(ip) == 0 {
		return ErrPodNoAssignedIP
	}
	p.Lock()
	defer p.Unlock()
	delete(p.nameByIP, ip)
	return nil
}

func (p *podCache) GetNameByIP(ip string) (name string, err error) {
	p.RLock()
	defer p.RUnlock()
	name, ok := p.nameByIP[ip]
	if !ok {
		err = ErrPodNotFound
	}
	return
}
