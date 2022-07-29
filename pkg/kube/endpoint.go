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
package kube

import (
	"errors"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	listersdiscoveryv1 "k8s.io/client-go/listers/discovery/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/log"
)

type HostPort struct {
	Host string
	Port int
}

// EndpointLister is an encapsulation for the lister of Kubernetes
// Endpoint and EndpointSlice.
type EndpointLister interface {
	// GetEndpoint fetches an Endpoint which entity is the Kubernetes
	// Endpoint according to the namespace and name.
	// Automatically choose from endpoint/endpointslice
	GetEndpoint(string, string) (Endpoint, error)
}

type endpointLister struct {
	useEndpointSlice bool
	epLister         listerscorev1.EndpointsLister
	epsLister        listersdiscoveryv1.EndpointSliceLister
}

func (lister *endpointLister) GetEndpoint(namespace, name string) (Endpoint, error) {
	if lister.useEndpointSlice {
		return lister.getEndpointSlices(namespace, name)
	}
	return lister.getEndpoint(namespace, name)
}

func (lister *endpointLister) getEndpoint(namespace, name string) (Endpoint, error) {
	if lister.epLister == nil {
		panic("not a endpoint lister")
	}
	ep, err := lister.epLister.Endpoints(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	if ep == nil {
		log.Warnw("get endpoints but found nil",
			zap.String("namespace", namespace),
			zap.String("name", name),
		)
	}
	return &endpoint{
		endpointType: endpointTypeEndpoints,
		endpoint:     ep,
	}, nil
}

func (lister *endpointLister) getEndpointSlices(namespace, svcName string) (Endpoint, error) {
	if lister.epsLister == nil {
		panic("not a endpointSlice lister")
	}
	selector := labels.SelectorFromSet(labels.Set{
		discoveryv1.LabelServiceName: svcName,
	})
	eps, err := lister.epsLister.EndpointSlices(namespace).List(selector)
	if err != nil {
		return nil, err
	}
	if len(eps) == 0 {
		log.Warnw("get endpoint slices but found empty slice",
			zap.String("namespace", namespace),
			zap.String("service", svcName),
			zap.Any("selector", selector),
		)
	}
	return &endpoint{
		endpointType:   endpointTypeEndpointSlices,
		endpointSlices: eps,
	}, nil
}

// Endpoint is an encapsulation for the Kubernetes Endpoint and EndpointSlice objects.
type Endpoint interface {
	// ServiceName returns the corresponding service owner of this endpoint.
	ServiceName() string
	// Namespace returns the residing namespace.
	Namespace() (string, error)
	// Endpoints returns the corresponding endpoints which matches the ServicePort.
	Endpoints(port *corev1.ServicePort) []HostPort
}

type endpointType string

const (
	endpointTypeEndpoints      endpointType = "Endpoint"
	endpointTypeEndpointSlices endpointType = "EndpointSlices"
)

type endpoint struct {
	endpointType   endpointType
	endpoint       *corev1.Endpoints
	endpointSlices []*discoveryv1.EndpointSlice
}

func (e *endpoint) ServiceName() string {
	if e.endpoint != nil {
		return e.endpoint.Name
	}
	return e.endpointSlices[0].Labels[discoveryv1.LabelServiceName]
}

func (e *endpoint) Namespace() (string, error) {
	switch e.endpointType {
	case endpointTypeEndpointSlices:
		if len(e.endpointSlices) > 0 {
			return e.endpointSlices[0].Namespace, nil
		} else {
			return "", errors.New("endpoint slice is empty")
		}
	case endpointTypeEndpoints:
		if e.endpoint != nil {
			return e.endpoint.Namespace, nil
		} else {
			return "", errors.New("endpoint is nil")
		}
	}
	return "", errors.New("unknown endpoint type " + string(e.endpointType))
}

func (e *endpoint) Endpoints(svcPort *corev1.ServicePort) []HostPort {
	var addrs []HostPort
	if e.endpoint != nil {
		for _, subset := range e.endpoint.Subsets {
			epPort := -1
			for _, subsetPort := range subset.Ports {
				if subsetPort.Name == svcPort.Name {
					epPort = int(subsetPort.Port)
					break
				}
			}
			if epPort != -1 {
				for _, addr := range subset.Addresses {
					addrs = append(addrs, HostPort{
						Host: addr.IP,
						Port: epPort,
					})
				}
			}
		}
	} else {
		for _, slice := range e.endpointSlices {
			epPort := -1
			for _, slicePort := range slice.Ports {
				// TODO Consider the case that port not restricted.
				if slicePort.Name != nil && *slicePort.Name == svcPort.Name && slicePort.Port != nil {
					epPort = int(*slicePort.Port)
					break
				}
			}
			if epPort != -1 {
				for _, ep := range slice.Endpoints {
					if ep.Conditions.Ready != nil && !*ep.Conditions.Ready {
						// Ignore not ready endpoints.
						continue
					}
					for _, addr := range ep.Addresses {
						addrs = append(addrs, HostPort{
							Host: addr,
							Port: epPort,
						})
					}
				}
			}
		}
	}
	return addrs
}

// NewEndpointListerAndInformer creates an EndpointLister and the sharedIndexInformer.
func NewEndpointListerAndInformer(factory informers.SharedInformerFactory, useEndpointSlice bool) (EndpointLister, cache.SharedIndexInformer) {
	var informer cache.SharedIndexInformer
	epLister := endpointLister{
		useEndpointSlice: useEndpointSlice,
	}
	if !useEndpointSlice {
		epLister.epLister = factory.Core().V1().Endpoints().Lister()
		informer = factory.Core().V1().Endpoints().Informer()
	} else {
		epLister.epsLister = factory.Discovery().V1().EndpointSlices().Lister()
		informer = factory.Discovery().V1().EndpointSlices().Informer()
	}
	return &epLister, informer
}

// NewEndpoint creates an Endpoint which entity is Kubernetes Endpoints.
func NewEndpoint(ep *corev1.Endpoints) Endpoint {
	return &endpoint{
		endpointType: endpointTypeEndpoints,
		endpoint:     ep,
	}
}

// NewEndpointWithSlice creates an Endpoint which entity is Kubernetes EndpointSlices.
func NewEndpointWithSlice(ep *discoveryv1.EndpointSlice) Endpoint {
	return &endpoint{
		endpointType:   endpointTypeEndpointSlices,
		endpointSlices: []*discoveryv1.EndpointSlice{ep},
	}
}
