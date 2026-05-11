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
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/util/intstr"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	sigsyaml "sigs.k8s.io/yaml"

	apisixv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

// routeSchemaValidator holds the parsed CRD schema for ApisixRoute
// and provides a Validate method for use in tests.
type routeSchemaValidator struct {
	structural *structuralschema.Structural
	internal   *apiextensions.JSONSchemaProps
}

func (v *routeSchemaValidator) Validate(t *testing.T, ar *apisixv2.ApisixRoute) error {
	t.Helper()

	data, err := json.Marshal(ar)
	require.NoError(t, err, "failed to marshal ApisixRoute")

	var obj map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &obj), "failed to unmarshal to map")

	schemaValidator, _, err := validation.NewSchemaValidator(v.internal)
	require.NoError(t, err, "failed to build schema validator")

	if errs := validation.ValidateCustomResource(nil, obj, schemaValidator); len(errs) > 0 {
		return errs.ToAggregate()
	}

	celValidator := cel.NewValidator(v.structural, false, celconfig.PerCallLimit)
	celErrs, _ := celValidator.Validate(context.Background(), nil, v.structural, obj, nil, celconfig.RuntimeCELCostBudget)
	if len(celErrs) > 0 {
		return celErrs.ToAggregate()
	}
	return nil
}

// loadApisixRouteSchema reads the ApisixRoute CRD YAML and returns a
// validator backed by the real generated schema.
func loadApisixRouteSchema(t *testing.T) *routeSchemaValidator {
	t.Helper()

	_, thisFile, _, _ := runtime.Caller(0)
	crdPath := filepath.Join(filepath.Dir(thisFile), "..", "..",
		"config", "crd", "bases", "apisix.apache.org_apisixroutes.yaml")

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
	return &routeSchemaValidator{structural: structural, internal: &internal}
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
func intPtr(i int) *int       { return &i }

func newRouteWithBodyExpr(namespace, ingressClass, fieldName, value string) *apisixv2.ApisixRoute {
	return &apisixv2.ApisixRoute{
		Spec: apisixv2.ApisixRouteSpec{
			IngressClassName: ingressClass,
			HTTP: []apisixv2.ApisixRouteHTTP{
				{
					Name:      "rule0",
					Websocket: boolPtr(false),
					Match: apisixv2.ApisixRouteHTTPMatch{
						Paths: []string{"/*"},
						NginxVars: apisixv2.ApisixRouteHTTPMatchExprs{
							{
								Subject: apisixv2.ApisixRouteHTTPMatchExprSubject{
									Scope: apisixv2.ScopeBody,
									Name:  fieldName,
								},
								Op:    apisixv2.OpEqual,
								Set:   []string{},
								Value: strPtr(value),
							},
						},
					},
					Backends: []apisixv2.ApisixRouteHTTPBackend{
						{ServiceName: "my-svc", ServicePort: intstr.FromInt(80), Weight: intPtr(100)},
					},
				},
			},
		},
	}
}

// TestApisixRoute_BodyScope_SimpleField verifies that a Body scope expr with a
// simple field name passes CRD schema validation.
func TestApisixRoute_BodyScope_SimpleField(t *testing.T) {
	v := loadApisixRouteSchema(t)
	ar := newRouteWithBodyExpr("default", "apisix", "action", "login")
	assert.NoError(t, v.Validate(t, ar))
}

// TestApisixRoute_BodyScope_NestedJSONPath verifies that a Body scope expr with
// a dot-notation JSON path passes CRD schema validation.
func TestApisixRoute_BodyScope_NestedJSONPath(t *testing.T) {
	v := loadApisixRouteSchema(t)
	ar := newRouteWithBodyExpr("default", "apisix", "model.version", "gpt-4")
	assert.NoError(t, v.Validate(t, ar))
}

// TestApisixRoute_BodyScope_EmptyName verifies that a Body scope expr with an
// empty name is rejected by the CEL XValidation rule.
func TestApisixRoute_BodyScope_EmptyName(t *testing.T) {
	v := loadApisixRouteSchema(t)
	ar := newRouteWithBodyExpr("default", "apisix", "", "login")
	err := v.Validate(t, ar)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required when scope is not Path")
}

// TestApisixRoute_PathScope_EmptyName verifies that Path scope without a name
// passes CRD schema validation (name is optional for Path).
func TestApisixRoute_PathScope_EmptyName(t *testing.T) {
	v := loadApisixRouteSchema(t)
	ar := &apisixv2.ApisixRoute{
		Spec: apisixv2.ApisixRouteSpec{
			HTTP: []apisixv2.ApisixRouteHTTP{
				{
					Name:      "rule0",
					Websocket: boolPtr(false),
					Match: apisixv2.ApisixRouteHTTPMatch{
						Paths: []string{"/*"},
						NginxVars: apisixv2.ApisixRouteHTTPMatchExprs{
							{
								Subject: apisixv2.ApisixRouteHTTPMatchExprSubject{
									Scope: apisixv2.ScopePath,
								},
								Op:    apisixv2.OpEqual,
								Set:   []string{},
								Value: strPtr("/api"),
							},
						},
					},
					Backends: []apisixv2.ApisixRouteHTTPBackend{
						{ServiceName: "my-svc", ServicePort: intstr.FromInt(80), Weight: intPtr(100)},
					},
				},
			},
		},
	}
	assert.NoError(t, v.Validate(t, ar))
}
