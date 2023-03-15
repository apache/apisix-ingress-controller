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

package validation

import (
	"github.com/hashicorp/go-multierror"

	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
)

// ApisixRouteValidator validates ApisixRoute and its plugins.
// When the validation of one plugin fails, it will continue to validate the rest of plugins.
func ValidateApisixRouteV2(ar *v2.ApisixRoute) (valid bool, resultErr error) {
	valid, resultErr = ValidateApisixRouteHTTPV2(ar.Spec.HTTP)
	return
}

func ValidateApisixRouteHTTPV2(httpRouteList []v2.ApisixRouteHTTP) (valid bool, resultErr error) {
	valid = true
	for _, http := range httpRouteList {
		if _, err := ValidateApisixRoutePlugins(http.Plugins); err != nil {
			valid = false
			resultErr = multierror.Append(resultErr, err)
		}
	}
	return
}
