// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package apisix

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/nettest"

	"github.com/apache/apisix-ingress-controller/pkg/metrics"
)

type fakeAPISIXSchemaSrv struct {
	schema map[string]string
}

var testData = map[string]string{
	// plugins' schema
	"plugins/key-auth":           `{"$comment":"this is a mark for our injected plugin schema","type":"object","additionalProperties":false,"properties":{"disable":{"type":"boolean"},"header":{"default":"apikey","type":"string"}}}`,
	"plugins/batch-requests":     `{"$comment":"this is a mark for our injected plugin schema","type":"object","additionalProperties":false,"properties":{"disable":{"type":"boolean"}}}`,
	"plugins/ext-plugin-pre-req": `{"properties":{"disable":{"type":"boolean"},"extra_info":{"items":{"type":"string","minLength":1,"maxLength":64},"minItems":1,"type":"array"},"conf":{"items":{"properties":{"value":{"type":"string"},"name":{"type":"string","minLength":1,"maxLength":128}},"type":"object"},"minItems":1,"type":"array"}},"$comment":"this is a mark for our injected plugin schema","type":"object"}`,

	"route":    `{"anyOf":[{"required":["plugins","uri"]},{"required":["upstream","uri"]},{"required":["upstream_id","uri"]},{"required":["service_id","uri"]},{"required":["plugins","uris"]},{"required":["upstream","uris"]},{"required":["upstream_id","uris"]},{"required":["service_id","uris"]},{"required":["script","uri"]},{"required":["script","uris"]}],"additionalProperties":false,"not":{"anyOf":[{"required":["script","plugins"]},{"required":["script","plugin_config_id"]}]},"properties":{"priority":{"default":0,"type":"integer"},"uris":{"minItems":1,"type":"array","items":{"type":"string","description":"HTTP uri"},"uniqueItems":true},"methods":{"type":"array","items":{"type":"string","enum":["GET","POST","PUT","DELETE","PATCH","HEAD","OPTIONS","CONNECT","TRACE"],"description":"HTTP method"},"uniqueItems":true},"name":{"type":"string","minLength":1,"maxLength":100},"remote_addrs":{"minItems":1,"type":"array","items":{"type":"string","anyOf":[{"type":"string","format":"ipv4","title":"IPv4"},{"type":"string","pattern":"^([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\/([12]?[0-9]|3[0-2])$","title":"IPv4\/CIDR"},{"type":"string","format":"ipv6","title":"IPv6"},{"type":"string","pattern":"^([a-fA-F0-9]{0,4}:){1,8}(:[a-fA-F0-9]{0,4}){0,8}([a-fA-F0-9]{0,4})?\/[0-9]{1,3}$","title":"IPv6\/CIDR"}],"description":"client IP"},"uniqueItems":true},"filter_func":{"type":"string","minLength":10,"pattern":"^function"},"enable_websocket":{"type":"boolean","description":"enable websocket for request"},"script_id":{"anyOf":[{"pattern":"^[a-zA-Z0-9-_.]+$","type":"string","minLength":1,"maxLength":64},{"minimum":1,"type":"integer"}]},"service_protocol":{"enum":["grpc","http"]},"service_id":{"anyOf":[{"pattern":"^[a-zA-Z0-9-_.]+$","type":"string","minLength":1,"maxLength":64},{"minimum":1,"type":"integer"}]},"hosts":{"minItems":1,"type":"array","items":{"pattern":"^\\*?[0-9a-zA-Z-._]+$","type":"string"},"uniqueItems":true},"vars":{"type":"array"},"upstream":{"oneOf":[{"required":["type","nodes"]},{"required":["type","service_name","discovery_type"]}],"properties":{"id":{"anyOf":[{"pattern":"^[a-zA-Z0-9-_.]+$","type":"string","minLength":1,"maxLength":64},{"minimum":1,"type":"integer"}]},"name":{"type":"string","minLength":1,"maxLength":100},"create_time":{"type":"integer"},"retries":{"minimum":0,"type":"integer"},"scheme":{"enum":["grpc","grpcs","http","https"],"default":"http"},"key":{"type":"string","description":"the key of chash for dynamic load balancing"},"hash_on":{"default":"vars","enum":["vars","header","cookie","consumer","vars_combinations"],"type":"string"},"tls":{"properties":{"client_key":{"type":"string","minLength":128,"maxLength":65536},"client_cert":{"type":"string","minLength":128,"maxLength":65536}},"type":"object","required":["client_cert","client_key"]},"labels":{"maxProperties":16,"type":"object","patternProperties":{".*":{"pattern":"^\\S+$","description":"value of label","type":"string","minLength":1,"maxLength":64}},"description":"key\/value pairs to specify attributes"},"discovery_type":{"type":"string","description":"discovery type"},"update_time":{"type":"integer"},"service_name":{"type":"string","minLength":1,"maxLength":256},"pass_host":{"default":"pass","type":"string","enum":["pass","node","rewrite"],"description":"mod of host passing"},"upstream_host":{"pattern":"^\\*?[0-9a-zA-Z-._]+$","type":"string"},"desc":{"maxLength":256,"type":"string"},"checks":{"anyOf":[{"required":["active"]},{"required":["active","passive"]}],"properties":{"active":{"properties":{"healthy":{"properties":{"http_statuses":{"items":{"minimum":200,"maximum":599,"type":"integer"},"default":[200,302],"type":"array","minItems":1,"uniqueItems":true},"successes":{"default":2,"minimum":1,"maximum":254,"type":"integer"},"interval":{"minimum":1,"default":1,"type":"integer"}},"type":"object"},"concurrency":{"default":10,"type":"integer"},"http_path":{"default":"\/","type":"string"},"https_verify_certificate":{"default":true,"type":"boolean"},"req_headers":{"minItems":1,"items":{"uniqueItems":true,"type":"string"},"type":"array"},"unhealthy":{"properties":{"http_statuses":{"items":{"minimum":200,"maximum":599,"type":"integer"},"default":[429,404,500,501,502,503,504,505],"type":"array","minItems":1,"uniqueItems":true},"http_failures":{"default":5,"minimum":1,"maximum":254,"type":"integer"},"tcp_failures":{"default":2,"minimum":1,"maximum":254,"type":"integer"},"timeouts":{"default":3,"minimum":1,"maximum":254,"type":"integer"},"interval":{"minimum":1,"default":1,"type":"integer"}},"type":"object"},"timeout":{"default":1,"type":"number"},"type":{"enum":["http","https","tcp"],"default":"http","type":"string"},"host":{"pattern":"^\\*?[0-9a-zA-Z-._]+$","type":"string"},"port":{"minimum":1,"maximum":65535,"type":"integer"}},"type":"object"},"passive":{"properties":{"type":{"enum":["http","https","tcp"],"default":"http","type":"string"},"healthy":{"properties":{"successes":{"default":5,"minimum":1,"maximum":254,"type":"integer"},"http_statuses":{"items":{"minimum":200,"maximum":599,"type":"integer"},"default":[200,201,202,203,204,205,206,207,208,226,300,301,302,303,304,305,306,307,308],"type":"array","minItems":1,"uniqueItems":true}},"type":"object"},"unhealthy":{"properties":{"timeouts":{"default":7,"minimum":1,"maximum":254,"type":"integer"},"http_statuses":{"items":{"minimum":200,"maximum":599,"type":"integer"},"default":[429,500,503],"type":"array","minItems":1,"uniqueItems":true},"http_failures":{"default":5,"minimum":1,"maximum":254,"type":"integer"},"tcp_failures":{"default":2,"minimum":1,"maximum":254,"type":"integer"}},"type":"object"}},"type":"object"}},"additionalProperties":false,"type":"object"},"type":{"type":"string","enum":["chash","roundrobin","ewma","least_conn"],"description":"algorithms of load balancing"},"nodes":{"anyOf":[{"patternProperties":{".*":{"type":"integer","minimum":0,"description":"weight of node"}},"type":"object"},{"items":{"properties":{"weight":{"type":"integer","minimum":0,"description":"weight of node"},"priority":{"type":"integer","default":0,"description":"priority of node"},"metadata":{"type":"object","description":"metadata of node"},"host":{"pattern":"^\\*?[0-9a-zA-Z-._]+$","type":"string"},"port":{"type":"integer","minimum":1,"description":"port of node"}},"type":"object","required":["host","port","weight"]},"type":"array"}]},"timeout":{"properties":{"send":{"exclusiveMinimum":0,"type":"number"},"read":{"exclusiveMinimum":0,"type":"number"},"connect":{"exclusiveMinimum":0,"type":"number"}},"type":"object","required":["connect","send","read"]}},"additionalProperties":false,"type":"object"},"id":{"anyOf":[{"pattern":"^[a-zA-Z0-9-_.]+$","type":"string","minLength":1,"maxLength":64},{"minimum":1,"type":"integer"}]},"upstream_id":{"anyOf":[{"pattern":"^[a-zA-Z0-9-_.]+$","type":"string","minLength":1,"maxLength":64},{"minimum":1,"type":"integer"}]},"labels":{"maxProperties":16,"type":"object","patternProperties":{".*":{"pattern":"^\\S+$","description":"value of label","type":"string","minLength":1,"maxLength":64}},"description":"key\/value pairs to specify attributes"},"uri":{"type":"string","minLength":1,"maxLength":4096},"update_time":{"type":"integer"},"plugin_config_id":{"anyOf":[{"pattern":"^[a-zA-Z0-9-_.]+$","type":"string","minLength":1,"maxLength":64},{"minimum":1,"type":"integer"}]},"desc":{"maxLength":256,"type":"string"},"status":{"default":1,"type":"integer","enum":[1,0],"description":"route status, 1 to enable, 0 to disable"},"remote_addr":{"type":"string","anyOf":[{"type":"string","format":"ipv4","title":"IPv4"},{"type":"string","pattern":"^([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\/([12]?[0-9]|3[0-2])$","title":"IPv4\/CIDR"},{"type":"string","format":"ipv6","title":"IPv6"},{"type":"string","pattern":"^([a-fA-F0-9]{0,4}:){1,8}(:[a-fA-F0-9]{0,4}){0,8}([a-fA-F0-9]{0,4})?\/[0-9]{1,3}$","title":"IPv6\/CIDR"}],"description":"client IP"},"plugins":{"type":"object"},"host":{"pattern":"^\\*?[0-9a-zA-Z-._]+$","type":"string"},"script":{"type":"string","minLength":10,"maxLength":102400},"create_time":{"type":"integer"}},"allOf":[{"oneOf":[{"required":["uri"]},{"required":["uris"]}]},{"oneOf":[{"not":{"anyOf":[{"required":["host"]},{"required":["hosts"]}]}},{"required":["host"]},{"required":["hosts"]}]},{"oneOf":[{"not":{"anyOf":[{"required":["remote_addr"]},{"required":["remote_addrs"]}]}},{"required":["remote_addr"]},{"required":["remote_addrs"]}]}],"type":"object"}`,
	"upstream": `{"oneOf":[{"required":["type","nodes"]},{"required":["type","service_name","discovery_type"]}],"properties":{"id":{"anyOf":[{"pattern":"^[a-zA-Z0-9-_.]+$","type":"string","minLength":1,"maxLength":64},{"minimum":1,"type":"integer"}]},"name":{"type":"string","minLength":1,"maxLength":100},"create_time":{"type":"integer"},"retries":{"minimum":0,"type":"integer"},"scheme":{"enum":["grpc","grpcs","http","https"],"default":"http"},"key":{"type":"string","description":"the key of chash for dynamic load balancing"},"hash_on":{"default":"vars","enum":["vars","header","cookie","consumer","vars_combinations"],"type":"string"},"tls":{"properties":{"client_key":{"type":"string","minLength":128,"maxLength":65536},"client_cert":{"type":"string","minLength":128,"maxLength":65536}},"type":"object","required":["client_cert","client_key"]},"labels":{"maxProperties":16,"type":"object","patternProperties":{".*":{"pattern":"^\\S+$","description":"value of label","type":"string","minLength":1,"maxLength":64}},"description":"key\/value pairs to specify attributes"},"discovery_type":{"type":"string","description":"discovery type"},"update_time":{"type":"integer"},"service_name":{"type":"string","minLength":1,"maxLength":256},"pass_host":{"default":"pass","type":"string","enum":["pass","node","rewrite"],"description":"mod of host passing"},"upstream_host":{"pattern":"^\\*?[0-9a-zA-Z-._]+$","type":"string"},"desc":{"maxLength":256,"type":"string"},"checks":{"anyOf":[{"required":["active"]},{"required":["active","passive"]}],"properties":{"active":{"properties":{"healthy":{"properties":{"http_statuses":{"items":{"minimum":200,"maximum":599,"type":"integer"},"default":[200,302],"type":"array","minItems":1,"uniqueItems":true},"successes":{"default":2,"minimum":1,"maximum":254,"type":"integer"},"interval":{"minimum":1,"default":1,"type":"integer"}},"type":"object"},"concurrency":{"default":10,"type":"integer"},"http_path":{"default":"\/","type":"string"},"https_verify_certificate":{"default":true,"type":"boolean"},"req_headers":{"minItems":1,"items":{"uniqueItems":true,"type":"string"},"type":"array"},"unhealthy":{"properties":{"http_statuses":{"items":{"minimum":200,"maximum":599,"type":"integer"},"default":[429,404,500,501,502,503,504,505],"type":"array","minItems":1,"uniqueItems":true},"http_failures":{"default":5,"minimum":1,"maximum":254,"type":"integer"},"tcp_failures":{"default":2,"minimum":1,"maximum":254,"type":"integer"},"timeouts":{"default":3,"minimum":1,"maximum":254,"type":"integer"},"interval":{"minimum":1,"default":1,"type":"integer"}},"type":"object"},"timeout":{"default":1,"type":"number"},"type":{"enum":["http","https","tcp"],"default":"http","type":"string"},"host":{"pattern":"^\\*?[0-9a-zA-Z-._]+$","type":"string"},"port":{"minimum":1,"maximum":65535,"type":"integer"}},"type":"object"},"passive":{"properties":{"type":{"enum":["http","https","tcp"],"default":"http","type":"string"},"healthy":{"properties":{"successes":{"default":5,"minimum":1,"maximum":254,"type":"integer"},"http_statuses":{"items":{"minimum":200,"maximum":599,"type":"integer"},"default":[200,201,202,203,204,205,206,207,208,226,300,301,302,303,304,305,306,307,308],"type":"array","minItems":1,"uniqueItems":true}},"type":"object"},"unhealthy":{"properties":{"timeouts":{"default":7,"minimum":1,"maximum":254,"type":"integer"},"http_statuses":{"items":{"minimum":200,"maximum":599,"type":"integer"},"default":[429,500,503],"type":"array","minItems":1,"uniqueItems":true},"http_failures":{"default":5,"minimum":1,"maximum":254,"type":"integer"},"tcp_failures":{"default":2,"minimum":1,"maximum":254,"type":"integer"}},"type":"object"}},"type":"object"}},"additionalProperties":false,"type":"object"},"type":{"type":"string","enum":["chash","roundrobin","ewma","least_conn"],"description":"algorithms of load balancing"},"nodes":{"anyOf":[{"patternProperties":{".*":{"type":"integer","minimum":0,"description":"weight of node"}},"type":"object"},{"items":{"properties":{"weight":{"type":"integer","minimum":0,"description":"weight of node"},"priority":{"type":"integer","default":0,"description":"priority of node"},"metadata":{"type":"object","description":"metadata of node"},"host":{"pattern":"^\\*?[0-9a-zA-Z-._]+$","type":"string"},"port":{"type":"integer","minimum":1,"description":"port of node"}},"type":"object","required":["host","port","weight"]},"type":"array"}]},"timeout":{"properties":{"send":{"exclusiveMinimum":0,"type":"number"},"read":{"exclusiveMinimum":0,"type":"number"},"connect":{"exclusiveMinimum":0,"type":"number"}},"type":"object","required":["connect","send","read"]}},"additionalProperties":false,"type":"object"}`,
	"consumer": `{"type":"object","properties":{"desc":{"maxLength":256,"type":"string"},"username":{"pattern":"^[a-zA-Z0-9_]+$","type":"string","minLength":1,"maxLength":32},"plugins":{"type":"object"},"labels":{"maxProperties":16,"type":"object","patternProperties":{".*":{"pattern":"^\\S+$","description":"value of label","type":"string","minLength":1,"maxLength":64}},"description":"key\/value pairs to specify attributes"},"update_time":{"type":"integer"},"create_time":{"type":"integer"}},"additionalProperties":false,"required":["username"]}`,
	"ssl":      `{"additionalProperties":false,"type":"object","oneOf":[{"required":["sni","key","cert"]},{"required":["snis","key","cert"]}],"properties":{"update_time":{"type":"integer"},"client":{"type":"object","required":["ca"],"properties":{"ca":{"minLength":128,"type":"string","maxLength":65536},"depth":{"default":1,"type":"integer","minimum":0}}},"status":{"default":1,"type":"integer","enum":[1,0],"description":"ssl status, 1 to enable, 0 to disable"},"sni":{"pattern":"^\\*?[0-9a-zA-Z-.]+$","type":"string"},"id":{"anyOf":[{"minLength":1,"type":"string","maxLength":64,"pattern":"^[a-zA-Z0-9-_.]+$"},{"minimum":1,"type":"integer"}]},"certs":{"items":{"minLength":128,"type":"string","maxLength":65536},"type":"array"},"create_time":{"type":"integer"},"labels":{"maxProperties":16,"type":"object","patternProperties":{".*":{"minLength":1,"maxLength":64,"description":"value of label","type":"string","pattern":"^\\S+$"}},"description":"key\/value pairs to specify attributes"},"exptime":{"minimum":1588262400,"type":"integer"},"snis":{"items":{"pattern":"^\\*?[0-9a-zA-Z-.]+$","type":"string"},"type":"array","minItems":1},"cert":{"minLength":128,"type":"string","maxLength":65536},"key":{"minLength":128,"type":"string","maxLength":65536},"keys":{"items":{"minLength":128,"type":"string","maxLength":65536},"type":"array"},"validity_end":{"type":"integer"},"validity_start":{"type":"integer"}}}`,
}

const errMsg = `{"error_msg":"not found schema"}`

func (srv *fakeAPISIXSchemaSrv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if !strings.HasPrefix(r.URL.Path, "/apisix/admin/schema") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if r.Method == http.MethodGet {
		name := strings.Trim(strings.TrimPrefix(r.URL.Path, "/apisix/admin/schema/"), "/")
		if len(name) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if resp, ok := srv.schema[name]; ok {
			_, _ = w.Write([]byte(resp))
		} else {
			_, _ = w.Write([]byte(errMsg))
		}
		w.WriteHeader(http.StatusOK)
		return
	}

}

func runFakeSchemaSrv(t *testing.T) *http.Server {
	srv := &fakeAPISIXSchemaSrv{
		schema: testData,
	}

	ln, _ := nettest.NewLocalListener("tcp")

	httpSrv := &http.Server{
		Addr:    ln.Addr().String(),
		Handler: srv,
	}

	go func() {
		if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			t.Errorf("failed to run http server: %s", err)
		}
	}()

	return httpSrv
}

func TestSchemaClient(t *testing.T) {
	srv := runFakeSchemaSrv(t)
	defer func() {
		assert.Nil(t, srv.Shutdown(context.Background()))
	}()

	u := url.URL{
		Scheme: "http",
		Host:   srv.Addr,
		Path:   "/apisix/admin",
	}

	closedCh := make(chan struct{})
	close(closedCh)
	cli := newSchemaClient(&cluster{
		baseURL:          u.String(),
		cli:              http.DefaultClient,
		cache:            &dummyCache{},
		cacheSynced:      closedCh,
		metricsCollector: metrics.NewPrometheusCollector(),
	})

	ctx := context.TODO()

	// Test `GetPluginSchema`.
	for name := range testData {
		if strings.HasPrefix(name, "plugins/") {
			list := strings.Split(name, "/")
			assert.Greater(t, len(list), 0)

			schema, err := cli.GetPluginSchema(ctx, list[len(list)-1])
			assert.Nil(t, err)
			assert.Equal(t, schema.Name, name)
			assert.Equal(t, schema.Content, testData[name])
		}
	}

	// Test getting a non-existent plugin's schema.
	schema, err := cli.GetPluginSchema(ctx, "not-a-plugin")
	assert.Nil(t, err)
	assert.Equal(t, schema.Content, errMsg)

	// Test `GetRouteSchema`
	routeSchema, err := cli.GetRouteSchema(ctx)
	assert.Nil(t, err)
	assert.Equal(t, routeSchema.Content, testData["route"])

	// Test `GetRouteSchema`
	upstreamSchema, err := cli.GetUpstreamSchema(ctx)
	assert.Nil(t, err)
	assert.Equal(t, upstreamSchema.Content, testData["upstream"])

	// Test `GetConsumerSchema`
	consumerSchema, err := cli.GetConsumerSchema(ctx)
	assert.Nil(t, err)
	assert.Equal(t, consumerSchema.Content, testData["consumer"])

	// Test `GetSslSchema`
	sslSchema, err := cli.GetSslSchema(ctx)
	assert.Nil(t, err)
	assert.Equal(t, sslSchema.Content, testData["ssl"])
}
