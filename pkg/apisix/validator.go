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

package apisix

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/xeipuuv/gojsonschema"

	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type APISIXSchema struct {
	Plugins       map[string]SchemaPlugin `json:"plugins"`
	StreamPlugins map[string]SchemaPlugin `json:"stream_plugins"`
}

type SchemaPlugin struct {
	SchemaContent any `json:"schema"`
}

type PluginSchemaDef map[string]gojsonschema.JSONLoader

type apisixSchemaReferenceValidator struct {
	StreamPlugins PluginSchemaDef
	HTTPPlugins   PluginSchemaDef
}

func NewReferenceFile(source string) (APISIXSchemaValidator, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var schemadef APISIXSchema
	err = json.Unmarshal(data, &schemadef)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	validator := &apisixSchemaReferenceValidator{
		HTTPPlugins:   make(PluginSchemaDef),
		StreamPlugins: make(PluginSchemaDef),
	}

	for _, plugin := range []struct {
		name   string
		schema map[string]SchemaPlugin
	}{
		{name: "HTTPPlugins", schema: schemadef.Plugins},
		{name: "StreamPlugins", schema: schemadef.StreamPlugins},
	} {
		for k, v := range plugin.schema {
			switch plugin.name {
			case "HTTPPlugins":
				validator.HTTPPlugins[k] = gojsonschema.NewGoLoader(v.SchemaContent)
			case "StreamPlugins":
				validator.StreamPlugins[k] = gojsonschema.NewGoLoader(v.SchemaContent)
			}
		}
	}

	return validator, nil
}

func (asv *apisixSchemaReferenceValidator) ValidateHTTPPluginSchema(plugins v1.Plugins) (bool, error) {
	var resultErrs error

	for pluginName, pluginConfig := range plugins {
		schema, ok := asv.HTTPPlugins[pluginName]
		if !ok {
			return false, fmt.Errorf("unknown plugin [%s]", pluginName)
		}
		result, err := gojsonschema.Validate(schema, gojsonschema.NewGoLoader(pluginConfig))
		if err != nil {
			return false, err
		}

		if result.Valid() {
			continue
		}

		fmt.Println("failed")

		resultErrs = multierror.Append(resultErrs, fmt.Errorf("plugin [%s] config is invalid", pluginName))
		for _, desc := range result.Errors() {
			resultErrs = multierror.Append(resultErrs, fmt.Errorf("- %s", desc))
		}
		return false, resultErrs
	}

	return true, nil
}

func (asv *apisixSchemaReferenceValidator) ValidateSteamPluginSchema(plugins v1.Plugins) (bool, error) {
	var resultErrs error

	for pluginName, pluginConfig := range plugins {
		schema, ok := asv.StreamPlugins[pluginName]
		if !ok {
			return false, fmt.Errorf("unknown stream plugin [%s]", pluginName)
		}
		result, err := gojsonschema.Validate(schema, gojsonschema.NewGoLoader(pluginConfig))
		if err != nil {
			return false, err
		}

		if result.Valid() {
			continue
		}

		resultErrs = multierror.Append(resultErrs, fmt.Errorf("stream plugin [%s] config is invalid", pluginName))
		for _, desc := range result.Errors() {
			resultErrs = multierror.Append(resultErrs, fmt.Errorf("- %s", desc))
		}
		return false, resultErrs
	}

	return true, nil
}
