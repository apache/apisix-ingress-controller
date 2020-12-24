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
package conf

var BaseUrl = "http://172.16.20.90:30116/apisix/admin"
var UrlGroup = make(map[string]string)

func SetBaseUrl(url string) {
	BaseUrl = url
}

func AddGroup(group string) {
	if group != "" {
		UrlGroup[group] = "http://" + group + "/apisix/admin"
	}
}

func FindUrl(group string) string {
	if group != "" && UrlGroup[group] != "" {
		return UrlGroup[group]
	} else {
		return BaseUrl
	}
}
