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

package framework

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/conformance/utils/kubernetes"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	v2 "github.com/apache/apisix-ingress-controller/api/v2"
)

func HTTPRouteMustHaveCondition(t testing.TestingT, cli client.Client, timeout time.Duration, refNN, hrNN types.NamespacedName, condition metav1.Condition) {
	err := PollUntilHTTPRouteHaveStatus(cli, timeout, hrNN, func(hr *gatewayv1.HTTPRoute) bool {
		for _, parent := range hr.Status.Parents {
			if err := kubernetes.ConditionsHaveLatestObservedGeneration(hr, parent.Conditions); err != nil {
				log.Printf("HTTPRoute %s (parentRef=%v) %v", hrNN, parentRefToString(parent.ParentRef), err)
				return false
			}
			if (refNN.Name == "" || parent.ParentRef.Name == gatewayv1.ObjectName(refNN.Name)) &&
				(refNN.Namespace == "" || (parent.ParentRef.Namespace != nil && string(*parent.ParentRef.Namespace) == refNN.Namespace)) {
				if findConditionInList(parent.Conditions, condition) {
					log.Printf("found condition %v in %v for %s reference %s", condition, parent.Conditions, hrNN, refNN)
					return true
				} else {
					log.Printf("NOT FOUND condition %v in %v for %s reference %s", condition, parent.Conditions, hrNN, refNN)
				}
			}
		}
		return false
	})
	require.NoError(t, err, "error waiting for HTTPRoute status to have a Condition matching %+v", condition)
}

func PollUntilHTTPRouteHaveStatus(cli client.Client, timeout time.Duration, hrNN types.NamespacedName, f func(route *gatewayv1.HTTPRoute) bool) error {
	if err := gatewayv1.Install(cli.Scheme()); err != nil {
		return err
	}
	return genericPollResource(new(gatewayv1.HTTPRoute), cli, timeout, hrNN, f)
}

func HTTPRoutePolicyMustHaveCondition(t testing.TestingT, client client.Client, timeout time.Duration, refNN, hrpNN types.NamespacedName,
	condition metav1.Condition) {
	err := PollUntilHTTPRoutePolicyHaveStatus(client, timeout, hrpNN, func(httpRoutePolicy *v1alpha1.HTTPRoutePolicy) bool {
		for _, ancestor := range httpRoutePolicy.Status.Ancestors {
			if err := kubernetes.ConditionsHaveLatestObservedGeneration(httpRoutePolicy, ancestor.Conditions); err != nil {
				log.Printf("HTTPRoutePolicy %s (parentRef=%v) %v", hrpNN, parentRefToString(ancestor.AncestorRef), err)
				return false
			}

			if ancestor.AncestorRef.Name == gatewayv1.ObjectName(refNN.Name) &&
				(refNN.Namespace == "" || (ancestor.AncestorRef.Namespace != nil && string(*ancestor.AncestorRef.Namespace) == refNN.Namespace)) {
				if findConditionInList(ancestor.Conditions, condition) {
					log.Printf("found condition %v in list %v for %s reference %s", condition, ancestor.Conditions, hrpNN, refNN)
					return true
				} else {
					log.Printf("NOT FOUND condition %v in %v for %s reference %s", condition, ancestor.Conditions, hrpNN, refNN)
				}
			}
		}
		return false
	})

	require.NoError(t, err, "error waiting for HTTPRoutePolicy %s status to have a Condition matching %+v", hrpNN, condition)
}

func PollUntilHTTPRoutePolicyHaveStatus(cli client.Client, timeout time.Duration, hrpNN types.NamespacedName,
	f func(httpRoutePolicy *v1alpha1.HTTPRoutePolicy) bool) error {
	if err := v1alpha1.AddToScheme(cli.Scheme()); err != nil {
		return err
	}
	return genericPollResource(new(v1alpha1.HTTPRoutePolicy), cli, timeout, hrpNN, f)
}

func ApisixConsumerMustHaveCondition(t testing.TestingT, cli client.Client, timeout time.Duration, nn types.NamespacedName, condition metav1.Condition) {
	err := PollUntilApisixConsumerMustHaveStatus(cli, timeout, nn, func(consumer *v2.ApisixConsumer) bool {
		if err := kubernetes.ConditionsHaveLatestObservedGeneration(consumer, consumer.Status.Conditions); err != nil {
			return false
		}
		if findConditionInList(consumer.Status.Conditions, condition) {
			return true
		}
		return false
	})

	require.NoError(t, err, "error waiting for ApisixConsumer %s status to have a Condition matching %+v", nn, condition)
}

func PollUntilApisixConsumerMustHaveStatus(cli client.Client, timeout time.Duration, nn types.NamespacedName, f func(consumer *v2.ApisixConsumer) bool) error {
	if err := v2.AddToScheme(cli.Scheme()); err != nil {
		return err
	}
	return genericPollResource(new(v2.ApisixConsumer), cli, timeout, nn, f)
}

func parentRefToString(p gatewayv1.ParentReference) string {
	if p.Namespace != nil && *p.Namespace != "" {
		return fmt.Sprintf("%v/%v", p.Namespace, p.Name)
	}
	return string(p.Name)
}

func findConditionInList(conditions []metav1.Condition, expected metav1.Condition) bool {
	return slices.ContainsFunc(conditions, func(item metav1.Condition) bool {
		// an empty Status string means "Match any status".
		// an empty Reason string means "Match any reason".
		return expected.Type == item.Type &&
			(expected.Status == "" || expected.Status == item.Status) &&
			(expected.Reason == "" || expected.Reason == item.Reason) &&
			(expected.Message == "" || strings.Contains(item.Message, expected.Message))
	})
}

func genericPollResource[Obj client.Object](obj Obj, cli client.Client, timeout time.Duration, nn types.NamespacedName, predicate func(Obj) bool) error {
	return wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (done bool, err error) {
		if err := cli.Get(ctx, nn, obj); err != nil {
			return false, errors.Wrapf(err, "error fetching Object %s", nn)
		}
		return predicate(obj), nil
	})
}

func NewApplier(t testing.TestingT, cli client.Client, apply func(string) error) Applier {
	return &applier{
		t:     t,
		cli:   cli,
		apply: apply,
	}
}

type Applier interface {
	MustApplyApisixConsumer(nn types.NamespacedName, spec string)
}

type applier struct {
	t     testing.TestingT
	cli   client.Client
	apply func(string) error
}

func (a *applier) MustApplyApisixConsumer(nn types.NamespacedName, spec string) {
	err := a.apply(spec)
	require.NoError(a.t, err, "creating ApisixConsumer", "request", nn, "spec", spec)

	ApisixConsumerMustHaveCondition(a.t, a.cli, 8*time.Second, nn, metav1.Condition{
		Type:   string(gatewayv1.RouteConditionAccepted),
		Status: metav1.ConditionTrue,
		Reason: string(gatewayv1.GatewayReasonAccepted),
	})
}
