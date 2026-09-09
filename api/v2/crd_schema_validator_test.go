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

package v2_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	sigsyaml "sigs.k8s.io/yaml"
)

// crdSchemaValidator holds the parsed CRD schema and validates objects against it,
// including both OpenAPI structural validation and CEL x-kubernetes-validations rules.
type crdSchemaValidator struct {
	structural *structuralschema.Structural
	internal   *apiextensions.JSONSchemaProps
}

// Validate marshals obj to JSON then runs the CRD's OpenAPI schema validator
// followed by any CEL x-kubernetes-validations rules.
func (v *crdSchemaValidator) Validate(t *testing.T, obj any) error {
	t.Helper()

	data, err := json.Marshal(obj)
	require.NoError(t, err, "failed to marshal object")

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw), "failed to unmarshal to map")

	schemaValidator, _, err := validation.NewSchemaValidator(v.internal)
	require.NoError(t, err, "failed to build schema validator")

	if errs := validation.ValidateCustomResource(nil, raw, schemaValidator); len(errs) > 0 {
		return errs.ToAggregate()
	}

	celValidator := cel.NewValidator(v.structural, false, celconfig.PerCallLimit)
	celErrs, _ := celValidator.Validate(context.Background(), nil, v.structural, raw, nil, celconfig.RuntimeCELCostBudget)
	if len(celErrs) > 0 {
		return celErrs.ToAggregate()
	}
	return nil
}

// loadCRDSchema reads a CRD YAML file and returns a validator for the "v2" version schema.
func loadCRDSchema(t *testing.T, crdPath string) *crdSchemaValidator {
	t.Helper()

	data, err := os.ReadFile(crdPath)
	require.NoError(t, err, "failed to read CRD file: %s", crdPath)

	jsonData, err := sigsyaml.YAMLToJSON(data)
	require.NoError(t, err, "failed to convert CRD YAML to JSON")

	var crd apiextensionsv1.CustomResourceDefinition
	require.NoError(t, json.Unmarshal(jsonData, &crd), "failed to unmarshal CRD")

	var v1Schema *apiextensionsv1.JSONSchemaProps
	for _, v := range crd.Spec.Versions {
		if v.Name == "v2" {
			v1Schema = v.Schema.OpenAPIV3Schema
			break
		}
	}
	require.NotNil(t, v1Schema, "v2 schema not found in CRD")

	var internal apiextensions.JSONSchemaProps
	require.NoError(t,
		apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(v1Schema, &internal, nil),
		"failed to convert v1 schema to internal",
	)

	structural, err := structuralschema.NewStructural(&internal)
	require.NoError(t, err, "failed to build structural schema")
	return &crdSchemaValidator{structural: structural, internal: &internal}
}
