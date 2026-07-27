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
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/apache/apisix-ingress-controller/internal/types"
)

const (
	// A discovery request that neither succeeds nor gives a definitive answer is
	// retried a few times to absorb short blips. A longer outage is reported to
	// the caller instead: it is not this function's job to decide how long the
	// process should wait for the API server.
	discoveryRetryInterval = 500 * time.Millisecond
	discoveryRetryFactor   = 2
	discoveryRetrySteps    = 4

	// Interval bounds used while waiting for the API server to become reachable.
	apiServerWaitInitialInterval = 1 * time.Second
	apiServerWaitMaxInterval     = 30 * time.Second
)

// HasAPIResource reports whether the API resource of obj is served by the cluster.
//
// An error is returned when discovery cannot produce a definitive answer, for
// instance because the API server is unreachable. Callers must not fall back to
// "resource absent" in that case: detection runs once at startup and gates field
// index, controller and readiness registration, so a temporary API server outage
// would otherwise leave the controller permanently degraded until it is
// restarted.
func HasAPIResource(mgr ctrl.Manager, obj client.Object) (bool, error) {
	return HasAPIResourceWithLogger(mgr, obj, ctrl.Log.WithName("api-detection"))
}

// HasAPIResourceWithLogger is the same as HasAPIResource but accepts a custom logger
// for more detailed debugging information.
func HasAPIResourceWithLogger(mgr ctrl.Manager, obj client.Object, logger logr.Logger) (bool, error) {
	gvk, err := apiutil.GVKForObject(obj, mgr.GetScheme())
	if err != nil {
		return false, fmt.Errorf("cannot derive GVK from scheme: %w", err)
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
		return false, fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Query server resources for the specific group/version
	var apiResources *metav1.APIResourceList
	err = discoverWithRetry(func() error {
		var err error
		apiResources, err = discoveryClient.ServerResourcesForGroupVersion(groupVersion)
		return err
	}, logger, time.Sleep)
	switch {
	case err == nil:
	case isResourceAbsent(err):
		logger.Info("group/version not available in cluster", "error", err)
		return false, nil
	default:
		return false, fmt.Errorf("failed to detect API resource %s: %w", gvk, err)
	}

	// Check if the specific kind exists in the resource list
	for _, res := range apiResources.APIResources {
		if res.Kind == gvk.Kind {
			return true, nil
		}
	}

	logger.Info("API resource kind not found in group/version")
	return false, nil
}

// WaitForAPIServer blocks until the API server answers a discovery request, or
// ctx is done.
//
// Capability detection (field indexes, optional CRDs, ReferenceGrant support) is
// decided once at startup and cannot be revised afterwards, so it must not run
// against an unreachable API server.
func WaitForAPIServer(ctx context.Context, cfg *rest.Config, logger logr.Logger) error {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}
	backoff := wait.Backoff{
		Duration: apiServerWaitInitialInterval,
		Factor:   discoveryRetryFactor,
		Jitter:   0.1,
		Cap:      apiServerWaitMaxInterval,
		// Retry until ctx is done rather than for a fixed number of attempts:
		// the controller cannot do anything useful without the API server.
		Steps: math.MaxInt32,
	}
	return waitForAPIServer(ctx, discoveryClient.ServerVersion, backoff, logger)
}

func waitForAPIServer(ctx context.Context, probe func() (*version.Info, error), backoff wait.Backoff, logger logr.Logger) error {
	var lastErr error
	if err := wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		if _, lastErr = probe(); lastErr != nil {
			logger.Info("waiting for the Kubernetes API server to become reachable", "error", lastErr)
			return false, nil
		}
		return true, nil
	}); err != nil {
		if lastErr != nil {
			err = lastErr
		}
		return fmt.Errorf("give up waiting for the Kubernetes API server: %w", err)
	}
	return nil
}

// discoverWithRetry runs probe until it returns nil or a definitive answer,
// retrying at most discoveryRetrySteps times. sleep is injected for testability.
func discoverWithRetry(probe func() error, logger logr.Logger, sleep func(time.Duration)) error {
	interval := discoveryRetryInterval
	for i := 0; ; i++ {
		err := probe()
		if err == nil || isResourceAbsent(err) || i == discoveryRetrySteps {
			return err
		}
		logger.Info("discovery request failed, retrying", "error", err, "retryIn", interval.String())
		sleep(interval)
		interval *= discoveryRetryFactor
	}
}

// isResourceAbsent reports whether err definitively means the group/version is
// not usable, as opposed to discovery having failed to reach a conclusion.
func isResourceAbsent(err error) bool {
	switch {
	// Discovery of a group/version that is not installed answers 404.
	case apierrors.IsNotFound(err):
		return true
	// RBAC forbids discovery: the resource is not usable by this controller and
	// waiting will not change that.
	case apierrors.IsForbidden(err):
		return true
	// Connection refused, timeouts, EOF, 5xx, throttling: the API server may
	// answer differently once it is reachable again, so no conclusion yet.
	default:
		return false
	}
}

func FormatGVK(obj client.Object) string {
	gvk := types.GvkOf(obj)
	return gvk.String()
}
