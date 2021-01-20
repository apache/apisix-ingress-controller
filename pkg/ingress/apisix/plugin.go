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
package apisix

import (
	"strconv"

	"github.com/api7/ingress-controller/pkg/seven/apisix"
)

type CorsYaml struct {
	Enable       bool   `json:"enable,omitempty"`
	AllowOrigin  string `json:"allow_origin,omitempty"`
	AllowHeaders string `json:"allow_headers,omitempty"`
	AllowMethods string `json:"allow_methods,omitempty"`
}

func (c *CorsYaml) SetEnable(enable string) {
	if b, err := strconv.ParseBool(enable); err != nil {
		c.Enable = false
	} else {
		c.Enable = b
	}
}

func (c *CorsYaml) SetOrigin(origin string) {
	c.AllowOrigin = origin
}

func (c *CorsYaml) SetHeaders(headers string) {
	c.AllowHeaders = headers
}
func (c *CorsYaml) SetMethods(methods string) {
	c.AllowMethods = methods
}

func (c *CorsYaml) Build() *apisix.Cors {
	maxAge := int64(3600)
	return apisix.BuildCors(c.Enable, &c.AllowOrigin, &c.AllowHeaders, &c.AllowMethods, &maxAge)
}
