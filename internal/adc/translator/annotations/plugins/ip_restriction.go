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

package plugins

import (
	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

type ipRestriction struct{}

// NewIPRestrictionHandler creates a handler to convert
// annotations about client IP control to APISIX ip-restriction plugin.
func NewIPRestrictionHandler() PluginAnnotationsHandler {
	return &ipRestriction{}
}

func (i *ipRestriction) PluginName() string {
	return "ip-restriction"
}

func (i *ipRestriction) Handle(e annotations.Extractor) (any, error) {
	allowlist := e.GetStringsAnnotation(annotations.AnnotationsAllowlistSourceRange)
	blocklist := e.GetStringsAnnotation(annotations.AnnotationsBlocklistSourceRange)

	if allowlist == nil && blocklist == nil {
		return nil, nil
	}

	return &adctypes.IPRestrictConfig{
		Allowlist: allowlist,
		Blocklist: blocklist,
	}, nil
}
