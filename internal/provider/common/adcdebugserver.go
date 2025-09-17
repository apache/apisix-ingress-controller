// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package common

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/cache"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

type ResourceInfo struct {
	ID   string
	Name string
	Type string
	Link string
}

type ADCDebugProvider struct {
	store         *cache.Store
	configManager *ConfigManager[types.NamespacedNameKind, adctypes.Config]
	pathPrefix    string
}

func newTemplate(name, body string) *template.Template {
	return template.Must(template.New(name).
		Funcs(template.FuncMap{"urlencode": url.QueryEscape}).
		Parse(body))
}

func (asrv *ADCDebugProvider) SetupHandler(pathPrefix string, mux *http.ServeMux) {
	asrv.pathPrefix = pathPrefix
	mux.HandleFunc("/config", asrv.handleConfig)
	mux.HandleFunc("/", asrv.handleIndex)
}

func NewADCDebugProvider(store *cache.Store, configManager *ConfigManager[types.NamespacedNameKind, adctypes.Config]) *ADCDebugProvider {
	return &ADCDebugProvider{store: store, configManager: configManager}
}

func (asrv *ADCDebugProvider) handleIndex(w http.ResponseWriter, r *http.Request) {
	configs := asrv.configManager.List()
	configNames := make([]string, 0, len(configs))
	for _, cfg := range configs {
		configNames = append(configNames, cfg.Name)
	}

	tmpl := newTemplate("index", `
		<html>
		<head><title>ADC Debug Server</title></head>
		<body>
			<h1>Configurations</h1>
			<ul>
				{{range .ConfigNames}}
				<li><a href="{{$.Prefix}}/config?name={{. | urlencode}}">{{.}}</a></li>
				{{end}}
			</ul>
		</body>
		</html>
	`)

	_ = tmpl.Execute(w, struct {
		ConfigNames []string
		Prefix      string
	}{ConfigNames: configNames, Prefix: asrv.pathPrefix})
}

func (asrv *ADCDebugProvider) handleConfig(w http.ResponseWriter, r *http.Request) {
	configNameEncoded := r.URL.Query().Get("name")
	if configNameEncoded == "" {
		http.Error(w, "Config name is required", http.StatusBadRequest)
		return
	}

	configName, err := url.QueryUnescape(configNameEncoded)
	if err != nil {
		http.Error(w, "Invalid config name encoding", http.StatusBadRequest)
		return
	}

	resourceIDEncoded := r.URL.Query().Get("id")
	resourceID := ""
	if resourceIDEncoded != "" {
		resourceID, err = url.QueryUnescape(resourceIDEncoded)
		if err != nil {
			http.Error(w, "Invalid resource ID encoding", http.StatusBadRequest)
			return
		}
	}

	resourceType := r.URL.Query().Get("type")

	if resourceType == "" {
		asrv.showResourceTypes(w, configName, url.QueryEscape(configName))
		return
	}

	if resourceID == "" {
		asrv.showResources(w, r, configName, url.QueryEscape(configName), resourceType)
		return
	}

	asrv.showResourceDetail(w, r, configName, resourceType, resourceID)
}

func (asrv *ADCDebugProvider) showResourceTypes(w http.ResponseWriter, configName, configNameEncoded string) {
	resourceTypes := []string{adctypes.TypeService, adctypes.TypeRoute, adctypes.TypeConsumer, adctypes.TypeSSL, adctypes.TypeGlobalRule, adctypes.TypePluginMetadata}

	tmpl := newTemplate("resources", `
        <html>
        <head><title>Resources for {{.ConfigName}}</title></head>
        <body>
            <h1>Resources for {{.ConfigName}}</h1>
            <ul>
                {{range .ResourceTypes}}
                <li><a href="{{$.Prefix}}/config?name={{$.ConfigNameEncoded}}&type={{. | urlencode}}">{{.}}</a></li>
                {{end}}
            </ul>
        </body>
        </html>
    `)

	_ = tmpl.Execute(w, struct {
		ConfigName        string
		ConfigNameEncoded string
		ResourceTypes     []string
		Prefix            string
	}{
		ConfigName:        configName,
		ConfigNameEncoded: configNameEncoded,
		ResourceTypes:     resourceTypes,
		Prefix:            asrv.pathPrefix,
	})
}

func (asrv *ADCDebugProvider) showResources(w http.ResponseWriter, r *http.Request, configName, configNameEncoded, resourceType string) {
	resources, err := asrv.store.GetResources(configName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resourceInfos []ResourceInfo
	switch resourceType {
	case adctypes.TypeService:
		for _, svc := range resources.Services {
			resourceInfos = append(resourceInfos, ResourceInfo{
				ID:   svc.ID,
				Name: svc.Name,
				Type: resourceType,
				Link: fmt.Sprintf("%s/config?name=%s&type=%s&id=%s",
					asrv.pathPrefix, configNameEncoded, url.QueryEscape(resourceType), url.QueryEscape(svc.ID)),
			})
		}
	case adctypes.TypeConsumer:
		for _, consumer := range resources.Consumers {
			resourceInfos = append(resourceInfos, ResourceInfo{
				ID:   consumer.Username,
				Name: consumer.Username,
				Type: resourceType,
				Link: fmt.Sprintf("%s/config?name=%s&type=%s&id=%s",
					asrv.pathPrefix, configNameEncoded, url.QueryEscape(resourceType), url.QueryEscape(consumer.Username)),
			})
		}
	case adctypes.TypeSSL:
		for _, ssl := range resources.SSLs {
			resourceInfos = append(resourceInfos, ResourceInfo{
				ID:   ssl.ID,
				Name: ssl.ID,
				Type: resourceType,
				Link: fmt.Sprintf("%s/config?name=%s&type=%s&id=%s",
					asrv.pathPrefix, configNameEncoded, url.QueryEscape(resourceType), url.QueryEscape(ssl.ID)),
			})
		}
	case adctypes.TypeGlobalRule:
		for key := range resources.GlobalRules {
			resourceInfos = append(resourceInfos, ResourceInfo{
				ID:   key,
				Name: key,
				Type: resourceType,
				Link: fmt.Sprintf("%s/config?name=%s&type=%s&id=%s",
					asrv.pathPrefix, configNameEncoded, url.QueryEscape(resourceType), url.QueryEscape(key)),
			})
		}
	case adctypes.TypePluginMetadata:
		if resources.PluginMetadata != nil {
			resourceInfos = append(resourceInfos, ResourceInfo{
				ID:   "pluginmetadata",
				Name: "Plugin Metadata",
				Type: resourceType,
				Link: fmt.Sprintf("%s/config?name=%s&type=%s&id=%s",
					asrv.pathPrefix, configNameEncoded, url.QueryEscape(resourceType), "pluginmetadata"),
			})
		}
	case adctypes.TypeRoute:
		for _, svc := range resources.Services {
			for _, route := range svc.Routes {
				resourceInfos = append(resourceInfos, ResourceInfo{
					ID:   route.ID,
					Name: route.Name,
					Type: resourceType,
					Link: fmt.Sprintf("%s/config?name=%s&type=%s&id=%s",
						asrv.pathPrefix, configNameEncoded, url.QueryEscape(resourceType), url.QueryEscape(route.ID)),
				})
			}
		}
	default:
		http.NotFound(w, r)
		return
	}

	tmpl := newTemplate("resourceList", `
		<html>
		<head><title>{{.ResourceType}} for {{.ConfigName}}</title></head>
		<body>
			<h1>{{.ResourceType}} for {{.ConfigName}}</h1>
			<ul>
				{{range .Resources}}
				<li><a href="{{.Link}}">{{.Name}} ({{.ID}})</a></li>
				{{end}}
			</ul>
		</body>
		</html>
	`)

	_ = tmpl.Execute(w, struct {
		ConfigName   string
		ResourceType string
		Resources    []ResourceInfo
		Prefix       string
	}{
		ConfigName:   configName,
		ResourceType: resourceType,
		Resources:    resourceInfos,
		Prefix:       asrv.pathPrefix,
	})
}

func (asrv *ADCDebugProvider) showResourceDetail(w http.ResponseWriter, r *http.Request, configName, resourceType, resourceID string) {
	resources, err := asrv.store.GetResources(configName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resource interface{}
	switch resourceType {
	case adctypes.TypeService:
		for _, svc := range resources.Services {
			if svc.ID == resourceID {
				resource = svc
				break
			}
		}
	case adctypes.TypeConsumer:
		for _, consumer := range resources.Consumers {
			if consumer.Username == resourceID {
				resource = consumer
				break
			}
		}
	case adctypes.TypeSSL:
		for _, ssl := range resources.SSLs {
			if ssl.ID == resourceID {
				resource = ssl
				break
			}
		}
	case adctypes.TypeGlobalRule:
		resource = resources.GlobalRules
	case adctypes.TypePluginMetadata:
		resource = resources.PluginMetadata
	case adctypes.TypeRoute:
		for _, svc := range resources.Services {
			for _, route := range svc.Routes {
				if route.ID == resourceID {
					resource = route
					break
				}
			}
		}
	default:
		http.NotFound(w, r)
		return
	}

	if resource == nil {
		http.NotFound(w, r)
		return
	}

	jsonData, err := json.MarshalIndent(resource, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := newTemplate("resourceDetail", `
		<html>
		<head><title>Resource Detail</title></head>
		<body>
			<h1>Resource Details: {{.ResourceType}}/{{.ResourceID}}</h1>
			<pre>{{.Resource}}</pre>
			<a href="{{.Prefix}}/config?name={{.ConfigName | urlencode}}&type={{.ResourceType}}">Back</a>
		</body>
		</html>
	`)

	_ = tmpl.Execute(w, struct {
		ConfigName   string
		Resource     string
		ResourceID   string
		ResourceType string
		Prefix       string
	}{
		ConfigName:   configName,
		Resource:     string(jsonData),
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Prefix:       asrv.pathPrefix,
	})
}
