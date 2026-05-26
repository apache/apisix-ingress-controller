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

package v1

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	v1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
	adctranslator "github.com/apache/apisix-ingress-controller/internal/adc/translator"
	"github.com/apache/apisix-ingress-controller/internal/controller"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

type adcAdmissionValidator struct {
	kubeClient             client.Client
	client                 *adcclient.Client
	translator             *adctranslator.Translator
	log                    logr.Logger
	defaultResolveEndpoint bool
}

func newADCAdmissionValidator(kubeClient client.Client, log logr.Logger) (*adcAdmissionValidator, error) {
	defaultMode := string(config.ControllerConfig.ProviderConfig.Type)
	cli, err := adcclient.New(log, defaultMode, config.ControllerConfig.ExecADCTimeout.Duration)
	if err != nil {
		return nil, err
	}

	return &adcAdmissionValidator{
		kubeClient:             kubeClient,
		client:                 cli,
		translator:             adctranslator.NewTranslator(log),
		log:                    log.WithName("adc-validation"),
		defaultResolveEndpoint: config.ControllerConfig.ProviderConfig.Type == config.ProviderTypeStandalone,
	}, nil
}

func (v *adcAdmissionValidator) Validate(ctx context.Context, obj client.Object) error {
	if v == nil {
		return nil
	}

	task, err := v.buildTask(ctx, obj)
	if err != nil {
		return err
	}
	if task == nil {
		return nil
	}

	if err := v.client.Validate(ctx, *task); err != nil {
		var validationErrs internaltypes.ADCValidationErrors
		if errors.As(err, &validationErrs) {
			return err
		}

		v.log.Error(err, "ADC validation unavailable, allowing admission", "resource", utils.NamespacedNameKind(obj))
		return nil
	}

	return nil
}

func (v *adcAdmissionValidator) buildTask(ctx context.Context, obj client.Object) (*adcclient.Task, error) {
	var (
		tctx          *provider.TranslateContext
		result        *adctranslator.TranslateResult
		resourceTypes []string
		err           error
	)

	switch resource := obj.(type) {
	case *apiv2.ApisixRoute:
		configs, err := v.buildIngressClassConfigs(ctx, resource.DeepCopy())
		if err != nil {
			return nil, err
		}
		if len(configs) == 0 {
			return nil, nil
		}
		tctx, err = controller.PrepareApisixRouteForValidation(ctx, v.kubeClient, v.log, resource.DeepCopy())
		if err != nil {
			return nil, err
		}
		result, err = v.translator.TranslateApisixRoute(tctx, resource.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return v.newTask(obj, configs, resourceTypes, result), nil
	case *apiv2.ApisixConsumer:
		configs, err := v.buildIngressClassConfigs(ctx, resource.DeepCopy())
		if err != nil {
			return nil, err
		}
		if len(configs) == 0 {
			return nil, nil
		}
		tctx, err = controller.PrepareApisixConsumerForValidation(ctx, v.kubeClient, v.log, resource.DeepCopy())
		if err != nil {
			return nil, err
		}
		result, err = v.translator.TranslateApisixConsumer(tctx, resource.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeConsumer)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return v.newTask(obj, configs, resourceTypes, result), nil
	case *v1alpha1.Consumer:
		tctx, err = controller.PrepareConsumerForValidation(ctx, v.kubeClient, v.log, resource.DeepCopy())
		if err != nil {
			return nil, err
		}
		result, err = v.translator.TranslateConsumerV1alpha1(tctx, resource.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeConsumer)
	case *apiv2.ApisixTls:
		configs, err := v.buildIngressClassConfigs(ctx, resource.DeepCopy())
		if err != nil {
			return nil, err
		}
		if len(configs) == 0 {
			return nil, nil
		}
		tctx, err = controller.PrepareApisixTlsForValidation(ctx, v.kubeClient, v.log, resource.DeepCopy())
		if err != nil {
			return nil, err
		}
		result, err = v.translator.TranslateApisixTls(tctx, resource.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeSSL)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return v.newTask(obj, configs, resourceTypes, result), nil
	default:
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	configs, err := v.buildConfigs(tctx)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return nil, nil
	}

	return v.newTask(obj, configs, resourceTypes, result), nil
}

func (v *adcAdmissionValidator) buildConfigs(tctx *provider.TranslateContext) (map[internaltypes.NamespacedNameKind]adctypes.Config, error) {
	configs := make(map[internaltypes.NamespacedNameKind]adctypes.Config, len(tctx.GatewayProxies))
	for key, gp := range tctx.GatewayProxies {
		cfg, err := v.translator.TranslateGatewayProxyToConfig(tctx, &gp, v.defaultResolveEndpoint)
		if err != nil {
			return nil, err
		}
		if cfg == nil {
			continue
		}
		configs[key] = *cfg
	}
	return configs, nil
}

func (v *adcAdmissionValidator) buildIngressClassConfigs(ctx context.Context, obj client.Object) (map[internaltypes.NamespacedNameKind]adctypes.Config, error) {
	tctx := provider.NewDefaultTranslateContext(ctx)

	ingressClass, err := controller.FindMatchingIngressClass(tctx, v.kubeClient, v.log, obj)
	if err != nil {
		return nil, err
	}
	if err := controller.ProcessIngressClassParameters(tctx, v.kubeClient, v.log, obj, ingressClass); err != nil {
		return nil, err
	}
	return v.buildConfigs(tctx)
}

func (v *adcAdmissionValidator) newTask(obj client.Object, configs map[internaltypes.NamespacedNameKind]adctypes.Config, resourceTypes []string, result *adctranslator.TranslateResult) *adcclient.Task {
	return &adcclient.Task{
		Key:           utils.NamespacedNameKind(obj),
		Name:          utils.NamespacedNameKind(obj).String(),
		Labels:        label.GenLabel(obj),
		Configs:       configs,
		ResourceTypes: resourceTypes,
		Resources: &adctypes.Resources{
			GlobalRules:    result.GlobalRules,
			PluginMetadata: result.PluginMetadata,
			Services:       result.Services,
			SSLs:           result.SSL,
			Consumers:      result.Consumers,
		},
	}
}
