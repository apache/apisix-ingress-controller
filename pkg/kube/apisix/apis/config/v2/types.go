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
package v2

import (
	"encoding/json"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// ApisixRoutePlugin represents an APISIX plugin.
type ApisixPlugin struct {
	// The plugin name.
	Name string `json:"name" yaml:"name"`
	// Whether this plugin is in use, default is true.
	Enable bool `json:"enable" yaml:"enable"`
	// Plugin configuration.
	//+kubebuilder:validation:Type=object
	Config apiextensionsv1.JSON `json:"config" yaml:"config"`
}

type Plugin struct {
	Name      string    `json:"name,omitempty" yaml:"name,omitempty"`
	Enable    bool      `json:"enable,omitempty" yaml:"enable,omitempty"`
	Config    Config    `json:"config,omitempty" yaml:"config,omitempty"`
	ConfigSet ConfigSet `json:"config_set,omitempty" yaml:"config_set,omitempty"`
}

type ConfigSet []interface{}

func (p ConfigSet) DeepCopyInto(out *ConfigSet) {
	b, _ := json.Marshal(&p)
	_ = json.Unmarshal(b, out)
}

func (p *ConfigSet) DeepCopy() *ConfigSet {
	if p == nil {
		return nil
	}
	out := new(ConfigSet)
	p.DeepCopyInto(out)
	return out
}

type Config map[string]interface{}

func (p Config) DeepCopyInto(out *Config) {
	b, _ := json.Marshal(&p)
	_ = json.Unmarshal(b, out)
}

func (p *Config) DeepCopy() *Config {
	if p == nil {
		return nil
	}
	out := new(Config)
	p.DeepCopyInto(out)
	return out
}
