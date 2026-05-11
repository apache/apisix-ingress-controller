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

package v2

import (
	"os"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func strPtr(s string) *string { return &s }

// celSubjectRule is the CEL expression used in the +kubebuilder:validation:XValidation
// marker on ApisixRouteHTTPMatchExprSubject.
const celSubjectRule = `self.scope == 'Path' || size(self.name) > 0`

func evalCELSubjectRule(t *testing.T, scope, name string) bool {
	t.Helper()
	env, err := cel.NewEnv(
		cel.Variable("self", cel.MapType(cel.StringType, cel.StringType)),
	)
	require.NoError(t, err)
	ast, issues := env.Compile(celSubjectRule)
	require.NoError(t, issues.Err())
	prg, err := env.Program(ast)
	require.NoError(t, err)
	out, _, err := prg.Eval(map[string]any{
		"self": map[string]any{"scope": scope, "name": name},
	})
	require.NoError(t, err)
	return out.Value().(bool)
}

// TestCEL_SubjectRule_Logic verifies the CEL expression used in the XValidation marker.
func TestCEL_SubjectRule_Logic(t *testing.T) {
	// Non-Path scopes with a non-empty name must pass.
	for _, scope := range []string{ScopeHeader, ScopeQuery, ScopeCookie, ScopeVariable, ScopeBody} {
		assert.True(t, evalCELSubjectRule(t, scope, "field"), "scope=%s with name should pass", scope)
	}
	// Path scope with empty name must pass (name is ignored for Path).
	assert.True(t, evalCELSubjectRule(t, ScopePath, ""), "Path with empty name should pass")
	// Non-Path scopes with empty name must fail.
	for _, scope := range []string{ScopeHeader, ScopeQuery, ScopeCookie, ScopeVariable, ScopeBody} {
		assert.False(t, evalCELSubjectRule(t, scope, ""), "scope=%s with empty name should fail", scope)
	}
}

// TestCEL_SubjectRule_InCRD verifies the generated CRD YAML contains the XValidation rule
// with correct (ASCII) quote characters and not typographic quotes.
func TestCEL_SubjectRule_InCRD(t *testing.T) {
	const crdPath = "../../config/crd/bases/apisix.apache.org_apisixroutes.yaml"
	data, err := os.ReadFile(crdPath)
	require.NoError(t, err, "CRD file should exist; run 'make manifests' if missing")

	var crd map[string]any
	require.NoError(t, yaml.Unmarshal(data, &crd))

	// Ensure no typographic/smart quotes crept in anywhere in the file.
	raw := string(data)
	assert.NotContains(t, raw, "\u2018", "CRD must not contain left single quotation mark \u2018")
	assert.NotContains(t, raw, "\u2019", "CRD must not contain right single quotation mark \u2019")
	assert.NotContains(t, raw, "\u201c", "CRD must not contain left double quotation mark \u201c")
	assert.NotContains(t, raw, "\u201d", "CRD must not contain right double quotation mark \u201d")

	// Walk the parsed CRD to extract the x-kubernetes-validations rule string directly,
	// which is more robust than substring matching against the raw YAML (line-wrapping safe).
	rule := extractXValidationRule(t, crd)
	assert.Equal(t, celSubjectRule, rule,
		"XValidation rule in CRD must match the expected CEL expression")
}

// extractXValidationRule walks the parsed CRD map to find the first
// x-kubernetes-validations rule on the subject property of HTTP match exprs.
func extractXValidationRule(t *testing.T, crd map[string]any) string {
	t.Helper()
	// Path: spec.versions[0].schema.openAPIV3Schema
	//   .properties.spec.properties.http.items
	//   .properties.match.properties.exprs.items
	//   .properties.subject.x-kubernetes-validations[0].rule
	get := func(m map[string]any, key string) map[string]any {
		v, ok := m[key]
		require.True(t, ok, "key %q not found", key)
		mv, ok := v.(map[string]any)
		require.True(t, ok, "key %q is not a map", key)
		return mv
	}
	spec := get(crd, "spec")
	versions := spec["versions"].([]any)
	require.NotEmpty(t, versions)
	schema := get(versions[0].(map[string]any), "schema")
	root := get(schema, "openAPIV3Schema")
	props := get(root, "properties")
	specProps := get(get(props, "spec"), "properties")
	httpItems := get(get(specProps, "http"), "items")
	matchProps := get(get(get(httpItems, "properties"), "match"), "properties")
	exprsItems := get(get(matchProps, "exprs"), "items")
	subject := get(get(exprsItems, "properties"), "subject")
	validations, ok := subject["x-kubernetes-validations"].([]any)
	require.True(t, ok, "x-kubernetes-validations not found or not a list")
	require.NotEmpty(t, validations)
	first := validations[0].(map[string]any)
	rule, ok := first["rule"].(string)
	require.True(t, ok, "rule field not found or not a string")
	return rule
}

func TestToVars_ScopeBody_SimpleField(t *testing.T) {
	exprs := ApisixRouteHTTPMatchExprs{
		{
			Subject: ApisixRouteHTTPMatchExprSubject{
				Scope: ScopeBody,
				Name:  "action",
			},
			Op:    OpEqual,
			Value: strPtr("login"),
		},
	}

	vars, err := exprs.ToVars()
	require.NoError(t, err)
	require.Len(t, vars, 1)

	// vars[0] is []StringOrSlice: [subject, op, value]
	// Should map to post_arg.action
	assert.Equal(t, "post_arg.action", vars[0][0].StrVal)
	assert.Equal(t, "==", vars[0][1].StrVal)
	assert.Equal(t, "login", vars[0][2].StrVal)
}

func TestToVars_ScopeBody_NestedJSONPath(t *testing.T) {
	exprs := ApisixRouteHTTPMatchExprs{
		{
			Subject: ApisixRouteHTTPMatchExprSubject{
				Scope: ScopeBody,
				Name:  "model.version",
			},
			Op:    OpEqual,
			Value: strPtr("gpt-4"),
		},
	}

	vars, err := exprs.ToVars()
	require.NoError(t, err)
	require.Len(t, vars, 1)

	// Should map to post_arg.model.version (dot-notation passthrough)
	assert.Equal(t, "post_arg.model.version", vars[0][0].StrVal)
}

func TestToVars_ScopeBody_EmptyName_ReturnsError(t *testing.T) {
	exprs := ApisixRouteHTTPMatchExprs{
		{
			Subject: ApisixRouteHTTPMatchExprSubject{
				Scope: ScopeBody,
				Name:  "",
			},
			Op:    OpEqual,
			Value: strPtr("login"),
		},
	}

	_, err := exprs.ToVars()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty subject.name")
}
