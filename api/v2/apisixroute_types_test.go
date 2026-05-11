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
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"

	apisixv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

func loadApisixRouteSchema(t *testing.T) *crdSchemaValidator {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	crdPath := filepath.Join(filepath.Dir(thisFile), "..", "..",
		"config", "crd", "bases", "apisix.apache.org_apisixroutes.yaml")
	return loadCRDSchema(t, crdPath)
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
func intPtr(i int) *int       { return &i }

func newRouteWithBodyExpr(ingressClass, fieldName, value string) *apisixv2.ApisixRoute {
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
	assert.NoError(t, v.Validate(t, newRouteWithBodyExpr("apisix", "action", "login")))
}

// TestApisixRoute_BodyScope_NestedJSONPath verifies that a Body scope expr with
// a dot-notation JSON path passes CRD schema validation.
func TestApisixRoute_BodyScope_NestedJSONPath(t *testing.T) {
	v := loadApisixRouteSchema(t)
	assert.NoError(t, v.Validate(t, newRouteWithBodyExpr("apisix", "model.version", "gpt-4")))
}

// TestApisixRoute_BodyScope_EmptyName verifies that a Body scope expr with an
// empty name is rejected by the CEL XValidation rule.
func TestApisixRoute_BodyScope_EmptyName(t *testing.T) {
	v := loadApisixRouteSchema(t)
	err := v.Validate(t, newRouteWithBodyExpr("apisix", "", "login"))
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
