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

package validation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	v1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta1"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

func NewPluginValidatorHandler() http.Handler {
	// Create a validating webhook.
	wh, err := kwhvalidating.NewWebhook(kwhvalidating.WebhookConfig{
		ID:        "apisixRoute-plugin",
		Validator: pluginValidator,
	})
	if err != nil {
		log.Errorf("failed to create webhook: %s", err)
	}

	h, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: wh})
	if err != nil {
		log.Errorf("failed to create webhook handle: %s", err)
	}

	return h
}

// ErrNotApisixRoute will be used when the validating object is not ApisixRoute.
var ErrNotApisixRoute = errors.New("object is not ApisixRoute")

type apisixRoutePlugin struct {
	Name   string
	Config interface{}
}

// pluginValidator validates plugins in ApisixRoute.
// When the validation of one plugin fails, it will continue to validate the rest of plugins.
var pluginValidator = kwhvalidating.ValidatorFunc(
	func(ctx context.Context, review *kwhmodel.AdmissionReview, object metav1.Object) (result *kwhvalidating.ValidatorResult, err error) {
		valid := true

		var plugins []apisixRoutePlugin

		switch ar := object.(type) {
		case *v2beta1.ApisixRoute:
			for _, h := range ar.Spec.HTTP {
				for _, p := range h.Plugins {
					// only check plugins that are enabled.
					if p.Enable {
						plugins = append(plugins, apisixRoutePlugin{
							p.Name, p.Config,
						})
					}
				}
			}
		case *v2alpha1.ApisixRoute:
			for _, h := range ar.Spec.HTTP {
				for _, p := range h.Plugins {
					if p.Enable {
						plugins = append(plugins, apisixRoutePlugin{
							p.Name, p.Config,
						})
					}
				}
			}
		case *v1.ApisixRoute:
			for _, r := range ar.Spec.Rules {
				for _, path := range r.Http.Paths {
					for _, p := range path.Plugins {
						if p.Enable {
							plugins = append(plugins, apisixRoutePlugin{
								p.Name, p.Config,
							})
						}
					}
				}
			}
		default:
			return &kwhvalidating.ValidatorResult{Valid: false, Message: ErrNotApisixRoute.Error()}, ErrNotApisixRoute
		}

		client, err := GetSchemaClient()
		if err != nil {
			log.Errorf("failed to get the schema client: %s", err)
			return &kwhvalidating.ValidatorResult{Valid: false, Message: "failed to get the schema client"}, err
		}

		var msg []string
		for _, p := range plugins {
			if v, m, err := validatePlugin(client, p.Name, p.Config); !v {
				valid = false
				msg = append(msg, m)
				log.Warnf("failed to validate plugin %s: %s", p.Name, err)
			}
		}

		return &kwhvalidating.ValidatorResult{Valid: valid, Message: strings.Join(msg, "\n")}, nil
	})

func validatePlugin(client apisix.Schema, pluginName string, pluginConfig interface{}) (valid bool, msg string, err error) {
	valid = true

	pluginSchema, err := client.GetPluginSchema(context.TODO(), pluginName)
	if err != nil {
		log.Errorf("failed to get the schema of plugin %s: %s", pluginName, err)
		valid = false
		msg = fmt.Sprintf("failed to get the schema of plugin %s", pluginName)
		return
	}

	var errorMsg []string
	if err, re := validateSchema(pluginSchema.Content, pluginConfig); err != nil {
		valid = false
		msg = fmt.Sprintf("%s plugin's config is invalid\n", pluginName)
		for _, desc := range re {
			errorMsg = append(errorMsg, desc.String())
		}
	}

	if len(errorMsg) > 0 {
		msg = strings.Join(errorMsg, "\n")
	}
	return
}
