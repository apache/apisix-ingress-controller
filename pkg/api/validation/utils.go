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

	"github.com/xeipuuv/gojsonschema"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

var (
	once         sync.Once
	onceErr      error
	schemaClient apisix.Schema
)

// GetSchemaClient returns a Schema client in the singleton way.
// It can query the schema of objects from APISIX.
func GetSchemaClient() (apisix.Schema, error) {
	once.Do(func() {
		client, err := apisix.NewClient()
		if err != nil {
			onceErr = err
			return
		}

		cfg := config.NewDefaultConfig()
		if err := cfg.Validate(); err != nil {
			onceErr = err
			return
		}

		clusterOpts := &apisix.ClusterOptions{
			Name:     cfg.APISIX.DefaultClusterName,
			AdminKey: cfg.APISIX.DefaultClusterAdminKey,
			BaseURL:  cfg.APISIX.DefaultClusterBaseURL,
		}
		if err := client.AddCluster(context.TODO(), clusterOpts); err != nil {
			onceErr = err
			return
		}

		schemaClient = client.Cluster(cfg.APISIX.DefaultClusterName).Schema()
	})
	return schemaClient, onceErr
}

// TODO: make this helper function more generic so that it can be used by other validating webhooks.
func validateSchema(schema string, config interface{}) (error, []gojsonschema.ResultError) {
	schemaLoader := gojsonschema.NewStringLoader(schema)
	configLoader := gojsonschema.NewGoLoader(config)

	result, err := gojsonschema.Validate(schemaLoader, configLoader)
	if err != nil {
		log.Errorf("failed to load and validate the schema: %s", err)
		return err, nil
	}

	if result.Valid() {
		log.Info("the plugin's configLoader is valid")
	} else {
		log.Error("the plugin's configLoader is not valid. see errors:")
		for _, desc := range result.Errors() {
			log.Errorf("- %s\n", desc)
		}
		return fmt.Errorf("invalid plugin config"), result.Errors()
	}

	return nil, nil
}
