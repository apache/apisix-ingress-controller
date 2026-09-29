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

package translator

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func TestTranslateHTTPRouteRequestRedirectPath(t *testing.T) {
	tests := []struct {
		name     string
		match    gatewayv1.HTTPPathMatch
		redirect gatewayv1.HTTPRequestRedirectFilter
		expected *adctypes.RedirectConfig
	}{
		{
			name: "redirect without path keeps request URI",
			match: gatewayv1.HTTPPathMatch{
				Type:  ptr.To(gatewayv1.PathMatchExact),
				Value: ptr.To("/headers"),
			},
			redirect: gatewayv1.HTTPRequestRedirectFilter{
				Scheme:     ptr.To("https"),
				Hostname:   ptr.To(gatewayv1.PreciseHostname("example.org")),
				Port:       ptr.To(gatewayv1.PortNumber(8443)),
				StatusCode: ptr.To(301),
			},
			expected: &adctypes.RedirectConfig{
				URI:     "https://example.org:8443$request_uri",
				RetCode: 301,
			},
		},
		{
			name: "replace full path",
			match: gatewayv1.HTTPPathMatch{
				Type:  ptr.To(gatewayv1.PathMatchExact),
				Value: ptr.To("/my-path"),
			},
			redirect: gatewayv1.HTTPRequestRedirectFilter{
				Path: &gatewayv1.HTTPPathModifier{
					Type:            gatewayv1.FullPathHTTPPathModifier,
					ReplaceFullPath: ptr.To("/my-path/"),
				},
				StatusCode: ptr.To(307),
			},
			expected: &adctypes.RedirectConfig{
				URI:               "$scheme://$host/my-path/",
				RetCode:           307,
				AppendQueryString: true,
			},
		},
		{
			name: "replace path prefix",
			match: gatewayv1.HTTPPathMatch{
				Type:  ptr.To(gatewayv1.PathMatchPathPrefix),
				Value: ptr.To("/original-prefix"),
			},
			redirect: gatewayv1.HTTPRequestRedirectFilter{
				Path: &gatewayv1.HTTPPathModifier{
					Type:               gatewayv1.PrefixMatchHTTPPathModifier,
					ReplacePrefixMatch: ptr.To("/replacement-prefix"),
				},
			},
			expected: &adctypes.RedirectConfig{
				RegexURI:          []string{"^/original-prefix(/.*)?$", "/replacement-prefix$1"},
				RetCode:           302,
				AppendQueryString: true,
			},
		},
		{
			name: "replace path prefix with root",
			match: gatewayv1.HTTPPathMatch{
				Type:  ptr.To(gatewayv1.PathMatchPathPrefix),
				Value: ptr.To("/original-prefix"),
			},
			redirect: gatewayv1.HTTPRequestRedirectFilter{
				Path: &gatewayv1.HTTPPathModifier{
					Type:               gatewayv1.PrefixMatchHTTPPathModifier,
					ReplacePrefixMatch: ptr.To("/"),
				},
			},
			expected: &adctypes.RedirectConfig{
				RegexURI:          []string{"^/original-prefix(?:/(.*))?$", "/$1"},
				RetCode:           302,
				AppendQueryString: true,
			},
		},
		{
			name: "replace path prefix and hostname",
			match: gatewayv1.HTTPPathMatch{
				Type:  ptr.To(gatewayv1.PathMatchPathPrefix),
				Value: ptr.To("/path-and-host"),
			},
			redirect: gatewayv1.HTTPRequestRedirectFilter{
				Hostname: ptr.To(gatewayv1.PreciseHostname("example.org")),
				Path: &gatewayv1.HTTPPathModifier{
					Type:               gatewayv1.PrefixMatchHTTPPathModifier,
					ReplacePrefixMatch: ptr.To("/replacement-prefix"),
				},
			},
			expected: &adctypes.RedirectConfig{
				RegexURI:          []string{"^/path-and-host(/.*)?$", "//example.org/replacement-prefix$1"},
				RetCode:           302,
				AppendQueryString: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Name: "redirect", Namespace: "default"},
				Spec: gatewayv1.HTTPRouteSpec{
					Rules: []gatewayv1.HTTPRouteRule{{
						Matches: []gatewayv1.HTTPRouteMatch{{Path: &tt.match}},
						Filters: []gatewayv1.HTTPRouteFilter{{
							Type:            gatewayv1.HTTPRouteFilterRequestRedirect,
							RequestRedirect: &tt.redirect,
						}},
					}},
				},
			}

			result, err := NewTranslator(logr.Discard(), "").TranslateHTTPRoute(
				provider.NewDefaultTranslateContext(context.Background()),
				route,
			)
			require.NoError(t, err)
			require.Len(t, result.Services, 1)

			plugin, ok := result.Services[0].Plugins[adctypes.PluginRedirect].(*adctypes.RedirectConfig)
			require.True(t, ok)
			assert.Equal(t, tt.expected, plugin)
		})
	}
}

func TestRedirectConfigPathFieldsJSON(t *testing.T) {
	data, err := json.Marshal(&adctypes.RedirectConfig{
		RegexURI:          []string{"^/old(/.*)?$", "/new$1"},
		RetCode:           302,
		AppendQueryString: true,
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"regex_uri": ["^/old(/.*)?$", "/new$1"],
		"ret_code": 302,
		"append_query_string": true
	}`, string(data))
}

func TestBuildPrefixMatchRegex(t *testing.T) {
	tests := []struct {
		name          string
		matchPrefix   string
		replacePrefix string
		requestPath   string
		expectedPath  string
	}{
		{
			name:          "replaces prefix and keeps remainder",
			matchPrefix:   "/old",
			replacePrefix: "/new",
			requestPath:   "/old/child",
			expectedPath:  "/new/child",
		},
		{
			name:          "ignores trailing slashes in configured prefixes",
			matchPrefix:   "/old/",
			replacePrefix: "/new/",
			requestPath:   "/old/child",
			expectedPath:  "/new/child",
		},
		{
			name:          "replaces an exact prefix without adding a slash",
			matchPrefix:   "/old",
			replacePrefix: "/new/",
			requestPath:   "/old",
			expectedPath:  "/new",
		},
		{
			name:          "replaces prefix with root",
			matchPrefix:   "/old",
			replacePrefix: "/",
			requestPath:   "/old/child",
			expectedPath:  "/child",
		},
		{
			name:          "empty replacement keeps a valid root path",
			matchPrefix:   "/old",
			replacePrefix: "",
			requestPath:   "/old",
			expectedPath:  "/",
		},
		{
			name:          "replaces the root prefix",
			matchPrefix:   "/",
			replacePrefix: "/new",
			requestPath:   "/child",
			expectedPath:  "/new/child",
		},
		{
			name:          "quotes regular expression characters in match prefix",
			matchPrefix:   "/v1.0/(users)+",
			replacePrefix: "/new",
			requestPath:   "/v1.0/(users)+/child",
			expectedPath:  "/new/child",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regexURI := buildPrefixMatchRegex(tt.matchPrefix, tt.replacePrefix)
			require.Len(t, regexURI, 2)
			re := regexp.MustCompile(regexURI[0])
			assert.Equal(t, tt.expectedPath, re.ReplaceAllString(tt.requestPath, regexURI[1]))
		})
	}
}
