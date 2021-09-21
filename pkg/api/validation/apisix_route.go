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
	"strings"

	"github.com/hashicorp/go-multierror"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	"github.com/xeipuuv/gojsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	v1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta1"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

// errNotApisixRoute will be used when the validating object is not ApisixRoute.
var errNotApisixRoute = errors.New("object is not ApisixRoute")

type apisixRoutePlugin struct {
	Name   string
	Config interface{}
}

// ApisixRouteValidator validates ApisixRoute and its plugins.
// When the validation of one plugin fails, it will continue to validate the rest of plugins.
var ApisixRouteValidator = kwhvalidating.ValidatorFunc(
	func(ctx context.Context, review *kwhmodel.AdmissionReview, object metav1.Object) (result *kwhvalidating.ValidatorResult, err error) {
		log.Debug("arrive ApisixRoute validator webhook")

		valid := true
		var plugins []apisixRoutePlugin
		var spec interface{}

		switch ar := object.(type) {
		case *v2beta1.ApisixRoute:
			spec = ar.Spec

			// validate plugins
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
			spec = ar.Spec

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
			spec = ar.Spec

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
			return &kwhvalidating.ValidatorResult{Valid: false, Message: errNotApisixRoute.Error()}, errNotApisixRoute
		}

		client, err := GetSchemaClient(&apisix.ClusterOptions{})
		if err != nil {
			msg := "failed to get the schema client"
			log.Errorf("%s: %s", msg, err)
			return &kwhvalidating.ValidatorResult{Valid: false, Message: msg}, err
		}

		rs, err := client.GetRouteSchema(ctx)
		if err != nil {
			msg := "failed to get route's schema"
			log.Errorf("%s: %s", msg, err)
			return &kwhvalidating.ValidatorResult{Valid: false, Message: msg}, err
		}
		arSchemaLoader := gojsonschema.NewStringLoader(rs.Content)

		var msgs []string
		if _, err := validateSchema(&arSchemaLoader, spec); err != nil {
			valid = false
			msgs = append(msgs, err.Error())
			log.Warnf("failed to validate ApisixRoute: %s", err)
		}

		for _, p := range plugins {
			if v, err := validatePlugin(client, p.Name, p.Config); !v {
				valid = false
				msgs = append(msgs, err.Error())
				log.Warnf("failed to validate plugin %s: %s", p.Name, err)
			}
		}

		return &kwhvalidating.ValidatorResult{Valid: valid, Message: strings.Join(msgs, "\n")}, nil
	},
)

func validatePlugin(client apisix.Schema, pluginName string, pluginConfig interface{}) (valid bool, result error) {
	valid = true

	pluginSchema, err := client.GetPluginSchema(context.TODO(), pluginName)
	if err != nil {
		result = fmt.Errorf("failed to get the schema of plugin %s: %s", pluginName, err)
		log.Error(result)
		valid = false
		return
	}

	pluginSchemaLoader := gojsonschema.NewStringLoader(pluginSchema.Content)
	if _, err := validateSchema(&pluginSchemaLoader, pluginConfig); err != nil {
		valid = false
		result = multierror.Append(result, fmt.Errorf("%s plugin's config is invalid", pluginName))
		result = multierror.Append(result, err)
		log.Warn(result)
	}

	return
}
