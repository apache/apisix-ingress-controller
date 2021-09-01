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

	"github.com/hashicorp/go-multierror"
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

// TODO: make this helper function more generic so that it can be used by other validating webhooks.
func validateSchema(schema string, config interface{}) (bool, error) {
	// TODO: cache the schema loader
	schemaLoader := gojsonschema.NewStringLoader(schema)
	configLoader := gojsonschema.NewGoLoader(config)

	result, err := gojsonschema.Validate(schemaLoader, configLoader)
	if err != nil {
		log.Errorf("failed to load and validate the schema: %s", err)
		return false, err
	}

	if result.Valid() {
		return true, nil
	}

	log.Warn("the given document is not valid. see errors:\n")
	var resultErr error
	for _, desc := range result.Errors() {
		resultErr = multierror.Append(resultErr, fmt.Errorf("%s\n", desc.Description()))
		log.Errorf("- %s\n", desc)
	}

	return false, resultErr
}
