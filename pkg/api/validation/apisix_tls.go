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
	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta2"
	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

// errNotApisixTls will be used when the validating object is not ApisixTls.
var errNotApisixTls = errors.New("object is not ApisixTls")

// ApisixTlsValidator validates ApisixTls's spec.
var ApisixTlsValidator = kwhvalidating.ValidatorFunc(
	func(ctx context.Context, review *kwhmodel.AdmissionReview, object metav1.Object) (result *kwhvalidating.ValidatorResult, err error) {
		log.Debug("arrive ApisixTls validator webhook")

		valid := true
		var spec interface{}

		switch at := object.(type) {
		case *v2beta2.ApisixRoute:
			spec = at.Spec
		case *v2beta3.ApisixRoute:
			spec = at.Spec
		case *v2.ApisixRoute:
			spec = at.Spec
		default:
			return &kwhvalidating.ValidatorResult{Valid: false, Message: errNotApisixTls.Error()}, errNotApisixTls
		}

		client, err := GetSchemaClient(&apisix.ClusterOptions{})
		if err != nil {
			msg := "failed to get the schema client"
			log.Errorf("%s: %s", msg, err)
			return &kwhvalidating.ValidatorResult{Valid: false, Message: msg}, err
		}

		ss, err := client.GetSslSchema(ctx)
		if err != nil {
			msg := "failed to get SSL's schema"
			log.Errorf("%s: %s", msg, err)
			return &kwhvalidating.ValidatorResult{Valid: false, Message: msg}, err
		}
		atSchemaLoader := gojsonschema.NewStringLoader(ss.Content)

		var msgs []string
		if _, err := validateSchema(&atSchemaLoader, spec); err != nil {
			valid = false
			msgs = append(msgs, err.Error())
		}

		return &kwhvalidating.ValidatorResult{Valid: valid, Message: strings.Join(msgs, "\n")}, nil
	},
)
