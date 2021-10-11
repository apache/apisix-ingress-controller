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
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-multierror"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	"github.com/xeipuuv/gojsonschema"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

var (
	once         sync.Once
	onceErr      error
	schemaClient apisix.Schema
)

// GetSchemaClient returns a Schema client in the singleton way.
// It can query the schema of objects from APISIX.
func GetSchemaClient(co *apisix.ClusterOptions) (apisix.Schema, error) {
	once.Do(func() {
		client, err := apisix.NewClient()
		if err != nil {
			onceErr = err
			return
		}

		if err := client.AddCluster(context.TODO(), co); err != nil {
			onceErr = err
			return
		}

		schemaClient = client.Cluster(co.Name).Schema()
	})
	return schemaClient, onceErr
}

// NewHandlerFunc returns a HandlerFunc to handle admission reviews using the given validator.
func NewHandlerFunc(ID string, validator kwhvalidating.Validator) gin.HandlerFunc {
	// Create a validating webhook.
	wh, err := kwhvalidating.NewWebhook(kwhvalidating.WebhookConfig{
		ID:        ID,
		Validator: validator,
	})
	if err != nil {
		log.Errorf("failed to create webhook: %s", err)
	}

	h, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: wh})
	if err != nil {
		log.Errorf("failed to create webhook handle: %s", err)
	}

	return gin.WrapH(h)
}

// validateSchema validates the schema of the given Go struct.
func validateSchema(schemaLoader *gojsonschema.JSONLoader, obj interface{}) (bool, error) {
	configLoader := gojsonschema.NewGoLoader(obj)

	result, err := gojsonschema.Validate(*schemaLoader, configLoader)
	if err != nil {
		log.Errorf("failed to load and validate the schema: %s", err)
		return false, err
	}

	if result.Valid() {
		return true, nil
	}

	log.Warn("the given document is not valid. see errors:\n")
	var resultErr error
	resultErr = multierror.Append(resultErr, fmt.Errorf("the given document is not valid"))
	for _, desc := range result.Errors() {
		resultErr = multierror.Append(resultErr, fmt.Errorf("%s", desc.Description()))
		log.Warnf("- %s", desc)
	}

	return false, resultErr
}
