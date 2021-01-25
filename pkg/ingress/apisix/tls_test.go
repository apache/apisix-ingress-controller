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
package apisix

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"

	apisixhttp "github.com/api7/ingress-controller/pkg/apisix"
	"github.com/api7/ingress-controller/pkg/seven/conf"
	"github.com/api7/ingress-controller/pkg/seven/utils"
	apisix "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

func TestConvert(t *testing.T) {
	atlsStr := `
apiVersion: apisix.apache.org/v1
kind: ApisixTls
metadata:
  name: foo
  namespace: helm
spec:
  hosts:
  - api6.com
  secret:
    name: test-atls
    namespace: helm
`
	id := "helm_foo"
	snis := []string{"api6.com"}
	status := 1
	cert := "root"
	key := "123456"
	group := ""
	sslExpect := &apisix.Ssl{
		ID:     id,
		Snis:   snis,
		Cert:   cert,
		Key:    key,
		Status: status,
		Group:  group,
	}
	atlsCRD := &ApisixTLSCRD{}
	err := yaml.Unmarshal([]byte(atlsStr), atlsCRD)
	assert.Nil(t, err, "yaml decode failed")
	sc := &SecretClientMock{}
	ssl, err := atlsCRD.Convert(sc)
	assert.Nil(t, err)
	assert.EqualValues(t, sslExpect.Key, ssl.Key, "key convert error")
	assert.EqualValues(t, sslExpect.ID, ssl.ID, "id convert error")
	assert.EqualValues(t, sslExpect.Cert, ssl.Cert, "cert convert error")
	assert.EqualValues(t, sslExpect.Snis, ssl.Snis, "snis convert error")
	assert.EqualValues(t, sslExpect.Group, ssl.Group, "group convert error")
}

func TestConvert_group_annotation(t *testing.T) {
	atlsStr := `
apiVersion: apisix.apache.org/v1
kind: ApisixTls
metadata:
  annotations:
    k8s.apisix.apache.org/ingress.class: 127.0.0.1:9080
  name: foo
  namespace: helm
spec:
  hosts:
  - api6.com
  secret:
    name: test-atls
    namespace: helm
`
	id := "helm_foo"
	snis := []string{"api6.com"}
	status := int(1)
	cert := "root"
	key := "123456"
	group := "127.0.0.1:9080"
	sslExpect := &apisix.Ssl{
		ID:     id,
		Snis:   snis,
		Cert:   cert,
		Key:    key,
		Status: status,
		Group:  group,
	}
	setDummyApisixClient(t)
	atlsCRD := &ApisixTLSCRD{}
	err := yaml.Unmarshal([]byte(atlsStr), atlsCRD)
	assert.Nil(t, err, "yaml decode failed")
	sc := &SecretClientMock{}
	ssl, err := atlsCRD.Convert(sc)
	assert.Nil(t, err)
	assert.EqualValues(t, sslExpect.Group, ssl.Group, "group convert error")
}

func TestConvert_Error(t *testing.T) {
	atlsStr := `
apiVersion: apisix.apache.org/v1
kind: ApisixTls
metadata:
  name: foo
  namespace: helm
spec:
  secret:
    name: test-atls
    namespace: helm
`
	setDummyApisixClient(t)
	atlsCRD := &ApisixTLSCRD{}
	err := yaml.Unmarshal([]byte(atlsStr), atlsCRD)
	assert.Nil(t, err, "yaml decode failed")
	sc := &SecretClientErrorMock{}
	ssl, err := atlsCRD.Convert(sc)
	assert.Nil(t, ssl)
	assert.NotNil(t, err)
}

type SecretClientMock struct{}

func (sc *SecretClientMock) FindByName(namespace, name string) (*v1.Secret, error) {
	secretStr := `
{
  "apiVersion": "v1",
  "kind": "Secret",
  "metadata": {
    "name": "test-atls",
    "namespace": "helm"
  },
  "data": {
    "cert": "cm9vdA==",
    "key": "MTIzNDU2"
  }
}
`
	secret := &v1.Secret{}
	if err := json.Unmarshal([]byte(secretStr), secret); err != nil {
		fmt.Errorf(err.Error())
	}
	return secret, nil
}

type SecretClientErrorMock struct{}

func (sc *SecretClientErrorMock) FindByName(namespace, name string) (*v1.Secret, error) {
	return nil, utils.ErrNotFound
}

func setDummyApisixClient(t *testing.T) {
	cli, err := apisixhttp.NewClient()
	assert.Nil(t, err)
	err = cli.AddCluster(&apisixhttp.ClusterOptions{
		Name:    "",
		BaseURL: "http://127.0.0.2:9080/apisix/admin",
	})
	assert.Nil(t, err)
	conf.SetAPISIXClient(cli)
}
