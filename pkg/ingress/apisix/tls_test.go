package apisix

import (
	"encoding/json"
	"fmt"
	a6Type "github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	"testing"
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
	host := "api6.com"
	snis := []*string{&host}
	status := int(1)
	cert := "root"
	key := "123456"
	sslExpect := &a6Type.Ssl{
		ID:     &id,
		Snis:   snis,
		Cert:   &cert,
		Key:    &key,
		Status: &status,
	}
	atlsCRD := &ApisixTlsCRD{}
	err := yaml.Unmarshal([]byte(atlsStr), atlsCRD)
	assert.Nil(t, err, "yaml decode failed")
	sc := &SecretClientMock{}
	ssl, err := atlsCRD.Convert(sc)
	assert.EqualValues(t, sslExpect.Key, ssl.Key, "ssl convert error")
	assert.EqualValues(t, sslExpect.ID, ssl.ID, "ssl convert error")
	assert.EqualValues(t, sslExpect.Cert, ssl.Cert, "ssl convert error")
	assert.EqualValues(t, sslExpect.Snis, ssl.Snis, "ssl convert error")
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
	atlsCRD := &ApisixTlsCRD{}
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
	return nil, fmt.Errorf("NOT FOUND")
}
