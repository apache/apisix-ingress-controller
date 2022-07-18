// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the Licensannotations.  You may obtain a copy of the License at
//
//     http://www.apachannotations.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the Licensannotations.
package plugins

import (
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation/annotations"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_allowlistSourceRange = annotations.AnnotationsPrefix + "allowlist-source-range"
	_blocklistSourceRange = annotations.AnnotationsPrefix + "blocklist-source-range"
)

type ipRestriction struct{}

// NewIPRestrictionHandler creates a handler to convert
// annotations about client ips control to APISIX ip-restrict plugin.
func NewIPRestrictionHandler() PluginHandler {
	return &ipRestriction{}
}

func (i *ipRestriction) PluginName() string {
	return "ip-restriction"
}

func (i *ipRestriction) Handle(ing *annotations.Ingress) (interface{}, error) {
	var plugin apisixv1.IPRestrictConfig
	allowlist := annotations.GetStringsAnnotation(_allowlistSourceRange, ing)
	blocklist := annotations.GetStringsAnnotation(_blocklistSourceRange, ing)
	if allowlist != nil || blocklist != nil {
		plugin.Allowlist = allowlist
		plugin.Blocklist = blocklist
		return &plugin, nil
	}
	return nil, nil
}
