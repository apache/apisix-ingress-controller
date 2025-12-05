// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package utils

import (
	"github.com/go-logr/logr"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/apache/apisix-ingress-controller/internal/types"
)

// HasAPIResource checks if a specific API resource is available in the current cluster.
// It uses the Discovery API to query the cluster's available resources and returns true
// if the resource is found, false otherwise.
func HasAPIResource(mgr ctrl.Manager, obj client.Object) bool {
	return HasAPIResourceWithLogger(mgr, obj, ctrl.Log.WithName("api-detection"))
}

// HasAPIResourceWithLogger is the same as HasAPIResource but accepts a custom logger
// for more detailed debugging information.
func HasAPIResourceWithLogger(mgr ctrl.Manager, obj client.Object, logger logr.Logger) bool {
	gvk, err := apiutil.GVKForObject(obj, mgr.GetScheme())
	if err != nil {
		logger.Info("cannot derive GVK from scheme", "error", err)
		return false
	}

	groupVersion := gvk.GroupVersion().String()

	logger = logger.WithValues(
		"kind", gvk.Kind,
		"group", gvk.Group,
		"version", gvk.Version,
		"groupVersion", groupVersion,
	)

	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		logger.Info("failed to create discovery client", "error", err)
		return false
	}

	// Query server resources for the specific group/version
	apiResources, err := discoveryClient.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		logger.Info("group/version not available in cluster", "error", err)
		return false
	}

	// Check if the specific kind exists in the resource list
	for _, res := range apiResources.APIResources {
		if res.Kind == gvk.Kind {
			return true
		}
	}

	logger.Info("API resource kind not found in group/version")
	return false
}

func FormatGVK(obj client.Object) string {
	gvk := types.GvkOf(obj)
	return gvk.String()
}
