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
package translation

import (
	"github.com/apache/apisix-ingress-controller/pkg/id"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) translatePluginConfig(namespace, name string, plugins apisixv1.Plugins) (*apisixv1.PluginConfig, error) {
	pc, err := t.TranslatePluginConfig(plugins)
	if err != nil {
		return nil, err
	}
	pc.Name = apisixv1.ComposePluginConfigName(namespace, name)
	pc.ID = id.GenID(pc.Name)
	return pc, nil
}

func (t *translator) TranslatePluginConfig(plugins apisixv1.Plugins) (*apisixv1.PluginConfig, error) {
	pc := apisixv1.NewDefaultPluginConfig()
	pc.Plugins = plugins
	return pc, nil
}
