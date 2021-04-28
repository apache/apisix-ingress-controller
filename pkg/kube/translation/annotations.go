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
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation/annotations"
	apisix "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslateAnnotations(anno map[string]string) apisix.Plugins {
	plugins := make(apisix.Plugins)
	if cors := annotations.BuildCorsPlugin(anno); cors != nil {
		plugins["cors"] = cors
	}
	if ipRestriction := annotations.BuildIpRestrictionPlugin(anno); ipRestriction != nil {
		plugins["ip-restriction"] = ipRestriction
	}
	if trafficSplit := annotations.BuildTrafficSplitPlugin(anno); trafficSplit != nil {
		plugins["traffic-split"] = trafficSplit
	}
	return plugins
}
