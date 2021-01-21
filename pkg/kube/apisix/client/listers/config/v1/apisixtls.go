/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/api7/ingress-controller/pkg/kube/apisix/apis/config/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ApisixTLSLister helps list ApisixTLSs.
// All objects returned here must be treated as read-only.
type ApisixTLSLister interface {
	// List lists all ApisixTLSs in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.ApisixTLS, err error)
	// ApisixTLSs returns an object that can list and get ApisixTLSs.
	ApisixTLSs(namespace string) ApisixTLSNamespaceLister
	ApisixTLSListerExpansion
}

// apisixTLSLister implements the ApisixTLSLister interface.
type apisixTLSLister struct {
	indexer cache.Indexer
}

// NewApisixTLSLister returns a new ApisixTLSLister.
func NewApisixTLSLister(indexer cache.Indexer) ApisixTLSLister {
	return &apisixTLSLister{indexer: indexer}
}

// List lists all ApisixTLSs in the indexer.
func (s *apisixTLSLister) List(selector labels.Selector) (ret []*v1.ApisixTLS, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.ApisixTLS))
	})
	return ret, err
}

// ApisixTLSs returns an object that can list and get ApisixTLSs.
func (s *apisixTLSLister) ApisixTLSs(namespace string) ApisixTLSNamespaceLister {
	return apisixTLSNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ApisixTLSNamespaceLister helps list and get ApisixTLSs.
// All objects returned here must be treated as read-only.
type ApisixTLSNamespaceLister interface {
	// List lists all ApisixTLSs in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.ApisixTLS, err error)
	// Get retrieves the ApisixTLS from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.ApisixTLS, error)
	ApisixTLSNamespaceListerExpansion
}

// apisixTLSNamespaceLister implements the ApisixTLSNamespaceLister
// interface.
type apisixTLSNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ApisixTLSs in the indexer for a given namespace.
func (s apisixTLSNamespaceLister) List(selector labels.Selector) (ret []*v1.ApisixTLS, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.ApisixTLS))
	})
	return ret, err
}

// Get retrieves the ApisixTLS from the indexer for a given namespace and name.
func (s apisixTLSNamespaceLister) Get(name string) (*v1.ApisixTLS, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("apisixtls"), name)
	}
	return obj.(*v1.ApisixTLS), nil
}
