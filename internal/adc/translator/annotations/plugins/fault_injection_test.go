package plugins

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

func TestFaultInjectionHttpAllowMethods(t *testing.T) {
	handler := NewFaultInjectionHandler()
	assert.Equal(t, "fault-injection", handler.PluginName())

	extractor := annotations.NewExtractor(map[string]string{
		annotations.AnnotationsHttpAllowMethods: "GET,POST",
	})

	plugin, err := handler.Handle(extractor)
	assert.NoError(t, err)
	assert.NotNil(t, plugin)

	data, err := json.Marshal(plugin)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"abort":{"http_status":405,"vars":[[["request_method","!","in",["GET","POST"]]]]}}`, string(data))
}

func TestFaultInjectionHttpBlockMethods(t *testing.T) {
	handler := NewFaultInjectionHandler()
	assert.Equal(t, "fault-injection", handler.PluginName())

	extractor := annotations.NewExtractor(map[string]string{
		annotations.AnnotationsHttpBlockMethods: "GET,POST",
	})

	plugin, err := handler.Handle(extractor)
	assert.NoError(t, err)
	assert.NotNil(t, plugin)

	data, err := json.Marshal(plugin)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"abort":{"http_status":405,"vars":[[["request_method","in",["GET","POST"]]]]}}`, string(data))
}
