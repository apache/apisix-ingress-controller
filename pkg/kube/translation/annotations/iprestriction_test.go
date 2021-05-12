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
	"testing"

	"github.com/stretchr/testify/assert"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestIPRestrictionHandler(t *testing.T) {
	annotations := map[string]string{
		_allowlistSourceRange: "10.2.2.2,192.168.0.0/16",
	}
	p := NewIPRestrictionHandler()
	out, err := p.Handle(NewExtractor(annotations))
	assert.Nil(t, err, "checking given error")
	config := out.(*apisixv1.IPRestrictConfig)
	assert.Len(t, config.Whitelist, 2, "checking size of white list")
	assert.Equal(t, config.Whitelist[0], "10.2.2.2")
	assert.Equal(t, config.Whitelist[1], "192.168.0.0/16")

	assert.Equal(t, p.PluginName(), "ip-restriction")
}
