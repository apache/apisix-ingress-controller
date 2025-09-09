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
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/cache"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

type ConfigListData struct {
	ConfigNames []string
}

type ResourceListData struct {
	ConfigName   string
	ResourceType string
	Resources    []ResourceInfo
}

type ResourceInfo struct {
	ID   string
	Name string
	Type string
	Link string
}

type ResourceDetailData struct {
	ConfigName   string
	Resource     interface{}
	ResourceID   string
	ResourceType string
}

type ADCDebugServer struct {
	store         *cache.Store
	handler       http.Handler
	configManager *ConfigManager[types.NamespacedNameKind, adctypes.Config]
	port          int
}

func (asrv *ADCDebugServer) setupHandler() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", asrv.handleIndex)
	mux.HandleFunc("/config", asrv.handleConfig)
	asrv.handler = mux
}

func NewADCDebugServer(store *cache.Store, configManager *ConfigManager[types.NamespacedNameKind, adctypes.Config], port int) *ADCDebugServer {
	srv := &ADCDebugServer{
		store:         store,
		configManager: configManager,
		port:          port,
	}
	srv.setupHandler()
	return srv
}

func (asrv *ADCDebugServer) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", asrv.port),
		Handler: asrv.handler,
	}

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}

func (asrv *ADCDebugServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	configs := asrv.configManager.List()
	configNames := make([]string, 0)
	for _, cfg := range configs {
		configNames = append(configNames, cfg.Name)
	}

	tmpl := template.New("index").Funcs(template.FuncMap{
		"urlencode": url.QueryEscape,
	})

	tmpl, err := tmpl.Parse(`
		<html>
		<head><title>ADC Debug Server</title></head>
		<body>
			<h1>Configurations</h1>
			<ul>
				{{range .ConfigNames}}
				<li><a href="/config?name={{. | urlencode}}">{{.}}</a></li>
				{{end}}
			</ul>
		</body>
		</html>
	`)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = tmpl.Execute(w, ConfigListData{ConfigNames: configNames})
}

func (asrv *ADCDebugServer) handleConfig(w http.ResponseWriter, r *http.Request) {
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

	resourceType := r.URL.Query().Get("type")
	resourceIDEncoded := r.URL.Query().Get("id")

	var resourceID string
	if resourceIDEncoded != "" {
		resourceID, err = url.QueryUnescape(resourceIDEncoded)
		if err != nil {
			http.Error(w, "Invalid resource ID encoding", http.StatusBadRequest)
			return
		}
	}

	if resourceType == "" {
		// Show resource types for this config
		asrv.showResourceTypes(w, configName, configNameEncoded)
		return
	}

	if resourceID == "" {
		asrv.showResources(w, r, configName, configNameEncoded, resourceType)
		return
	}

	asrv.showResourceDetail(w, r, configName, resourceType, resourceID)
}

func (asrv *ADCDebugServer) showResourceTypes(w http.ResponseWriter, configName, configNameEncoded string) {
	resourceTypes := []string{
		"services", "routes", "consumers", "ssls",
		"globalrules", "pluginmetadata",
	}

	tmpl := template.New("resources").Funcs(template.FuncMap{
		"urlencode": url.QueryEscape,
	})

	tmpl, err := tmpl.Parse(`
		<html>
		<head><title>Resources for {{.ConfigName}}</title></head>
		<body>
			<h1>Resources for {{.ConfigName}}</h1>
			<ul>
				{{range .ResourceTypes}}
				<li><a href="/config?name={{$.ConfigNameEncoded}}&type={{.}}">{{.}}</a></li>
				{{end}}
			</ul>
		</body>
		</html>
	`)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		ConfigName        string
		ConfigNameEncoded string
		ResourceTypes     []string
	}{
		ConfigName:        configName,
		ConfigNameEncoded: configNameEncoded,
		ResourceTypes:     resourceTypes,
	}

	_ = tmpl.Execute(w, data)
}

func (asrv *ADCDebugServer) showResources(w http.ResponseWriter, r *http.Request, configName, configNameEncoded, resourceType string) {
	resources, err := asrv.store.GetResources(configName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resourceInfos []ResourceInfo

	switch resourceType {
	case "services":
		for _, svc := range resources.Services {
			resourceInfos = append(resourceInfos, ResourceInfo{
				ID:   svc.ID,
				Name: svc.Name,
				Type: resourceType,
				Link: fmt.Sprintf("/config?name=%s&type=%s&id=%s",
					configNameEncoded, resourceType, url.QueryEscape(svc.ID)),
			})
		}
	case "consumers":
		for _, consumer := range resources.Consumers {
			resourceInfos = append(resourceInfos, ResourceInfo{
				ID:   consumer.Username,
				Name: consumer.Username,
				Type: resourceType,
				Link: fmt.Sprintf("/config?name=%s&type=%s&id=%s",
					configNameEncoded, resourceType, url.QueryEscape(consumer.Username)),
			})
		}
	case "ssls":
		for _, ssl := range resources.SSLs {
			resourceInfos = append(resourceInfos, ResourceInfo{
				ID:   ssl.ID,
				Name: ssl.ID,
				Type: resourceType,
				Link: fmt.Sprintf("/config?name=%s&type=%s&id=%s",
					configNameEncoded, resourceType, url.QueryEscape(ssl.ID)),
			})
		}
	case "globalrules":
		for key := range resources.GlobalRules {
			resourceInfos = append(resourceInfos, ResourceInfo{
				ID:   key,
				Name: key,
				Type: resourceType,
				Link: fmt.Sprintf("/config?name=%s&type=%s&id=%s",
					configNameEncoded, resourceType, url.QueryEscape(key)),
			})
		}
	case "pluginmetadata":
		if resources.PluginMetadata != nil {
			resourceInfos = append(resourceInfos, ResourceInfo{
				ID:   "pluginmetadata",
				Name: "Plugin Metadata",
				Type: resourceType,
				Link: fmt.Sprintf("/config?name=%s&type=%s&id=%s",
					configNameEncoded, resourceType, "pluginmetadata"),
			})
		}
	case "routes":
		for _, svc := range resources.Services {
			for _, route := range svc.Routes {
				resourceInfos = append(resourceInfos, ResourceInfo{
					ID:   route.ID,
					Name: route.Name,
					Type: resourceType,
					Link: fmt.Sprintf("/config?name=%s&type=%s&id=%s",
						configNameEncoded, resourceType, url.QueryEscape(route.ID)),
				})
			}
		}
	default:
		http.NotFound(w, r)
		return
	}

	tmpl := template.Must(template.New("resourceList").Parse(`
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
	`))

	_ = tmpl.Execute(w, ResourceListData{
		ConfigName:   configName,
		ResourceType: resourceType,
		Resources:    resourceInfos,
	})
}

func (asrv *ADCDebugServer) showResourceDetail(w http.ResponseWriter, r *http.Request, configName, resourceType, resourceID string) {
	resources, err := asrv.store.GetResources(configName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resource interface{}

	switch resourceType {
	case "services":
		for _, svc := range resources.Services {
			if svc.ID == resourceID {
				resource = svc
				break
			}
		}
	case "consumers":
		for _, consumer := range resources.Consumers {
			if consumer.Username == resourceID {
				resource = consumer
				break
			}
		}
	case "ssls":
		for _, ssl := range resources.SSLs {
			if ssl.ID == resourceID {
				resource = ssl
				break
			}
		}
	case "globalrules":
		resource = resources.GlobalRules
	case "pluginmetadata":
		resource = resources.PluginMetadata
	case "routes":
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

	tmpl := template.Must(template.New("resourceDetail").Parse(`
		<html>
		<head><title>Resource Detail</title></head>
		<body>
			<h1>Resource Details: {{.ResourceType}}/{{.ResourceID}}</h1>
			<pre>{{.Resource}}</pre>
		</body>
		</html>
	`))

	_ = tmpl.Execute(w, ResourceDetailData{
		ConfigName:   configName,
		Resource:     string(jsonData),
		ResourceID:   resourceID,
		ResourceType: resourceType,
	})
}
