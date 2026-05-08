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
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	sigsyaml "sigs.k8s.io/yaml"

	apisixv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

// consumerSchemaValidator holds the parsed CRD schema for ApisixConsumer
// and provides a Validate method for use in tests.
type consumerSchemaValidator struct {
	structural *structuralschema.Structural
	internal   *apiextensions.JSONSchemaProps
}

func (v *consumerSchemaValidator) Validate(t *testing.T, ac *apisixv2.ApisixConsumer) error {
	t.Helper()

	data, err := json.Marshal(ac)
	require.NoError(t, err, "failed to marshal ApisixConsumer")

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

// loadApisixConsumerSchema reads the ApisixConsumer CRD YAML and returns a
// validator backed by the real generated schema.
func loadApisixConsumerSchema(t *testing.T) *consumerSchemaValidator {
	t.Helper()

	_, thisFile, _, _ := runtime.Caller(0)
	crdPath := filepath.Join(filepath.Dir(thisFile), "..", "..",
		"config", "crd", "bases", "apisix.apache.org_apisixconsumers.yaml")

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
	return &consumerSchemaValidator{structural: structural, internal: &internal}
}

func TestApisixConsumer_JwtAuth_SymmetricHS256(t *testing.T) {
	v := loadApisixConsumerSchema(t)
	ac := &apisixv2.ApisixConsumer{
		Spec: apisixv2.ApisixConsumerSpec{
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					Value: &apisixv2.ApisixConsumerJwtAuthValue{
						Key:       "my-key",
						Secret:    "my-secret",
						Algorithm: "HS256",
					},
				},
			},
		},
	}
	assert.NoError(t, v.Validate(t, ac))
}

// TestApisixConsumer_JwtAuth_AsymmetricWithWhitespaceOnlyPublicKey verifies
// that a whitespace-only public_key is treated as absent and rejected for
// asymmetric algorithms.
func TestApisixConsumer_JwtAuth_AsymmetricWithWhitespaceOnlyPublicKey(t *testing.T) {
	v := loadApisixConsumerSchema(t)
	ac := &apisixv2.ApisixConsumer{
		Spec: apisixv2.ApisixConsumerSpec{
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					Value: &apisixv2.ApisixConsumerJwtAuthValue{
						Key:       "my-key",
						Algorithm: "RS256",
						PublicKey: "   ",
					},
				},
			},
		},
	}
	err := v.Validate(t, ac)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "algorithms other than HS256/HS384/HS512")
}

func TestApisixConsumer_JwtAuth_SymmetricHS512(t *testing.T) {
	v := loadApisixConsumerSchema(t)
	ac := &apisixv2.ApisixConsumer{
		Spec: apisixv2.ApisixConsumerSpec{
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					Value: &apisixv2.ApisixConsumerJwtAuthValue{
						Key:       "my-key",
						Secret:    "my-secret",
						Algorithm: "HS512",
					},
				},
			},
		},
	}
	assert.NoError(t, v.Validate(t, ac))
}

func TestApisixConsumer_JwtAuth_NoAlgorithmDefaultsToSymmetric(t *testing.T) {
	v := loadApisixConsumerSchema(t)
	ac := &apisixv2.ApisixConsumer{
		Spec: apisixv2.ApisixConsumerSpec{
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					Value: &apisixv2.ApisixConsumerJwtAuthValue{
						Key:    "my-key",
						Secret: "my-secret",
					},
				},
			},
		},
	}
	assert.NoError(t, v.Validate(t, ac))
}

func TestApisixConsumer_JwtAuth_AsymmetricRS256WithPublicKey(t *testing.T) {
	v := loadApisixConsumerSchema(t)
	ac := &apisixv2.ApisixConsumer{
		Spec: apisixv2.ApisixConsumerSpec{
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					Value: &apisixv2.ApisixConsumerJwtAuthValue{
						Key:       "my-key",
						PublicKey: "test-public-key",
						Algorithm: "RS256",
					},
				},
			},
		},
	}
	assert.NoError(t, v.Validate(t, ac))
}

func TestApisixConsumer_JwtAuth_AsymmetricRS256WithPrivateKey(t *testing.T) {
	v := loadApisixConsumerSchema(t)
	ac := &apisixv2.ApisixConsumer{
		Spec: apisixv2.ApisixConsumerSpec{
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					Value: &apisixv2.ApisixConsumerJwtAuthValue{
						Key:        "my-key",
						PrivateKey: "test-private-key",
						Algorithm:  "RS256",
					},
				},
			},
		},
	}
	assert.NoError(t, v.Validate(t, ac))
}

func TestApisixConsumer_JwtAuth_AsymmetricRS256WithBothKeys(t *testing.T) {
	v := loadApisixConsumerSchema(t)
	ac := &apisixv2.ApisixConsumer{
		Spec: apisixv2.ApisixConsumerSpec{
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					Value: &apisixv2.ApisixConsumerJwtAuthValue{
						Key:        "my-key",
						PublicKey:  "test-public-key",
						PrivateKey: "test-private-key",
						Algorithm:  "RS256",
					},
				},
			},
		},
	}
	assert.NoError(t, v.Validate(t, ac))
}

func TestApisixConsumer_JwtAuth_AsymmetricRS256WithoutAnyKey(t *testing.T) {
	v := loadApisixConsumerSchema(t)
	ac := &apisixv2.ApisixConsumer{
		Spec: apisixv2.ApisixConsumerSpec{
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					Value: &apisixv2.ApisixConsumerJwtAuthValue{
						Key:       "my-key",
						Algorithm: "RS256",
					},
				},
			},
		},
	}
	err := v.Validate(t, ac)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "algorithms other than HS256/HS384/HS512")
}

func TestApisixConsumer_JwtAuth_AsymmetricES256WithoutAnyKey(t *testing.T) {
	v := loadApisixConsumerSchema(t)
	ac := &apisixv2.ApisixConsumer{
		Spec: apisixv2.ApisixConsumerSpec{
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					Value: &apisixv2.ApisixConsumerJwtAuthValue{
						Key:       "my-key",
						Algorithm: "ES256",
					},
				},
			},
		},
	}
	err := v.Validate(t, ac)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "algorithms other than HS256/HS384/HS512")
}

func TestApisixConsumer_JwtAuth_AsymmetricEdDSAWithoutAnyKey(t *testing.T) {
	v := loadApisixConsumerSchema(t)
	ac := &apisixv2.ApisixConsumer{
		Spec: apisixv2.ApisixConsumerSpec{
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					Value: &apisixv2.ApisixConsumerJwtAuthValue{
						Key:       "my-key",
						Algorithm: "EdDSA",
					},
				},
			},
		},
	}
	err := v.Validate(t, ac)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "algorithms other than HS256/HS384/HS512")
}

func TestApisixConsumer_JwtAuth_AsymmetricWithEmptyPublicKey(t *testing.T) {
	v := loadApisixConsumerSchema(t)
	ac := &apisixv2.ApisixConsumer{
		Spec: apisixv2.ApisixConsumerSpec{
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					Value: &apisixv2.ApisixConsumerJwtAuthValue{
						Key:       "my-key",
						Algorithm: "RS256",
						// PublicKey is empty string — omitempty means it won't appear
						// in the serialized JSON, same effect as not set
					},
				},
			},
		},
	}
	err := v.Validate(t, ac)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "algorithms other than HS256/HS384/HS512")
}

// TestApisixConsumer_JwtAuth_EmptyAlgorithmTreatedAsSymmetric verifies that an
// explicitly empty algorithm string is treated the same as an unset algorithm
// (defaults to HS256) and does not require public_key or private_key.
func TestApisixConsumer_JwtAuth_EmptyAlgorithmTreatedAsSymmetric(t *testing.T) {
	v := loadApisixConsumerSchema(t)
	ac := &apisixv2.ApisixConsumer{
		Spec: apisixv2.ApisixConsumerSpec{
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					Value: &apisixv2.ApisixConsumerJwtAuthValue{
						Key:    "my-key",
						Secret: "my-secret",
						// Algorithm is explicitly empty string — should be treated as
						// unset and not require asymmetric keys.
					},
				},
			},
		},
	}
	assert.NoError(t, v.Validate(t, ac))
}
