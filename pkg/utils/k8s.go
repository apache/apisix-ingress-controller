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
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/apache/apisix-ingress-controller/internal/types"
)

const (
	// A discovery request that gives no definitive answer is retried a few times
	// to absorb short blips. A longer outage is reported to the caller instead:
	// it is not this function's job to decide how long to wait for the API server.
	discoveryRetryInitialInterval = 500 * time.Millisecond
	discoveryRetrySteps           = 5 // ~7.5s in total

	// The startup wait for the API server is deliberately finite. The health
	// probe endpoint is only served once mgr.Start runs, so a pod still waiting
	// here fails its liveness probe and is killed anyway; giving up with an
	// explicit error and letting the pod restart says more in the logs than
	// being killed mid-wait. Longer outages are absorbed by the restart backoff.
	apiServerWaitInitialInterval = 1 * time.Second
	apiServerWaitSteps           = 6 // ~31s in total

	backoffFactor = 2
	backoffJitter = 0.1
)

// Neither backoff sets Cap: wait.Backoff zeroes the remaining steps as soon as
// the capped interval is reached, which would silently cut the retries short.

func discoveryBackoff() wait.Backoff {
	return wait.Backoff{
		Duration: discoveryRetryInitialInterval,
		Factor:   backoffFactor,
		Jitter:   backoffJitter,
		Steps:    discoveryRetrySteps,
	}
}

func apiServerWaitBackoff() wait.Backoff {
	return wait.Backoff{
		Duration: apiServerWaitInitialInterval,
		Factor:   backoffFactor,
		Jitter:   backoffJitter,
		Steps:    apiServerWaitSteps,
	}
}

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

	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return false, fmt.Errorf("failed to create discovery client: %w", err)
	}

	return hasAPIResource(discoveryClient, gvk, discoveryBackoff(), logger)
}

func hasAPIResource(
	discoveryClient discovery.DiscoveryInterface,
	gvk schema.GroupVersionKind,
	backoff wait.Backoff,
	logger logr.Logger,
) (bool, error) {
	groupVersion := gvk.GroupVersion().String()

	logger = logger.WithValues(
		"kind", gvk.Kind,
		"group", gvk.Group,
		"version", gvk.Version,
		"groupVersion", groupVersion,
	)

	// Query server resources for the specific group/version
	var apiResources *metav1.APIResourceList
	err := retryUntilDefinitive(backoff, logger, func() error {
		var err error
		apiResources, err = discoveryClient.ServerResourcesForGroupVersion(groupVersion)
		return err
	})
	switch {
	case err == nil:
	case apierrors.IsNotFound(err):
		logger.Info("group/version not available in cluster", "error", err)
		return false, nil
	case apierrors.IsForbidden(err):
		// Definitive, but a misconfiguration rather than an absent CRD: log loudly
		// so it is not mistaken for "the CRD was never installed".
		logger.Error(err, "discovery is forbidden, treating the API resource as not installed; "+
			"check the discovery permissions of the controller service account")
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

// WaitForAPIServer blocks until the API server answers a discovery request, ctx
// is done, or the wait budget is exhausted.
//
// Capability detection (field indexes, optional CRDs, ReferenceGrant support) is
// decided once at startup and cannot be revised afterwards, so it must not run
// against an unreachable API server.
func WaitForAPIServer(ctx context.Context, cfg *rest.Config, logger logr.Logger) error {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}
	return waitForAPIServer(ctx, func() error {
		_, err := discoveryClient.ServerVersion()
		return err
	}, apiServerWaitBackoff(), logger)
}

func waitForAPIServer(ctx context.Context, probe func() error, backoff wait.Backoff, logger logr.Logger) error {
	var lastErr error
	if err := wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		if lastErr = probe(); lastErr != nil {
			logger.Info("waiting for the Kubernetes API server to become reachable", "error", lastErr)
			return false, nil
		}
		return true, nil
	}); err != nil {
		// Report why the API server did not answer; a cancelled ctx is the
		// caller shutting us down and must stay recognizable as such.
		if ctx.Err() == nil && lastErr != nil {
			err = lastErr
		}
		return fmt.Errorf("give up waiting for the Kubernetes API server: %w", err)
	}
	return nil
}

// retryUntilDefinitive runs probe until it returns nil or a definitive answer,
// and reports the last failure once the retry budget is exhausted.
func retryUntilDefinitive(backoff wait.Backoff, logger logr.Logger, probe func() error) error {
	var lastErr error
	//nolint:errcheck // the loop only ever exhausts its steps; lastErr carries the outcome.
	_ = wait.ExponentialBackoff(backoff, func() (bool, error) {
		if lastErr = probe(); lastErr == nil || isResourceAbsent(lastErr) {
			return true, nil
		}
		logger.Info("discovery request failed, retrying", "error", lastErr)
		return false, nil
	})
	return lastErr
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
