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
	"strings"

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

// errNotApisixConsumer will be used when the validating object is not ApisixConsumer.
var errNotApisixConsumer = errors.New("object is not ApisixConsumer")

// ApisixConsumerValidator validates ApisixConsumer's spec.
var ApisixConsumerValidator = kwhvalidating.ValidatorFunc(
	func(ctx context.Context, review *kwhmodel.AdmissionReview, object metav1.Object) (result *kwhvalidating.ValidatorResult, err error) {
		log.Debug("arrive ApisixConsumer validator webhook")

		valid := true
		var spec interface{}

		switch ac := object.(type) {
		case *v2beta1.ApisixRoute:
			spec = ac.Spec
		case *v2alpha1.ApisixRoute:
			spec = ac.Spec
		case *v1.ApisixRoute:
			spec = ac.Spec
		default:
			return &kwhvalidating.ValidatorResult{Valid: false, Message: errNotApisixConsumer.Error()}, errNotApisixConsumer
		}

		client, err := GetSchemaClient(&apisix.ClusterOptions{})
		if err != nil {
			msg := "failed to get the schema client"
			log.Errorf("%s: %s", msg, err)
			return &kwhvalidating.ValidatorResult{Valid: false, Message: msg}, err
		}

		cs, err := client.GetConsumerSchema(ctx)
		if err != nil {
			msg := "failed to get consumer's schema"
			log.Errorf("%s: %s", msg, err)
			return &kwhvalidating.ValidatorResult{Valid: false, Message: msg}, err
		}
		acSchemaLoader := gojsonschema.NewStringLoader(cs.Content)

		var msgs []string
		if _, err := validateSchema(&acSchemaLoader, spec); err != nil {
			valid = false
			msgs = append(msgs, err.Error())
		}

		return &kwhvalidating.ValidatorResult{Valid: valid, Message: strings.Join(msgs, "\n")}, nil
	},
)
