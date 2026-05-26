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

package v1

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	apisixv1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/webhook/v1/reference"
)

var consumerLog = logf.Log.WithName("consumer-resource")

func SetupConsumerWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&apisixv1alpha1.Consumer{}).
		WithValidator(NewConsumerCustomValidator(mgr.GetClient())).
		Complete()
}

// +kubebuilder:webhook:path=/validate-apisix-apache-org-v1alpha1-consumer,mutating=false,failurePolicy=Ignore,sideEffects=None,groups=apisix.apache.org,resources=consumers,verbs=create;update,versions=v1alpha1,name=vconsumer-v1alpha1.kb.io,admissionReviewVersions=v1

type ConsumerCustomValidator struct {
	Client       client.Client
	checker      reference.Checker
	adcValidator *adcAdmissionValidator
	initErr      error
}

var _ webhook.CustomValidator = &ConsumerCustomValidator{}

func NewConsumerCustomValidator(c client.Client) *ConsumerCustomValidator {
	adcValidator, err := newADCAdmissionValidator(c, consumerLog)
	return &ConsumerCustomValidator{
		Client:       c,
		checker:      reference.NewChecker(c, consumerLog),
		adcValidator: adcValidator,
		initErr:      err,
	}
}

func (v *ConsumerCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	consumer, ok := obj.(*apisixv1alpha1.Consumer)
	if !ok {
		return nil, fmt.Errorf("expected a Consumer object but got %T", obj)
	}
	consumerLog.Info("Validation for Consumer upon creation", "name", consumer.GetName(), "namespace", consumer.GetNamespace())
	if !controller.MatchConsumerGatewayRef(ctx, v.Client, consumerLog, consumer) {
		return nil, nil
	}

	warnings := v.collectWarnings(ctx, consumer)
	if v.initErr != nil {
		consumerLog.Error(v.initErr, "ADC validator init failed, skipping ADC validation")
		return warnings, nil
	}
	if err := v.validateDuplicateKeyAuthCredentials(ctx, consumer); err != nil {
		return warnings, err
	}
	return warnings, v.adcValidator.Validate(ctx, consumer)
}

func (v *ConsumerCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	consumer, ok := newObj.(*apisixv1alpha1.Consumer)
	if !ok {
		return nil, fmt.Errorf("expected a Consumer object for the newObj but got %T", newObj)
	}
	consumerLog.Info("Validation for Consumer upon update", "name", consumer.GetName(), "namespace", consumer.GetNamespace())
	if !controller.MatchConsumerGatewayRef(ctx, v.Client, consumerLog, consumer) {
		return nil, nil
	}

	warnings := v.collectWarnings(ctx, consumer)
	if v.initErr != nil {
		consumerLog.Error(v.initErr, "ADC validator init failed, skipping ADC validation")
		return warnings, nil
	}
	if err := v.validateDuplicateKeyAuthCredentials(ctx, consumer); err != nil {
		return warnings, err
	}
	return warnings, v.adcValidator.Validate(ctx, consumer)
}

func (*ConsumerCustomValidator) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *ConsumerCustomValidator) collectWarnings(ctx context.Context, consumer *apisixv1alpha1.Consumer) admission.Warnings {
	defaultNamespace := consumer.GetNamespace()

	visited := make(map[types.NamespacedName]struct{})
	var warnings admission.Warnings

	for _, credential := range consumer.Spec.Credentials {
		if credential.SecretRef == nil || credential.SecretRef.Name == "" {
			continue
		}

		namespace := defaultNamespace
		if credential.SecretRef.Namespace != nil && *credential.SecretRef.Namespace != "" {
			namespace = *credential.SecretRef.Namespace
		}

		nn := types.NamespacedName{Namespace: namespace, Name: credential.SecretRef.Name}
		if _, ok := visited[nn]; ok {
			continue
		}
		visited[nn] = struct{}{}

		warnings = append(warnings, v.checker.Secret(ctx, reference.SecretRef{
			Object:         consumer,
			NamespacedName: nn,
		})...)
	}

	return warnings
}

func (v *ConsumerCustomValidator) validateDuplicateKeyAuthCredentials(ctx context.Context, consumer *apisixv1alpha1.Consumer) error {
	keys, err := v.extractKeyAuthKeys(ctx, consumer)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}

	// Use the consumerGatewayRef field index to list only Consumers sharing the same gateway.
	ns := consumer.Namespace
	if consumer.Spec.GatewayRef.Namespace != nil && *consumer.Spec.GatewayRef.Namespace != "" {
		ns = *consumer.Spec.GatewayRef.Namespace
	}
	indexKey := indexer.GenIndexKey(ns, consumer.Spec.GatewayRef.Name)

	var consumers apisixv1alpha1.ConsumerList
	if err := v.Client.List(ctx, &consumers, client.MatchingFields{indexer.ConsumerGatewayRef: indexKey}); err != nil {
		return err
	}

	for i := range consumers.Items {
		existing := &consumers.Items[i]
		if existing.Namespace == consumer.Namespace && existing.Name == consumer.Name {
			continue
		}

		existingKeys, err := v.extractKeyAuthKeys(ctx, existing)
		if err != nil {
			return err
		}
		for key := range existingKeys {
			if _, ok := keys[key]; ok {
				return fmt.Errorf("duplicate key-auth credential key %q already used by Consumer %s/%s", key, existing.Namespace, existing.Name)
			}
		}
	}

	return nil
}

func (v *ConsumerCustomValidator) extractKeyAuthKeys(ctx context.Context, consumer *apisixv1alpha1.Consumer) (map[string]struct{}, error) {
	keys := make(map[string]struct{})

	for _, credential := range consumer.Spec.Credentials {
		if credential.Type != "key-auth" {
			continue
		}

		key, err := v.extractCredentialKey(ctx, consumer, credential)
		if err != nil {
			return nil, err
		}
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
	}

	return keys, nil
}

func (v *ConsumerCustomValidator) extractCredentialKey(ctx context.Context, consumer *apisixv1alpha1.Consumer, credential apisixv1alpha1.Credential) (string, error) {
	if credential.SecretRef != nil && credential.SecretRef.Name != "" {
		namespace := consumer.Namespace
		if credential.SecretRef.Namespace != nil && *credential.SecretRef.Namespace != "" {
			namespace = *credential.SecretRef.Namespace
		}

		var secret corev1.Secret
		err := v.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: credential.SecretRef.Name}, &secret)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return "", nil
			}
			return "", err
		}
		return string(secret.Data["key"]), nil
	}

	if len(credential.Config.Raw) == 0 {
		return "", nil
	}

	var cfg struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(credential.Config.Raw, &cfg); err != nil {
		// Malformed JSON is not a hard error: skip duplicate detection for this
		// credential so existing consumers with bad config are not suddenly denied.
		consumerLog.V(1).Info("skipping duplicate key-auth check: malformed credential config",
			"consumer", consumer.Name, "error", err)
		return "", nil
	}
	return cfg.Key, nil
}
