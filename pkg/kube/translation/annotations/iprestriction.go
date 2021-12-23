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
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_allowlistSourceRange = AnnotationsPrefix + "allowlist-source-range"
	_blocklistSourceRange = AnnotationsPrefix + "blocklist-source-range"
)

type ipRestriction struct{}

// NewIPRestrictionHandler creates a handler to convert
// annotations about client ips control to APISIX ip-restrict plugin.
func NewIPRestrictionHandler() Handler {
	return &ipRestriction{}
}

func (i *ipRestriction) PluginName() string {
	return "ip-restriction"
}

func (i *ipRestriction) Handle(e Extractor) (interface{}, error) {
	var plugin apisixv1.IPRestrictConfig
	allowlist := e.GetStringsAnnotation(_allowlistSourceRange)
	blocklist := e.GetStringsAnnotation(_blocklistSourceRange)
	if allowlist != nil || blocklist != nil {
		plugin.Allowlist = allowlist
		plugin.Blocklist = blocklist
		return &plugin, nil
	}
	return nil, nil
}
