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
package annotations

import (
	"strconv"
)

// TrafficPlugin is the traffic-split plugin.
type TrafficSplitPlugin struct {
	Header        string `json:"Header,omitempty"`
	HeaderValue   string `json:"header-value,omitempty"`
	HeaderPattern string `json:"header-pattern,omitempty"`
	Weight        int    `json:"weight,omitempty"`
}

var (
	_enableTrafficPlugin        = "k8s.apisix.apache.org/canary"
	_trafficPluginMatchHeader   = "k8s.apisix.apache.org/canary-by-header"
	_trafficPluginMatchHeaderEq = "k8s.apisix.apache.org/canary-by-header-value"
	_trafficPluginMatchHaderReg = "k8s.apisix.apache.org/canary-by-header-pattern"
	// TODO: support canary by cookie
	// _                    = "k8s.apisix.apache.org/canary-by-cookie"
	_trafficPluginWeight = "k8s.apisix.apache.org/canary-weight"
)

// BuildTrafficSplitPlugin build the traffic-split plugin config body.
func BuildTrafficSplitPlugin(annotations map[string]string) *TrafficSplitPlugin {
	enable, ok := annotations[_enableTrafficPlugin]
	if !ok || enable == "false" {
		return nil
	}

	var tsp TrafficSplitPlugin
	if mh, ok := annotations[_trafficPluginMatchHeader]; ok {
		tsp.Header = mh
	}
	if mhe, ok := annotations[_trafficPluginMatchHeaderEq]; ok {
		tsp.HeaderValue = mhe
	}
	if mhr, ok := annotations[_trafficPluginMatchHaderReg]; ok {
		tsp.HeaderPattern = mhr
	}
	if weightStr, ok := annotations[_trafficPluginWeight]; ok {
		if weight, err := strconv.Atoi(weightStr); err == nil {
			tsp.Weight = weight
		}
	}
	return &tsp
}
