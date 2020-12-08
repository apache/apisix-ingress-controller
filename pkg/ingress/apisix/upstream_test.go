package apisix

import (
	"fmt"
	"testing"

	ingress "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	"github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestApisixUpstreamCRD_Convert(t *testing.T) {
	assert := assert.New(t)

	// get yaml from string
	var crd ingress.ApisixUpstream
	bytes := []byte(upstreamYaml)
	if err := yaml.Unmarshal(bytes, &crd); err != nil {
		assert.Error(err)
	} else {
		au3 := &ApisixUpstreamBuilder{CRD: &crd, Ep: &EndpointRequestTest{}} // mock endpoints
		// convert
		if upstreams, err := au3.Convert(); err != nil {
			assert.Error(err)
		} else {
			// equals or deepCompare
			upstreamExpect := buildExpectUpstream()
			//upstreamsExpect := []*v1.Upstream{upstreamExpect}
			b := equals(upstreams[0], upstreamExpect)
			//b := reflect.DeepEqual(upstreams, []*v1.Upstream{upstreamExpect})
			if !b {
				assert.True(b, "convert upstream not expected")
				assert.Error(fmt.Errorf("convert upstream not expect"))
			}
			t.Log("[upstream convert] ok")
		}
	}
}

func equals(s, d *v1.Upstream) bool {
	if *s.Name != *d.Name || *s.FullName != *d.FullName || *s.Group != *d.Group {
		return false
	}

	if *s.FromKind != *d.FromKind || *s.Type != *d.Type || *s.Key != *d.Key || *s.HashOn != *d.HashOn {
		return false
	}

	return true
}

// mock BuildEps
type EndpointRequestTest struct{}

func (epr *EndpointRequestTest) BuildEps(ns, name string, port int) []*v1.Node {
	nodes := make([]*v1.Node, 0)
	return nodes
}

func buildExpectUpstream() *v1.Upstream {
	fullName := "default_httpserver_8080"
	LBType := "chash"
	HashOn := "header"
	Key := "hello_key"
	fromKind := "ApisixUpstream"
	upstreamExpect := &v1.Upstream{
		FullName: &fullName,
		Name:     &fullName,
		Type:     &LBType,
		HashOn:   &HashOn,
		Key:      &Key,
		FromKind: &fromKind,
	}
	return upstreamExpect
}

var upstreamYaml = `
kind: ApisixUpstream
apiVersion: apisix.apache.org/v1
metadata:
  name: httpserver
spec:
  ports:
  - loadbalancer:
      hashOn: header
      key: hello_key
      type: chash
    port: 8080
`
